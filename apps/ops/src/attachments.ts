// Attachments collector (FEAT-pEWA6N6 phase 1, read-only inventory): the
// custom domains attached to the account's Pages and Workers projects, plus
// a DNS record count per zone. No mutations — every sub-fetch soft-fails
// into errors (house rule), so one permission gap never blanks the view.
// Per-page/per-zone calls run sequentially per group through fetchRetry.

import { fetchRetry } from "./retry";
import { fetchZonesList, rest } from "./summary";

export interface PageDomainRow {
  hostname: string;
  status: string;
  /** SSL state when the API reports one (validation_data/ssl). */
  ssl: string | null;
  /** Verification status when the API reports one (validation_data). */
  verification: string | null;
}

export interface PagesProjectAttachments {
  project: string;
  domains: PageDomainRow[];
}

export interface WorkerDomainRow {
  hostname: string;
  service: string;
  environment: string;
}

export interface ZoneAttachment {
  id: string;
  name: string;
  status: string;
  /** DNS record count (result_info.total_count); null when the count
   *  sub-fetch failed — the zone still shows, the count is the gap. */
  recordCount: number | null;
}

export interface AttachmentsPayload {
  generatedAt: string;
  pages: PagesProjectAttachments[];
  workers: WorkerDomainRow[];
  zones: ZoneAttachment[];
  errors: string[];
}

type PagesProject = { name: string };
type PagesDomain = {
  name: string;
  status?: string;
  created_on?: string;
  validation_data?: { status?: string; ssl?: { status?: string } };
  ssl?: { status?: string };
};
type WorkerCustomDomain = { hostname: string; service: string; environment: string };

/** Collect the attachments inventory: (a) Pages projects then each
 *  project's custom domains, (b) account-level Workers custom domains,
 *  (c) one dns_records?per_page=1 probe per zone for result_info.total_count.
 *  Every sub-fetch soft-fails into errors; the payload never throws. */
export async function collectAttachments(accountId: string, token: string, now = new Date()): Promise<AttachmentsPayload> {
  const out: AttachmentsPayload = { generatedAt: now.toISOString(), pages: [], workers: [], zones: [], errors: [] };

  // One soft GET: parses the v4 envelope, failures land in out.errors.
  const getEnvelope = async <T>(label: string, path: string): Promise<{ result: T; resultInfo?: { totalCount?: number } } | null> => {
    try {
      const res = await fetchRetry(`https://api.cloudflare.com/client/v4${path}`, { headers: { Authorization: `Bearer ${token}` } });
      const body = (await res.json()) as { success: boolean; result: T; errors?: { message: string }[]; result_info?: { total_count?: number } };
      if (!body.success) throw new Error(body.errors?.[0]?.message ?? `HTTP ${res.status}`);
      return { result: body.result, resultInfo: body.result_info ? { totalCount: body.result_info.total_count } : undefined };
    } catch (err) {
      out.errors.push(`${label}: ${err instanceof Error ? err.message : String(err)}`);
      return null;
    }
  };

  // (a) Pages projects, then each project's custom domains — sequential.
  try {
    const projects = await rest<PagesProject>(token, `/accounts/${accountId}/pages/projects`);
    for (const p of projects) {
      const got = await getEnvelope<PagesDomain[]>(`pages/${p.name}`, `/accounts/${accountId}/pages/projects/${p.name}/domains`);
      out.pages.push({
        project: p.name,
        domains: (got?.result ?? []).map((d) => ({
          hostname: d.name,
          status: d.status ?? "",
          ssl: d.validation_data?.ssl?.status ?? d.ssl?.status ?? null,
          verification: d.validation_data?.status ?? null,
        })),
      });
    }
  } catch (err) {
    out.errors.push(`pages projects: ${err instanceof Error ? err.message : String(err)}`);
  }

  // (b) Account-level Workers custom domains.
  try {
    const domains = await rest<WorkerCustomDomain>(token, `/accounts/${accountId}/workers/domains`);
    out.workers = domains.map((d) => ({ hostname: d.hostname, service: d.service, environment: d.environment }));
  } catch (err) {
    out.errors.push(`workers domains: ${err instanceof Error ? err.message : String(err)}`);
  }

  // (c) Per-zone DNS record counts: per_page=1, total_count is the answer.
  try {
    const zones = await fetchZonesList(token, accountId);
    for (const z of zones) {
      const got = await getEnvelope<unknown[]>(`zone ${z.name}`, `/zones/${z.id}/dns_records?per_page=1`);
      out.zones.push({ id: z.id, name: z.name, status: z.status, recordCount: got?.resultInfo?.totalCount ?? null });
    }
  } catch (err) {
    out.errors.push(`zones: ${err instanceof Error ? err.message : String(err)}`);
  }

  return out;
}
