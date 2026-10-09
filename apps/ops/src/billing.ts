// Cosmoflare Ops billing collector (FEAT-052): month-to-date usage over the
// billing period, per-product included vs used vs projected, projected USD
// beyond the Workers Paid allowance, and the CosmoLabs project that drives
// it. Every GraphQL dataset and field below was introspected from the live
// schema on 2026-10-09 (see the per-dataset comments). Independent datasets
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
// Dataset queries — fields introspected from the live GraphQL schema on
// 2026-10-09 (orchestrator pass; replaces the earlier "not exposed" notes):
// - workersInvocationsAdaptive: dims scriptName; sum{requests errors cpuTimeUs}
//   ("Sum of cpu time in us") → converted to ms.
// - durableObjectsInvocationsAdaptiveGroups: dims scriptName; sum{requests}.
// - durableObjectsPeriodicGroups: sum{duration} ("Sum of Duration - GB*s");
//   groups by date only (introspected live 2026-10-09 — scriptName is NOT a
//   dimension), so duration is an account total under one synthetic
//   "all scripts" consumer.
// - d1AnalyticsAdaptiveGroups: dims databaseId; sum{rowsRead rowsWritten
//   readQueries writeQueries}; datetime filter, ISO times.
// - d1StorageAdaptiveGroups: dims databaseId; max{databaseSizeBytes} → GB;
//   Date filter (YYYY-MM-DD) only.
// - r2OperationsAdaptiveGroups: dims bucketName actionType; sum{requests}.
// - r2StorageAdaptiveGroups: dims bucketName; max{payloadSize metadataSize}
//   → GB (payload + metadata).
// - kvOperationsAdaptiveGroups: dims namespaceId actionType ("read"/"write"
//   seen live; delete is free, list prices like a write); sum{requests}.
// - kvStorageAdaptiveGroups: dims namespaceId; max{byteCount} → GB.
// R2 and KV datasets document a max query range of 31 days (docs, read
// 2026-10-09); the storage window stays inside that bound.

const WORKERS_Q = `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){
w: workersInvocationsAdaptive(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){dimensions{scriptName} sum{requests errors cpuTimeUs}}}}}`;

const D1_Q = `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){
d: d1AnalyticsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){dimensions{databaseId} sum{rowsRead rowsWritten readQueries writeQueries}}}}}`;

// Storage snapshot datasets — Date-typed filters over a 2-day window ending
// today (the latest day may not be sampled yet); the aggregation takes the
// max sample per resource, so the window does not double-count.
const D1_STORAGE_Q = `query($a:String!,$s:Date!,$e:Date!){viewer{accounts(filter:{accountTag:$a}){
s: d1StorageAdaptiveGroups(limit:10000,filter:{date_geq:$s,date_leq:$e}){dimensions{databaseId} max{databaseSizeBytes}}}}}`;

const R2_STORAGE_Q = `query($a:String!,$s:Date!,$e:Date!){viewer{accounts(filter:{accountTag:$a}){
rs: r2StorageAdaptiveGroups(limit:10000,filter:{date_geq:$s,date_leq:$e}){dimensions{bucketName} max{payloadSize metadataSize}}}}}`;

const KV_STORAGE_Q = `query($a:String!,$s:Date!,$e:Date!){viewer{accounts(filter:{accountTag:$a}){
ks: kvStorageAdaptiveGroups(limit:10000,filter:{date_geq:$s,date_leq:$e}){dimensions{namespaceId} max{byteCount}}}}}`;

const R2_OPS_Q = `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){
r: r2OperationsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){dimensions{bucketName actionType} sum{requests}}}}}`;

const KV_OPS_Q = `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){
k: kvOperationsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){dimensions{namespaceId actionType} sum{requests}}}}}`;

const DO_Q = `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){
o: durableObjectsInvocationsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){dimensions{scriptName} sum{requests}}}}}`;

// DO duration — the periodic dataset is Date-filtered and groups by date, not
// scriptName (introspected live 2026-10-09), so it needs its own Date
// variables ($ds/$de) alongside the flow datasets' Time variables.
const DO_PERIODIC_Q = `query($a:String!,$ds:Date!,$de:Date!){viewer{accounts(filter:{accountTag:$a}){
p: durableObjectsPeriodicGroups(limit:10000,filter:{date_geq:$ds,date_leq:$de}){dimensions{date} sum{duration}}}}}`;

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

// Synthetic consumers hold account-wide totals that cannot be attributed to
// one project. d1.storage is per-database (databaseId dimension, introspected
// 2026-10-09) so it no longer needs one; DO duration uses one because its
// periodic dataset groups by date only, not scriptName.
const SYNTHETIC_PREFIXES = ["all databases", "all scripts"];

export function isSyntheticConsumer(name: string): boolean {
  return SYNTHETIC_PREFIXES.some((p) => name.startsWith(p));
}

export function topConsumers(accs: Map<string, ConsumerAcc>, pmap: Record<string, string>, total: number): ProductConsumer[] {
  return [...accs.values()]
    .map((c) => ({
      name: c.name,
      project: isSyntheticConsumer(c.name) ? "shared" : projectFor(c.name, pmap),
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
  const today = now.toISOString().slice(0, 10);
  const from = new Date(now.getTime() - MS_DAY).toISOString().slice(0, 10);
  const vars = { a: accountId, s: period.start, e: now.toISOString(), ds: period.start.slice(0, 10), de: today };

  type Row = { dimensions: Record<string, string>; sum?: Record<string, number>; max?: Record<string, number> };
  type Acc = Record<string, Row[]>;
  type Resp = { viewer: { accounts: Acc[] } };

  // Merge every flow dataset into ONE aliased GraphQL call (the operator's
  // explicit priority: fewer upstream calls). The storage snapshot is a second
  // merged call (Date-typed variables). If a merged call fails, each dataset
  // is retried alone so one bad dataset cannot drop the rest — each retry
  // appends only its own error.
  const strip = (q: string) =>
    q
      .replace(/query\([^)]*\)\{viewer\{accounts\(filter:\{accountTag:\$a\}\)\{/, "")
      .replace(/\}\}\}$/, "");
  const flowBody = [WORKERS_Q, D1_Q, R2_OPS_Q, KV_OPS_Q, DO_Q, DO_PERIODIC_Q].map(strip).join("");
  const flowQuery = `query($a:String!,$s:Time!,$e:Time!,$ds:Date!,$de:Date!){viewer{accounts(filter:{accountTag:$a}){${flowBody}}}}`;
  const snapBody = [D1_STORAGE_Q, R2_STORAGE_Q, KV_STORAGE_Q].map(strip).join("");
  const snapQuery = `query($a:String!,$s:Date!,$e:Date!){viewer{accounts(filter:{accountTag:$a}){${snapBody}}}}`;

  const acc: Acc = {};
  const singles: Promise<void>[] = [];
  const mergeInto = (d: Resp) => {
    Object.assign(acc, d.viewer.accounts[0]);
  };
  const single = (key: string, q: string, vs: Record<string, unknown>) =>
    gql<Resp>(token, q, vs, calls)
      .then(mergeInto)
      .catch((e: Error) => {
        errors.push(`${key}: ${e.message}`);
      });

  const flow = await gql<Resp>(token, flowQuery, vars, calls).catch(() => null);
  if (flow) {
    mergeInto(flow);
  } else {
    for (const [key, q] of [
      ["w", WORKERS_Q],
      ["d", D1_Q],
      ["r", R2_OPS_Q],
      ["k", KV_OPS_Q],
      ["o", DO_Q],
      ["p", DO_PERIODIC_Q],
    ] as const) {
      singles.push(single(key, q, vars));
    }
  }

  const snapVars = { a: accountId, s: from, e: today };
  const snap = await gql<Resp>(token, snapQuery, snapVars, calls).catch(() => null);
  if (snap) {
    mergeInto(snap);
  } else {
    for (const [key, q] of [
      ["d1.storage", D1_STORAGE_Q],
      ["r2.storage", R2_STORAGE_Q],
      ["kv.storage", KV_STORAGE_Q],
    ] as const) {
      singles.push(single(key, q, snapVars));
    }
  }

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
  for (const g of rows("w")) {
    bump("workers.requests", g.dimensions.scriptName, g.sum?.requests ?? 0);
    bump("workers.cpu_ms", g.dimensions.scriptName, g.sum?.cpuTimeUs ?? 0);
  }
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
  for (const g of rows("p")) bump("do.duration", "all scripts", g.sum?.duration ?? 0); // account total (per-day GB-s)

  // Storage snapshots: keep the max sample per resource across the window
  // (a resource appears once per sampled day — never sum across days).
  const maxBy = (key: string, nameOf: (d: Row["dimensions"]) => string, valueOf: (g: Row) => number) => {
    const best = new Map<string, ConsumerAcc>();
    for (const g of rows(key)) {
      const name = nameOf(g.dimensions);
      const v = valueOf(g);
      const cur = best.get(name);
      if (!cur || v > cur.used) best.set(name, { name, used: v });
    }
    return best;
  };
  const d1Store = maxBy("s", (d) => names.d1[d.databaseId] ?? d.databaseId, (g) => g.max?.databaseSizeBytes ?? 0);
  const r2Store = maxBy("rs", (d) => d.bucketName, (g) => (g.max?.payloadSize ?? 0) + (g.max?.metadataSize ?? 0));
  const kvStore = maxBy("ks", (d) => names.kv[d.namespaceId] ?? d.namespaceId, (g) => g.max?.byteCount ?? 0);
  if (d1Store.size) accs.set("d1.storage", d1Store);
  if (r2Store.size) accs.set("r2.storage", r2Store);
  if (kvStore.size) accs.set("kv.storage", kvStore);

  const products: ProductUsage[] = [];
  for (const price of PRICES) {
    const m = accs.get(price.id);
    const total = m ? [...m.values()].reduce((n, c) => n + c.used, 0) : 0;
    const isStorage = price.id === "r2.storage" || price.id === "kv.storage" || price.id === "d1.storage";
    const toUnit = isStorage ? 1_000_000_000 : price.id === "workers.cpu_ms" ? 1_000 : 1; // bytes → GB, µs → ms
    const used = total / toUnit;
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
      topConsumers: m ? topConsumers(m, pmap, total).slice(0, 5).map((c) => ({ ...c, used: c.used / toUnit })) : [],
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
      if (isSyntheticConsumer(c.name)) continue; // account-wide totals are not a project
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
