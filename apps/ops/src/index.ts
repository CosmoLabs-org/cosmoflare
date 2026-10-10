// Cosmoflare Ops Worker (FEAT-052): serves the pager PWA (static assets)
// plus a small private API. Cloudflare Access guard: the Worker verifies the
// Access JWT on every /api/* request (signature, aud, iss, exp), so the API
// stays closed even if the Access app is removed.

import { verifyAccessJwt } from "./access";
import { collectBilling } from "./billing";
import { cached } from "./cache";
import { collectDomainDetail, collectDomains } from "./domains";
import { collectSummary, fetchD1List, fetchZonesList } from "./summary";
import { runScheduled, testFire, type OpsEnv } from "./scheduled";
import { handleRules } from "./rules-api";
import { handleSubscribe } from "./subscriptions";

// Env is the OpsEnv shape (OPS_KV required, VAPID keypair + contact, Access
// config, assets binding) plus the standalone billing anchor override.
export interface Env extends OpsEnv {
  BILLING_ANCHOR_DAY?: string; // "1"-"31"; unset → collectBilling's fallback chain
}

// Upstream caching (FEAT-052): /api/summary and /api/billing plus the REST
// lists they read go through cached() — L1 module memo, L2 KV when OPS_KV is
// bound, else the Cache API. The Cache API IS functional on Workers Custom
// Domains, but the docs state: "For Workers fronted by Cloudflare Access, the
// Cache API is not currently available"
// (https://developers.cloudflare.com/workers/runtime-apis/cache/, checked
// 2026-10-09). This hostname is Access-fronted, so the Cache-API layer is a
// best-effort no-op here and KV is the real L2 once OPS_KV is bound.
// TTLs (docs/planning-mode/2026-10-09-ops-billing-ui-caching.md, cache.ts
// header): summary is 24h analytics — fresh 5 min, stale 1 h; zone/D1 lists
// change on a days timescale — fresh 1 h, stale 24 h; MTD billing usage moves
// slowly — fresh 15 min, stale 6 h.
const SUMMARY_TTL = { ttlSec: 5 * 60, staleSec: 60 * 60 };
const LIST_TTL = { ttlSec: 60 * 60, staleSec: 24 * 60 * 60 };
const BILLING_TTL = { ttlSec: 15 * 60, staleSec: 6 * 60 * 60 };

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

// GET /api/domains — the account's full zone list (every status, registrar
// expiry kept), through the same cached() contract as the other views:
// 1h fresh / 24h stale, ?refresh behind the 60s storm guard.
export async function domainsResponse(ctx: ExecutionContext, env: Env, refresh: boolean): Promise<Response> {
  const result = await cached(ctx, env, "domains", { ...LIST_TTL, force: refresh }, () =>
    collectDomains(env.CF_ACCOUNT_ID, env.CF_API_TOKEN, new Date()),
  );
  return json({ ...result.value, cache: { ageSec: Math.round(result.ageSec), stale: result.stale } });
}

// GET /api/billing — same contract as /api/summary: served through cached()
// (15 min fresh / 6 h stale), ?refresh forces a reload behind the 60 s
// refresh-storm guard, and the response carries `cache: { ageSec, stale }`.
// BILLING_ANCHOR_DAY (optional secret/var, "1"-"31") is the subscription
// fallback: the day of month the billing period restarts.
export async function billingResponse(ctx: ExecutionContext, env: Env, refresh: boolean): Promise<Response> {
  const result = await cached(ctx, env, "billing", { ...BILLING_TTL, force: refresh }, () =>
    collectBilling(env.CF_ACCOUNT_ID, env.CF_API_TOKEN, new Date(), {
      anchorDay: Number(env.BILLING_ANCHOR_DAY) || undefined,
    }),
  );
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
      case "/api/domains":
        return domainsResponse(ctx, env, url.searchParams.has("refresh"));
      case "/api/domains/detail": {
        // /api/domains/detail?id=<zoneId> — the per-domain profile (zone
        // detail + DNSSEC + SSL), cached 1h per zone.
        const id = url.searchParams.get("id");
        if (!id) return json({ error: "missing id" }, 400);
        const result = await cached(ctx, env, `domain:${id}`, LIST_TTL, () =>
          collectDomainDetail(env.CF_API_TOKEN, id),
        );
        return json({ ...result.value, cache: { ageSec: Math.round(result.ageSec), stale: result.stale } });
      }
      case "/api/billing":
        return billingResponse(ctx, env, url.searchParams.has("refresh"));
      case "/api/subscribe": // POST add / DELETE remove / GET count — auth already checked above
        return handleSubscribe(request, env);
      case "/api/rules": // GET read / PUT replace — auth already checked above
        return handleRules(request, env);
      case "/api/test-fire": // POST only: fire a real push at every subscribed device
        if (request.method !== "POST") return json({ error: "method not allowed" }, 405);
        return json(await testFire(env));
      default:
        return json({ error: "not found" }, 404);
    }
  },

  // Cron trigger (every 5 min, wrangler.jsonc): evaluate the alert rules and
  // page every subscribed device. waitUntil keeps the isolate alive for the
  // whole run; the result line is the cron's audit log.
  async scheduled(_controller: ScheduledController, env: Env, ctx: ExecutionContext): Promise<void> {
    ctx.waitUntil(
      runScheduled(env)
        .then((r) => console.log(JSON.stringify({ cron: "alerts", ...r })))
        // A rejected runScheduled (e.g. a KV failure) must still leave an
        // audit line, not become an unhandled rejection inside waitUntil.
        .catch((error: unknown) => console.log(JSON.stringify({ cron: "alerts", error: String(error) }))),
    );
  },
} satisfies ExportedHandler<Env>;
