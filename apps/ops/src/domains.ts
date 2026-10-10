// Domains collector (FEAT-p9GD588 wave 1): the account's full zone list —
// every status, with expiry — shaped for the pager's domains view. Unlike
// summary.ts's fetchZonesList (active-only, analytics-scoped), this keeps
// paused/pending zones and the registrar-managed expires_at field.

import { rest } from "./summary";
import { fetchRetry } from "./retry";

export interface DomainRecord {
  id: string;
  name: string;
  status: string;
  paused: boolean;
  /** Registrar expiry (ISO date) for Cloudflare-registered domains; null
   *  when the zone's DNS is managed here but the domain lives at another
   *  registrar. */
  expiresAt: string | null;
  /** Zone setup type: "full" (DNS here) or "partial" (CNAME setup). */
  type: string;
  /** Plan display name, e.g. "Free Website". */
  plan: string;
  developmentMode: boolean;
  createdOn: string;
  modifiedOn: string;
  /** Assigned Cloudflare nameservers. */
  nameServers: string[];
}

export interface Domains {
  generatedAt: string;
  domains: DomainRecord[];
  errors: string[];
}

/** Fetch every zone on the account (all statuses, pagination handled by
 *  rest) with the detail fields the tap-to-expand sheet shows — the list
 *  response already carries them, so no per-zone calls. */
export async function fetchDomainsList(token: string, accountId: string): Promise<DomainRecord[]> {
  const zones = await rest<{
    id: string;
    name: string;
    status: string;
    paused: boolean;
    expires_at: string | null;
    type: string;
    plan?: { name?: string };
    development_mode?: number | boolean;
    created_on: string;
    modified_on: string;
    name_servers?: string[];
  }>(token, `/zones?account.id=${accountId}`);
  return zones.map((z) => ({
    id: z.id,
    name: z.name,
    status: z.status,
    paused: Boolean(z.paused),
    expiresAt: z.expires_at ?? null,
    type: z.type,
    plan: z.plan?.name ?? "",
    developmentMode: Boolean(z.development_mode),
    createdOn: z.created_on,
    modifiedOn: z.modified_on,
    nameServers: z.name_servers ?? [],
  }));
}

/** Collect the domains payload; a list failure becomes an errors entry,
 *  never a failed response (house rule). */
export async function collectDomains(accountId: string, token: string, now = new Date()): Promise<Domains> {
  try {
    const domains = await fetchDomainsList(token, accountId);
    return { generatedAt: now.toISOString(), domains, errors: [] };
  } catch (err) {
    return { generatedAt: now.toISOString(), domains: [], errors: [err instanceof Error ? err.message : String(err)] };
  }
}

export interface DomainDetail {
  zone: DomainRecord & {
    originalNameServers: string[];
    activatedOn: string | null;
    ownerType: string;
    dnssecStatus: string | null;
    sslMode: string | null;
  };
  errors: string[];
}

/** Per-domain profile data (operator ask 2026-10-10): the zone detail plus
 *  DNSSEC and SSL status — three GETs, each soft-failed into errors so one
 *  permission gap never blanks the profile. */
export async function collectDomainDetail(token: string, zoneId: string): Promise<DomainDetail> {
  const errors: string[] = [];
  const get = async <T>(path: string): Promise<T | null> => {
    try {
      const res = await fetchRetry(`https://api.cloudflare.com/client/v4${path}`, { headers: { Authorization: `Bearer ${token}` } });
      const body = (await res.json()) as { success: boolean; result: T; errors?: { message: string }[] };
      if (!body.success) throw new Error(body.errors?.[0]?.message ?? `HTTP ${res.status}`);
      return body.result;
    } catch (err) {
      errors.push(err instanceof Error ? err.message : String(err));
      return null;
    }
  };
  const [zone, dnssec, ssl] = await Promise.all([
    get<{ id: string; name: string; status: string; paused: boolean; expires_at: string | null; type: string; plan?: { name?: string }; development_mode?: number | boolean; created_on: string; modified_on: string; name_servers?: string[]; original_name_servers?: string[] | null; activated_on?: string | null; owner?: { type?: string } }>(`/zones/${zoneId}`),
    get<{ status: string }>(`/zones/${zoneId}/dnssec`),
    get<{ value: string }>(`/zones/${zoneId}/settings/ssl`),
  ]);
  return {
    zone: {
      id: zone?.id ?? zoneId,
      name: zone?.name ?? "",
      status: zone?.status ?? "",
      paused: Boolean(zone?.paused),
      expiresAt: zone?.expires_at ?? null,
      type: zone?.type ?? "",
      plan: zone?.plan?.name ?? "",
      developmentMode: Boolean(zone?.development_mode),
      createdOn: zone?.created_on ?? "",
      modifiedOn: zone?.modified_on ?? "",
      nameServers: zone?.name_servers ?? [],
      originalNameServers: zone?.original_name_servers ?? [],
      activatedOn: zone?.activated_on ?? null,
      ownerType: zone?.owner?.type ?? "",
      dnssecStatus: dnssec?.status ?? null,
      sslMode: ssl?.value ?? null,
    },
    errors,
  };
}
