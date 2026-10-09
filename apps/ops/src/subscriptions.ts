// Cosmoflare Ops push subscriptions (FEAT-052): the private /api/subscribe
// endpoint handlers, backed by KV key "subs". NOTE: index.ts verifies the
// Cloudflare Access JWT (signature, aud, iss, exp) on every /api/* request
// BEFORE calling handleSubscribe — the handler itself trusts that gate, so it
// performs no authentication of its own.

import type { OpsEnv } from "./scheduled";

// SUBS_KEY is the KV key holding the StoredSubscription[] array.
export const SUBS_KEY = "subs";

// MAX_SUBS caps the array — this is a single-operator pager.
export const MAX_SUBS = 10;

// StoredSubscription is one Web Push endpoint registration, stored flat
// (same JSON keys as the Go alertspush.Subscription: endpoint, p256dh, auth).
export interface StoredSubscription {
  endpoint: string;
  p256dh: string;
  auth: string;
}

export function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store" },
  });
}

/** readSubs loads the subscription list; a missing key yields []. */
export async function readSubs(env: OpsEnv): Promise<StoredSubscription[]> {
  return (await env.OPS_KV.get(SUBS_KEY, "json")) ?? [];
}

/** writeSubs persists the subscription list. */
export async function writeSubs(env: OpsEnv, subs: StoredSubscription[]): Promise<void> {
  await env.OPS_KV.put(SUBS_KEY, JSON.stringify(subs));
}

// validateSubscription checks the browser's PushSubscription shape: an https
// endpoint plus the two client keys the aes128gcm encryption needs. Anything
// else is rejected with a reason the caller can show.
export function validateSubscription(raw: unknown): { sub: StoredSubscription } | { error: string } {
  if (typeof raw !== "object" || raw === null) return { error: "body must be a JSON object" };
  const r = raw as { endpoint?: unknown; keys?: { p256dh?: unknown; auth?: unknown } };
  if (typeof r.endpoint !== "string" || !r.endpoint.startsWith("https://")) return { error: "endpoint must be an https URL" };
  if (typeof r.keys?.p256dh !== "string" || !r.keys.p256dh) return { error: "keys.p256dh is required" };
  if (typeof r.keys?.auth !== "string" || !r.keys.auth) return { error: "keys.auth is required" };
  return { sub: { endpoint: r.endpoint, p256dh: r.keys.p256dh, auth: r.keys.auth } };
}

/**
 * handleSubscribe serves the /api/subscribe route: POST adds (replacing an
 * existing subscription with the same endpoint — keys may have been
 * refreshed), DELETE {endpoint} removes, GET returns the count. Max 10
 * subscriptions; a new endpoint beyond the cap is a 409.
 */
export async function handleSubscribe(request: Request, env: OpsEnv): Promise<Response> {
  switch (request.method) {
    case "GET": {
      const subs = await readSubs(env);
      return json({ count: subs.length });
    }
    case "POST": {
      let raw: unknown;
      try {
        raw = await request.json();
      } catch {
        return json({ error: "body must be valid JSON" }, 400);
      }
      const checked = validateSubscription(raw);
      if ("error" in checked) return json({ error: checked.error }, 400);
      const subs = await readSubs(env);
      const existing = subs.findIndex((s) => s.endpoint === checked.sub.endpoint);
      if (existing === -1 && subs.length >= MAX_SUBS) {
        return json({ error: `subscription limit reached (${MAX_SUBS}); remove one first` }, 409);
      }
      if (existing === -1) subs.push(checked.sub);
      else subs[existing] = checked.sub; // same endpoint: refresh the keys
      await writeSubs(env, subs);
      return json({ ok: true, count: subs.length }, 201);
    }
    case "DELETE": {
      let raw: unknown;
      try {
        raw = await request.json();
      } catch {
        return json({ error: "body must be valid JSON" }, 400);
      }
      const endpoint = (raw as { endpoint?: unknown } | null)?.endpoint;
      if (typeof endpoint !== "string" || !endpoint) return json({ error: "endpoint is required" }, 400);
      const subs = await readSubs(env);
      const next = subs.filter((s) => s.endpoint !== endpoint);
      await writeSubs(env, next);
      return json({ ok: true, count: next.length });
    }
    default:
      return json({ error: "method not allowed" }, 405);
  }
}
