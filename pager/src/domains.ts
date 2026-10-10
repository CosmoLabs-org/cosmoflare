// Domains view (FEAT-p9GD588 wave 1): every domain on the account — name,
// status, and registrar expiry with days-to-expiry attention coloring.
// Wave 2 (registrar detail: auto-renew, lock, prices) waits on Registrar
// Read for the token.

import { el, skeleton, statusLine } from "./dom";
import { api, LoginExpiredError, totalAgeSec, type FetchResult } from "./api";
import { refreshButton } from "./refresh";

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

function domainRow(d: DomainRecord, nowMs: number): HTMLElement {
  // One item: the tappable summary row plus the smooth-expanding detail
  // sheet beneath it. All fields come from the zones list itself — no
  // per-domain upstream call.
  const item = el("div", "cf-domain-item");

  const row = el("button", "cf-domain-row") as HTMLButtonElement;
  row.type = "button";
  row.setAttribute("aria-expanded", "false");
  const name = el("div", "cf-domain-name");
  name.append(el("strong", undefined, d.name));
  const badges = el("span", "cf-domain-badges");
  if (d.paused) badges.append(el("span", "cf-badge is-muted", "paused"));
  if (d.status !== "active") badges.append(el("span", "cf-badge is-muted", d.status));
  name.append(badges);

  const days = d.expiresAt !== null ? daysUntil(d.expiresAt, nowMs) : null;
  const level = expiryLevel(days);
  const expiry = el("span", `cf-domain-expiry ${level ? `cf-level-${level}-text` : ""}`);
  if (days === null) {
    expiry.textContent = "external registrar";
  } else if (days < 0) {
    expiry.textContent = `expired ${formatDate(d.expiresAt!)}`;
  } else {
    expiry.textContent = `${formatDate(d.expiresAt!)} · ${days === 0 ? "today" : `${days}d`}`;
  }
  const chevron = el("span", "cf-domain-chevron", "›");
  row.append(name, expiry, chevron);
  row.addEventListener("click", () => {
    const open = item.classList.toggle("is-open");
    row.setAttribute("aria-expanded", String(open));
  });

  // Detail sheet: grid-rows 0fr→1fr transition (see styles) for the smooth
  // expand; the inner wrapper carries the overflow clip.
  const detail = el("div", "cf-domain-detail");
  const inner = el("div", "cf-domain-detail-inner");
  const field = (label: string, value: string): HTMLElement => {
    const f = el("div", "cf-domain-field");
    f.append(el("span", "cf-domain-field-label", label), el("span", "cf-domain-field-value", value));
    return f;
  };
  inner.append(
    field("Registration", days === null ? "External registrar — renewal happens there" : `Cloudflare Registrar · renews ${formatDate(d.expiresAt!)}`),
    field("Plan", d.plan || "—"),
    field("Setup", d.type === "partial" ? "Partial (CNAME setup)" : "Full (DNS on Cloudflare)"),
  );
  if (d.nameServers && d.nameServers.length > 0) inner.append(field("Nameservers", d.nameServers.join(", ")));
  if (d.developmentMode) inner.append(field("Development mode", "on"));
  if (d.createdOn) inner.append(field("Added", formatDate(d.createdOn)));
  if (d.modifiedOn) inner.append(field("Last change", formatDate(d.modifiedOn)));
  detail.append(inner);

  item.append(row, detail);
  return item;
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
  const bar = el("div", "cf-dash-bar");
  const line = statusLine(totalAgeSec(res.ageSec, d.cache), { cached: Boolean(d.cache?.stale), demo: res.demo });
  const refresh = refreshButton(() => renderDomains(root, { refresh: true }));
  bar.append(line, refresh);

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
        : `${sorted.length} domains · none expiring within 30 days`);

  const card = el("section", "cf-card");
  if (sorted.length === 0) {
    card.append(el("p", "cf-empty cf-empty-quiet", "No domains found on this account."));
  } else {
    for (const dom of sorted) card.append(domainRow(dom, nowMs));
  }
  root.replaceChildren(bar, heading, sub, card);
}
