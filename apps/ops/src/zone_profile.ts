// Zone profile collector (profiles program instance 4): one zone's
// metadata, SSL mode, and DNSSEC — shaped for the pager's zone profile
// view. Follows the house rule every collector obeys: each section fails
// independently — a failure becomes an errors[] line and an omitted
// section, never a failed response and never an invented number.

import { fetchRetry } from "./retry";
import { rest } from "./summary";
import type { OpsEnv } from "./scheduled";

const API = "https://api.cloudflare.com/client/v4";

export interface ZoneMetadataSection {
  name: string;
  status: string;
  planName: string;
  paused: boolean;
  type: string;
  nameServers: string[];
  createdOn: string | null;
}

export interface ZoneDnssecSection {
  status: string | null;
  algorithm: string | null;
}

export interface ZoneProfilePayload {
  generatedAt: string;
  /** The requested zone name. */
  name: string;
  /** Resolved from the cached zone list; null when the name is unknown. */
  zoneId: string | null;
  metadata: ZoneMetadataSection | null;
  sslMode: string | null;
  dnssec: ZoneDnssecSection | null;
  /** One line per failed section — an omitted section is data absent, never invented. */
  errors: string[];
}

// The scheduled engine caches the account's zone list in KV under
// "cron:zones" ({at, value}, 1h fresh / 24h KV expiry) so its 5-min run
// costs only analytics calls. The profile resolves the requested name's id
// from that same cached list — same key, same freshness contract — instead
// of paying a second /zones page per profile view. Same constants and
// helper as worker_profile.ts's routes section (kept module-local there,
// so this is a deliberate copy, not an import).
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
    // A KV read failure must not kill the profile — fetch live below.
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
// Error the caller's catch turns into an errors[] line (worker_profile.ts's
// getSection pattern).
async function getSection<T>(token: string, path: string): Promise<T> {
  const res = await fetchRetry(`${API}${path}`, { headers: { Authorization: `Bearer ${token}` } });
  const body = (await res.json()) as { success?: boolean; result?: T; errors?: { message: string }[] };
  if (!res.ok || body.success !== true) throw new Error(body.errors?.[0]?.message ?? `HTTP ${res.status}`);
  return body.result as T;
}

export async function collectZoneProfile(env: OpsEnv, name: string, now = new Date()): Promise<ZoneProfilePayload> {
  const out: ZoneProfilePayload = { generatedAt: now.toISOString(), name, zoneId: null, metadata: null, sslMode: null, dnssec: null, errors: [] };

  // 0. Resolve the zone id from the cached zone list — without an id no
  //    section can be fetched, so a failed list or an unknown name is one
  //    errors line and an empty profile. Names compare case-insensitively
  //    (DNS names are case-insensitive; defensive against odd-case links).
  let zoneId: string | null = null;
  try {
    const zones = await zonesFromCacheOrFetch(env);
    const lower = name.toLowerCase();
    const hit = zones.find((z) => z.name.toLowerCase() === lower);
    if (hit) zoneId = hit.id;
  } catch (err) {
    out.errors.push(`zone list unavailable: ${err instanceof Error ? err.message : String(err)}`);
    return out;
  }
  if (zoneId === null) {
    out.errors.push(`zone not found: ${name}`);
    return out;
  }
  out.zoneId = zoneId;

  // 1. Metadata: GET /zones/{id} — the fields the metadata card shows
  //    (plan, status, created, paused, type, nameservers). A 404/403
  //    (zone gone, missing Zone Read) degrades to an errors line.
  try {
    const zone = await getSection<{
      name?: string;
      status?: string;
      plan?: { name?: string };
      paused?: boolean;
      type?: string;
      name_servers?: string[];
      created_on?: string | null;
    }>(env.CF_API_TOKEN, `/zones/${zoneId}`);
    out.metadata = {
      name: zone.name ?? "",
      status: zone.status ?? "",
      planName: zone.plan?.name ?? "",
      paused: Boolean(zone.paused),
      type: zone.type ?? "",
      nameServers: zone.name_servers ?? [],
      createdOn: zone.created_on ?? null,
    };
  } catch (err) {
    out.errors.push(`metadata unavailable: ${err instanceof Error ? err.message : String(err)}`);
  }

  // 2. SSL mode: GET /zones/{id}/settings/ssl → {value} — e.g. "strict".
  try {
    const ssl = await getSection<{ value?: string }>(env.CF_API_TOKEN, `/zones/${zoneId}/settings/ssl`);
    out.sslMode = ssl.value ?? null;
  } catch (err) {
    out.errors.push(`SSL mode unavailable: ${err instanceof Error ? err.message : String(err)}`);
  }

  // 3. DNSSEC: GET /zones/{id}/dnssec → {status, algorithm}. The API may
  //    answer result: null for never-enabled DNSSEC — nulls carry through,
  //    the view omits the row.
  try {
    const dnssec = await getSection<{ status?: string | null; algorithm?: string | null }>(env.CF_API_TOKEN, `/zones/${zoneId}/dnssec`);
    out.dnssec = { status: dnssec.status ?? null, algorithm: dnssec.algorithm ?? null };
  } catch (err) {
    out.errors.push(`DNSSEC unavailable: ${err instanceof Error ? err.message : String(err)}`);
  }

  return out;
}
