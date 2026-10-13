// Worker profile collector (worker profile expansion): one script's
// configuration — bindings, cron schedules, zone routes, recent versions —
// shaped for the pager's worker profile view. Follows the house rule every
// collector obeys: each section fails independently, a failure becomes an
// errors[] line and an omitted section, never a failed response and never an
// invented number.

import { fetchRetry } from "./retry";
import { rest } from "./summary";
import type { OpsEnv } from "./scheduled";

const API = "https://api.cloudflare.com/client/v4";

export interface WorkerBindingRow {
  type: string;
  name: string;
}

export interface WorkerRouteRow {
  zone: string;
  pattern: string;
}

export interface WorkerVersionRow {
  /** Deployment version number, newest first in the payload. */
  number: number;
  createdOn: string | null;
}

export interface WorkerProfilePayload {
  generatedAt: string;
  bindings: WorkerBindingRow[];
  cronTriggers: string[];
  routes: WorkerRouteRow[];
  versions: WorkerVersionRow[];
  /** One line per failed section — an omitted section is data absent, never invented. */
  errors: string[];
}

// The scheduled engine caches the account's zone list in KV under
// "cron:zones" ({at, value}, 1h fresh / 24h KV expiry) so its 5-min run
// costs only analytics calls. The routes section reuses that same cached
// list — same key, same freshness contract — instead of paying a second
// /zones page per profile view.
const ZONES_CACHE_KEY = "cron:zones";
const ZONES_TTL_MS = 60 * 60 * 1000;
// KV-side expiry matching the 1h wall-time check (KV has no read-time expiry).
const ZONES_KV_TTL_SEC = 86400;

interface CachedZones {
  at: number;
  value: { id: string; name: string }[];
}

async function zonesFromCacheOrFetch(env: OpsEnv): Promise<{ id: string; name: string }[]> {
  try {
    const cached = await env.OPS_KV.get<CachedZones>(ZONES_CACHE_KEY, "json");
    if (cached && typeof cached.at === "number" && Date.now() - cached.at < ZONES_TTL_MS) return cached.value;
  } catch {
    // A KV read failure must not kill the routes section — fetch live below.
  }
  const zones = await rest<{ id: string; name: string }>(env.CF_API_TOKEN, `/zones?account.id=${env.CF_ACCOUNT_ID}`);
  try {
    await env.OPS_KV.put(ZONES_CACHE_KEY, JSON.stringify({ at: Date.now(), value: zones }), { expirationTtl: ZONES_KV_TTL_SEC });
  } catch {
    // Cache-write failure is non-fatal; the list is still usable.
  }
  return zones;
}

// One soft-failing GET: !success or a transport error becomes a thrown
// Error the caller's catch turns into an errors[] line (collectDomainDetail's
// get() pattern).
async function getSection<T>(token: string, path: string): Promise<T> {
  const res = await fetchRetry(`${API}${path}`, { headers: { Authorization: `Bearer ${token}` } });
  const body = (await res.json()) as { success?: boolean; result?: T; errors?: { message: string }[] };
  if (!res.ok || body.success !== true) throw new Error(body.errors?.[0]?.message ?? `HTTP ${res.status}`);
  return body.result as T;
}

export async function collectWorkerProfile(env: OpsEnv, name: string, now = new Date()): Promise<WorkerProfilePayload> {
  const out: WorkerProfilePayload = { generatedAt: now.toISOString(), bindings: [], cronTriggers: [], routes: [], versions: [], errors: [] };
  const script = encodeURIComponent(name);
  const base = `/accounts/${env.CF_ACCOUNT_ID}/workers/scripts/${script}`;

  // 1. Bindings (script settings): [{type, name}] — a 404/403 (script gone,
  //    missing Workers Scripts Read) degrades to an errors line.
  try {
    const settings = await getSection<{ bindings?: { type?: string; name?: string }[] | null }>(env.CF_API_TOKEN, `${base}/settings`);
    out.bindings = (settings.bindings ?? []).map((b) => ({ type: b.type ?? "", name: b.name ?? "" }));
  } catch (err) {
    out.errors.push(`bindings unavailable: ${err instanceof Error ? err.message : String(err)}`);
  }

  // 2. Cron schedules: [{cron}]. Defensively accept result.items too.
  try {
    const schedules = await getSection<{ cron?: string }[] | { schedules?: { cron?: string }[] }>(env.CF_API_TOKEN, `${base}/schedules`);
    const rows = Array.isArray(schedules) ? schedules : (schedules.schedules ?? []);
    out.cronTriggers = rows.map((r) => r.cron ?? "").filter((c) => c !== "");
  } catch (err) {
    out.errors.push(`cron schedules unavailable: ${err instanceof Error ? err.message : String(err)}`);
  }

  // 3. Recent versions: [{id, number, metadata:{created_on}}] — keep the 5
  //    most recent by number desc. Defensively accept result.items.
  try {
    const versions = await getSection<{ number?: number; metadata?: { created_on?: string | null } }[] | { items?: { number?: number; metadata?: { created_on?: string | null } }[] }>(env.CF_API_TOKEN, `${base}/versions`);
    const rows = Array.isArray(versions) ? versions : (versions.items ?? []);
    out.versions = rows
      .map((v) => ({ number: v.number ?? 0, createdOn: v.metadata?.created_on ?? null }))
      .filter((v) => v.number > 0)
      .sort((a, b) => b.number - a.number)
      .slice(0, 5);
  } catch (err) {
    out.errors.push(`versions unavailable: ${err instanceof Error ? err.message : String(err)}`);
  }

  // 4. Zone-scoped routes: every custom domain routed to this script. The
  //    zone list rides the scheduled engine's cached one; a failed list or a
  //    failed per-zone routes call degrades to an errors line.
  try {
    const zones = await zonesFromCacheOrFetch(env);
    await Promise.all(zones.map(async (z) => {
      try {
        const routes = await getSection<{ pattern?: string; script?: string }[]>(env.CF_API_TOKEN, `/zones/${z.id}/workers/routes`);
        for (const r of routes) {
          if (r.script === name && r.pattern) out.routes.push({ zone: z.name, pattern: r.pattern });
        }
      } catch (err) {
        out.errors.push(`routes unavailable for ${z.name}: ${err instanceof Error ? err.message : String(err)}`);
        return;
      }
    }));
    out.routes.sort((a, b) => a.zone.localeCompare(b.zone) || a.pattern.localeCompare(b.pattern));
  } catch (err) {
    out.errors.push(`routes unavailable: ${err instanceof Error ? err.message : String(err)}`);
  }

  return out;
}
