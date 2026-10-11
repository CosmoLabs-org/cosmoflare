// Hard-cap rate limit for the Cosmoflare Ops Worker /api/* surface
// (operator directive 2026-10-11, BUG-pZKJ5VB / FEAT-pT0AFDM): at most
// MAX_REQUESTS requests per rolling WINDOW_MS per client IP, checked BEFORE
// the Access JWT verification in index.ts (cheap rejection first).
//
// The bucket map is module scope — per-isolate, exactly like cache.ts's
// `mem` (the one sanctioned request-independent global in this Worker).
// Why not KV: counting every request in KV would put a KV read+write (a
// billed operation class) on every /api call, so the "guard" would itself
// generate spend. Per-isolate state is free and needs no writes at all.
// Consequence: this is a best-effort per-isolate cap, NOT a global quota —
// the true billing backstop is an edge WAF rate-limiting rule on the zone
// (operator note: keep that rule configured; this Worker cap only stops a
// single isolate from fanning out upstream GraphQL spend).

export const WINDOW_MS = 60_000; // rolling window length (60 s)
export const MAX_REQUESTS = 120; // hard cap per IP per window

// Entries older than this many windows are pruned on every check, so a
// scanner rotating source IPs cannot grow the map without bound.
const PRUNE_WINDOWS = 2;

// One client's window: count of requests since windowStart (ms epoch).
export interface Bucket {
  count: number;
  windowStart: number;
}

export interface RateVerdict {
  ok: boolean;
  // Seconds until the client may retry; always 0 when allowed and in
  // (0, WINDOW_MS/1000] when rejected (feeds the Retry-After header).
  retryAfterSec: number;
}

// Module-scope bucket map (per-isolate; see the header for why deliberate).
const buckets = new Map<string, Bucket>();

/**
 * Pure, clock-injectable core: slide the window forward for `key`, prune
 * buckets older than PRUNE_WINDOWS windows from `map`, and return whether
 * the request is allowed. Maintains at most one small entry per active IP.
 */
export function allowed(map: Map<string, Bucket>, key: string, nowMs: number): RateVerdict {
  for (const [k, b] of map) {
    if (nowMs - b.windowStart >= PRUNE_WINDOWS * WINDOW_MS) map.delete(k);
  }
  const b = map.get(key);
  if (!b || nowMs - b.windowStart >= WINDOW_MS) {
    map.set(key, { count: 1, windowStart: nowMs });
    return { ok: true, retryAfterSec: 0 };
  }
  if (b.count < MAX_REQUESTS) {
    b.count += 1;
    return { ok: true, retryAfterSec: 0 };
  }
  const retryAfterSec = Math.max(1, Math.ceil((b.windowStart + WINDOW_MS - nowMs) / 1000));
  return { ok: false, retryAfterSec };
}

// Thin wrapper for the fetch handler: keyed by client IP. CF-Connecting-IP
// is set by Cloudflare on every request reaching the zone; the "unknown"
// fallback collapses header-less hits (e.g. local `wrangler dev`) into one
// bucket rather than letting them bypass the cap.
export function rateLimited(request: Request, nowMs: number = Date.now()): RateVerdict {
  return allowed(buckets, request.headers.get("CF-Connecting-IP") ?? "unknown", nowMs);
}
