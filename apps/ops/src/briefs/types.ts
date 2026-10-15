// Brief types (P-15, FEAT-pC1EV32): the contract every later briefs task
// compiles against. A Brief is the deterministic advisory for one zone —
// same ZoneFacts in, byte-identical rendered brief out (no LLM, no clock:
// generatedAt lives in the API envelope, never inside these values).
//
// TemplateMap is declared here (not in B2's templates.ts) so compile.ts can
// take the registry as an injected argument without importing the registry
// module — the import lives in the endpoint (B3), which keeps the
// templates ↔ compile cycle from ever forming.

import type { AlertFeedEntry } from "../scheduled";

/** The v1 finding kinds. v2 candidates: 'worker' | 'project' | 'account' scopes. */
export type BriefKind = "low-cache-hit" | "html-cache-trap" | "cache-rule-gap" | "alert-firing" | "quota-pressure";

/** v1 is per-zone only. */
export type BriefScope = "zone";

/** v1 renders agent-brief text only; v2: 'slack' | 'ticket'. */
export type BriefFormat = "agent-brief";

export interface BriefFinding {
  kind: BriefKind;
  severity: "critical" | "warning" | "info";
  /** Every figure traces to a collector fact — real numbers only. */
  evidence: string;
}

export interface BriefAction {
  directive: string;
  /** Real CLI invocation only — templates must never invent flags. */
  command?: string;
}

export interface Brief {
  scope: BriefScope;
  zone: string;
  zoneId: string;
  /** Measurement window label, e.g. "7d" — mirrored into evidence lines. */
  window: string;
  /** Severity-ranked (critical → warning → info, kind alphabetical), stable for identical facts. */
  findings: BriefFinding[];
  /** Numbered at render time, not here. */
  actions: BriefAction[];
  /** Acceptance commands. */
  verify: string[];
}

/** One zone's measured facts — the only input the compiler is allowed to read. */
export interface ZoneFacts {
  zone: string;
  zoneId: string;
  window: string;
  /** httpRequests1dGroups 7d coverage (analytics.go ZoneCacheCoverage port). */
  coverage?: { requests: number; cached: number; hitRatePct: number };
  /** Origin probe headers from https://<zone>/ (caching.go ProbeResult port). */
  probe?: { contentType: string; cacheControl: string; cfCacheStatus: string };
  /** The zone's Cache Rules count; undefined when the rules were not visible
   *  (an authorized:false quota family never becomes a fabricated 0). */
  cacheRuleCount?: number;
  /** The REAL /api/alerts/feed entries scoped to this zone (live KV read, never cached). */
  firingAlerts: AlertFeedEntry[];
  /** RulesQuotaPayload.families, counters only (rules_quota.ts self-caches 5 min). */
  quota?: { families: { family: string; label: string; used: number; allowance: number; authorized: boolean }[] };
}

/**
 * The per-kind playbook registry: given the facts, one entry per BriefKind
 * returns that kind's actions and verify commands. The findings slot in the
 * return type is reserved for B2's self-describing templates; compileBrief
 * owns the verdict/threshold logic and orders findings itself.
 */
export type TemplateMap = Record<BriefKind, (f: ZoneFacts) => { findings: BriefFinding[]; actions: BriefAction[]; verify: string[] }>;
