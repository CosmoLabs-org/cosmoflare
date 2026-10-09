// Upstream cache for the Cosmoflare Ops Worker (FEAT-052): a two-tier cache
// in front of every Cloudflare API call so the operator dashboard and its
// auto-refresh cost almost no upstream requests.
//
// L1 = module-scope Map (per-isolate memo; the one sanctioned request-independent
// global in this Worker). L2 = KV (env.OPS_KV) when bound, else the Cache API
// (caches.default) with a synthetic GET-key URL https://ops-cache.internal/<key>.
// Entry JSON: { storedAt, value }.
//
// Verified 2026-10-09, https://developers.cloudflare.com/workers/runtime-apis/cache/:
// - "Workers deployed to custom domains have access to functional `cache` operations."
// - "For Workers fronted by Cloudflare Access, the Cache API is not currently available."
// ops.cosmolabs.org IS fronted by Access, so the Cache API here is best-effort
// (every call wrapped in try/catch — a throw means "no value"); KV is the real
// L2 once OPS_KV is bound. Cache-API rules honored: put() takes a GET request
// as key; match() has no ignoreSearch (our key URL has no query, so fine).
//
// Final TTLs (docs/planning-mode/2026-10-09-ops-billing-ui-caching.md):
// | Dataset                       | Fresh TTL | Stale window |
// | Zone / D1 lists               | 1 h       | 24 h         |
// | 24h analytics (summary)       | 5 min     | 1 h          |
// | MTD billing usage             | 15 min    | 6 h          |
// | Billing subscription (period) | 12 h      | 7 d          |

export interface CacheEnv {
  OPS_KV?: KVNamespace;
}

export interface CachedResult<T> {
  value: T;
  ageSec: number;
  stale: boolean;
}

export interface CacheOptions {
  ttlSec: number;
  staleSec: number;
  // Reload instead of serving the cache - but at most once per 60 s per key
  // (refresh-storm guard): a second force inside 60 s degrades to a normal
  // read and returns the current value.
  force?: boolean;
  // Injectable clock for tests.
  now?: () => number;
}

interface CacheEntry {
  storedAt: number;
  value: unknown;
}

// Minimum window between forced reloads of the same key.
const FORCE_MIN_MS = 60_000;
// KV expirationTtl has a 60 s minimum; clamp so short stale windows do not
// fail silently.
const KV_MIN_TTL_SEC = 60;

// L1 memo + per-key in-flight dedup + force bookkeeping. Module scope is
// deliberate (per-isolate cache); nothing here is request-scoped.
const mem = new Map<string, CacheEntry>();
const inflight = new Map<string, Promise<CacheEntry>>();
const lastForce = new Map<string, number>();

// Deduplicates concurrent operations per key: the first caller's factory
// runs, everyone else awaits (and shares) the same promise. Settled promises
// leave the map synchronously, so a later caller always starts fresh work.
function singleFlight(key: string, factory: () => Promise<CacheEntry>): Promise<CacheEntry> {
  const existing = inflight.get(key);
  return existing ?? register(key, factory());
}

// Registers a fresh promise in the in-flight map and removes it on settle.
function register(key: string, p: Promise<CacheEntry>): Promise<CacheEntry> {
  inflight.set(key, p);
  const settle = (): void => {
    if (inflight.get(key) === p) inflight.delete(key);
  };
  p.then(settle, settle);
  return p;
}

// L2 read: KV when bound, else the Cache API (wrapped - see header note:
// Access-fronted Workers have no functional Cache API, so any throw or any
// absence of caches.default means "no value").
async function l2Get(env: CacheEnv, key: string): Promise<CacheEntry | null> {
  if (env.OPS_KV) {
    const raw = await env.OPS_KV.get(key);
    return raw ? (JSON.parse(raw) as CacheEntry) : null;
  }
  try {
    const store = (caches as unknown as { default?: Cache }).default;
    if (!store) return null;
    const res = await store.match(`https://ops-cache.internal/${key}`);
    if (!res) return null;
    return (await res.json()) as CacheEntry;
  } catch {
    return null; // Access-fronted (docs 2026-10-09): Cache API unavailable
  }
}

// L2 write: KV with expirationTtl = staleSec (clamped to the 60 s KV
// minimum), else Cache API with Cache-Control max-age = staleSec.
async function l2Put(env: CacheEnv, key: string, entry: CacheEntry, staleSec: number): Promise<void> {
  const body = JSON.stringify(entry);
  if (env.OPS_KV) {
    await env.OPS_KV.put(key, body, { expirationTtl: Math.max(staleSec, KV_MIN_TTL_SEC) });
    return;
  }
  try {
    const store = (caches as unknown as { default?: Cache }).default;
    if (!store) return;
    await store.put(
      `https://ops-cache.internal/${key}` as unknown as Request,
      new Response(body, {
        headers: { "Content-Type": "application/json", "Cache-Control": `public, max-age=${staleSec}` },
      }),
    );
  } catch {
    // Access-fronted (docs 2026-10-09): Cache API unavailable - L2 stays cold
  }
}

// Build a fresh entry from a load, update L1, best-effort update L2.
async function loadEntry<T>(
  env: CacheEnv,
  key: string,
  staleSec: number,
  now: () => number,
  load: () => Promise<T>,
): Promise<CacheEntry> {
  const value = await load();
  const entry: CacheEntry = { storedAt: now(), value };
  mem.set(key, entry);
  try {
    await l2Put(env, key, entry, staleSec);
  } catch {
    // L2 is best-effort; the fresh value is already in L1
  }
  return entry;
}

/**
 * Returns the cached value for key when younger than ttlSec. When older but
 * younger than staleSec, returns the stale value and refreshes in the
 * background via ctx.waitUntil (stale-while-revalidate). Concurrent misses
 * for the same key share one in-flight load (single-flight). A failed
 * background refresh keeps the stale value; a failed load with no value at
 * all rejects. L1 = module memory, L2 = Cache API (caches.default) or KV
 * when bound.
 */
export async function cached<T>(
  ctx: ExecutionContext,
  env: CacheEnv,
  key: string,
  opts: CacheOptions,
  load: () => Promise<T>,
): Promise<CachedResult<T>> {
  const now = opts.now ?? Date.now;
  const t = now();
  const refresh = (): Promise<CacheEntry> =>
    singleFlight(key, () => loadEntry(env, key, opts.staleSec, now, load));

  let force = opts.force === true;
  if (force && t - (lastForce.get(key) ?? Number.NEGATIVE_INFINITY) < FORCE_MIN_MS) {
    force = false; // refresh-storm guard: degrade to a normal read below
  }
  if (force) lastForce.set(key, t);

  if (force) {
    const entry = await refresh();
    return { value: entry.value as T, ageSec: (now() - entry.storedAt) / 1000, stale: false };
  }

  // L1 hit.
  const l1 = mem.get(key);
  if (l1) {
    const ageMs = t - l1.storedAt;
    if (ageMs < opts.ttlSec * 1000) return { value: l1.value as T, ageSec: ageMs / 1000, stale: false };
    if (ageMs < opts.staleSec * 1000) {
      // Stale-while-revalidate: serve the stale value, refresh off the
      // request path. A failed refresh leaves the stale entry in place.
      ctx.waitUntil(refresh().then(() => undefined, () => undefined));
      return { value: l1.value as T, ageSec: ageSec(l1, now), stale: true };
    }
  }

  // Miss (L1 empty or too old): one L2 read + at most one load, shared by
  // all concurrent misses. A load failure propagates to every waiter.
  // (loadEntry directly, NOT refresh(): the factory runs before singleFlight
  // registers it, so a nested singleFlight for the same key would see no
  // in-flight promise and chain the promise onto itself.)
  const entry = await singleFlight(key, async () => {
    const fromL2 = await l2Get(env, key);
    if (fromL2) {
      mem.set(key, fromL2);
      return fromL2;
    }
    return loadEntry(env, key, opts.staleSec, now, load);
  });
  return { value: entry.value as T, ageSec: ageSec(entry, now), stale: false };
}

// Age in seconds of an entry relative to the injectable clock.
function ageSec(entry: CacheEntry, now: () => number): number {
  return (now() - entry.storedAt) / 1000;
}
