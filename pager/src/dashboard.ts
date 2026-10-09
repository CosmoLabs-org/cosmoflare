// Overview view (FEAT-052): KPI tiles (projected overage, billing-period
// progress, D1 and Workers pacing, attention count), then the merged
// "Needs attention" list. Data: /api/summary v2 + /api/billing through the
// shared ApiClient. Levels mirror the starter alert thresholds (O10) so the
// screen and the pager agree on what is "red".

import { el, skeleton, statusLine } from "./dom";
import { formatAmount, formatCount, formatPct, formatUsd, formatDateShort, periodProgress } from "./format";

import { api, LoginExpiredError, totalAgeSec, type Billing, type BillingPeriod, type FetchResult, type ProductUsage, type Summary } from "./api";
import { ATTENTION_CAP, capAttention, collectAttention } from "./attention";
import { capTopRings, gaugeLevel, productLabel, ringGauge, sortByProjectedDesc, type RingGaugeProps } from "./gauges";
import { hrefFor } from "./routes";
import { refreshButton } from "./refresh";

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
  return `${productLabel(p.id, p.product, p.metric)} · projected % of ${formatCount(p.included)} ${p.unit}`;
}

// pacingSublabel is the KPI tile's second line: "projected 106% of 25.0B
// rows" — the tile value is the projected %, the sublabel repeats what that
// number is a projection OF.
export function pacingSublabel(p: ProductUsage): string {
  const pct = p.included > 0 ? (p.projected / p.included) * 100 : 0;
  return `projected ${formatPct(pct, 0)} of ${formatCount(p.included)} ${p.unit}`;
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
    label: productLabel(p.id, p.product, p.metric),
    // IMP-002: storage projection === usage, so the size line under the
    // center value carries BOTH numbers ("3.9 GB of 1.0 GB") — the sublabel
    // would repeat the allowance 14px below it.
    sublabel: p.id.endsWith(".storage")
      ? undefined
      : `${formatCount(p.included)} ${p.unit}`,
    usedLineText: p.id.endsWith(".storage")
      ? `${formatAmount(p.used, p.unit)} of ${formatAmount(p.included, p.unit)}`
      : undefined,
  };
}

// periodEndsLabel marks a calendar-sourced period as an assumption — the
// operator should know "(calendar month)" is not the invoice's own date.
export function periodEndsLabel(end: string, source: BillingPeriod["source"]): string {
  return `ends ${formatDateShort(end)}${source === "calendar" ? " (calendar month)" : ""}`;
}

function kpiTile(value: string, label: string, level: Level = "ok", sublabel?: string): HTMLElement {
  const tile = el("div", `cf-kpi cf-level-${level}`);
  tile.append(el("div", "cf-kpi-value", value), el("div", "cf-kpi-label", label));
  if (sublabel) tile.append(el("div", "cf-kpi-sub", sublabel));
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
  // A background revalidation repaints the view with the fresh data
  // (BUG-057 defect 2). The repaint fetches both endpoints again — both are
  // then fresh in the client cache, so it is instant — and keeps the old
  // DOM when it fails.
  const repaint = (): void => {
    void Promise.all([
      api.fetchJson<Summary>("api/summary"),
      api.fetchJson<Billing>("api/billing"),
    ]).then(([s, b]) => paintOverview(root, s, b), () => undefined);
  };
  try {
    const [sumRes, billRes] = await Promise.all([
      api.fetchJson<Summary>("api/summary", { refresh: opts.refresh, onRevalidate: repaint }),
      api.fetchJson<Billing>("api/billing", { refresh: opts.refresh, onRevalidate: repaint }),
    ]);
    paintOverview(root, sumRes, billRes);
  } catch (err) {
    // A failed refresh rethrows so the clicked Refresh button keeps its node
    // and surfaces the error itself; good data is never replaced by an error
    // card (BUG-057 defect 3). Only a first (skeleton) load shows the error.
    if (opts.refresh) throw err;
    errorState(root, err);
  }
}

// paintOverview renders one (summary, billing) pair into the view host —
// the try-body of renderOverview, split out so background revalidations can
// repaint without touching the skeleton/error flow.
// The plan this account runs on and its base price — neutral grey — plus
// the month's projected overage in its severity color (operator request
// 2026-10-10: "$5.00 in neutral grey then + whatever overage").
// Workers Paid base: $5.00/mo (cloudflare.com/workers/pricing, the same
// source pricing.ts verifies product prices against).
const WORKERS_PAID_BASE_USD = 5;

function planLine(billing: Billing): HTMLElement {
  const over = billing.totalProjectedOverageUsd;
  const line = el("p", "cf-plan-line");
  line.append(
    el("span", "cf-plan-base", `Workers Paid · $${WORKERS_PAID_BASE_USD}.00/mo`),
    el("span", `cf-plan-overage ${over >= 0.01 ? (over >= 50 ? "cf-level-critical-text" : "cf-level-warning-text") : "cf-ok-text"}`,
      over >= 0.01 ? `+ ${formatUsd(over)} overage this month` : "no overage projected"),
  );
  return line;
}

function paintOverview(root: HTMLElement, sumRes: FetchResult<Summary>, billRes: FetchResult<Billing>): void {
  {
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
      kpiTile(d1Prod ? formatPct(pacingPct(d1Prod), 0) : "—", d1Prod ? productLabel(d1Prod.id, d1Prod.product, d1Prod.metric) : "D1 rows-read pacing", d1Prod ? usageLevelFromPct(pacingPct(d1Prod)) : "ok", d1Prod ? pacingSublabel(d1Prod) : undefined),
      kpiTile(wProd ? formatPct(pacingPct(wProd), 0) : "—", wProd ? productLabel(wProd.id, wProd.product, wProd.metric) : "Workers requests pacing", wProd ? usageLevelFromPct(pacingPct(wProd)) : "ok", wProd ? pacingSublabel(wProd) : undefined),
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

    // Workers Paid allowances — the 8 rings closest to their limit,
    // worst-first (projected % desc), the rest behind a "Show all N"
    // toggle. Each ring taps through to the billing view.
    const ringsCard = el("section", "cf-card");
    ringsCard.append(el("h2", undefined, "Workers Paid allowances"));
    const sorted = sortByProjectedDesc(billing.products);
    const { shown, hiddenCount } = capTopRings(sorted);
    const rings = el("div", "cf-rings");
    const ringFor = (p: (typeof sorted)[number]): HTMLElement => {
      const props = ringPropsFor(p, billing.period);
      const { level, overLimit } = gaugeLevel(props.usedPct, props.projectedPct);
      const link = el("a", `cf-ring cf-level-${level}${overLimit ? " cf-over" : ""}`);
      link.href = hrefFor("billing");
      link.append(ringGauge(props));
      return link;
    };
    for (const p of shown) rings.append(ringFor(p));
    if (hiddenCount > 0) {
      let expanded = false;
      const toggle = el("button", "cf-btn cf-show-all", `Show all ${sorted.length}`);
      toggle.addEventListener("click", () => {
        expanded = !expanded;
        if (expanded) {
          for (const p of sorted.slice(shown.length)) rings.append(ringFor(p));
          toggle.textContent = "Show top 8";
        } else {
          rings.replaceChildren(...shown.map(ringFor));
          toggle.textContent = `Show all ${sorted.length}`;
        }
      });
      ringsCard.append(rings, toggle);
    } else {
      ringsCard.append(rings);
    }
    root.replaceChildren(statusBar(sumRes, billRes, root), planLine(billing), grid, ringsCard, attentionCard);
  }
}

// usageLevelFromPct colors the pacing KPI tiles with the same bands as
// levelForUsage (the projected % of limit from the summary rows).
export function usageLevelFromPct(pct: number): Level {
  return levelForUsage(pct);
}

// statusBar builds the "Updated … / cached / demo" line plus the Refresh
// button. The age is honest (BUG-057 defect 2): client copy age + server
// cache age, for both endpoints.
function statusBar(sumRes: FetchResult<Summary>, billRes: FetchResult<Billing>, root: HTMLElement): HTMLElement {
  const bar = el("div", "cf-dash-bar");
  const maxAge = Math.max(
    totalAgeSec(sumRes.ageSec, sumRes.data.cache),
    totalAgeSec(billRes.ageSec, billRes.data.cache),
  );
  const line = statusLine(maxAge, { cached: Boolean(sumRes.data.cache?.stale || billRes.data.cache?.stale), demo: sumRes.demo || billRes.demo });
  const refresh = refreshButton(() => renderOverview(root, { refresh: true }));
  bar.append(line, refresh);
  return bar;
}
