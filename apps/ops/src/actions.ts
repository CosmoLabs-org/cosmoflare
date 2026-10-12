// Operator action buttons (operator 2026-10-12: "action buttons to make
// changes to our Cloudflare account"): the first write-path action beyond
// domain lifecycle — zone cache purge. Same contract as lifecycle.ts's
// handlers: Access + rate limit run first in index.ts, then the write-token
// gate, then the mutation with the WRITE token, then the audit entry. The
// CF API's error text surfaces verbatim — no silent failures.

import { appendAudit, cfWrite, NO_WRITE_TOKEN, parseProtectedHosts, type LifecycleEnv } from "./lifecycle";
import { cached, invalidate } from "./cache";
import { fetchZonesList } from "./summary";

const LIST_TTL = { ttlSec: 60 * 60, staleSec: 24 * 60 * 60 };

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store" },
  });
}

interface PurgeBody {
  zone?: string; // zone NAME (what the pager rows carry) or ID
}

// POST /api/zones/purge {zone} — purge_everything on one zone. Needs
// "Zone > Cache Purge" on the CF_API_TOKEN_WRITE token. Cache purge is not
// destructive (the edge re-fetches from origin), so protected hosts are NOT
// refused — every action is audited either way.
export async function handlePurge(request: Request, env: LifecycleEnv, ctx: ExecutionContext, operator: string): Promise<Response> {
  let body: PurgeBody;
  try {
    body = (await request.json()) as PurgeBody;
  } catch {
    return json({ error: "invalid JSON body" }, 400);
  }
  if (!body.zone) return json({ error: "purge needs zone (name or id)" }, 400);
  if (!env.CF_API_TOKEN_WRITE) {
    await appendAudit(env, { ts: new Date().toISOString(), action: "purge", target: body.zone, operator, ok: false, error: NO_WRITE_TOKEN });
    return json({ error: NO_WRITE_TOKEN }, 503);
  }
  // Resolve name → id through the cached zones list (the same 1h list the
  // other handlers use); a 32-hex id passes through.
  const zones = (await cached(ctx, env, "zones-list", LIST_TTL, () => fetchZonesList(env.CF_API_TOKEN, env.CF_ACCOUNT_ID))).value;
  const zone = zones.find((z) => z.id === body.zone || z.name === body.zone);
  if (!zone) return json({ error: "zone not found in the cached zone list — refresh and retry" }, 404);

  const result = await cfWrite(env, "POST", `/zones/${encodeURIComponent(zone.id)}/purge_cache`, { purge_everything: true });
  await appendAudit(env, {
    ts: new Date().toISOString(),
    action: "purge",
    target: zone.name,
    operator,
    ok: result.ok,
    ...(result.ok ? {} : { error: result.error }),
  });
  if (!result.ok) {
    // The common cause is the WRITE token missing the Cache Purge permission.
    return json({ error: result.error, hint: "the write token may be missing the 'Zone > Cache Purge' permission" }, 502);
  }
  return json({ zone: zone.name, purged: true });
}

interface DnsToggleBody {
  zone?: string; // zone NAME or id
  record?: string; // dns record id
  name?: string; // record display name (audit readability)
  paused?: boolean; // true = pause (stops resolving), false = re-enable
  override?: boolean;
}

// POST /api/dns/toggle {zone, record, paused} — pause or re-enable ONE DNS
// record without deleting it (operator action buttons wave 2). Needs "Zone >
// DNS > Edit" on the WRITE token. Pausing a record on a PROTECTED host can
// take the site down, so it refuses like zone-remove (same
// ZONE_REMOVE_OVERRIDE operator override); RE-ENABLING on a protected host
// restores service and is always allowed.
export async function handleDnsToggle(request: Request, env: LifecycleEnv, ctx: ExecutionContext, operator: string): Promise<Response> {
  let body: DnsToggleBody;
  try {
    body = (await request.json()) as DnsToggleBody;
  } catch {
    return json({ error: "invalid JSON body" }, 400);
  }
  const action = body.paused === true ? "pause" : body.paused === false ? "enable" : null;
  if (!body.zone || !body.record || action === null) {
    return json({ error: "dns toggle needs zone, record and paused (true|false)" }, 400);
  }
  const target = `${body.zone}: ${body.name ?? body.record}`;
  if (!env.CF_API_TOKEN_WRITE) {
    await appendAudit(env, { ts: new Date().toISOString(), action: "dns-toggle", target, operator, ok: false, error: NO_WRITE_TOKEN });
    return json({ error: NO_WRITE_TOKEN }, 503);
  }
  const zones = (await cached(ctx, env, "zones-list", LIST_TTL, () => fetchZonesList(env.CF_API_TOKEN, env.CF_ACCOUNT_ID))).value;
  const zone = zones.find((z) => z.id === body.zone || z.name === body.zone);
  if (!zone) return json({ error: "zone not found in the cached zone list — refresh and retry" }, 404);

  if (action === "pause" && parseProtectedHosts(env.PROTECTED_HOSTS).includes(zone.name)) {
    const overrideAllowed = body.override === true && env.ZONE_REMOVE_OVERRIDE === "1";
    if (!overrideAllowed) {
      const error = `${zone.name} is a protected host — pausing its DNS requires an operator override (ZONE_REMOVE_OVERRIDE=1 and override:true)`;
      await appendAudit(env, { ts: new Date().toISOString(), action: "dns-toggle", target, operator, ok: false, error });
      return json({ error }, 403);
    }
  }

  const result = await cfWrite(env, "PATCH", `/zones/${encodeURIComponent(zone.id)}/dns_records/${encodeURIComponent(body.record)}`, { paused: action === "pause" });
  await appendAudit(env, {
    ts: new Date().toISOString(),
    action: "dns-toggle",
    target: `${target} → ${action === "pause" ? "paused" : "active"}`,
    operator,
    ok: result.ok,
    ...(result.ok ? {} : { error: result.error }),
  });
  if (!result.ok) {
    return json({ error: result.error, hint: "the write token may be missing the 'Zone > DNS > Edit' permission" }, 502);
  }
  // The records list serves cached for 5 min — a just-toggled record must
  // read back immediately.
  await invalidate(env, `dns:${zone.id}`);
  return json({ zone: zone.name, record: body.record, paused: action === "pause" });
}
