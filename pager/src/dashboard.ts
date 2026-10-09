// Overview view (FEAT-052): KPI tiles (projected overage, billing-period
// progress, D1 and Workers pacing, attention count), then the merged
// "Needs attention" list. Data: /api/summary v2 + /api/billing through the
// shared ApiClient. Levels mirror the starter alert thresholds (O10) so the
// screen and the pager agree on what is "red".

import { el, skeleton, statusLine } from "./dom";
import { formatCount, formatPct, formatUsd, formatDateShort, formatAge, periodProgress } from "./format";
import { api, LoginExpiredError, type Billing, type BillingPeriod, type ProductUsage, type Summary } from "./api";
import { ATTENTION_CAP, capAttention, collectAttention } from "./attention";
import { gaugeLevel, ringGauge, sortByProjectedDesc, type RingGaugeProps } from "./gauges";
import { hrefFor } from "./routes";

export type Level = "ok" | "warning" | "critical";

export const levelForUncached = (n: number): Level => (n >= 10_000 ? "critical" : n >= 5_000 ? "warning" : "ok");
export const levelForMiss = (pct: number | null): Level => (pct === null ? "ok" : pct >= 80 ? "critical" : pct >= 50 ? "warning" : "ok");
export const levelForD1 = (rows24h: number): Level => (rows24h >= 1e9 ? "critical" : rows24h >= 1e8 ? "warning" : "ok");
export const levelForUsage = (projectedPct: number | null): Level =>
  projectedPct === null ? "ok" : projectedPct >= 100 ? "critical" : projectedPct >= 80 ? "warning" : "ok";
// Workers error rate: 1% is "look at this", 5% is "act now".
export const levelForErrorPct = (pct: number): Level => (pct >= 5 ? "critical" : pct >= 1 ? "warning" : "ok");

export { formatCount } from "./format";

// pacingPct is the KPI tile number: where a product's linear projection
// lands relative to its included allowance (100% = exactly at allowance at
// period end; 184% = will use the allowance plus most of an extra one).
export function pacingPct(p: ProductUsage): number {
  return p.included > 0 ? (p.projected / p.included) * 100 : 0;
}

// pacingLabel states the projection honestly: what metric, that the number
// is a projection, and which allowance it is a percentage OF.
export function pacingLabel(p: ProductUsage): string {
  return `${p.product} ${p.metric.toLowerCase()} · projected % of ${formatCount(p.included)} ${p.unit}`;
}

// ringPropsFor maps one billing product + the period into ring-gauge props:
// the percentages are of the product's allowance, the tick is the
// expected-to-date share of the period, and the sublabel reads "10M requests".
export function ringPropsFor(p: ProductUsage, period: BillingPeriod): RingGaugeProps {
  const included = p.included > 0 ? p.included : 0;
  return {
    usedPct: included > 0 ? (p.used / included) * 100 : 0,
    projectedPct: pacingPct(p),
    expectedPct: periodProgress(period.day, period.days).elapsedPct,
    label: `${p.product} ${p.metric.toLowerCase()}`,
    sublabel: `${formatCount(p.included)} ${p.unit}`,
  };
}

// periodEndsLabel marks a calendar-sourced period as an assumption — the
// operator should know "(calendar month)" is not the invoice's own date.
export function periodEndsLabel(end: string, source: BillingPeriod["source"]): string {
  return `period ends ${formatDateShort(end)}${source === "calendar" ? " (calendar month)" : ""}`;
}

function kpiTile(value: string, label: string, level: Level = "ok"): HTMLElement {
  const tile = el("div", `cf-kpi cf-level-${level}`);
  tile.append(el("div", "cf-kpi-value", value), el("div", "cf-kpi-label", label));
  tile.setAttribute("role", "group");
  return tile;
}

function errorState(root: HTMLElement, err: unknown): void {
  if (err instanceof LoginExpiredError) {
    root.replaceChildren(el("p", "cf-empty cf-login-expired", err.message));
    return;
  }
  const box = el("section", "cf-card cf-level-warning");
  box.append(
    el("h2", undefined, "Could not load the overview"),
    el("p", "cf-row-detail", err instanceof Error ? err.message : String(err)),
  );
  const retry = el("button", "cf-btn", "Retry");
  retry.addEventListener("click", () => void renderOverview(root));
  box.append(retry);
  root.replaceChildren(box);
}

// findProduct locates a billing product by id, for the pacing KPI tiles.
const findProduct = (b: Billing | null, id: string): ProductUsage | null => b?.products.find((p) => p.id === id) ?? null;

export async function renderOverview(root: HTMLElement, opts: { refresh?: boolean } = {}): Promise<void> {
  if (!opts.refresh) root.replaceChildren(skeleton());
  try {
    const [sumRes, billRes] = await Promise.all([
      api.fetchJson<Summary>("api/summary", { refresh: opts.refresh }),
      api.fetchJson<Billing>("api/billing", { refresh: opts.refresh }),
    ]);
    const summary = sumRes.data;
    const billing = billRes.data;
    const grid = el("div", "cf-kpis");
    const attention = collectAttention(summary, billing);

    // KPI tiles. The billing-period tile embeds its own thin progress bar —
    // the orphan full-width bar that used to sit under the grid is gone.
    const d1Prod = findProduct(billing, "d1.rows_read");
    const wProd = findProduct(billing, "workers.requests");
    const periodTile = kpiTile(`day ${billing.period.day} of ${billing.period.days}`, periodEndsLabel(billing.period.end, billing.period.source));
    const prog = el("div", "cf-progress");
    prog.setAttribute("aria-label", `Billing period: day ${billing.period.day} of ${billing.period.days}`);
    const fill = el("div", "cf-progress-fill");
    fill.style.width = `${periodProgress(billing.period.day, billing.period.days).elapsedPct}%`;
    prog.append(fill);
    periodTile.append(prog);
    grid.append(
      kpiTile(formatUsd(billing.totalProjectedOverageUsd), "projected overage",
        billing.totalProjectedOverageUsd >= 0.01 ? (billing.totalProjectedOverageUsd >= 50 ? "critical" : "warning") : "ok"),
      periodTile,
      kpiTile(d1Prod ? formatPct(pacingPct(d1Prod), 0) : "—", d1Prod ? pacingLabel(d1Prod) : "D1 rows-read pacing", d1Prod ? usageLevelFromPct(pacingPct(d1Prod)) : "ok"),
      kpiTile(wProd ? formatPct(pacingPct(wProd), 0) : "—", wProd ? pacingLabel(wProd) : "Workers requests pacing", wProd ? usageLevelFromPct(pacingPct(wProd)) : "ok"),
      kpiTile(String(attention.length), attention.length === 1 ? "item needs attention" : "items need attention",
        attention.some((i) => i.level === "critical") ? "critical" : attention.length ? "warning" : "ok"),
    );

    // Needs attention list — the worst rows up to the cap, then "Show all N".
    const attentionCard = el("section", "cf-card");
    attentionCard.append(el("h2", undefined, "Needs attention"));
    if (attention.length === 0) {
      attentionCard.append(el("p", "cf-empty cf-empty-quiet", "All clear — nothing above threshold."));
    } else {
      const { shown, extra } = capAttention(attention, ATTENTION_CAP);
      for (const item of shown) {
        const link = el("a", `cf-attention cf-level-${item.level}`);
        link.href = hrefFor(item.route);
        link.append(
          el("span", "cf-attention-mark", item.level === "critical" ? "!" : "•"),
          el("span", "cf-attention-body",
            `${item.title} — ${item.detail}`),
          el("span", "cf-attention-chevron", "→"),
        );
        attentionCard.append(link);
      }
      if (extra > 0) {
        const more = el("a", "cf-attention cf-attention-more");
        more.href = hrefFor("billing");
        more.append(el("span", "cf-attention-body", `Show all ${attention.length}`));
        attentionCard.append(more);
      }
    }

    // Workers Paid allowances — one ring per product metric, worst-first
    // (projected % desc), each tapping through to the billing view.
    const ringsCard = el("section", "cf-card");
    ringsCard.append(el("h2", undefined, "Workers Paid allowances"));
    const rings = el("div", "cf-rings");
    for (const p of sortByProjectedDesc(billing.products)) {
      const props = ringPropsFor(p, billing.period);
      const { level, overLimit } = gaugeLevel(props.usedPct, props.projectedPct);
      const link = el("a", `cf-ring cf-level-${level}${overLimit ? " cf-over" : ""}`);
      link.href = hrefFor("billing");
      link.append(ringGauge(props));
      rings.append(link);
    }
    ringsCard.append(rings);
    root.replaceChildren(statusBar(sumRes, billRes, root), grid, ringsCard, attentionCard);
  } catch (err) {
    errorState(root, err);
  }
}

// usageLevelFromPct colors the pacing KPI tiles with the same bands as
// levelForUsage (the projected % of limit from the summary rows).
export function usageLevelFromPct(pct: number): Level {
  return levelForUsage(pct);
}

// statusBar builds the "Updated … / cached / demo" line plus the Refresh
// button. Ages come from the responses' server cache field when present,
// else the client-side copy age.
function statusBar(sumRes: { data: Summary; demo: boolean; ageSec: number }, billRes: { data: Billing; demo: boolean; ageSec: number }, root: HTMLElement): HTMLElement {
  const bar = el("div", "cf-dash-bar");
  const ages = [sumRes, billRes];
  const maxAge = Math.max(...ages.map((r) => r.ageSec));
  const serverAge = sumRes.data.cache ? sumRes.data.cache.ageSec : null;
  const line = statusLine(maxAge, { cached: Boolean(sumRes.data.cache?.stale || billRes.data.cache?.stale), demo: sumRes.demo || billRes.demo });
  if (serverAge !== null) line.textContent = `Updated ${formatAge(serverAge)}${sumRes.data.cache?.stale ? " · cached" : ""}`;
  const refresh = el("button", "cf-btn", "Refresh");
  refresh.addEventListener("click", () => void renderOverview(root, { refresh: true }));
  bar.append(line, refresh);
  return bar;
}
