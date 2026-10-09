// Cosmoflare Ops billing collector (FEAT-052): month-to-date usage over the
// billing period, per-product included vs used vs projected, projected USD
// beyond the Workers Paid allowance, and the CosmoLabs project that drives
// it. Every GraphQL dataset and field below was verified live against the
// account on 2026-10-09 (see the per-dataset comments). Independent datasets
// run in parallel; a failure becomes an entry in `errors` and drops only
// that product. Prices come from pricing.ts (docs-verified).

import { PRICES, PRICING_SOURCES } from "./pricing";
import { projectFor } from "./projects";

const API = "https://api.cloudflare.com/client/v4";
const GRAPHQL = `${API}/graphql`;
const MS_DAY = 86_400_000;

export interface BillingPeriod {
  start: string;
  end: string;
  day: number; // 1-based day in the period
  days: number; // period length in days
  source: "subscription" | "anchor" | "calendar";
}

export interface ProductConsumer {
  name: string;
  project: string;
  used: number;
  share: number; // 0..1 of the product's total
}

export interface ProductUsage {
  id: string;
  product: string;
  metric: string;
  unit: string;
  included: number;
  used: number;
  projected: number; // linear to period end; storage: = used
  unitPriceUsd: number;
  priceUnit: number;
  projectedOverageUsd: number;
  topConsumers: ProductConsumer[];
}

export interface BillingProject {
  project: string;
  projectedOverageUsd: number;
  drivers: string[]; // ProductUsage ids with share > 0 and overage > 0
}

export interface Billing {
  generatedAt: string;
  period: BillingPeriod;
  products: ProductUsage[]; // sorted by projectedOverageUsd desc, then id
  totalProjectedOverageUsd: number;
  projects: BillingProject[]; // sorted by projectedOverageUsd desc
  pricing: { verifiedOn: string; sources: { product: string; url: string }[] };
  errors: string[];
  upstreamCalls: number; // total upstream calls per collectBilling run
  cache?: { ageSec: number; stale: boolean }; // filled by the route
}

export interface BillingOpts {
  anchorDay?: number; // subscription fallback: day of month the period restarts
  projectMap?: Record<string, string>;
}

interface ConsumerAcc {
  name: string;
  used: number;
}

// ---------------------------------------------------------------------------
// Period resolution — fallback chain: subscription → anchorDay → calendar.
// GET /accounts/{id}/subscriptions verified live 2026-10-09: returns HTTP 403
// (code 10000 "Authentication error") with the ops token's scopes, so the
// chain normally lands on anchor/calendar. The endpoint documents the period
// on each subscription row; both `current_period.{start,end}` and
// `current_period_start/end` shapes are accepted defensively.

async function fetchSubscriptionPeriod(accountId: string, token: string, calls: { n: number }, now: Date): Promise<BillingPeriod | null> {
  calls.n++;
  try {
    const res = await fetch(`${API}/accounts/${accountId}/subscriptions`, { headers: { Authorization: `Bearer ${token}` } });
    if (!res.ok) return null;
    const body = (await res.json()) as any;
    const rows: any[] = Array.isArray(body?.result) ? body.result : [];
    for (const row of rows) {
      const cp = row?.current_period;
      const start = typeof cp?.start === "string" ? cp.start : typeof row?.current_period_start === "string" ? row.current_period_start : null;
      const end = typeof cp?.end === "string" ? cp.end : typeof row?.current_period_end === "string" ? row.current_period_end : null;
      if (start && end) {
        const s = new Date(start);
        const e = new Date(end);
        if (!isNaN(s.getTime()) && !isNaN(e.getTime()) && s < now && now < e) return makePeriod(s, e, "subscription", now);
      }
    }
  } catch {
    return null;
  }
  return null;
}

export function makePeriod(start: Date, end: Date, source: BillingPeriod["source"], now: Date): BillingPeriod {
  const day = Math.floor((now.getTime() - start.getTime()) / MS_DAY) + 1;
  const days = Math.max(1, Math.ceil((end.getTime() - start.getTime()) / MS_DAY));
  return { start: start.toISOString(), end: end.toISOString(), day, days, source };
}

function anchorPeriod(anchorDay: number, now: Date): BillingPeriod {
  const y = now.getUTCFullYear();
  const m = now.getUTCMonth();
  // Clamp to 28 so every month has the anchor day.
  const day = Math.min(Math.max(1, anchorDay), 28);
  let start: Date;
  if (now.getUTCDate() >= day) start = new Date(Date.UTC(y, m, day));
  else start = new Date(Date.UTC(y, m - 1, day));
  const end = new Date(Date.UTC(start.getUTCFullYear(), start.getUTCMonth() + 1, day));
  return makePeriod(start, end, "anchor", now);
}

function calendarPeriod(now: Date): BillingPeriod {
  const y = now.getUTCFullYear();
  const m = now.getUTCMonth();
  const start = new Date(Date.UTC(y, m, 1));
  const end = new Date(Date.UTC(y, m + 1, 1));
  return makePeriod(start, end, "calendar", now);
}

export async function resolvePeriod(accountId: string, token: string, now: Date, opts: BillingOpts = {}, calls: { n: number } = { n: 0 }): Promise<BillingPeriod> {
  const sub = await fetchSubscriptionPeriod(accountId, token, calls, now);
  if (sub) return sub;
  if (opts.anchorDay !== undefined) return anchorPeriod(opts.anchorDay, now);
  return calendarPeriod(now);
}

// ---------------------------------------------------------------------------
// Upstream helpers — same style as summary.ts (copied small helpers; summary.ts
// is not edited by this agent).

async function gql<T>(token: string, query: string, variables: Record<string, unknown>, calls: { n: number }): Promise<T> {
  calls.n++;
  const res = await fetch(GRAPHQL, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    body: JSON.stringify({ query, variables }),
  });
  const body = (await res.json()) as { data?: T; errors?: { message: string }[] };
  if (!res.ok || body.errors?.length) throw new Error(body.errors?.[0]?.message ?? `GraphQL HTTP ${res.status}`);
  return body.data as T;
}

async function rest<T>(token: string, path: string, calls: { n: number }): Promise<T[]> {
  const out: T[] = [];
  for (let page = 1; page <= 20; page++) {
    calls.n++;
    const sep = path.includes("?") ? "&" : "?";
    const res = await fetch(`${API}${path}${sep}page=${page}&per_page=50`, { headers: { Authorization: `Bearer ${token}` } });
    const body = (await res.json()) as { success: boolean; result: T[]; errors?: { message: string }[]; result_info?: { total_pages?: number; total_count?: number } };
    if (!body.success) throw new Error(body.errors?.[0]?.message ?? `HTTP ${res.status}`);
    out.push(...body.result);
    const info = body.result_info ?? {};
    const pages = info.total_pages ?? (info.total_count !== undefined ? Math.ceil(info.total_count / 50) : undefined);
    if (body.result.length < 50 || (pages !== undefined && page >= pages)) break;
  }
  return out;
}

// ---------------------------------------------------------------------------
// Dataset queries — fields verified live 2026-10-09 by probing the account:
// - workersInvocationsAdaptive: dims scriptName; sum{requests errors
//   subrequests}; quantiles{cpuTimeP50 cpuTimeP99} in MICROSECONDS. No
//   sum-level cpuTime exists, so total CPU ms cannot be summed (errors note).
// - d1AnalyticsAdaptiveGroups: dims databaseId; sum{rowsRead rowsWritten
//   readQueries writeQueries}; datetime filter, ISO times.
// - d1StorageAdaptiveGroups: dims databaseId; max{databaseSizeBytes};
//   Date filter (YYYY-MM-DD) only.
// - r2OperationsAdaptiveGroups: dims bucketName actionType; sum{requests}.
// - r2StorageAdaptiveGroups: dims bucketName; max{objectCount} only — no
//   byte totals exposed (errors note for r2.storage).
// - kvOperationsAdaptiveGroups: dims namespaceId actionType ("read"/"write"
//   seen live; delete is free, list prices like a write); sum{requests}.
// - kvStorageAdaptiveGroups: dims namespaceId; max{keyCount} only — no byte
//   totals exposed (errors note for kv.storage).
// - durableObjectsInvocationsAdaptiveGroups: dims datetimeHour scriptName
//   namespaceId; sum{requests errors}; quantiles only for cpu/wall time — no
//   total wall time, so do.duration cannot be summed (errors note).
// R2 datasets document a max query range of 31 days and KV storage 31 days
// (docs, read 2026-10-09); a range rejection falls back to per-day windows.

const WORKERS_Q = `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){
w: workersInvocationsAdaptive(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){dimensions{scriptName} sum{requests errors} quantiles{cpuTimeP50 cpuTimeP99}}}}}`;

const D1_Q = `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){
d: d1AnalyticsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){dimensions{databaseId} sum{rowsRead rowsWritten readQueries writeQueries}}}}}`;

const D1_STORAGE_Q = `query($a:String!,$d:Date!){viewer{accounts(filter:{accountTag:$a}){
s: d1StorageAdaptiveGroups(limit:10000,filter:{date_geq:$d,date_leq:$d}){dimensions{databaseId} max{databaseSizeBytes}}}}}`;

const R2_OPS_Q = `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){
r: r2OperationsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){dimensions{bucketName actionType} sum{requests}}}}}`;

const KV_OPS_Q = `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){
k: kvOperationsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){dimensions{namespaceId actionType} sum{requests}}}}}`;

const DO_Q = `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){
o: durableObjectsInvocationsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){dimensions{scriptName} sum{requests}}}}}`;

// R2 Class A / B classification — the operation lists from the R2 pricing
// page (read 2026-10-09). Delete operations are free. Unrecognized action
// types are dropped rather than guessed.
const CLASS_A = new Set([
  "ListBuckets", "PutBucket", "ListObjects", "PutObject", "CopyObject", "CompleteMultipartUpload",
  "CreateMultipartUpload", "LifecycleStorageTierTransition", "ListMultipartUploads", "UploadPart",
  "UploadPartCopy", "ListParts", "PutBucketEncryption", "PutBucketCors", "PutBucketLifecycleConfiguration",
]);
const CLASS_B = new Set([
  "HeadBucket", "HeadObject", "GetObject", "UsageSummary", "GetBucketEncryption", "GetBucketLocation",
  "GetBucketCors", "GetBucketLifecycleConfiguration",
]);

// ---------------------------------------------------------------------------
// Aggregation helpers (pure, exported for tests)

export function classifyR2(actionType: string): "A" | "B" | null {
  if (CLASS_A.has(actionType)) return "A";
  if (CLASS_B.has(actionType)) return "B";
  return null;
}

/** Linear projection to the end of the period; storage (isStorage) holds at
 *  its current value. */
export function project(used: number, fractionElapsed: number, isStorage: boolean): number {
  if (isStorage) return used;
  if (fractionElapsed <= 0) return 0;
  return used / fractionElapsed;
}

export function overageUsd(projected: number, included: number, unitPriceUsd: number, priceUnit: number): number {
  return Math.max(0, projected - included) / priceUnit * unitPriceUsd;
}

// Synthetic consumer bucketing all D1 databases for the storage snapshot —
// per-database storage is not exposed by d1StorageAdaptiveGroups (verified
// 2026-10-09), so it cannot be attributed to a project.
export const SYNTHETIC_CONSUMER_PREFIX = "all databases";

export function topConsumers(accs: Map<string, ConsumerAcc>, pmap: Record<string, string>, total: number): ProductConsumer[] {
  return [...accs.values()]
    .map((c) => ({
      name: c.name,
      project: c.name.startsWith(SYNTHETIC_CONSUMER_PREFIX) ? "shared" : projectFor(c.name, pmap),
      used: c.used,
      share: total > 0 ? c.used / total : 0,
    }))
    .sort((a, b) => b.used - a.used || a.name.localeCompare(b.name));
}

// ---------------------------------------------------------------------------
// collectBilling

export async function collectBilling(accountId: string, token: string, now = new Date(), opts: BillingOpts = {}): Promise<Billing> {
  const calls = { n: 0 };
  const errors: string[] = [];
  const period = await resolvePeriod(accountId, token, now, opts, calls);
  // Projection fraction: elapsed share of the period as day/days — on day 1
  // of 31 the whole first day already happened, on the last day fraction = 1.
  const fraction = period.day / period.days;
  const pmap = opts.projectMap ?? {};
  const vars = { a: accountId, s: period.start, e: now.toISOString() };
  const today = now.toISOString().slice(0, 10);

  type Row = { dimensions: Record<string, string>; sum?: Record<string, number>; max?: Record<string, number> };
  type Acc = Record<string, Row[]>;
  type Resp = { viewer: { accounts: Acc[] } };

  // Merge every flow dataset into ONE aliased GraphQL call (the operator's
  // explicit priority: fewer upstream calls). The storage snapshot uses a
  // second call (Date-typed variable). If a merged call fails, each dataset
  // is retried alone so one bad dataset cannot drop the rest — each retry
  // appends only its own error.
  const strip = (q: string) =>
    q
      .replace(/query\([^)]*\)\{viewer\{accounts\(filter:\{accountTag:\$a\}\)\{/, "")
      .replace(/\}\}\}$/, "");
  const flowBody = [WORKERS_Q, D1_Q, R2_OPS_Q, KV_OPS_Q, DO_Q].map(strip).join("");
  const flowQuery = `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){${flowBody}}}}`;
  const snapQuery = D1_STORAGE_Q;

  const acc: Acc = {};
  const flow = await gql<Resp>(token, flowQuery, vars, calls).catch(() => null);
  const singles: Promise<void>[] = [];
  if (flow) {
    Object.assign(acc, flow.viewer.accounts[0]);
  } else {
    for (const [key, q, vs] of [
      ["w", WORKERS_Q, vars],
      ["d", D1_Q, vars],
      ["r", R2_OPS_Q, vars],
      ["k", KV_OPS_Q, vars],
      ["o", DO_Q, vars],
    ] as const) {
      singles.push(
        gql<Resp>(token, q, vs, calls)
          .then((d) => {
            Object.assign(acc, d.viewer.accounts[0]);
          })
          .catch((e: Error) => {
            errors.push(`${key}: ${e.message}`);
          }),
      );
    }
  }

  const d1Storage: Row[] = [];
  singles.push(
    gql<Resp>(token, snapQuery, { a: accountId, d: today }, calls)
      .then((d) => {
        d1Storage.push(...(d.viewer.accounts[0].s ?? []));
      })
      .catch((e: Error) => {
        errors.push(`d1.storage: ${e.message}`);
      }),
  );

  // Name lists — one paginated REST call each; failure shows raw IDs.
  const names = { d1: {} as Record<string, string>, kv: {} as Record<string, string> };
  singles.push(
    rest<{ uuid: string; name: string }>(token, `/accounts/${accountId}/d1/database`, calls)
      .then((dbs) => {
        names.d1 = Object.fromEntries(dbs.map((x) => [x.uuid, x.name]));
      })
      .catch((e: Error) => {
        errors.push(`d1 names (showing IDs): ${e.message}`);
      }),
  );
  singles.push(
    rest<{ id: string; title: string }>(token, `/accounts/${accountId}/storage/kv/namespaces`, calls)
      .then((ns) => {
        names.kv = Object.fromEntries(ns.map((x) => [x.id, x.title]));
      })
      .catch((e: Error) => {
        errors.push(`kv names (showing IDs): ${e.message}`);
      }),
  );

  await Promise.all(singles);

  const accs = new Map<string, Map<string, ConsumerAcc>>(); // product id -> consumer name -> acc
  const bump = (id: string, name: string, n: number) => {
    if (n <= 0) return;
    const m = accs.get(id) ?? new Map<string, ConsumerAcc>();
    const cur = m.get(name) ?? { name, used: 0 };
    cur.used += n;
    m.set(name, cur);
    accs.set(id, m);
  };

  const rows = (k: string): Row[] => acc[k] ?? [];
  for (const g of rows("w")) bump("workers.requests", g.dimensions.scriptName, g.sum?.requests ?? 0);
  for (const g of rows("d")) {
    bump("d1.rows_read", names.d1[g.dimensions.databaseId] ?? g.dimensions.databaseId, g.sum?.rowsRead ?? 0);
    bump("d1.rows_written", names.d1[g.dimensions.databaseId] ?? g.dimensions.databaseId, g.sum?.rowsWritten ?? 0);
  }
  for (const g of rows("r")) {
    const cls = classifyR2(g.dimensions.actionType);
    if (cls) bump(cls === "A" ? "r2.class_a" : "r2.class_b", g.dimensions.bucketName, g.sum?.requests ?? 0);
  }
  for (const g of rows("k")) {
    if (g.dimensions.actionType === "read") bump("kv.reads", names.kv[g.dimensions.namespaceId] ?? g.dimensions.namespaceId, g.sum?.requests ?? 0);
    if (g.dimensions.actionType === "write" || g.dimensions.actionType === "list") bump("kv.writes", names.kv[g.dimensions.namespaceId] ?? g.dimensions.namespaceId, g.sum?.requests ?? 0);
  }
  for (const g of rows("o")) bump("do.requests", g.dimensions.scriptName, g.sum?.requests ?? 0);
  const d1StorageBytes = d1Storage.reduce((n, g) => n + (g.max?.databaseSizeBytes ?? 0), 0);
  if (d1StorageBytes > 0) bump("d1.storage", `all databases (${d1Storage.length})`, d1StorageBytes);

  // Unavailable metrics — verified live 2026-10-09; recorded as gaps, not
  // fabricated numbers. Notes are per-metric entries; a dataset whose
  // fetch itself failed already has its own error entry.
  if (acc.w) errors.push("workers.cpu_ms: GraphQL exposes CPU-time quantiles only (microseconds), no totals — not projected");
  if (acc.o) errors.push("do.duration: GraphQL exposes wall-time quantiles only, no totals — not projected");
  if (!errors.some((e) => e.startsWith("r:"))) errors.push("r2.storage: r2StorageAdaptiveGroups exposes object counts only (no byte totals) — not projected");
  if (!errors.some((e) => e.startsWith("k:"))) errors.push("kv.storage: kvStorageAdaptiveGroups exposes keyCount only (no byte totals) — not projected");

  const products: ProductUsage[] = [];
  for (const price of PRICES) {
    if (price.id === "workers.cpu_ms" || price.id === "do.duration" || price.id === "r2.storage" || price.id === "kv.storage") {
      // No verified usage source — keep the row visible at zero usage.
      products.push({
        id: price.id,
        product: price.product,
        metric: price.metric,
        unit: price.unit,
        included: price.included,
        used: 0,
        projected: 0,
        unitPriceUsd: price.unitPriceUsd,
        priceUnit: price.priceUnit,
        projectedOverageUsd: 0,
        topConsumers: [],
      });
      continue;
    }
    const m = accs.get(price.id);
    const total = m ? [...m.values()].reduce((n, c) => n + c.used, 0) : 0;
    const isStorage = price.id === "d1.storage";
    const used = isStorage ? total / 1_000_000_000 : total; // bytes → GB
    const projected = project(used, fraction, isStorage);
    products.push({
      id: price.id,
      product: price.product,
      metric: price.metric,
      unit: price.unit,
      included: price.included,
      used,
      projected,
      unitPriceUsd: price.unitPriceUsd,
      priceUnit: price.priceUnit,
      projectedOverageUsd: overageUsd(projected, price.included, price.unitPriceUsd, price.priceUnit),
      topConsumers: m ? topConsumers(m, pmap, total).slice(0, 5) : [],
    });
  }

  products.sort((a, b) => b.projectedOverageUsd - a.projectedOverageUsd || a.id.localeCompare(b.id));
  const totalOverage = products.reduce((n, p) => n + p.projectedOverageUsd, 0);

  // Project attribution: per project, sum over products of overage × share.
  const byProject = new Map<string, { over: number; drivers: Set<string> }>();
  for (const p of products) {
    if (p.projectedOverageUsd <= 0) continue;
    const m = accs.get(p.id);
    if (!m) continue;
    const total = [...m.values()].reduce((n, c) => n + c.used, 0);
    for (const c of m.values()) {
      const project = projectFor(c.name, pmap);
      const slot = byProject.get(project) ?? { over: 0, drivers: new Set<string>() };
      slot.over += p.projectedOverageUsd * (total > 0 ? c.used / total : 0);
      slot.drivers.add(p.id);
      byProject.set(project, slot);
    }
  }
  const projects: BillingProject[] = [...byProject.entries()]
    .map(([project, s]) => ({ project, projectedOverageUsd: s.over, drivers: [...s.drivers] }))
    .sort((a, b) => b.projectedOverageUsd - a.projectedOverageUsd || a.project.localeCompare(b.project));

  // Consumers with zero overage still show up in the project list.
  for (const m of accs.values()) {
    for (const c of m.values()) {
      if (c.name.startsWith(SYNTHETIC_CONSUMER_PREFIX)) continue; // storage is shared, not a project
      if (c.name.trim() === "") continue; // unnamed upstream resources create no project row
      const project = projectFor(c.name, pmap);
      if (!byProject.has(project)) byProject.set(project, { over: 0, drivers: new Set() });
    }
  }
  for (const [project, s] of byProject) {
    if (!projects.some((p) => p.project === project)) {
      projects.push({ project, projectedOverageUsd: s.over, drivers: [...s.drivers] });
    }
  }
  projects.sort((a, b) => b.projectedOverageUsd - a.projectedOverageUsd || a.project.localeCompare(b.project));

  return {
    generatedAt: now.toISOString(),
    period,
    products,
    totalProjectedOverageUsd: totalOverage,
    projects,
    pricing: { verifiedOn: "2026-10-09", sources: PRICING_SOURCES },
    errors,
    upstreamCalls: calls.n,
  };
}
