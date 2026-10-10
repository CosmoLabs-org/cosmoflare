import { fetchRetry } from "./retry";
// Cosmoflare Ops summary (FEAT-052): the same telemetry the CLI's
// FEAT-048/049 alerts read — zone cache status, D1 rows read, monthly
// pacing — fetched live from the Cloudflare GraphQL Analytics API by the
// Worker. Pure transforms are exported for tests; the token never leaves
// the Worker.

const GRAPHQL = "https://api.cloudflare.com/client/v4/graphql";
const API = "https://api.cloudflare.com/client/v4";

// Cache status sets — mirror pkg/cosmoflare/analytics_cache.go (design D3-D5).
const ELIGIBLE = new Set(["hit", "miss", "expired", "revalidated", "updating", "stale"]);
const MISSED = new Set(["miss", "expired"]);
const UNCACHED = new Set(["dynamic", "bypass"]);
export const ZONE_CACHE_FLOOR = 100;

// Monthly allowances (Workers Paid), from the verified limits catalog
// (pkg/cosmoflare/limitsdata/catalog.json: workers.requests_monthly,
// d1.rows_read_monthly).
const WORKERS_REQUESTS_MONTHLY = 10_000_000;
const D1_ROWS_READ_MONTHLY = 25_000_000_000;

export interface ZoneGroups {
  zoneTag: string;
  groups: { count: number; cacheStatus: string }[];
}
export interface ZoneRow {
  zone: string;
  total: number;
  uncached: number;
  eligible: number;
  missPct: number | null; // null under the 100-eligible floor
  byStatus: Record<string, number>; // every cacheStatus count, v2
}
export interface D1Group {
  databaseId: string;
  rowsRead: number;
  readQueries: number;
  rowsWritten: number; // v2
}
export interface D1Row {
  name: string;
  databaseId: string; // v2
  rowsRead: number;
  readQueries: number;
  rowsWritten: number; // v2
  rowsPerQuery: number;
}
// 24h per-script row (v2). CPU quantiles are stored in ms: the GraphQL
// quantiles are microseconds ("CPU time 99th percentile - microseconds",
// live schema description 2026-10-09; Cloudflare's own exporter also treats
// them as µs) and a real query on 2026-10-09 returned e.g. cpuTimeP50=1108.
export interface WorkerRow {
  script: string;
  requests: number;
  errors: number;
  errorPct: number;
  cpuP50Ms: number | null;
  cpuP99Ms: number | null;
}
export interface UsageRow {
  id: string;
  name: string;
  used: number;
  limit: number;
  pct: number;
  projectedPct: number | null;
}
export interface Summary {
  generatedAt: string;
  windowHours: number;
  usage: UsageRow[];
  d1: D1Row[];
  zones: ZoneRow[];
  errors: string[];
  workers: WorkerRow[]; // v2, sorted by requests desc
  cache?: { ageSec: number; stale: boolean }; // filled by the route from cached()
}

export function summarizeZones(data: ZoneGroups[], names: Record<string, string>): ZoneRow[] {
  const byId = new Map<string, Map<string, number>>();
  for (const z of data) {
    const m = byId.get(z.zoneTag) ?? new Map<string, number>();
    for (const g of z.groups) m.set(g.cacheStatus, (m.get(g.cacheStatus) ?? 0) + g.count);
    byId.set(z.zoneTag, m);
  }
  const rows: ZoneRow[] = Object.entries(names).map(([id, zone]) => {
    const m = byId.get(id) ?? new Map<string, number>();
    let total = 0, uncached = 0, eligible = 0, missed = 0;
    for (const [status, n] of m) {
      total += n;
      if (UNCACHED.has(status)) uncached += n;
      if (ELIGIBLE.has(status)) eligible += n;
      if (MISSED.has(status)) missed += n;
    }
    const missPct = eligible >= ZONE_CACHE_FLOOR ? (100 * missed) / eligible : null;
    const byStatus: Record<string, number> = {};
    for (const [status, n] of m) byStatus[status] = n;
    return { zone, total, uncached, eligible, missPct, byStatus };
  });
  return rows.sort((a, b) => b.uncached - a.uncached || a.zone.localeCompare(b.zone));
}

export function summarizeD1(groups: D1Group[], names: Record<string, string>): D1Row[] {
  const byId = new Map<string, { rowsRead: number; readQueries: number; rowsWritten: number }>();
  for (const g of groups) {
    const acc = byId.get(g.databaseId) ?? { rowsRead: 0, readQueries: 0, rowsWritten: 0 };
    acc.rowsRead += g.rowsRead;
    acc.readQueries += g.readQueries;
    acc.rowsWritten += g.rowsWritten;
    byId.set(g.databaseId, acc);
  }
  return [...byId.entries()]
    .map(([id, a]) => ({
      name: names[id] || id,
      databaseId: id,
      rowsRead: a.rowsRead,
      readQueries: a.readQueries,
      rowsWritten: a.rowsWritten,
      rowsPerQuery: a.readQueries > 0 ? Math.round(a.rowsRead / a.readQueries) : 0,
    }))
    .sort((a, b) => b.rowsRead - a.rowsRead || a.name.localeCompare(b.name));
}

/** Per-script 24h rows: requests, errors, error % and CPU quantiles in ms.
 *  Sorted by requests desc (v2 contract). CPU quantiles arrive as
 *  microseconds (live schema description, 2026-10-09) — divided by 1000. */
export function summarizeWorkers(
  rows: { scriptName: string; requests: number; errors: number; cpuP50Us: number | null; cpuP99Us: number | null }[],
): WorkerRow[] {
  return rows
    .map((r) => ({
      script: r.scriptName,
      requests: r.requests,
      errors: r.errors,
      errorPct: r.requests > 0 ? (100 * r.errors) / r.requests : 0,
      cpuP50Ms: r.cpuP50Us === null ? null : r.cpuP50Us / 1000,
      cpuP99Ms: r.cpuP99Us === null ? null : r.cpuP99Us / 1000,
    }))
    .sort((a, b) => b.requests - a.requests || a.script.localeCompare(b.script));
}

/** Percent used and linear projection to the end of the UTC calendar month. */
export function monthPacing(used: number, limit: number, now: Date): { pct: number; projectedPct: number | null } {
  const start = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1);
  const end = Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1);
  const pct = limit > 0 ? (100 * used) / limit : 0;
  const elapsed = now.getTime() - start;
  return { pct, projectedPct: elapsed > 0 && limit > 0 ? (pct * (end - start)) / elapsed : null };
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

export async function rest<T>(token: string, path: string): Promise<T[]> {
  const out: T[] = [];
  for (let page = 1; page <= 20; page++) {
    const sep = path.includes("?") ? "&" : "?";
    const res = await fetchRetry(`${API}${path}${sep}page=${page}&per_page=50`, { headers: { Authorization: `Bearer ${token}` } });
    const body = (await res.json()) as { success: boolean; result: T[]; errors?: { message: string }[]; result_info?: { total_pages?: number; total_count?: number } };
    if (!body.success) throw new Error(body.errors?.[0]?.message ?? `HTTP ${res.status}`);
    out.push(...body.result);
    const info = body.result_info ?? {};
    const pages = info.total_pages ?? (info.total_count !== undefined ? Math.ceil(info.total_count / 50) : undefined);
    if (body.result.length < 50 || (pages !== undefined && page >= pages)) break;
  }
  return out;
}

// REST list loaders, exported so index.ts can wrap them in `cached` (the
// 1 h / 24 h list TTLs) and pass them back through SummaryDeps.
export async function fetchZonesList(token: string, accountId: string): Promise<{ id: string; name: string; status: string }[]> {
  return rest(token, `/zones?account.id=${accountId}&status=active`);
}

export async function fetchD1List(token: string, accountId: string): Promise<{ uuid: string; name: string }[]> {
  return rest(token, `/accounts/${accountId}/d1/database`);
}

// Optional list-loader overrides: the route passes cached() wrappers here.
export interface SummaryDeps {
  zonesList?: typeof fetchZonesList;
  d1List?: typeof fetchD1List;
}

/** Collects the dashboard summary. Each section is independent: a failure
 *  becomes an entry in errors (the gap), never a failed page. */
export async function collectSummary(accountId: string, token: string, now = new Date(), deps: SummaryDeps = {}): Promise<Summary> {
  const end = now.toISOString();
  const start = new Date(now.getTime() - 24 * 3600 * 1000).toISOString();
  const monthStart = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1)).toISOString();
  const summary: Summary = { generatedAt: end, windowHours: 24, usage: [], d1: [], zones: [], errors: [], workers: [] };
  const zonesList = deps.zonesList ?? ((tok: string, acct: string) => fetchZonesList(tok, acct));
  const d1List = deps.d1List ?? ((tok: string, acct: string) => fetchD1List(tok, acct));

  // Two independent queries: a D1 failure (missing scope, dataset error)
  // must not hide Workers pacing — D1 is additive, as in the CLI (D15).
  const monthVars = { a: accountId, s: monthStart, e: end };
  const workersUsage = gql<{ viewer: { accounts: { w: { sum: { requests: number } }[] }[] } }>(token,
    `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){w: workersInvocationsAdaptive(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){sum{requests}}}}}`, monthVars)
    .then((data) => {
      const reqs = (data.viewer.accounts[0]?.w ?? []).reduce((n, g) => n + g.sum.requests, 0);
      return { id: "workers.requests_monthly", name: "Workers requests", used: reqs, limit: WORKERS_REQUESTS_MONTHLY, ...monthPacing(reqs, WORKERS_REQUESTS_MONTHLY, now) } as UsageRow;
    })
    .catch((e: Error) => { summary.errors.push(`usage (workers): ${e.message}`); return null; });
  const d1Usage = gql<{ viewer: { accounts: { d: { sum: { rowsRead: number } }[] }[] } }>(token,
    `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){d: d1AnalyticsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){sum{rowsRead}}}}}`, monthVars)
    .then((data) => {
      const rows = (data.viewer.accounts[0]?.d ?? []).reduce((n, g) => n + g.sum.rowsRead, 0);
      return { id: "d1.rows_read_monthly", name: "D1 rows read", used: rows, limit: D1_ROWS_READ_MONTHLY, ...monthPacing(rows, D1_ROWS_READ_MONTHLY, now) } as UsageRow;
    })
    .catch((e: Error) => { summary.errors.push(`usage (d1): ${e.message}`); return null; });
  const usage = Promise.all([workersUsage, d1Usage]).then((rows) => {
    summary.usage = rows.filter((r): r is UsageRow => r !== null);
  });

  const d1 = (async () => {
    const [dbs, data] = await Promise.all([
      d1List(token, accountId).catch((e: Error) => {
        summary.errors.push(`d1 names (showing IDs): ${e.message}`);
        return [] as { uuid: string; name: string }[];
      }),
      gql<{ viewer: { accounts: { g: { sum: { rowsRead: number; readQueries: number; rowsWritten: number }; dimensions: { databaseId: string } }[] }[] } }>(
        token,
        `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){g: d1AnalyticsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){sum{rowsRead readQueries rowsWritten} dimensions{databaseId}}}}}`,
        { a: accountId, s: start, e: end },
      ),
    ]);
    const names = Object.fromEntries(dbs.map((d) => [d.uuid, d.name]));
    summary.d1 = summarizeD1(
      (data.viewer.accounts[0]?.g ?? []).map((g) => ({ databaseId: g.dimensions.databaseId, rowsRead: g.sum.rowsRead, readQueries: g.sum.readQueries, rowsWritten: g.sum.rowsWritten })),
      names,
    );
  })().catch((e: Error) => summary.errors.push(`d1: ${e.message}`));

  // v2: per-script 24h rows. Fields verified live 2026-10-09 (a real query
  // returned dimensions.scriptName, sum.{requests,errors},
  // quantiles.{cpuTimeP50,cpuTimeP99} for 16 scripts). CPU quantiles are
  // microseconds (live schema description "CPU time 99th percentile -
  // microseconds") — converted to ms by summarizeWorkers.
  const workers = (async () => {
    const data = await gql<{ viewer: { accounts: { w: { dimensions: { scriptName: string }; sum: { requests: number; errors: number }; quantiles: { cpuTimeP50: number | null; cpuTimeP99: number | null } }[] }[] } }>(
      token,
      `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){w: workersInvocationsAdaptive(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){dimensions{scriptName} sum{requests errors} quantiles{cpuTimeP50 cpuTimeP99}}}}}`,
      { a: accountId, s: start, e: end },
    );
    summary.workers = summarizeWorkers(
      (data.viewer.accounts[0]?.w ?? []).map((r) => ({
        scriptName: r.dimensions.scriptName,
        requests: r.sum.requests,
        errors: r.sum.errors,
        cpuP50Us: r.quantiles.cpuTimeP50,
        cpuP99Us: r.quantiles.cpuTimeP99,
      })),
    );
  })().catch((e: Error) => summary.errors.push(`workers: ${e.message}`));

  const zones = (async () => {
    const list = await zonesList(token, accountId);
    const names = Object.fromEntries(list.map((z) => [z.id, z.name]));
    const ids = Object.keys(names);
    const q = `query($t:[String!],$s:Time!,$e:Time!){viewer{zones(filter:{zoneTag_in:$t}){zoneTag
      httpRequestsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e,requestSource:"eyeball"}){count dimensions{cacheStatus}}}}}`;
    const batches: Promise<ZoneGroups[]>[] = [];
    for (let i = 0; i < ids.length; i += 10) {
      batches.push(
        gql<{ viewer: { zones: { zoneTag: string; httpRequestsAdaptiveGroups: { count: number; dimensions: { cacheStatus: string } }[] }[] } }>(
          token, q, { t: ids.slice(i, i + 10), s: start, e: end },
        ).then((d) => d.viewer.zones.map((z) => ({ zoneTag: z.zoneTag, groups: z.httpRequestsAdaptiveGroups.map((g) => ({ count: g.count, cacheStatus: g.dimensions.cacheStatus })) }))),
      );
    }
    summary.zones = summarizeZones((await Promise.all(batches)).flat(), names);
  })().catch((e: Error) => summary.errors.push(`zones: ${e.message}`));

  await Promise.all([usage, d1, zones, workers]);
  return summary;
}
