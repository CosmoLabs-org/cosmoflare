// Domains helpers (FEAT-p9GD588 wave 1): the pure layer behind the React
// domains view — the payload types, days-to-expiry math and the
// soonest-expiry-first sort. Wave 2 (registrar detail: auto-renew, lock,
// prices) waits on Registrar Read for the token. The imperative vanilla
// renderer that used to live here was retired (TASK-pD575FN);
// pager/src/app/views/DomainsView.tsx is the only render path now.

export interface DomainRecord {
  id: string;
  name: string;
  status: string;
  paused: boolean;
  expiresAt: string | null;
  type?: string;
  plan?: string;
  developmentMode?: boolean;
  createdOn?: string;
  modifiedOn?: string;
  nameServers?: string[];
}

export interface DomainsPayload {
  generatedAt: string;
  domains: DomainRecord[];
  errors: string[];
  cache?: { ageSec: number; stale: boolean };
}

/** /api/domains/detail?id= — the enriched profile payload. */
export interface DomainDetailPayload {
  zone: DomainRecord & {
    originalNameServers?: string[];
    activatedOn?: string | null;
    ownerType?: string;
    dnssecStatus?: string | null;
    sslMode?: string | null;
  };
  errors: string[];
}

/** One DNS record row (operator action buttons wave 2, 2026-10-12):
 *  /api/dns/records?id=. `paused` is the pause lever — a paused record
 *  stops resolving without being deleted. */
export interface DnsRecord {
  id: string;
  type: string;
  name: string;
  content: string;
  proxied: boolean;
  paused: boolean;
  ttl: number;
}

/** /api/dns/records?id= response. */
export interface DnsRecordsPayload {
  records: DnsRecord[];
  error?: string | null;
  cache?: { ageSec: number; stale: boolean };
}

/** One WAF custom rule (action buttons wave 3, 2026-10-12): the zone's
 *  custom-firewall phase ruleset. */
export interface WafRule {
  id: string;
  description: string;
  expression: string;
  action: string; // block / skip / log / challenge …
  enabled: boolean;
}

/** /api/waf/rules?id= response. */
export interface WafRulesPayload {
  rulesetId: string | null;
  rules: WafRule[];
  error?: string | null;
  cache?: { ageSec: number; stale: boolean };
}

// Attachment inventory (FEAT-pEWA6N6 phase 1): mirrors the ops worker's
// attachments.ts payload — read-only, no mutations.
export interface PageDomainAttachment {
  hostname: string;
  status: string;
  ssl?: string | null;
  verification?: string | null;
}

export interface PagesProjectAttachments {
  project: string;
  domains: PageDomainAttachment[];
}

export interface WorkerDomainAttachment {
  hostname: string;
  service: string;
  environment: string;
  /** Immutable domain id — the Workers detach endpoint is BY ID, not by
   *  hostname (FEAT-pEWA6N6 phase 2); absent when the worker omitted it. */
  id?: string;
}

export interface ZoneAttachment {
  id: string;
  name: string;
  status: string;
  recordCount: number | null;
  /** True when the zone name is on the operator's protected-hosts list —
   *  removal hard-refuses without an operator override. */
  protected?: boolean;
}

// Lifecycle audit log (FEAT-pEWA6N6 phase 3): GET /api/audit returns the
// entries newest-first, capped at the 200 newest on the worker side.
export interface AuditEntry {
  ts: string;
  action: string;
  target: string;
  operator: string;
  ok: boolean;
  error?: string;
}

export interface AuditPayload {
  audit: AuditEntry[];
}

export interface AttachmentsPayload {
  generatedAt: string;
  pages: PagesProjectAttachments[];
  workers: WorkerDomainAttachment[];
  zones: ZoneAttachment[];
  errors: string[];
  cache?: { ageSec: number; stale: boolean };
}

/** Whole days from `now` to an ISO date (negative once past). */
export function daysUntil(iso: string, nowMs: number = Date.now()): number {
  return Math.floor((Date.parse(iso) - nowMs) / 86_400_000);
}

/** Attention level for an expiry: expiring inside 14 days (or already
 *  past) is critical, inside 30 warning, otherwise ok. Null expiry (domain
 *  at another registrar) carries no level. */
export function expiryLevel(days: number | null): "critical" | "warning" | "ok" | null {
  if (days === null) return null;
  if (days <= 14) return "critical";
  if (days <= 30) return "warning";
  return "ok";
}

/** Soonest expiry first; domains without expiry sort last, by name. */
export function sortDomains<T extends { name: string; expiresAt: string | null }>(domains: T[]): T[] {
  return [...domains].sort((a, b) => {
    const da = a.expiresAt !== null ? Date.parse(a.expiresAt) : null;
    const db = b.expiresAt !== null ? Date.parse(b.expiresAt) : null;
    if (da !== null && db !== null) return da - db || a.name.localeCompare(b.name);
    if (da !== null) return -1;
    if (db !== null) return 1;
    return a.name.localeCompare(b.name);
  });
}
