// Cosmoflare Ops Worker (FEAT-052): serves the pager PWA (static assets)
// plus a small private API. Cloudflare Access guard: the Worker verifies the
// Access JWT on every /api/* request (signature, aud, iss, exp), so the API
// stays closed even if the Access app is removed.

import { verifyAccessJwt } from "./access";
import { collectAttachments } from "./attachments";
import { collectAIGateways } from "./ai_gateway";
import { collectBilling } from "./billing";
import { collectEmailMetrics } from "./email";
import { cached } from "./cache";
import { collectDnsRecords, collectDomainDetail, collectDomains } from "./domains";
import { collectDurableObjects } from "./durable_objects";
import { handleAttach, auditResponse, handleDetach, handleZoneRemove, parseProtectedHosts } from "./lifecycle";
import { collectWafRules, handleDnsToggle, handlePurge, handleWafToggle } from "./actions";
import { collectSummary, fetchD1List, fetchZonesList } from "./summary";
import { collectWorkerProfile } from "./worker_profile";
import { handleWorkerTail } from "./tail";
import { collectD1Profile } from "./d1_profile";
import { collectDoProfile } from "./do_profile";
import { collectZoneProfile } from "./zone_profile";
import { collectRulesQuota } from "./rules_quota";
import { runScheduled, testFire, ALERT_FEED_KEY, type AlertFeedEntry, type OpsEnv } from "./scheduled";
import { rateLimited } from "./ratelimit";
import { handleRules } from "./rules-api";
import { handleSubscribe } from "./subscriptions";

// Env is the OpsEnv shape (OPS_KV required, VAPID keypair + contact, Access
// config, assets binding) plus the standalone billing anchor override and the
// lifecycle write-path knobs (FEAT-pEWA6N6 phases 2-3): CF_API_TOKEN_WRITE
// (optional secret — every mutation 503s without it), PROTECTED_HOSTS (the
// hard-refused zone names) and ZONE_REMOVE_OVERRIDE (the operator-override
// gate for protected zones).
export interface Env extends OpsEnv {
  BILLING_ANCHOR_DAY?: string; // "1"-"31"; unset → collectBilling's fallback chain
  CF_API_TOKEN_WRITE?: string; // lifecycle mutations (Pages/Workers/Zone Write scopes)
  PROTECTED_HOSTS?: string; // comma-separated; e.g. "cosmolabs.org,noble.coffee,cosmoflare.dev"
  ZONE_REMOVE_OVERRIDE?: string; // "1" lets a request carry override:true past the protected-host refusal
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

function json(body: unknown, status = 200, extra: Record<string, string> = {}): Response {
  return new Response(typeof body === "string" ? body : JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store", ...extra },
  });
}

// GET /api/summary — served through the upstream cache; ?refresh forces a
// reload but at most once per 60 s per key (refresh-storm guard, in cached()).
// The REST lists are cached independently (1 h / 24 h) so a summary refresh
// after the lists have been seen costs only the three GraphQL queries.
/** The shared cached D1 name list — the summary and the D1 profile endpoint
 *  ride one cache key, never a second /d1/database page per profile view. */
function cachedD1List(ctx: ExecutionContext, env: Env): Promise<{ uuid: string; name: string }[]> {
  return cached(ctx, env, "d1-list", LIST_TTL, () => fetchD1List(env.CF_API_TOKEN, env.CF_ACCOUNT_ID)).then((r) => r.value);
}

export async function summaryResponse(ctx: ExecutionContext, env: Env, refresh: boolean): Promise<Response> {
  const result = await cached(ctx, env, "summary", { ...SUMMARY_TTL, force: refresh }, () =>
    collectSummary(env.CF_ACCOUNT_ID, env.CF_API_TOKEN, new Date(), {
      zonesList: (tok, acct) => cached(ctx, env, "zones-list", LIST_TTL, () => fetchZonesList(tok, acct)).then((r) => r.value),
      d1List: () => cachedD1List(ctx, env),
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

// GET /api/durable-objects — namespaces (REST, slow-changing) + per-script
// 24h usage (analytics), through cached() at the summary cadence: 5 min
// fresh / 1 h stale.
export async function doResponse(ctx: ExecutionContext, env: Env, refresh: boolean): Promise<Response> {
  const result = await cached(ctx, env, "durable-objects", { ...SUMMARY_TTL, force: refresh }, () =>
    collectDurableObjects(env.CF_ACCOUNT_ID, env.CF_API_TOKEN, new Date()),
  );
  return json({ ...result.value, cache: { ageSec: Math.round(result.ageSec), stale: result.stale } });
}

// GET /api/attachments — the read-only custom-domain inventory (FEAT-pEWA6N6
// phase 1: Pages per-project domains, Workers custom domains, per-zone DNS
// record counts), through the same cached() contract as /api/domains:
// 1 h fresh / 24 h stale, ?refresh behind the 60 s storm guard.
export async function attachmentsResponse(ctx: ExecutionContext, env: Env, refresh: boolean): Promise<Response> {
  const result = await cached(ctx, env, "attachments", { ...LIST_TTL, force: refresh }, () =>
    collectAttachments(env.CF_ACCOUNT_ID, env.CF_API_TOKEN, new Date(), { protectedHosts: parseProtectedHosts(env.PROTECTED_HOSTS) }),
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

    // Hard cap first (operator directive 2026-10-11, BUG-pZKJ5VB): the
    // per-isolate per-IP rate limit runs before the Access JWT check because
    // it is cheaper than signature verification — a flooding client is
    // rejected without any crypto work. Best-effort only (per-isolate state,
    // see ratelimit.ts): the edge WAF rate-limiting rule remains the real
    // global backstop.
    const limit = rateLimited(request);
    if (!limit.ok) return json({ error: "rate limited" }, 429, { "Retry-After": String(limit.retryAfterSec) });

    const auth = await verifyAccessJwt(request.headers.get("Cf-Access-Jwt-Assertion") ?? "", env.ACCESS_TEAM_DOMAIN, env.ACCESS_AUD);
    if (!auth.ok) return json({ error: "forbidden" }, 403);
    // The verified identity email doubles as the lifecycle audit operator
    // (FEAT-pEWA6N6 phase 2): every mutation attempt records who did it.
    const operator = auth.email;

    switch (url.pathname) {
      case "/api/vapid-public-key":
        return json({ key: env.VAPID_PUBLIC_KEY ?? "" });
      case "/api/summary":
        return summaryResponse(ctx, env, url.searchParams.has("refresh"));
      case "/api/durable-objects":
        return doResponse(ctx, env, url.searchParams.has("refresh"));
      case "/api/ai": {
        // BR-02 (ROAD-112 phase 1): per-gateway model usage with the
        // KV-cached field probe, at the summary cadence (5 min / 1 h).
        const result = await cached(ctx, env, "ai", { ttlSec: 5 * 60, staleSec: 60 * 60, force: url.searchParams.has("refresh") }, () =>
          collectAIGateways(env.CF_ACCOUNT_ID, env.CF_API_TOKEN, new Date(), env.OPS_KV),
        );
        return json({ ...result.value, cache: { ageSec: Math.round(result.ageSec), stale: result.stale } });
      }
      case "/api/email": {
        // FEAT-059 (ROAD-107): per-zone Email Routing status, operator-gated
        // (a missing scope degrades to per-zone error strings), at the
        // /api/ai cadence (5 min fresh / 1 h stale).
        const result = await cached(ctx, env, "email", { ttlSec: 5 * 60, staleSec: 60 * 60, force: url.searchParams.has("refresh") }, () =>
          collectEmailMetrics(env.CF_ACCOUNT_ID, env.CF_API_TOKEN, new Date()),
        );
        return json({ ...result.value, cache: { ageSec: Math.round(result.ageSec), stale: result.stale } });
      }
      case "/api/domains":
        return domainsResponse(ctx, env, url.searchParams.has("refresh"));
      case "/api/attachments":
        return attachmentsResponse(ctx, env, url.searchParams.has("refresh"));
      // Lifecycle mutations (FEAT-pEWA6N6 phases 2-3): the rate-limit guard
      // and the Access check above run first for ALL of these, exactly as
      // for the reads; the handlers add their own write-token gate + audit.
      case "/api/domains/detach":
        if (request.method !== "POST") return json({ error: "method not allowed" }, 405);
        return handleDetach(request, env, ctx, operator);
      case "/api/domains/attach":
        if (request.method !== "POST") return json({ error: "method not allowed" }, 405);
        return handleAttach(request, env, ctx, operator);
      case "/api/zones/remove":
        if (request.method !== "POST") return json({ error: "method not allowed" }, 405);
        return handleZoneRemove(request, env, ctx, operator);
      case "/api/zones/purge": // POST purge_everything on one zone (operator action button)
        if (request.method !== "POST") return json({ error: "method not allowed" }, 405);
        return handlePurge(request, env, ctx, operator);
      case "/api/dns/records": { // GET one zone's DNS records (operator action buttons wave 2)
        if (request.method !== "GET") return json({ error: "method not allowed" }, 405);
        const zoneId = url.searchParams.get("id");
        if (!zoneId) return json({ error: "missing id" }, 400);
        // 5 min fresh / 1 h stale: records change on operator edits, and a
        // toggle invalidates its zone's key immediately.
        const result = await cached(ctx, env, `dns:${zoneId}`, { ttlSec: 5 * 60, staleSec: 60 * 60 }, () =>
          collectDnsRecords(env.CF_API_TOKEN, zoneId),
        );
        return json({ ...result.value, cache: { ageSec: Math.round(result.ageSec), stale: result.stale } });
      }
      case "/api/dns/toggle": // POST pause/enable one DNS record (operator action button)
        if (request.method !== "POST") return json({ error: "method not allowed" }, 405);
        return handleDnsToggle(request, env, ctx, operator);
      case "/api/waf/rules": { // GET one zone's WAF custom rules (action buttons wave 3)
        if (request.method !== "GET") return json({ error: "method not allowed" }, 405);
        const zoneId = url.searchParams.get("id");
        if (!zoneId) return json({ error: "missing id" }, 400);
        const result = await cached(ctx, env, `waf:${zoneId}`, { ttlSec: 5 * 60, staleSec: 60 * 60 }, () =>
          collectWafRules(env.CF_API_TOKEN, zoneId),
        );
        return json({ ...result.value, cache: { ageSec: Math.round(result.ageSec), stale: result.stale } });
      }
      case "/api/waf/toggle": // POST enable/disable one WAF custom rule (operator action button)
        if (request.method !== "POST") return json({ error: "method not allowed" }, 405);
        return handleWafToggle(request, env, ctx, operator);
      case "/api/audit": // GET the mutation audit log, newest first
        if (request.method !== "GET") return json({ error: "method not allowed" }, 405);
        return auditResponse(env);
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
      case "/api/worker-profile": {
        // /api/worker-profile?name=<script> — one script's configuration
        // (bindings, crons, routes, versions), cached 5 min per script like
        // the other per-entity detail endpoints.
        const name = url.searchParams.get("name");
        if (!name) return json({ error: "missing name" }, 400);
        const result = await cached(ctx, env, `worker-profile:${name}`, { ttlSec: 5 * 60, staleSec: 60 * 60 }, () =>
          collectWorkerProfile(env, name),
        );
        return json({ ...result.value, cache: { ageSec: Math.round(result.ageSec), stale: result.stale } });
      }
      case "/api/worker-tail": {
        // GET /api/worker-tail?name=<script> — one BOUNDED live-tail stream
        // (NDJSON): per-isolate cap of one active tail, 30 s duration cutoff,
        // 200-event cutoff (see tail.ts — the operator's paid-plan budget
        // rule). Never cached; every stream pays its own mint POST. The
        // Access check and rate limit above run first, exactly as for reads.
        if (request.method !== "GET") return json({ error: "method not allowed" }, 405);
        return handleWorkerTail(request, env, url.searchParams.get("name"));
      }
      case "/api/d1-profile": {
        // /api/d1-profile?name=<db> — one database's REST metadata (created,
        // version, region, tables, file size, read replication), cached 5 min
        // per database. The name->uuid lookup rides the shared cached
        // d1-list, so a profile view costs one metadata GET.
        const name = url.searchParams.get("name");
        if (!name) return json({ error: "missing name" }, 400);
        const result = await cached(ctx, env, `d1-profile:${name}`, { ttlSec: 5 * 60, staleSec: 60 * 60 }, () =>
          collectD1Profile(env, name, { d1List: () => cachedD1List(ctx, env) }),
        );
        return json({ ...result.value, cache: { ageSec: Math.round(result.ageSec), stale: result.stale } });
      }
      case "/api/do-profile": {
        // /api/do-profile?name=<namespace script> — one DO namespace's 24h
        // analytics, cached 5 min per namespace.
        const name = url.searchParams.get("name");
        if (!name) return json({ error: "missing name" }, 400);
        const result = await cached(ctx, env, `do-profile:${name}`, { ttlSec: 5 * 60, staleSec: 60 * 60 }, () =>
          collectDoProfile(env.CF_ACCOUNT_ID, env.CF_API_TOKEN, name),
        );
        return json({ ...result.value, cache: { ageSec: Math.round(result.ageSec), stale: result.stale } });
      }
      case "/api/zone-profile": {
        // /api/zone-profile?name=<zone> — one zone's metadata, SSL mode and
        // DNSSEC status, cached 5 min per zone.
        const name = url.searchParams.get("name");
        if (!name) return json({ error: "missing name" }, 400);
        const result = await cached(ctx, env, `zone-profile:${name}`, { ttlSec: 5 * 60, staleSec: 60 * 60 }, () =>
          collectZoneProfile(env, name),
        );
        return json({ ...result.value, cache: { ageSec: Math.round(result.ageSec), stale: result.stale } });
      }
      case "/api/rules-quota": {
        // /api/rules-quota?name=<zone> — per-family rule inventory against
        // the plan's free allowances (ROAD-p9X66J9). The collector caches
        // 5 min per zone in KV itself; rulesets families degrade to
        // authorized:false without the Zone Rulesets scope.
        const name = url.searchParams.get("name");
        if (!name) return json({ error: "missing name" }, 400);
        return json(await collectRulesQuota(env, name));
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
      case "/api/alerts/feed": // GET the fired-alert history, newest first — the pager's Notifications center reads this
        if (request.method !== "GET") return json({ error: "method not allowed" }, 405);
        return json({ feed: (await env.OPS_KV.get<AlertFeedEntry[]>(ALERT_FEED_KEY, "json")) ?? [] });
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
