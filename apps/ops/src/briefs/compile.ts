// Brief compiler (P-15, FEAT-pC1EV32): the PURE verdict pass from ZoneFacts
// to a Brief. No env, no fetch, no Date.now — determinism is the contract,
// and every threshold below mirrors the CLI exactly (pkg/cosmoflare's
// efficiency checks; the HTML trap is the assessHTMLCacheTrap port). The
// template registry is injected (the endpoint passes it) so this module
// stays dependency-free; only the fired kinds' actions/verify are consumed.

import type { AlertFeedEntry } from "../scheduled";
import type { Brief, BriefAction, BriefFinding, TemplateMap, ZoneFacts } from "./types";

// low-cache-hit: the CLI's 7d hit-rate check.
const MIN_REQUESTS = 1000;
const HIT_RATE_FLOOR_PCT = 5;
const HIT_RATE_CRITICAL_PCT = 1;

// html-cache-trap: caching.go's minCacheableMaxAgeSec and edge-served set.
const MIN_CACHEABLE_MAX_AGE_SEC = 60;
const EDGE_CACHE_HITS = new Set(["HIT", "STALE", "REVALIDATED", "UPDATING"]);

// quota-pressure: warn at 80% used, escalate at 95%.
const QUOTA_WARN_PCT = 80;
const QUOTA_CRIT_PCT = 95;

const SEVERITY_RANK = { critical: 0, warning: 1, info: 2 } as const;

// Two-decimal fixed rendering: deterministic, and long floats from the
// 100*x/y division never leak into evidence strings.
function fmtNum(n: number): string {
  return String(Math.round(n * 100) / 100);
}

function fmtWithUnit(value: number, unit: string): string {
  if (unit === "%") return `${fmtNum(value)}%`;
  return unit ? `${fmtNum(value)} ${unit}` : fmtNum(value);
}

/** cache-control "cacheable attempt" parse — the assessHTMLCacheTrap port:
 *  needs an explicit public directive, then s-maxage anywhere, or a
 *  max-age>=60 part; all comparisons lowercase (Go lowercases too). */
function isCacheableCacheControl(cacheControl: string): boolean {
  const cc = cacheControl.toLowerCase();
  if (!cc.includes("public")) return false;
  if (cc.includes("s-maxage")) return true;
  for (const part of cc.split(",")) {
    const t = part.trim();
    if (t.startsWith("max-age=")) {
      const secs = Number.parseInt(t.slice("max-age=".length), 10);
      if (!Number.isNaN(secs) && secs >= MIN_CACHEABLE_MAX_AGE_SEC) return true;
    }
  }
  return false;
}

function isCacheableHTML(probe: ZoneFacts["probe"]): boolean {
  return !!probe && probe.contentType.toLowerCase().includes("text/html") && isCacheableCacheControl(probe.cacheControl);
}

/** The html-cache-trap verdict (assessHTMLCacheTrap port): cacheable HTML the
 *  edge did NOT serve from cache. cacheRuleCount === 0 escalates to critical
 *  (nothing can fix the headers at the edge); undefined — rules not visible —
 *  stays warning and says so, never fabricating a zero. */
function assessHTMLTrap(facts: ZoneFacts): BriefFinding | null {
  const probe = facts.probe;
  if (!probe || !isCacheableHTML(probe)) return null;
  if (EDGE_CACHE_HITS.has(probe.cfCacheStatus)) return null; // the edge IS caching it

  const signals: string[] = [`origin "${probe.cacheControl}" on HTML`];
  signals.push(probe.cfCacheStatus === "" ? "no cf-cache-status header" : `cf-cache-status ${probe.cfCacheStatus}`);
  if (facts.cacheRuleCount === undefined) signals.push("zone Cache Rules not visible with this token");
  else if (facts.cacheRuleCount === 0) signals.push("no zone Cache Rule exists");
  else signals.push(`${facts.cacheRuleCount} zone Cache Rule(s) exist — check they cover HTML paths`);
  return {
    kind: "html-cache-trap",
    severity: facts.cacheRuleCount === 0 ? "critical" : "warning",
    evidence: signals.join("; "),
  };
}

/** One alert's evidence, from the feed entry's own fields (value/threshold/
 *  unit + the ratio base when present) — a percentage never travels alone. */
function alertEvidence(a: AlertFeedEntry): string {
  const base = `${a.title}: ${fmtWithUnit(a.value, a.unit)} vs threshold ${fmtWithUnit(a.threshold, a.unit)} in 24h`;
  return a.context ? `${base} · ${a.context}` : base;
}

/**
 * Compile ZoneFacts into a Brief. Pure: identical facts in, identical Brief
 * out. Findings order: critical → warning → info, then kind alphabetical
 * (stable sort keeps same-kind entries — e.g. several alerts — in feed
 * order). Zero findings yields a Brief with empty arrays; the endpoint
 * decides to present that as brief: null.
 */
export function compileBrief(facts: ZoneFacts, templates: TemplateMap): Brief {
  const findings: BriefFinding[] = [];

  // 1. low-cache-hit — meaningful volume with a broken hit rate.
  if (facts.coverage && facts.coverage.requests >= MIN_REQUESTS && facts.coverage.hitRatePct < HIT_RATE_FLOOR_PCT) {
    findings.push({
      kind: "low-cache-hit",
      severity: facts.coverage.hitRatePct < HIT_RATE_CRITICAL_PCT ? "critical" : "warning",
      evidence: `${facts.window} cache hit rate ${fmtNum(facts.coverage.hitRatePct)}% — ${fmtNum(facts.coverage.cached)} of ${fmtNum(facts.coverage.requests)} requests served from edge cache`,
    });
  }

  // 2. html-cache-trap (the assessHTMLCacheTrap verdict).
  const trap = assessHTMLTrap(facts);
  if (trap) findings.push(trap);

  // 3. cache-rule-gap — cacheable HTML the edge already serves, yet zero
  // zone Cache Rules exist (default cache behavior is doing the work).
  // Dedupe: when the trap fired, the gap is the same story — skip it.
  const gapProbe = facts.probe;
  if (!trap && gapProbe && isCacheableHTML(gapProbe) && facts.cacheRuleCount === 0) {
    findings.push({
      kind: "cache-rule-gap",
      severity: "info",
      evidence: `origin "${gapProbe.cacheControl}" on HTML; cf-cache-status ${gapProbe.cfCacheStatus || "absent"}; no zone Cache Rule exists`,
    });
  }

  // 4. alert-firing — one finding per currently firing alert.
  for (const a of facts.firingAlerts) {
    findings.push({ kind: "alert-firing", severity: "warning", evidence: alertEvidence(a) });
  }

  // 5. quota-pressure — per authorized family with a real allowance.
  if (facts.quota) {
    for (const f of facts.quota.families) {
      if (!(f.allowance > 0) || !f.authorized) continue;
      const usedPct = (100 * f.used) / f.allowance;
      if (usedPct >= QUOTA_WARN_PCT) {
        findings.push({
          kind: "quota-pressure",
          severity: usedPct >= QUOTA_CRIT_PCT ? "critical" : "warning",
          evidence: `${f.label}: ${f.used}/${f.allowance} used (${fmtNum(usedPct)}% of allowance)`,
        });
      }
    }
  }

  findings.sort((a, b) => SEVERITY_RANK[a.severity] - SEVERITY_RANK[b.severity] || (a.kind < b.kind ? -1 : a.kind > b.kind ? 1 : 0));

  // Actions/verify come from the fired kinds' templates, in findings order,
  // deduped (two fired alerts must not duplicate one directive).
  const actions: BriefAction[] = [];
  const verify: string[] = [];
  const seenAction = new Set<string>();
  const seenVerify = new Set<string>();
  for (const f of findings) {
    const tpl = templates[f.kind](facts);
    for (const a of tpl.actions) {
      const key = `${a.directive}\u0000${a.command ?? ""}`;
      if (!seenAction.has(key)) {
        seenAction.add(key);
        actions.push(a);
      }
    }
    for (const v of tpl.verify) {
      if (!seenVerify.has(v)) {
        seenVerify.add(v);
        verify.push(v);
      }
    }
  }

  return { scope: "zone", zone: facts.zone, zoneId: facts.zoneId, window: facts.window, findings, actions, verify };
}
