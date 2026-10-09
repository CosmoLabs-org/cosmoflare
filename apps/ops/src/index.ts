// Cosmoflare Ops Worker (FEAT-052): serves the pager PWA (static assets)
// plus a small private API. Cloudflare Access sits in front of the whole
// hostname (only the owner's email, 30-day session); the Worker also refuses
// /api/* without the Access assertion header as defense in depth.

import { collectSummary } from "./summary";

export interface Env {
  CF_ACCOUNT_ID: string;
  CF_API_TOKEN: string;
  VAPID_PUBLIC_KEY: string;
  ASSETS: Fetcher;
}

const SUMMARY_TTL_SECONDS = 300; // GraphQL budget: at most one collection per 5 min
const SUMMARY_CACHE_KEY = "https://cosmoflare-ops.internal/api/summary";

function json(body: unknown, status = 200, extra: HeadersInit = {}): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store", ...extra },
  });
}

async function summary(env: Env, ctx: ExecutionContext, refresh: boolean): Promise<Response> {
  const cache = caches.default;
  const key = new Request(SUMMARY_CACHE_KEY);
  if (!refresh) {
    const hit = await cache.match(key);
    if (hit) return new Response(hit.body, { headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store", "X-Ops-Cache": "hit" } });
  }
  const data = await collectSummary(env.CF_ACCOUNT_ID, env.CF_API_TOKEN);
  const body = JSON.stringify(data);
  ctx.waitUntil(cache.put(key, new Response(body, { headers: { "Content-Type": "application/json", "Cache-Control": `max-age=${SUMMARY_TTL_SECONDS}` } })));
  return new Response(body, { headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store", "X-Ops-Cache": "miss" } });
}

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);
    if (!url.pathname.startsWith("/api/")) return env.ASSETS.fetch(request);

    if (!request.headers.get("Cf-Access-Jwt-Assertion")) return json({ error: "forbidden" }, 403);
    switch (url.pathname) {
      case "/api/vapid-public-key":
        return json({ key: env.VAPID_PUBLIC_KEY ?? "" });
      case "/api/summary":
        return summary(env, ctx, url.searchParams.has("refresh"));
      default:
        return json({ error: "not found" }, 404);
    }
  },
} satisfies ExportedHandler<Env>;
