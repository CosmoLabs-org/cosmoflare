// Zone-fact collectors (P-15, FEAT-pC1EV32): five independent reads fanned
// out with Promise.all into one ZoneFacts — the only input compileBrief may
// read. House rule for every collector: a best-effort section degrades to
// an omitted field (probe failure, alert-feed read failure), while the
// coverage analytics read is the brief's hard spine — its failure
// propagates so the endpoint can answer 502 instead of inventing a brief.
//
// Caching budget (plan Global Constraints): coverage and the origin probe
// go through cached() (1 h fresh / 24 h stale); the quota collector
// (rules_quota.ts) KV-caches 5 min per zone itself and is NOT double-wrapped;
// the alert feed is a live KV read every call — an advisory must see the
// alerts that are firing now.

import { cached } from "../cache";
import { collectRulesQuota } from "../rules_quota";
import { activeZones, ALERT_FEED_KEY, type AlertFeedEntry, type OpsEnv } from "../scheduled";

/** The brief's measurement window, in whole days (httpRequests1dGroups is
 *  date-granular — it rejects datetime filters, verified live 2026-10-14). */
const WINDOW_DAYS = 7;
export const BRIEF_WINDOW = "7d";

/** Coverage + probe TTLs: 1 h fresh, 24 h stale — advisory cadence, not dashboard cadence. */
const BRIEFS_TTL = { ttlSec: 3600, staleSec: 86_400 };

const PROBE_TIMEOUT_MS = 10_000;

// cached() refreshes stale entries through ctx.waitUntil; the optional ctx
// defaults to a no-op so tests and future callers can pass just (env, zone).
const NOOP_CTX: ExecutionContext = {
  waitUntil: () => undefined,
  passThroughOnException: () => undefined,
};

// Private GraphQL helper — the summary.ts gql() pattern (scheduled.ts keeps
// the same private copy; the helper is deliberately not shared).
async function gql<T>(token: string, query: string, variables: Record<string, unknown>): Promise<T> {
  const res = await fetch("https://api.cloudflare.com/client/v4/graphql", {
    method: "POST",
    headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    body: JSON.stringify({ query, variables }),
  });
  const body = (await res.json()) as { data?: T; errors?: { message: string }[] };
  if (!res.ok || body.errors?.length) throw new Error(body.errors?.[0]?.message ?? `GraphQL HTTP ${res.status}`);
  return body.data as T;
}

// One httpRequests1dGroups query summing requests + cachedRequests over the
// 7 whole UTC days ending today — the pkg/cosmoflare/analytics.go
// ZoneCacheCoverage port (shape verified live 2026-10-14). Whole-day
// convention: end date is exclusive, +1 day so today is fully covered.
const COVERAGE_QUERY = `query($zoneTag: String!, $start: Date!, $end: Date!) {
  viewer {
    zones(filter: {zoneTag: $zoneTag}) {
      httpRequests1dGroups(limit: 10000, filter: {date_geq: $start, date_lt: $end}) {
        sum { requests cachedRequests }
        dimensions { date }
      }
    }
  }
}`;

function dayUtc(d: Date): string {
  return d.toISOString().slice(0, 10);
}

async function loadCoverage(token: string, zoneId: string): Promise<ZoneFacts["coverage"]> {
  const end = new Date();
  const start = new Date(end.getTime() - WINDOW_DAYS * 86_400_000);
  const vars = { zoneTag: zoneId, start: dayUtc(start), end: dayUtc(new Date(end.getTime() + 86_400_000)) };
  const data = await gql<{ viewer: { zones: { httpRequests1dGroups: { sum: { requests: number; cachedRequests: number } }[] }[] } }>(
    token, COVERAGE_QUERY, vars,
  );
  let requests = 0;
  let cachedReqs = 0;
  // A zone absent from the response decodes as zeros, never an error (Go).
  for (const z of data.viewer.zones) {
    for (const g of z.httpRequests1dGroups) {
      requests += g.sum.requests;
      cachedReqs += g.sum.cachedRequests;
    }
  }
  return { requests, cached: cachedReqs, hitRatePct: requests > 0 ? (100 * cachedReqs) / requests : 0 };
}

// Plain fetch of the zone apex, headers only — caching.go's defaultProbe
// identity (same UA/Accept so an origin cannot serve the worker something
// different from what the CLI measured). Never throws: a dead origin is a
// missing probe field, and the null is cached so a downed origin is not
// re-probed on every advisory read.
async function probeOrigin(zoneName: string): Promise<ZoneFacts["probe"]> {
  try {
    const res = await fetch(`https://${zoneName}/`, {
      headers: { "User-Agent": "cosmoflare-efficiency/1.0", Accept: "text/html,application/xhtml+xml" },
      signal: AbortSignal.timeout(PROBE_TIMEOUT_MS),
    });
    const headers = {
      contentType: res.headers.get("content-type") ?? "",
      cacheControl: res.headers.get("cache-control") ?? "",
      cfCacheStatus: res.headers.get("cf-cache-status") ?? "",
    };
    // Headers only: release the body without buffering it.
    await res.body?.cancel().catch(() => undefined);
    return headers;
  } catch {
    return null;
  }
}

// Live KV read of the alert feed, filtered to the queried zone. ruleId is
// "<ruleName>/<zoneKey>" (rules.ts alertId) where zoneKey is the zone ID,
// falling back to the zone name when the engine had no ID — so both
// segments match. Never cached; a corrupt feed degrades to [].
async function readFiringAlerts(env: OpsEnv, zone: string, zoneId: string): Promise<AlertFeedEntry[]> {
  try {
    const feed = (await env.OPS_KV.get<AlertFeedEntry[]>(ALERT_FEED_KEY, "json")) ?? [];
    if (!Array.isArray(feed)) return [];
    return feed.filter((e) => {
      const seg = e.ruleId.slice(e.ruleId.lastIndexOf("/") + 1);
      return seg === zoneId || seg === zone;
    });
  } catch {
    return []; // a broken feed read must not fail the brief — alerts are additive
  }
}

/**
 * Collect one zone's advisory facts. The zone name is resolved against the
 * shared "cron:zones" KV-cached list (activeZones — the same list
 * rules_quota.ts reads, so no extra REST call); an unknown zone throws so
 * the endpoint can 404. Coverage is the one hard-failing read; probe,
 * quota, and alerts degrade inside the facts.
 */
export async function collectZoneFacts(env: OpsEnv, zone: string, ctx: ExecutionContext = NOOP_CTX): Promise<ZoneFacts> {
  const zones = await activeZones(env);
  const match = zones.find((z) => z.name === zone);
  if (!match) throw new Error(`zone ${zone} not found`);
  const zoneId = match.id;

  const [coverage, probe, quota, firingAlerts] = await Promise.all([
    cached(ctx, env, `briefs-coverage:${zoneId}`, BRIEFS_TTL, () => loadCoverage(env.CF_API_TOKEN, zoneId)),
    cached(ctx, env, `briefs-probe:${zoneId}`, BRIEFS_TTL, () => probeOrigin(zone)),
    collectRulesQuota(env, zone), // self-caches 5 min per zone in KV — never double-wrapped
    readFiringAlerts(env, zone, zoneId), // live, never cached
  ]);

  // Only an authorized family carries a real count: used=0 with
  // authorized:false means "token cannot read the rules", not "no rules"
  // (caching.go: invisible rules never mask probe evidence).
  const cacheFamily = quota.families.find((f) => f.family === "cache-rules");
  const cacheRuleCount = cacheFamily?.authorized ? cacheFamily.used : undefined;

  return {
    zone,
    zoneId,
    window: BRIEF_WINDOW,
    coverage: coverage.value,
    probe: probe.value ?? undefined,
    cacheRuleCount,
    firingAlerts,
    quota: {
      families: quota.families.map(({ family, label, used, allowance, authorized }) => ({ family, label, used, allowance, authorized })),
    },
  };
}
