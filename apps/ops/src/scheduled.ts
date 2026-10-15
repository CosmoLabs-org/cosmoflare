// Cosmoflare Ops cron alert engine (FEAT-052): the scheduled entry the cron
// trigger calls every 5 minutes. It evaluates the FEAT-049 zone/D1 alert
// rules (rules.ts) against live Cloudflare telemetry and pages every
// subscribed device through Web Push (webpush.ts). A separate integration
// step wires runScheduled into the worker's `scheduled` handler and the
// route into index.ts — this module only exports entry points.

import type { ZoneGroups } from "./summary";
import { summarizeD1, summarizeZones } from "./summary";
import { CONDITIONS, evaluate, formatValue, pruneState, type D1Obs, type Fire, type FireState, type KVObs, type Rule, type ZoneObs } from "./rules";
import { readSubs, removeSubscriptions } from "./subscriptions";
import { sendPushes } from "./webpush";
import { fetchRetry } from "./retry";

const GRAPHQL = "https://api.cloudflare.com/client/v4/graphql";
const API = "https://api.cloudflare.com/client/v4";

// KV keys. Rules and fire-state follow the agent brief; "cron:zones" and
// "cron:d1-list" cache the two REST name lists for 1h so the per-5min run
// costs only the analytics GraphQL. The "cron:" prefix keeps them clear of
// cache.ts's API-side keys (BUG-058): those expect {storedAt, value}, and a
// cron {at, value} write over a shared key once parsed as storedAt=undefined
// → NaN age → served fresh forever.
export const RULES_KEY = "rules";
export const STATE_KEY = "fire-state";
// ALERT_FEED_KEY holds the fired-alert history the pager's Notifications
// center reads through /api/alerts/feed: newest-first AlertFeedEntry[], kept
// capped at MAX_FEED so the KV blob cannot grow without bound.
export const ALERT_FEED_KEY = "alert-feed";
export const MAX_FEED = 100;
const ZONES_CACHE_KEY = "cron:zones";
const D1_CACHE_KEY = "cron:d1-list";
const KV_CACHE_KEY = "cron:kv-list";
// KV-side expiry matching the 1h LIST_TTL_MS check above (written as 24h so
// the wall-time check inside kvCache stays the authority on freshness).
const KV_LIST_TTL_SEC = 86400;
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

// AlertFeedEntry is one fired-alert record in the KV "alert-feed" history and
// the wire shape of /api/alerts/feed — the pager's Notifications center
// renders these verbatim (pager/src/alerts.ts mirrors the type).
export interface AlertFeedEntry {
  ts: string; // fire time, ISO
  ruleId: string; // stable alert ID ("rule/scope" or "telemetry-gap/dataset")
  condition: string; // rules.ts condition id, or "telemetry-gap"
  title: string; // rule name
  value: number; // observed value at fire time
  threshold: number; // rule threshold
  unit: string; // "%" | "requests" | "rows" | "writes" ("" for gaps)
  message: string; // the page detail line
  sent: boolean; // whether the push reached at least one device
  context: string; // absolute base for ratio fires ("46 misses of 139 eligible requests"); "" for counts
}

// toFeedEntry projects one fire into the feed shape; sent starts false and is
// flipped after a delivery that reached at least one device.
function toFeedEntry(fire: Fire, ts: string, sent: boolean): AlertFeedEntry {
  return {
    ts,
    ruleId: fire.alertId,
    condition: fire.condition,
    title: fire.ruleName,
    value: fire.value,
    threshold: fire.threshold,
    unit: fire.unit,
    message: fire.message,
    sent,
    context: fire.context,
  };
}

// pagePayload composes THE push for a run (operator 2026-10-12: one
// notification per run, human strings). A single fire pages under its own
// headline with a value/threshold body; multiple fires page as a digest —
// title "N alerts fired", body listing up to DIGEST_LINES headlines with
// their numbers, then a "+N more" tail. Severity escalates to warning when
// any fire escalated (2× re-page) so the phone can weight it.
const DIGEST_LINES = 3;

function fireBody(fire: Fire): string {
  if (fire.condition === "telemetry-gap") {
    return `Cloudflare ${fire.dataset} analytics were unavailable this cycle, so ${fire.dataset} rules could not run.`;
  }
  // % reads as "63.6% — threshold 50.0%"; counts keep their unit word.
  const base = fire.unit === "%"
    ? `${formatValue(fire.value, fire.unit)}% in 24h — threshold ${formatValue(fire.threshold, fire.unit)}%`
    : `${formatValue(fire.value, fire.unit)} ${fire.unit} in 24h — threshold ${formatValue(fire.threshold, fire.unit)}`;
  // A ratio never travels without its base (operator 2026-10-12).
  return fire.context ? `${base} · ${fire.context}` : base;
}

function pagePayload(fires: Fire[], end: string): { id: string; severity: string; service: string; title: string; detail: string; fired_at: string } {
  if (fires.length === 1) {
    const fire = fires[0];
    return { id: fire.alertId, severity: fire.severity, service: "cloudflare", title: fire.title, detail: fireBody(fire), fired_at: end };
  }
  const lines = fires.slice(0, DIGEST_LINES).map((fire) => `• ${fire.title} — ${fireBody(fire)}`);
  const more = fires.length - DIGEST_LINES;
  if (more > 0) lines.push(`+${more} more`);
  const severity = fires.some((fire) => fire.severity === "warning") ? "warning" : "info";
  return { id: `alerts/${end}`, severity, service: "cloudflare", title: `${fires.length} alerts fired`, detail: lines.join("\n"), fired_at: end };
}

// appendFeed prepends the new entries to KV "alert-feed" (newest first) and
// caps the list at MAX_FEED. Best effort: a feed failure becomes an issue,
// never a cron crash.
async function appendFeed(env: OpsEnv, entries: AlertFeedEntry[], issues: string[]): Promise<void> {
  if (entries.length === 0) return;
  try {
    const existing = (await env.OPS_KV.get<AlertFeedEntry[]>(ALERT_FEED_KEY, "json")) ?? [];
    const feed = [...entries, ...existing].slice(0, MAX_FEED);
    await env.OPS_KV.put(ALERT_FEED_KEY, JSON.stringify(feed));
  } catch (err) {
    issues.push(`alert-feed write failed: ${(err as Error).message}`);
  }
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
  await env.OPS_KV.put(key, JSON.stringify({ at: Date.now(), value }), { expirationTtl: KV_LIST_TTL_SEC });
  return value;
}

// activeZones is the ONE writer of the shared "cron:zones" key (BUG-pP1YH9W):
// the paginated (restPages 20×50) active-only zone list through kvCache. Both
// the cron engine and rules_quota's rotation consume this exact list, so the
// key can never flip between different semantics (20-zone unpaginated vs the
// full active fleet).
export async function activeZones(env: OpsEnv): Promise<{ id: string; name: string }[]> {
  return kvCache<{ id: string; name: string }[]>(env, ZONES_CACHE_KEY, async () =>
    restPages<{ id: string; name: string; status: string }>(env.CF_API_TOKEN, `/zones?account.id=${env.CF_ACCOUNT_ID}&status=active`).then((rows) =>
      rows.map(({ id, name }) => ({ id, name })),
    ),
  );
}

// restPages walks the v4 REST API's page/per_page pagination like
// summary.ts's rest(); returns every result row.
async function restPages<T>(token: string, path: string): Promise<T[]> {
  const out: T[] = [];
  for (let page = 1; page <= 20; page++) {
    const sep = path.includes("?") ? "&" : "?";
    const res = await fetchRetry(`${API}${path}${sep}page=${page}&per_page=50`, { headers: { Authorization: `Bearer ${token}` } });
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
  const res = await fetchRetry(GRAPHQL, {
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

  // No paired device: nothing can be delivered, so skip every upstream call
  // (telemetry included) and return before the code path that used to crash
  // on unset VAPID vars.
  const subs = await readSubs(env);
  if (subs.length === 0) return result;

  const needZones = enabled.some((r) => CONDITIONS[r.condition]?.dataset === "zone");
  const needD1 = enabled.some((r) => CONDITIONS[r.condition]?.dataset === "d1");
  const needKV = enabled.some((r) => CONDITIONS[r.condition]?.dataset === "kv");
  if (!needZones && !needD1 && !needKV) return result; // rules exist but need no telemetry we collect

  const state = await loadState(env);
  const stateBefore = JSON.stringify(state);
  const start = new Date(now.getTime() - 24 * 3600 * 1000).toISOString();
  const end = now.toISOString();

  const zones: ZoneObs[] = [];
  const d1: D1Obs[] = [];
  const kv: KVObs[] = [];
  const gapMap: Record<string, string> = {};

  if (needZones) {
    try {
      const list = await activeZones(env);
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

  if (needKV) {
    // KV writes/day (FB-29): namespace names KV-cached 1h like the other
    // lists (a failed name list is NOT a gap — rows keep the namespace ID),
    // then kvOperationsAdaptiveGroups over the 24h window; only actionType
    // "write" rows count (the dataset also carries "read"/"delete" — deletes
    // are unmetered upstream, reads belong to a future condition).
    const nsList = await kvCache<{ id: string; title: string }[]>(env, KV_CACHE_KEY, () =>
      restPages<{ id: string; title: string }>(env.CF_API_TOKEN, `/accounts/${env.CF_ACCOUNT_ID}/storage/kv/namespaces`),
    ).catch((err: Error) => {
      result.issues.push(`kv name list failed (showing IDs): ${err.message}`);
      return [] as { id: string; title: string }[];
    });
    try {
      const data = await gql<{ viewer: { accounts: { g: { sum: { requests: number }; dimensions: { namespaceId: string; actionType: string } }[] }[] } }>(
        env.CF_API_TOKEN,
        `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){g: kvOperationsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){sum{requests} dimensions{namespaceId actionType}}}}}`,
        { a: env.CF_ACCOUNT_ID, s: start, e: end },
      );
      const names = Object.fromEntries(nsList.map((n) => [n.id, n.title]));
      const writes = new Map<string, number>();
      for (const row of data.viewer.accounts[0]?.g ?? []) {
        if (row.dimensions.actionType !== "write") continue;
        writes.set(row.dimensions.namespaceId, (writes.get(row.dimensions.namespaceId) ?? 0) + row.sum.requests);
      }
      for (const [id, count] of writes) {
        kv.push({ name: names[id] ?? id, id, writes: count });
      }
    } catch (err) {
      gapMap["kv"] = (err as Error).message;
    }
  }

  const fires = evaluate(enabled, zones, d1, gapMap, state, now.getTime(), kv);
  result.gaps = Object.keys(gapMap).sort().map((ds) => `${ds}: ${gapMap[ds]}`);

  // Bounded fire-state: drop cooldown entries older than 24h before the
  // changed-check, so a prune alone still persists the smaller blob.
  pruneState(state, now.getTime());

  const prunedEndpoints: string[] = [];
  if (fires.length > 0) {
    // Every fire lands in the alert-feed history, delivered or not: entries
    // start sent:false and flip once their push reached at least one device;
    // a delivery crash below leaves the entries honestly marked
    // "not delivered" instead of dropping them from the history.
    const feedEntries = fires.map((fire) => toFeedEntry(fire, end, false));
    // Delivery problems must never throw out of runScheduled (the cron
    // crashed here when VAPID vars were unset, skipping the fire-state write
    // below and re-evaluating the same fires every run): a push error becomes
    // an issue and the state write still happens.
    try {
      // ONE push per run (operator 2026-10-12: N simultaneous fires arrived
      // as N separate notifications): a single fire pages under its own
      // headline; multiple fires page as one digest listing them. The feed
      // above keeps per-fire granularity — the push is the poke, the feed is
      // the history.
      const payload = pagePayload(fires, end);
      const outcome = await sendPushes(env, subs, payload);
      result.sent += outcome.sent;
      result.pruned += outcome.pruned;
      prunedEndpoints.push(...outcome.prunedEndpoints);
      result.issues.push(...outcome.issues);
      const delivered = outcome.sent > 0;
      for (const entry of feedEntries) entry.sent = delivered;
      // Dead devices leave in ONE read-modify-write for the whole run.
      await removeSubscriptions(env, prunedEndpoints);
    } catch (err) {
      result.issues.push(`push delivery failed: ${(err as Error).message}`);
    }
    await appendFeed(env, feedEntries, result.issues);
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
    // Same shape a real alert now uses (operator 2026-10-12): headline +
    // plain-language body, so the test shows exactly what a breach will
    // look like on the phone.
    title: "Test alert · Cosmoflare",
    detail: "Pairing works. Real alerts look exactly like this — a headline naming the condition, then the numbers.",
    fired_at: new Date().toISOString(),
  };
  const outcome = await sendPushes(env, subs, payload);
  // A test fire also prunes dead endpoints the push service reports gone.
  await removeSubscriptions(env, outcome.prunedEndpoints);
  return { sent: outcome.sent, pruned: outcome.pruned, issues: outcome.issues };
}
