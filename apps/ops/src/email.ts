// Email Routing collector (FEAT-059, ROAD-107): per-zone Email Routing
// status from /zones/{id}/email/routing. The endpoint is operator-gated like
// Billing Read — verified live 2026-10-10 it returns code 10000 without the
// Email Routing Read scope — so every failure is a per-zone error string,
// never a throw: once the token gains the scope the zones fill in, and until
// then the payload is errors-only and the pager stays silent.

import { fetchRetry } from "./retry";
import { fetchZonesList } from "./summary";

const API = "https://api.cloudflare.com/client/v4";

export interface EmailZoneRow {
  zone: string;
  enabled: boolean;
  name?: string;
}

export interface EmailRoutingPayload {
  generatedAt: string;
  zones: EmailZoneRow[];
  errors: string[];
}

export async function collectEmailMetrics(accountId: string, token: string, now: Date): Promise<EmailRoutingPayload> {
  const out: EmailRoutingPayload = { generatedAt: now.toISOString(), zones: [], errors: [] };

  // 1. Zone list (same paginated REST loader the summary uses). A failed
  //    list means nothing is collectable — one error string, no throw.
  let zones: { id: string; name: string }[] = [];
  try {
    zones = await fetchZonesList(token, accountId);
  } catch (err) {
    out.errors.push(`zone list unavailable: ${err instanceof Error ? err.message : String(err)}`);
    return out;
  }

  // 2. Per-zone Email Routing status, each independent: a 403 (code 10000,
  //    missing Email Routing Read) degrades to that zone's error string.
  await Promise.all(zones.map(async (z) => {
    try {
      const res = await fetchRetry(`${API}/zones/${z.id}/email/routing`, { headers: { Authorization: `Bearer ${token}` } });
      const body = (await res.json()) as { success?: boolean; result?: { enabled?: boolean; name?: string } };
      if (!res.ok || body.success !== true) throw new Error(`HTTP ${res.status}`);
      out.zones.push({ zone: z.name, enabled: body.result?.enabled === true, name: body.result?.name || undefined });
    } catch (err) {
      out.errors.push(`email routing unavailable for ${z.name}: ${err instanceof Error ? err.message : String(err)}`);
    }
  }));

  out.zones.sort((a, b) => a.zone.localeCompare(b.zone));
  return out;
}
