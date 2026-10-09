// Billing view (FEAT-052): billing period progress, per-product allowance
// meters (used solid, projected hatched, allowance marker), expandable top
// consumers, projects sorted by projected cost, pricing footnote, telemetry
// gaps notice. Data: /api/billing via the shared ApiClient.

import { el, skeleton, statusLine } from "./dom";
import { formatCount, formatPct, formatUsd, formatDateShort, formatAge, periodProgress } from "./format";
import { api, LoginExpiredError, type Billing, type FetchResult, type ProductUsage } from "./api";

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

// productRow renders one product's meter card. The meter: a track scaled to
// fit projected (so an over-allowance projection is visible), solid fill up
// to used, hatched fill from used to projected, and a marker line at the
// included allowance.
function productRow(p: ProductUsage): HTMLElement {
  const m = meterPct(p);
  const card = el("section", `cf-card cf-product ${p.projectedOverageUsd > 0 ? "cf-over" : ""}`);
  const head = el("div", "cf-product-head");
  head.append(
    el("strong", "cf-product-name", `${p.product} — ${p.metric}`),
    el("span", `cf-product-overage ${p.projectedOverageUsd > 0 ? "cf-level-critical-text" : "cf-ok-text"}`,
      p.projectedOverageUsd > 0 ? `+${formatUsd(p.projectedOverageUsd)} overage` : "within allowance"),
  );

  const meter = el("div", "cf-meter");
  const track = el("div", "cf-meter-track");
  const used = el("div", "cf-meter-used");
  used.style.width = `${m.usedPct}%`;
  const projected = el("div", "cf-meter-projected");
  projected.style.width = `${Math.max(0, m.projectedPct - m.usedPct)}%`;
  projected.style.insetInlineStart = `${m.usedPct}%`;
  const marker = el("div", "cf-meter-marker");
  marker.style.insetInlineStart = `${m.includedPct}%`;
  track.append(used, projected, marker);

  const legend = el("div", "cf-meter-legend");
  legend.append(
    el("span", "cf-legend-swatch cf-swatch-used", "used"),
    el("span", "cf-legend-swatch cf-swatch-projected", "projected"),
    el("span", "cf-legend-swatch cf-swatch-allowance", "allowance"),
  );

  const detail = el("p", "cf-row-detail",
    `${formatCount(p.used)} of ${formatCount(p.included)} ${p.unit} used · projected ${formatCount(p.projected)} (${formatPct(p.included > 0 ? (p.projected / p.included) * 100 : 0, 0)} of allowance)`);
  const consumers = el("details", "cf-consumers");
  const summary = el("summary", "cf-consumers-summary", `Top consumers (${p.topConsumers.length})`);
  const list = el("ul", "cf-consumers-list");
  for (const c of p.topConsumers) {
    const li = el("li", "cf-consumer");
    li.append(
      el("span", "cf-consumer-name", c.name),
      el("span", "cf-consumer-project", c.project),
      el("span", "cf-consumer-share", formatPct(c.share * 100, 0)),
    );
    list.append(li);
  }
  consumers.append(summary, list);
  meter.append(track, legend);
  card.append(head, meter, detail, consumers);
  return card;
}

export async function renderBilling(root: HTMLElement, opts: { refresh?: boolean } = {}): Promise<void> {
  if (!opts.refresh) root.replaceChildren(skeleton());
  try {
    const res = await api.fetchJson<Billing>("api/billing", { refresh: opts.refresh });
    renderBillingInto(root, res);
  } catch (err) {
    errorState(root, err);
  }
}

// renderBillingInto paints one Billing result into the view host.
function renderBillingInto(root: HTMLElement, res: FetchResult<Billing>): void {
  const b = res.data;

  // Status bar + refresh
  const bar = el("div", "cf-dash-bar");
  const serverAge = b.cache ? b.cache.ageSec : null;
  const line = statusLine(res.ageSec, { cached: Boolean(b.cache?.stale), demo: res.demo });
  if (serverAge !== null) {
    line.textContent = `Updated ${formatAge(serverAge)}${b.cache?.stale ? " · cached" : ""}`;
  }
  const refresh = el("button", "cf-btn", "Refresh");
  refresh.addEventListener("click", () => void renderBilling(root, { refresh: true }));
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
  for (const p of b.products) products.append(productRow(p));

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
  root.replaceChildren(bar, period, products);
  if (b.errors.length) {
    const gaps = el("section", "cf-card cf-level-warning");
    gaps.append(el("h2", undefined, "Telemetry gaps"));
    for (const e of b.errors) gaps.append(el("p", "cf-row-detail", e));
    root.append(gaps);
  }
  root.append(projectsCard, pricing);
}
