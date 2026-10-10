// Domains view (FEAT-p9GD588 wave 1): every domain on the account — name,
// status, and registrar expiry with days-to-expiry attention coloring.
// Wave 2 (registrar detail: auto-renew, lock, prices) waits on Registrar
// Read for the token.

import { el, skeleton } from "./dom";
import { api, LoginExpiredError, type FetchResult, type Summary } from "./api";
import { formatCount } from "./format";

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

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

function errorState(root: HTMLElement, err: unknown): void {
  if (err instanceof LoginExpiredError) {
    root.replaceChildren(el("p", "cf-empty cf-login-expired", err.message));
    return;
  }
  const box = el("section", "cf-card cf-level-warning");
  box.append(
    el("h2", undefined, "Could not load domains"),
    el("p", "cf-row-detail", err instanceof Error ? err.message : String(err)),
  );
  const retry = el("button", "cf-btn", "Retry");
  retry.addEventListener("click", () => void renderDomains(root));
  box.append(retry);
  root.replaceChildren(box);
}

// The tapped domain, for the master-detail drill-down: set by a row tap,
// cleared by the profile's back control.
let activeDomainId: string | null = null;

function domainRow(d: DomainRecord, nowMs: number, onOpen: (d: DomainRecord) => void): HTMLElement {
  const row = el("button", "cf-domain-row") as HTMLButtonElement;
  row.type = "button";
  const name = el("span", "cf-domain-name", d.name);
  const days = d.expiresAt !== null ? daysUntil(d.expiresAt, nowMs) : null;
  const level = expiryLevel(days);
  const expiry = el("span", `cf-domain-expiry ${level ? `cf-level-${level}-text` : ""}`);
  if (days === null) {
    expiry.textContent = "external";
  } else if (days < 0) {
    expiry.textContent = `expired ${formatDate(d.expiresAt!)}`;
  } else {
    expiry.textContent = `${days}d · ${formatDate(d.expiresAt!)}`;
  }
  const badges = el("span", "cf-domain-badges");
  if (d.paused) badges.append(el("span", "cf-badge is-muted", "paused"));
  if (d.status !== "active") badges.append(el("span", "cf-badge is-muted", d.status));
  row.append(name, expiry, badges);
  row.addEventListener("click", () => onOpen(d));
  return row;
}

/** The domain profile: full-page detail with a back control. Paints from
 *  the list data immediately, then enriches from /api/domains/detail
 *  (original nameservers, DNSSEC, SSL mode, activation) when it lands. */
function domainProfile(root: HTMLElement, payload: DomainsPayload, d: DomainRecord): void {
  const back = el("button", "cf-btn cf-btn-ghost cf-profile-back", "All domains");
  back.type = "button";
  back.addEventListener("click", () => {
    activeDomainId = null;
    renderDomainsInto(root, { data: payload, source: "memory", ageSec: 0, demo: false });
  });

  const days = d.expiresAt !== null ? daysUntil(d.expiresAt) : null;
  const level = expiryLevel(days);
  const title = el("h2", "cf-profile-title", d.name);
  const eyebrow = el("p", "cf-profile-eyebrow", "Domain profile");

  const statusCard = el("section", `cf-card cf-profile-status ${level ? `cf-level-${level}` : ""}`);
  statusCard.append(el("p", "cf-profile-line", d.paused ? "Paused — Cloudflare is bypassed for this zone" : "Active — DNS resolves through Cloudflare"));
  if (days !== null) {
    statusCard.append(el("p", `cf-profile-line ${level ? `cf-level-${level}-text` : ""}`,
      days < 0 ? `Expired ${formatDate(d.expiresAt!)}` : `Renews ${formatDate(d.expiresAt!)} — ${days}d`));
  } else {
    statusCard.append(el("p", "cf-profile-line", "Registered at an external registrar — renewal happens there"));
  }

  // Sectioned fields: Registration / DNS / Security. The detail fetch adds
  // its rows into the same grid when it resolves.
  const field = (label: string, value: string): HTMLElement => {
    const f = el("div", "cf-domain-field");
    f.append(el("span", "cf-domain-field-label", label), el("span", "cf-domain-field-value", value));
    return f;
  };
  const section = (heading: string, ...rows: (HTMLElement | null)[]): HTMLElement => {
    const card = el("section", "cf-card cf-profile-section");
    card.append(el("h3", "cf-profile-section-title", heading));
    const grid = el("div", "cf-profile-fields");
    grid.append(...rows.filter((r): r is HTMLElement => r !== null));
    card.append(grid);
    return card;
  };

  const regCard = section("Registration",
    field("Plan", d.plan || "—"),
    field("Setup", d.type === "partial" ? "Partial (CNAME)" : "Full (DNS on Cloudflare)"),
    d.createdOn ? field("Added to Cloudflare", formatDate(d.createdOn)) : null,
  );
  const dnsCard = section("DNS",
    field("Nameservers", d.nameServers && d.nameServers.length > 0 ? d.nameServers.join("\n") : "—"),
    d.modifiedOn ? field("Last change", formatDate(d.modifiedOn)) : null,
  );
  const secCard = section("Security", d.developmentMode ? field("Development mode", "on") : null);

  // Renewal & traffic: expiration, days away, and the zone's 24h request
  // total joined from the cached summary by zone name (soft: absent when
  // the summary has not loaded).
  const renewCard = section("Renewal",
    field("Expiration", d.expiresAt !== null ? formatDate(d.expiresAt) : "External registrar"),
    days !== null
      ? field("Days away", days < 0 ? `${-days} days ago` : days === 0 ? "today" : `${days} days`)
      : null,
  );
  void api.fetchJson<Summary>("api/summary")
    .then((res) => {
      const z = res.data.zones.find((x) => x.zone === d.name);
      if (z) renewCard.querySelector<HTMLElement>(".cf-profile-fields")?.append(field("Traffic (24h)", formatCount(z.total)));
    })
    .catch(() => undefined);

  const view = el("div", "cf-profile");
  view.append(back, eyebrow, title, statusCard, renewCard, regCard, dnsCard, secCard);
  root.replaceChildren(view);
  root.scrollIntoView({ block: "start" });

  // Enrichment: original NS, activation, DNSSEC, SSL — from the cached
  // per-zone endpoint; failures leave the base profile untouched.
  void api.fetchJson<DomainDetailPayload>(`api/domains/detail?id=${encodeURIComponent(d.id)}`)
    .then((res) => {
      const z = res.data.zone;
      const dnsFields = dnsCard.querySelector<HTMLElement>(".cf-profile-fields");
      const regFields = regCard.querySelector<HTMLElement>(".cf-profile-fields");
      const secFields = secCard.querySelector<HTMLElement>(".cf-profile-fields");
      if (z.originalNameServers && z.originalNameServers.length > 0 && dnsFields) {
        // Historical: where DNS lived BEFORE the move to Cloudflare (live
        // check 2026-10-10: all 43 zones' current NS are *.ns.cloudflare.com;
        // 19 came from DreamHost). Worded so it never reads as current.
        dnsFields.append(field("Previous nameservers (before Cloudflare)", z.originalNameServers.join("\n")));
      }
      if (z.activatedOn && regFields) regFields.append(field("Activated", formatDate(z.activatedOn)));
      if (z.ownerType && regFields) regFields.append(field("Owner", z.ownerType));
      if (z.dnssecStatus && secFields) secFields.append(field("DNSSEC", z.dnssecStatus));
      if (z.sslMode && secFields) secFields.append(field("SSL mode", z.sslMode));
    })
    .catch(() => undefined);
}

export async function renderDomains(root: HTMLElement, opts: { refresh?: boolean } = {}): Promise<void> {
  if (!opts.refresh) root.replaceChildren(skeleton());
  try {
    const res = await api.fetchJson<DomainsPayload>("api/domains", {
      refresh: opts.refresh,
      onRevalidate: (fresh) => renderDomainsInto(root, { data: fresh, source: "network", ageSec: 0, demo: false }),
    });
    renderDomainsInto(root, res);
  } catch (err) {
    if (opts.refresh) throw err;
    errorState(root, err);
  }
}

function renderDomainsInto(root: HTMLElement, res: FetchResult<DomainsPayload>): void {
  const d = res.data;
  if (activeDomainId !== null) {
    const active = d.domains.find((x) => x.id === activeDomainId);
    if (active) {
      domainProfile(root, d, active);
      return;
    }
    activeDomainId = null;
  }
  const heading = el("h2", undefined, "Domains");
  const nowMs = Date.now();
  const sorted = sortDomains(d.domains);
  const expiring = sorted.filter((x) => {
    const days = x.expiresAt !== null ? daysUntil(x.expiresAt, nowMs) : null;
    return days !== null && days <= 30;
  }).length;
  const sub = el("p", "cf-row-detail",
    d.errors.length > 0
      ? `Partial load — ${d.errors[0]}`
      : expiring > 0
        ? `${sorted.length} domains · ${expiring} expiring within 30 days`
        : `${sorted.length} domains`);

  const card = el("section", "cf-card");
  if (sorted.length === 0) {
    card.append(el("p", "cf-empty cf-empty-quiet", "No domains found on this account."));
  } else {
    for (const dom of sorted) card.append(domainRow(dom, nowMs, (x) => {
      activeDomainId = x.id;
      domainProfile(root, d, x);
    }));
  }
  root.replaceChildren(heading, sub, card);
}
