// Domains collector (FEAT-p9GD588 wave 1): the account's full zone list —
// every status, with expiry — shaped for the pager's domains view. Unlike
// summary.ts's fetchZonesList (active-only, analytics-scoped), this keeps
// paused/pending zones and the registrar-managed expires_at field.

import { rest } from "./summary";

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
