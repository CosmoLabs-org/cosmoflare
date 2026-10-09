// Cosmoflare Ops cron alert engine (FEAT-052): the scheduled entry the cron
// trigger calls every 5 minutes. It evaluates the FEAT-049 zone/D1 alert
// rules (rules.ts) against live Cloudflare telemetry and pages every
// subscribed device through Web Push (webpush.ts). A separate integration
// step wires runScheduled into the worker's `scheduled` handler and the
// route into index.ts — this module only exports entry points.

import type { ZoneGroups } from "./summary";
import { summarizeD1, summarizeZones } from "./summary";
import { CONDITIONS, evaluate, pruneState, type D1Obs, type FireState, type Rule, type ZoneObs } from "./rules";
import { readSubs, removeSubscriptions } from "./subscriptions";
import { sendPushes } from "./webpush";

const GRAPHQL = "https://api.cloudflare.com/client/v4/graphql";
const API = "https://api.cloudflare.com/client/v4";

// KV keys. Rules and fire-state follow the agent brief; "zones" and
// "d1-list" cache the two REST name lists for 1h so the per-5min run costs
// only the analytics GraphQL.
export const RULES_KEY = "rules";
export const STATE_KEY = "fire-state";
const ZONES_CACHE_KEY = "zones";
const D1_CACHE_KEY = "d1-list";
const LIST_TTL_MS = 60 * 60 * 1000;

// OpsEnv extends the worker Env with everything the cron path needs: the KV
// namespace, the VAPID keypair and contact, and (unused here, required by the
// shared Env shape) the Access config and assets binding.
export interface OpsEnv {
  CF_ACCOUNT_ID: string;
  CF_API_TOKEN: string;
  VAPID_PUBLIC_KEY: string;
  VAPID_PRIVATE_KEY: string;
  VAPID_SUBJECT: string;
  OPS_KV: KVNamespace;
  ACCESS_TEAM_DOMAIN: string;
  ACCESS_AUD: string;
  ASSETS: Fetcher;
}

export interface RunResult {
  fired: number;
  sent: number;
  pruned: number;
  /** Telemetry gaps that fired or would have fired, "dataset: error". */
  gaps: string[];
  /** Non-prune delivery problems (status + push-service reason, transport errors). */
  issues: string[];
}

// Starter rules when KV key "rules" is absent (operator decision O10): the
// same three conditions the CLI watch seeds.
export function starterRules(): Rule[] {
  return [
    { name: "uncached requests", condition: "zone-uncached-requests", threshold: 10000, enabled: true },
    { name: "cache miss %", condition: "zone-cache-miss-pct", threshold: 50, enabled: true },
    { name: "d1 rows read", condition: "d1-rows-read", threshold: 1e9, enabled: true },
  ];
}

// getRules loads rules from KV, falling back to starters when absent.
export async function getRules(env: OpsEnv): Promise<Rule[]> {
  const stored = await env.OPS_KV.get<Rule[]>(RULES_KEY, "json");
  if (Array.isArray(stored) && stored.length > 0) return stored;
  return starterRules();
}

// loadState reads the cooldown memory; a missing key yields fresh state.
function loadState(env: OpsEnv): Promise<FireState> {
  return env.OPS_KV.get<FireState>(STATE_KEY, "json").then((s) => s ?? { lastFired: {}, lastValue: {}, fires: {} });
}

// kvCache caches a REST name list in KV for LIST_TTL_MS: {at, value} wrapper
// checked against wall time (KV has no read-time expiry).
async function kvCache<T>(env: OpsEnv, key: string, fetcher: () => Promise<T>): Promise<T> {
  const cached = await env.OPS_KV.get<{ at: number; value: T }>(key, "json");
  if (cached && typeof cached.at === "number" && Date.now() - cached.at < LIST_TTL_MS) return cached.value;
  const value = await fetcher();
  await env.OPS_KV.put(key, JSON.stringify({ at: Date.now(), value }));
  return value;
}

// restPages walks the v4 REST API's page/per_page pagination like
// summary.ts's rest(); returns every result row.
async function restPages<T>(token: string, path: string): Promise<T[]> {
  const out: T[] = [];
  for (let page = 1; page <= 20; page++) {
    const sep = path.includes("?") ? "&" : "?";
    const res = await fetch(`${API}${path}${sep}page=${page}&per_page=50`, { headers: { Authorization: `Bearer ${token}` } });
    const body = (await res.json()) as { success: boolean; result: T[]; errors?: { message: string }[]; result_info?: { total_pages?: number; total_count?: number } };
    if (!res.ok || !body.success) throw new Error(body.errors?.[0]?.message ?? `HTTP ${res.status}`);
    out.push(...body.result);
    const info = body.result_info ?? {};
    const pages = info.total_pages ?? (info.total_count !== undefined ? Math.ceil(info.total_count / 50) : undefined);
    if (body.result.length < 50 || (pages !== undefined && page >= pages)) break;
  }
  return out;
}

async function gql<T>(token: string, query: string, variables: Record<string, unknown>): Promise<T> {
  const res = await fetch(GRAPHQL, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    body: JSON.stringify({ query, variables }),
  });
  const body = (await res.json()) as { data?: T; errors?: { message: string }[] };
  if (!res.ok || body.errors?.length) throw new Error(body.errors?.[0]?.message ?? `GraphQL HTTP ${res.status}`);
  return body.data as T;
}

// ZONE_QUERY groups eyeball requests by cacheStatus per zone — the exact
// shape summary.ts uses, ⌈zones/10⌉ zones per query.
const ZONE_QUERY = `query($t:[String!],$s:Time!,$e:Time!){viewer{zones(filter:{zoneTag_in:$t}){zoneTag
  httpRequestsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e,requestSource:"eyeball"}){count dimensions{cacheStatus}}}}}`;

/**
 * runScheduled is the cron entry point: load rules (KV "rules", starters
 * when absent) → skip early when nothing is enabled (zero upstream calls) →
 * collect only the telemetry an enabled rule needs (zone list + D1 list are
 * KV-cached 1h; the analytics GraphQL runs every cycle) → evaluate → push
 * each fire to every subscription → persist fire-state only when changed.
 */
export async function runScheduled(env: OpsEnv, now: Date = new Date()): Promise<RunResult> {
  const result: RunResult = { fired: 0, sent: 0, pruned: 0, gaps: [], issues: [] };

  const rules = await getRules(env);
  const enabled = rules.filter((r) => r.enabled);
  if (enabled.length === 0) return result; // stay offline: zero fetch calls

  const needZones = enabled.some((r) => CONDITIONS[r.condition]?.dataset === "zone");
  const needD1 = enabled.some((r) => CONDITIONS[r.condition]?.dataset === "d1");
  if (!needZones && !needD1) return result; // rules exist but need no telemetry we collect

  const state = await loadState(env);
  const stateBefore = JSON.stringify(state);
  const start = new Date(now.getTime() - 24 * 3600 * 1000).toISOString();
  const end = now.toISOString();

  const zones: ZoneObs[] = [];
  const d1: D1Obs[] = [];
  const gapMap: Record<string, string> = {};

  if (needZones) {
    try {
      const list = await kvCache<{ id: string; name: string }[]>(env, ZONES_CACHE_KEY, async () =>
        restPages<{ id: string; name: string; status: string }>(env.CF_API_TOKEN, `/zones?account.id=${env.CF_ACCOUNT_ID}&status=active`).then((rows) => rows.map(({ id, name }) => ({ id, name }))),
      );
      const names = Object.fromEntries(list.map((z) => [z.id, z.name]));
      const ids = list.map((z) => z.id);
      const batches: Promise<ZoneGroups[]>[] = [];
      for (let i = 0; i < ids.length; i += 10) {
        batches.push(
          gql<{ viewer: { zones: { zoneTag: string; httpRequestsAdaptiveGroups: { count: number; dimensions: { cacheStatus: string } }[] }[] } }>(
            env.CF_API_TOKEN, ZONE_QUERY, { t: ids.slice(i, i + 10), s: start, e: end },
          ).then((d) => d.viewer.zones.map((z) => ({ zoneTag: z.zoneTag, groups: z.httpRequestsAdaptiveGroups.map((g) => ({ count: g.count, cacheStatus: g.dimensions.cacheStatus })) }))),
        );
      }
      const data = (await Promise.all(batches)).flat();
      zones.push(...summarizeZones(data, names));
    } catch (err) {
      gapMap["zone"] = (err as Error).message;
    }
  }

  if (needD1) {
    // A failed D1 name list is NOT a gap: rows keep the database ID as their
    // name (Go design D8 fallback) — only the analytics query failing pages.
    const dbList = await kvCache<{ uuid: string; name: string }[]>(env, D1_CACHE_KEY, () =>
      restPages<{ uuid: string; name: string }>(env.CF_API_TOKEN, `/accounts/${env.CF_ACCOUNT_ID}/d1/database`),
    ).catch((err: Error) => {
      result.issues.push(`d1 name list failed (showing IDs): ${err.message}`);
      return [] as { uuid: string; name: string }[];
    });
    try {
      const data = await gql<{ viewer: { accounts: { g: { sum: { rowsRead: number; readQueries: number; rowsWritten: number }; dimensions: { databaseId: string } }[] }[] } }>(
        env.CF_API_TOKEN,
        `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){g: d1AnalyticsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){sum{rowsRead readQueries rowsWritten} dimensions{databaseId}}}}}`,
        { a: env.CF_ACCOUNT_ID, s: start, e: end },
      );
      const names = Object.fromEntries(dbList.map((d) => [d.uuid, d.name]));
      d1.push(...summarizeD1(
        (data.viewer.accounts[0]?.g ?? []).map((g) => ({ databaseId: g.dimensions.databaseId, rowsRead: g.sum.rowsRead, readQueries: g.sum.readQueries, rowsWritten: g.sum.rowsWritten })),
        names,
      ));
    } catch (err) {
      gapMap["d1"] = (err as Error).message;
    }
  }

  const fires = evaluate(enabled, zones, d1, gapMap, state, now.getTime());
  result.gaps = Object.keys(gapMap).sort().map((ds) => `${ds}: ${gapMap[ds]}`);

  // Bounded fire-state: drop cooldown entries older than 24h before the
  // changed-check, so a prune alone still persists the smaller blob.
  pruneState(state, now.getTime());

  const prunedEndpoints: string[] = [];
  if (fires.length > 0) {
    const subs = await readSubs(env);
    for (const fire of fires) {
      const payload = {
        id: fire.alertId,
        severity: "info",
        service: "cloudflare",
        title: fire.ruleName,
        detail: fire.message,
        fired_at: end,
      };
      const outcome = await sendPushes(env, subs, payload);
      result.sent += outcome.sent;
      result.pruned += outcome.pruned;
      prunedEndpoints.push(...outcome.prunedEndpoints);
      result.issues.push(...outcome.issues);
    }
    // Dead devices leave in ONE read-modify-write for the whole run.
    await removeSubscriptions(env, prunedEndpoints);
  }

  if (JSON.stringify(state) !== stateBefore) {
    await env.OPS_KV.put(STATE_KEY, JSON.stringify(state));
  }
  result.fired = fires.length;
  return result;
}

// testFire sends the same canned info payload as
// 'cosmoflare alerts watch --test-fire' (cmd/alerts_watch.go), so the
// operator can verify pairing from the Worker side too.
export async function testFire(env: OpsEnv): Promise<{ sent: number; pruned: number; issues: string[] }> {
  const subs = await readSubs(env);
  if (subs.length === 0) return { sent: 0, pruned: 0, issues: [] };
  const payload = {
    id: "test-fire",
    severity: "info",
    service: "cloudflare",
    title: "Cosmoflare pager test",
    detail: "Test fire from 'cosmoflare alerts watch --test-fire'. Seeing this on your pager means pairing works.",
    fired_at: new Date().toISOString(),
  };
  const outcome = await sendPushes(env, subs, payload);
  // A test fire also prunes dead endpoints the push service reports gone.
  await removeSubscriptions(env, outcome.prunedEndpoints);
  return { sent: outcome.sent, pruned: outcome.pruned, issues: outcome.issues };
}
