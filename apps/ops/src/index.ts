// Cosmoflare Ops Worker (FEAT-052): serves the pager PWA (static assets)
// plus a small private API. Cloudflare Access sits in front of the whole
// hostname (only the owner's email, 30-day session); the Worker verifies the
// Access JWT on every /api/* request as well (signature, aud, iss, exp), so
// the API stays closed even if the Access app is removed.

import { verifyAccessJwt } from "./access";
import { collectSummary } from "./summary";

export interface Env {
  CF_ACCOUNT_ID: string;
  CF_API_TOKEN: string;
  VAPID_PUBLIC_KEY: string;
  ACCESS_TEAM_DOMAIN: string; // e.g. "team.cloudflareaccess.com"
  ACCESS_AUD: string; // the Access application's audience tag
  ASSETS: Fetcher;
}

// GraphQL budget: at most one collection per 5 minutes per isolate. The
// Cache API is not available to Workers fronted by Access, so the summary
// is memoized in module scope instead.
const SUMMARY_TTL_MS = 5 * 60 * 1000;
let memo: { at: number; body: string } | null = null;

function json(body: unknown, status = 200): Response {
  return new Response(typeof body === "string" ? body : JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store" },
  });
}

async function summary(env: Env, refresh: boolean): Promise<Response> {
  if (!refresh && memo && Date.now() - memo.at < SUMMARY_TTL_MS) return json(memo.body);
  const body = JSON.stringify(await collectSummary(env.CF_ACCOUNT_ID, env.CF_API_TOKEN));
  memo = { at: Date.now(), body };
  return json(body);
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);
    if (!url.pathname.startsWith("/api/")) return env.ASSETS.fetch(request);

    const auth = await verifyAccessJwt(request.headers.get("Cf-Access-Jwt-Assertion") ?? "", env.ACCESS_TEAM_DOMAIN, env.ACCESS_AUD);
    if (!auth.ok) return json({ error: "forbidden" }, 403);

    switch (url.pathname) {
      case "/api/vapid-public-key":
        return json({ key: env.VAPID_PUBLIC_KEY ?? "" });
      case "/api/summary":
        return summary(env, url.searchParams.has("refresh"));
      default:
        return json({ error: "not found" }, 404);
    }
  },
} satisfies ExportedHandler<Env>;
