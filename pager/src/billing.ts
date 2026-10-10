// Billing view (FEAT-052): billing period progress, per-product allowance
// bar gauges (FEAT-pDQ8JET — used solid, projected lighter, allowance +
// today markers, level badge), expandable top consumers, projects sorted by
// projected cost, pricing footnote, telemetry gaps notice. Data: /api/billing
// via the shared ApiClient.

import { el, skeleton, statusLine } from "./dom";
import { formatAmount, formatPct, formatUsd, formatDateShort, periodProgress } from "./format";
import { api, LoginExpiredError, totalAgeSec, type Billing, type BillingPeriod, type FetchResult, type ProductUsage } from "./api";
import { barGauge, productLabel, scaleFitPct } from "./gauges";
import { refreshButton } from "./refresh";

// meterPct computes the meter geometry for one product row: solid used
// width, hatched projected extension, and the allowance marker position —
// all as percentages of a scale that fits both used and projected.
export function meterPct(p: ProductUsage): { usedPct: number; projectedPct: number; includedPct: number } {
  const scale = Math.max(p.included, p.projected, p.used, 1e-9);
  return {
    usedPct: (p.used / scale) * 100,
    projectedPct: (p.projected / scale) * 100,
    includedPct: (p.included / scale) * 100,
  };
}

function errorState(root: HTMLElement, err: unknown): void {
  if (err instanceof LoginExpiredError) {
    root.replaceChildren(el("p", "cf-empty cf-login-expired", err.message));
    return;
  }
  const box = el("section", "cf-card cf-level-warning");
  box.append(
    el("h2", undefined, "Could not load billing"),
    el("p", "cf-row-detail", err instanceof Error ? err.message : String(err)),
  );
  const retry = el("button", "cf-btn", "Retry");
  retry.addEventListener("click", () => void renderBilling(root));
  box.append(retry);
  root.replaceChildren(box);
}

/**
 * IMP-002: ONE status per billing card — the gauge's level badge is that
 * status. The head carries only the projected cost beyond the allowance,
 * and only when there is one; a zero-overage card says nothing here (the
 * badge already reads ok/near/over).
 */
export function overageHeadText(p: ProductUsage): string | null {
  return p.projectedOverageUsd > 0 ? `+${formatUsd(p.projectedOverageUsd)} overage` : null;
}

// productRow renders one product's bar-gauge card: name + overage USD head,
// the gauge (solid used fill, projected fill, allowance + today markers,
// level badge, text row), then the expandable top consumers.
function productRow(p: ProductUsage, period: BillingPeriod): HTMLElement {
  const card = el("section", `cf-card cf-product ${p.projectedOverageUsd > 0 ? "cf-over" : ""}`);
  const head = el("div", "cf-product-head");
  head.append(el("strong", "cf-product-name", productLabel(p.id, p.product, p.metric)));
  const overage = overageHeadText(p);
  if (overage) head.append(el("span", "cf-product-overage cf-level-critical-text", overage));

  // Bar percentages are of the fitted track scale (max of used/projected/
  // included) so an over-allowance projection stays on the track; the level
  // and badge run on the % of allowance. The today marker sits at the
  // expected-to-date use (allowance × period elapsed).
  const scale = Math.max(p.used, p.projected, p.included, 1e-9);
  const gauge = barGauge({
    usedPct: p.included > 0 ? (p.used / p.included) * 100 : 0,
    projectedPct: p.included > 0 ? (p.projected / p.included) * 100 : 0,
    usedScalePct: scaleFitPct(p.used, scale),
    projectedScalePct: scaleFitPct(p.projected, scale),
    includedScalePct: scaleFitPct(p.included, scale),
    expectedPct: scaleFitPct(p.included * (period.day / period.days), scale),
    label: "",
    detailText: `${formatAmount(p.used, p.unit)} of ${formatAmount(p.included, p.unit)} used · projected ${formatAmount(p.projected, p.unit)} (${formatPct(p.included > 0 ? (p.projected / p.included) * 100 : 0, 0)} of allowance)`,
  });

  const consumers = el("details", "cf-consumers");
  const summary = el("summary", "cf-consumers-summary", `Top consumers (${p.topConsumers.length})`);
  const list = el("ol", "cf-consumers-list cf-consumers-ranked");
  p.topConsumers.forEach((c, i) => {
    // Ranking rows (operator 2026-10-10): rank number, name + project,
    // share bar and percentage on the right — the share reads at a glance
    // instead of a flat three-column list.
    const li = el("li", "cf-consumer");
    li.append(el("span", "cf-consumer-rank", String(i + 1)));
    const who = el("span", "cf-consumer-who");
    who.append(el("span", "cf-consumer-name", c.name), el("span", "cf-consumer-project", c.project));
    const share = el("span", "cf-consumer-sharewrap");
    const bar = el("span", "cf-consumer-sharebar");
    const fill = el("span", "cf-consumer-sharefill");
    fill.style.width = `${Math.min(100, Math.max(0, c.share * 100))}%`;
    bar.append(fill);
    share.append(bar, el("span", "cf-consumer-share", formatPct(c.share * 100, 0)));
    li.append(who, share);
    list.append(li);
  });
  consumers.append(summary, list);
  card.append(head, gauge, consumers);
  return card;
}

export async function renderBilling(root: HTMLElement, opts: { refresh?: boolean } = {}): Promise<void> {
  if (!opts.refresh) root.replaceChildren(skeleton());
  try {
    const res = await api.fetchJson<Billing>("api/billing", {
      refresh: opts.refresh,
      // A background revalidation repaints the view with the fresh copy
      // (BUG-057 defect 2) instead of leaving hours-old data on screen.
      onRevalidate: (fresh) => renderBillingInto(root, { data: fresh, source: "network", ageSec: 0, demo: false }),
    });
    renderBillingInto(root, res);
  } catch (err) {
    // A failed refresh rethrows so the clicked Refresh button keeps its node
    // and surfaces the error itself; good data is never replaced by an error
    // card (BUG-057 defect 3). Only a first (skeleton) load shows the error.
    if (opts.refresh) throw err;
    errorState(root, err);
  }
}

// billingLegend is the one-line key under the period card: solid = used,
// hatched = projected, line = allowance, dot = today. The bar gauges repeat
// these encodings; the legend spells them out once.
/** Rolling-bytes metrics (BUG-pFAKDN3): kv.storage's number is a rolling
 *  cumulative, not live size — it must never drive allowance visuals
 *  (rings, over-limit cards, attention) even though the payload carries it
 *  for awareness. */
export function isRollingStorage(id: string): boolean {
  return id === "kv.storage";
}

export function billingLegend(): HTMLElement {
  const row = el("div", "cf-legend");
  row.setAttribute("role", "list");
  row.setAttribute("aria-label", "Gauge legend: solid used, hatched projected, line allowance, dot today");
  const item = (swatchClass: string, text: string): HTMLElement => {
    const s = el("span", "cf-legend-item");
    s.append(el("span", swatchClass), document.createTextNode(text));
    return s;
  };
  row.append(
    item("cf-legend-swatch cf-legend-used", "used"),
    item("cf-legend-swatch cf-legend-projected", "projected"),
    item("cf-legend-allowance", "allowance"),
    item("cf-legend-dot", "today"),
  );
  return row;
}

// renderBillingInto paints one Billing result into the view host.
function renderBillingInto(root: HTMLElement, res: FetchResult<Billing>): void {
  const b = res.data;

  // Status bar + refresh. The age is honest (BUG-057 defect 2): client copy
  // age + server cache age.
  const bar = el("div", "cf-dash-bar");
  const line = statusLine(totalAgeSec(res.ageSec, b.cache), { cached: Boolean(b.cache?.stale), demo: res.demo });
  const refresh = refreshButton(() => renderBilling(root, { refresh: true }));
  bar.append(line, refresh);

  // Period progress
  const period = el("section", "cf-card");
  period.append(el("h2", undefined, `Billing period — day ${b.period.day} of ${b.period.days}`));
  const prog = el("div", "cf-progress");
  const fill = el("div", "cf-progress-fill");
  fill.style.width = `${periodProgress(b.period.day, b.period.days).elapsedPct}%`;
  prog.append(fill);
  period.append(prog, el("p", "cf-row-detail",
    `${b.period.source === "calendar" ? "Calendar month" : b.period.source === "anchor" ? "Anchor day" : "Subscription"} · ends ${formatDateShort(b.period.end)} · ${b.period.days - b.period.day} full days left`));

  // Products (sorted by projected overage desc — the server sends them so)
  const products = el("div", "cf-products");
  for (const p of b.products) {
    if (isRollingStorage(p.id)) continue; // rolling bytes: no allowance card
    products.append(productRow(p, b.period));
  }
  products.append(el("p", "cf-empty cf-empty-quiet",
    "KV storage hidden: Cloudflare's dataset reports rolling bytes, not live size (BUG-pFAKDN3) — no overage is priced on it."));

  // Projects table — "who to optimize"
  const projectsCard = el("section", "cf-card");
  projectsCard.append(el("h2", undefined, "Projects by projected overage"));
  if (b.projects.length === 0) {
    projectsCard.append(el("p", "cf-empty cf-empty-quiet", "No project is projected past its allowance."));
  } else {
    const table = el("table", "cf-table");
    const thead = el("thead");
    const headerRow = el("tr");
    for (const label of ["Project", "Projected overage", "Drivers"]) headerRow.append(el("th", undefined, label));
    thead.append(headerRow);
    const tbody = el("tbody");
    for (const proj of b.projects) {
      const tr = el("tr");
      tr.append(
        el("td", undefined, proj.project),
        el("td", "cf-num", formatUsd(proj.projectedOverageUsd)),
        el("td", "cf-row-detail", proj.drivers.join(", ")),
      );
      tbody.append(tr);
    }
    table.append(thead, tbody);
    projectsCard.append(table);
  }

  // Pricing footnote
  const pricing = el("p", "cf-pricing-note");
  pricing.append("Prices verified ", b.pricing.verifiedOn, " · ");
  b.pricing.sources.forEach((s, i) => {
    const a = el("a", "cf-link", s.product);
    a.href = s.url;
    a.target = "_blank";
    a.rel = "noreferrer";
    pricing.append(a);
    if (i < b.pricing.sources.length - 1) pricing.append(", ");
  });

  // Telemetry gaps notice
  root.replaceChildren(bar, period, billingLegend(), products);
  if (b.errors.length) {
    const gaps = el("section", "cf-card cf-level-warning");
    gaps.append(el("h2", undefined, "Telemetry gaps"));
    for (const e of b.errors) gaps.append(el("p", "cf-row-detail", e));
    root.append(gaps);
  }
  root.append(projectsCard, pricing);
}
