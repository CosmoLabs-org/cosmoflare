// Cosmoflare Ops Worker (FEAT-052): serves the pager PWA (static assets)
// plus a small private API. Cloudflare Access sits in front of the whole
// hostname (only the owner's email, 30-day session); the Worker verifies the
// Access JWT on every /api/* request as well (signature, aud, iss, exp), so
// the API stays closed even if the Access app is removed.

import { verifyAccessJwt } from "./access";
import { cached, type CacheEnv } from "./cache";
import { collectSummary, fetchD1List, fetchZonesList } from "./summary";

export interface Env extends CacheEnv {
  CF_ACCOUNT_ID: string;
  CF_API_TOKEN: string;
  VAPID_PUBLIC_KEY: string;
  ACCESS_TEAM_DOMAIN: string; // e.g. "team.cloudflareaccess.com"
  ACCESS_AUD: string; // the Access application's audience tag
  ASSETS: Fetcher;
}

// Upstream caching (FEAT-052): /api/summary and the REST lists it reads go
// through cached() — L1 module memo, L2 KV when OPS_KV is bound, else the
// Cache API. The Cache API IS functional on Workers Custom Domains, but the
// docs state: "For Workers fronted by Cloudflare Access, the Cache API is
// not currently available"
// (https://developers.cloudflare.com/workers/runtime-apis/cache/, checked
// 2026-10-09). This hostname is Access-fronted, so the Cache-API layer is a
// best-effort no-op here and KV is the real L2 once OPS_KV is bound.
// TTLs (docs/planning-mode/2026-10-09-ops-billing-ui-caching.md): summary is
// 24h analytics — fresh 5 min, stale 1 h; zone/D1 lists change on a days
// timescale — fresh 1 h, stale 24 h.
const SUMMARY_TTL = { ttlSec: 5 * 60, staleSec: 60 * 60 };
const LIST_TTL = { ttlSec: 60 * 60, staleSec: 24 * 60 * 60 };

function json(body: unknown, status = 200): Response {
  return new Response(typeof body === "string" ? body : JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store" },
  });
}

// GET /api/summary — served through the upstream cache; ?refresh forces a
// reload but at most once per 60 s per key (refresh-storm guard, in cached()).
// The REST lists are cached independently (1 h / 24 h) so a summary refresh
// after the lists have been seen costs only the three GraphQL queries.
export async function summaryResponse(ctx: ExecutionContext, env: Env, refresh: boolean): Promise<Response> {
  const result = await cached(ctx, env, "summary", { ...SUMMARY_TTL, force: refresh }, () =>
    collectSummary(env.CF_ACCOUNT_ID, env.CF_API_TOKEN, new Date(), {
      zonesList: (tok, acct) => cached(ctx, env, "zones-list", LIST_TTL, () => fetchZonesList(tok, acct)).then((r) => r.value),
      d1List: (tok, acct) => cached(ctx, env, "d1-list", LIST_TTL, () => fetchD1List(tok, acct)).then((r) => r.value),
    }),
  );
  // cache metadata is filled here from the cache result, per the v2 contract.
  return json({ ...result.value, cache: { ageSec: Math.round(result.ageSec), stale: result.stale } });
}

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);
    if (!url.pathname.startsWith("/api/")) return env.ASSETS.fetch(request);

    const auth = await verifyAccessJwt(request.headers.get("Cf-Access-Jwt-Assertion") ?? "", env.ACCESS_TEAM_DOMAIN, env.ACCESS_AUD);
    if (!auth.ok) return json({ error: "forbidden" }, 403);

    switch (url.pathname) {
      case "/api/vapid-public-key":
        return json({ key: env.VAPID_PUBLIC_KEY ?? "" });
      case "/api/summary":
        return summaryResponse(ctx, env, url.searchParams.has("refresh"));
      default:
        return json({ error: "not found" }, 404);
    }
  },
} satisfies ExportedHandler<Env>;
