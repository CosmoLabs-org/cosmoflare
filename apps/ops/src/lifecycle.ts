// Domain & zone lifecycle write path (FEAT-pEWA6N6 phases 2-3): the guarded
// mutations behind the pager's Domains view — reversible Pages/Workers
// custom-domain detach + re-attach, and the hard-gated destructive zone
// removal. Dry-run preview WAIVED by the operator (2026-10-11).
//
// Safety model:
// - Every mutation runs on CF_API_TOKEN_WRITE (optional secret). Unset → the
//   mutation answers 503 and nothing touches the Cloudflare API; reads keep
//   using CF_API_TOKEN.
// - Zone removal is server-side truth only: the zone is resolved from the
//   cached zones list, the typed confirmation must equal the zone name
//   EXACTLY, and zones named on PROTECTED_HOSTS are refused unless the
//   request carries override:true AND the ZONE_REMOVE_OVERRIDE var is "1".
// - Every mutation attempt (including refusals and the 503 no-token case)
//   appends to the KV 'audit' log, newest first, capped at AUDIT_CAP.
// - Successful mutations invalidate the affected cached() keys so the next
//   read refetches.
//
// Workers note (Cloudflare docs, verified 2026-10-12): deleting a Workers
// custom domain does NOT auto-delete its advanced certificate — surfaced in
// the UI detach success state for workers rows.

import { cached, invalidate, type CacheEnv } from "./cache";
import { fetchZonesList } from "./summary";

const API = "https://api.cloudflare.com/client/v4";

// Same contract as index.ts's LIST_TTL: the cached zones list read for zone
// removal / workers attach resolves server-side truth at 1 h fresh / 24 h
// stale. Kept local to avoid an import cycle with index.ts.
const LIST_TTL = { ttlSec: 60 * 60, staleSec: 24 * 60 * 60 };

export const AUDIT_KEY = "audit";
export const AUDIT_CAP = 200; // cap the array at the 200 newest entries

export const NO_WRITE_TOKEN = "write token not configured — set CF_API_TOKEN_WRITE";

export interface LifecycleEnv extends CacheEnv {
  CF_ACCOUNT_ID: string;
  CF_API_TOKEN: string; // reads (cached zones list)
  CF_API_TOKEN_WRITE?: string; // mutations; unset → 503
  /** Comma-separated protected hostnames (wrangler.jsonc var). */
  PROTECTED_HOSTS?: string;
  /** Set to "1" to allow override:true through the protected-host refusal. */
  ZONE_REMOVE_OVERRIDE?: string;
}

export interface AuditEntry {
  ts: string; // ISO
  action: string; // detach | attach | zone-remove
  target: string; // hostname / project/name / zone name
  operator: string; // email from the Access JWT
  ok: boolean;
  error?: string; // verbatim failure reason, absent on success
}

/** "a.b, c ,d" → ["a.b","c","d"] — trims and drops empties. */
export function parseProtectedHosts(raw?: string): string[] {
  return (raw ?? "")
    .split(",")
    .map((h) => h.trim())
    .filter((h) => h.length > 0);
}

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store" },
  });
}

// ---- Audit log (KV 'audit', newest first, capped) ----

export async function readAudit(env: LifecycleEnv): Promise<AuditEntry[]> {
  if (!env.OPS_KV) return [];
  const raw = await env.OPS_KV.get(AUDIT_KEY);
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as unknown;
    return Array.isArray(parsed) ? (parsed as AuditEntry[]) : [];
  } catch {
    return [];
  }
}

/** Read-modify-write append; the stored array is newest-first and capped. */
export async function appendAudit(env: LifecycleEnv, entry: AuditEntry): Promise<void> {
  if (!env.OPS_KV) return; // audit is best-effort; never blocks the mutation
  const list = await readAudit(env);
  list.unshift(entry);
  await env.OPS_KV.put(AUDIT_KEY, JSON.stringify(list.slice(0, AUDIT_CAP)));
}

// GET /api/audit — the audit log, newest first.
export async function auditResponse(env: LifecycleEnv): Promise<Response> {
  return json({ audit: await readAudit(env) });
}

// ---- Cloudflare mutation call ----

type WriteResult = { ok: true } | { ok: false; error: string };

// One v4 mutation call with the WRITE token. On failure the API's first
// error message is surfaced VERBATIM (no silent failures); non-JSON bodies
// degrade to "HTTP <status>".
export async function cfWrite(env: LifecycleEnv, method: string, path: string, body?: unknown): Promise<WriteResult> {
  let res: Response;
  try {
    res = await fetch(`${API}${path}`, {
      method,
      headers: {
        Authorization: `Bearer ${env.CF_API_TOKEN_WRITE ?? ""}`,
        ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
      },
      ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
    });
  } catch (err) {
    return { ok: false, error: err instanceof Error ? err.message : String(err) };
  }
  let parsed: { success?: boolean; errors?: { message?: string }[] } | null = null;
  try {
    parsed = (await res.json()) as { success?: boolean; errors?: { message?: string }[] };
  } catch {
    // non-JSON body — fall through to the status check
  }
  if (!res.ok || parsed?.success === false) {
    const message = parsed?.errors?.[0]?.message;
    return { ok: false, error: message ?? `HTTP ${res.status}` };
  }
  return { ok: true };
}

interface DetachBody {
  kind?: string;
  project?: string;
  name?: string;
  domainId?: string;
}

// POST /api/domains/detach {kind:'pages'|'workers', project?, name, domainId?}
// - pages:   DELETE /accounts/{id}/pages/projects/{project}/domains/{name}   (Pages Write)
// - workers: DELETE /accounts/{id}/workers/domains/{domainId} — BY ID, not
//            hostname                                                          (Workers Scripts Write)
// (_ctx is part of the handler signature for symmetry with attach/remove,
// which read the cached zones list through it.)
export async function handleDetach(request: Request, env: LifecycleEnv, _ctx: ExecutionContext, operator: string): Promise<Response> {
  let body: DetachBody;
  try {
    body = (await request.json()) as DetachBody;
  } catch {
    return json({ error: "invalid JSON body" }, 400);
  }
  if (!env.CF_API_TOKEN_WRITE) {
    await appendAudit(env, { ts: new Date().toISOString(), action: "detach", target: body.name ?? "?", operator, ok: false, error: NO_WRITE_TOKEN });
    return json({ error: NO_WRITE_TOKEN }, 503);
  }
  let path: string;
  let target: string;
  if (body.kind === "pages") {
    if (!body.project || !body.name) return json({ error: "pages detach needs project and name" }, 400);
    path = `/accounts/${env.CF_ACCOUNT_ID}/pages/projects/${encodeURIComponent(body.project)}/domains/${encodeURIComponent(body.name)}`;
    target = `${body.project}/${body.name}`;
  } else if (body.kind === "workers") {
    if (!body.domainId) return json({ error: "workers detach needs domainId (the detach endpoint is by id, not hostname)" }, 400);
    path = `/accounts/${env.CF_ACCOUNT_ID}/workers/domains/${encodeURIComponent(body.domainId)}`;
    target = body.name ?? body.domainId;
  } else {
    return json({ error: "kind must be 'pages' or 'workers'" }, 400);
  }
  const result = await cfWrite(env, "DELETE", path);
  await appendAudit(env, {
    ts: new Date().toISOString(),
    action: "detach",
    target,
    operator,
    ok: result.ok,
    ...(result.ok ? {} : { error: result.error }),
  });
  if (!result.ok) return json({ error: result.error }, 502);
  // Success: drop the read caches so the next view refetches.
  await invalidate(env, "attachments");
  await invalidate(env, "domains");
  return json({ ok: true, target });
}

interface AttachBody {
  kind?: string;
  project?: string;
  name?: string;
  service?: string;
}

// POST /api/domains/attach {kind, project?, name, service?}
// - pages:   POST /accounts/{id}/pages/projects/{project}/domains  body {name}
// - workers: PUT /accounts/{id}/workers/domains body {hostname, service,
//   zone_id} — live API shape (OpenAPI schema verified 2026-10-12; the
//   collection attach is a PUT upsert, `environment` is readOnly/deprecated,
//   and the required members are hostname + service). zone_id is resolved
//   from the cached zones list by longest-suffix match on the hostname.
export async function handleAttach(request: Request, env: LifecycleEnv, ctx: ExecutionContext, operator: string): Promise<Response> {
  let body: AttachBody;
  try {
    body = (await request.json()) as AttachBody;
  } catch {
    return json({ error: "invalid JSON body" }, 400);
  }
  if (!env.CF_API_TOKEN_WRITE) {
    await appendAudit(env, { ts: new Date().toISOString(), action: "attach", target: body.name ?? "?", operator, ok: false, error: NO_WRITE_TOKEN });
    return json({ error: NO_WRITE_TOKEN }, 503);
  }
  if (body.kind === "pages") {
    if (!body.project || !body.name) return json({ error: "pages attach needs project and name" }, 400);
    const result = await cfWrite(env, "POST", `/accounts/${env.CF_ACCOUNT_ID}/pages/projects/${encodeURIComponent(body.project)}/domains`, { name: body.name });
    await appendAudit(env, {
      ts: new Date().toISOString(),
      action: "attach",
      target: `${body.project}/${body.name}`,
      operator,
      ok: result.ok,
      ...(result.ok ? {} : { error: result.error }),
    });
    if (!result.ok) return json({ error: result.error }, 502);
    await invalidate(env, "attachments");
    await invalidate(env, "domains");
    return json({ ok: true, target: `${body.project}/${body.name}` });
  }
  if (body.kind === "workers") {
    if (!body.name || !body.service) return json({ error: "workers attach needs name (hostname) and service" }, 400);
    const zones = (
      await cached(ctx, env, "zones-list", LIST_TTL, () => fetchZonesList(env.CF_API_TOKEN, env.CF_ACCOUNT_ID))
    ).value;
    const host = body.name.toLowerCase();
    const zone = zones
      .filter((z) => host === z.name || host.endsWith(`.${z.name}`))
      .sort((a, b) => b.name.length - a.name.length)[0];
    if (!zone) {
      const error = `no zone found for ${body.name} — the hostname's zone must exist in this account`;
      await appendAudit(env, { ts: new Date().toISOString(), action: "attach", target: body.name, operator, ok: false, error });
      return json({ error }, 400);
    }
    const result = await cfWrite(env, "PUT", `/accounts/${env.CF_ACCOUNT_ID}/workers/domains`, {
      hostname: body.name,
      service: body.service,
      zone_id: zone.id,
    });
    await appendAudit(env, {
      ts: new Date().toISOString(),
      action: "attach",
      target: body.name,
      operator,
      ok: result.ok,
      ...(result.ok ? {} : { error: result.error }),
    });
    if (!result.ok) return json({ error: result.error }, 502);
    await invalidate(env, "attachments");
    await invalidate(env, "domains");
    return json({ ok: true, target: body.name });
  }
  return json({ error: "kind must be 'pages' or 'workers'" }, 400);
}

interface ZoneRemoveBody {
  zoneId?: string;
  typedName?: string;
  override?: boolean;
}

// POST /api/zones/remove {zoneId, typedName, override} — the destructive,
// hard-gated path:
// 1. the zone is resolved from the CACHED ZONES LIST (server-side truth
//    only — never trust a client-supplied zone name);
// 2. refuse 403 unless typedName === zone.name EXACTLY;
// 3. refuse 403 when the zone name is on PROTECTED_HOSTS unless
//    override === true AND env.ZONE_REMOVE_OVERRIDE === "1";
// 4. only then DELETE /zones/{zoneId} (Zone Write).
export async function handleZoneRemove(request: Request, env: LifecycleEnv, ctx: ExecutionContext, operator: string): Promise<Response> {
  let body: ZoneRemoveBody;
  try {
    body = (await request.json()) as ZoneRemoveBody;
  } catch {
    return json({ error: "invalid JSON body" }, 400);
  }
  if (!env.CF_API_TOKEN_WRITE) {
    await appendAudit(env, { ts: new Date().toISOString(), action: "zone-remove", target: body.zoneId ?? "?", operator, ok: false, error: NO_WRITE_TOKEN });
    return json({ error: NO_WRITE_TOKEN }, 503);
  }
  if (!body.zoneId || typeof body.typedName !== "string") {
    return json({ error: "zone remove needs zoneId and typedName" }, 400);
  }
  const zones = (
    await cached(ctx, env, "zones-list", LIST_TTL, () => fetchZonesList(env.CF_API_TOKEN, env.CF_ACCOUNT_ID))
  ).value;
  const zone = zones.find((z) => z.id === body.zoneId);
  const refuse = async (error: string): Promise<Response> => {
    await appendAudit(env, {
      ts: new Date().toISOString(),
      action: "zone-remove",
      target: zone?.name ?? body.zoneId ?? "?",
      operator,
      ok: false,
      error,
    });
    return json({ error }, 403);
  };
  if (!zone) {
    return json({ error: "zone not found in the cached zone list — refresh and retry" }, 404);
  }
  if (body.typedName !== zone.name) {
    return refuse(`typed name does not match — type "${zone.name}" exactly to confirm removal`);
  }
  const protectedHosts = parseProtectedHosts(env.PROTECTED_HOSTS);
  if (protectedHosts.includes(zone.name)) {
    const overrideAllowed = body.override === true && env.ZONE_REMOVE_OVERRIDE === "1";
    if (!overrideAllowed) {
      return refuse(
        `${zone.name} is a protected host — removal requires an operator override (ZONE_REMOVE_OVERRIDE=1 and override:true)`,
      );
    }
  }
  const result = await cfWrite(env, "DELETE", `/zones/${encodeURIComponent(zone.id)}`);
  await appendAudit(env, {
    ts: new Date().toISOString(),
    action: "zone-remove",
    target: zone.name,
    operator,
    ok: result.ok,
    ...(result.ok ? {} : { error: result.error }),
  });
  if (!result.ok) return json({ error: result.error }, 502);
  // The zone is gone: every list that mentions it must refetch.
  await invalidate(env, "zones-list");
  await invalidate(env, "domains");
  await invalidate(env, "attachments");
  return json({ ok: true, target: zone.name });
}
