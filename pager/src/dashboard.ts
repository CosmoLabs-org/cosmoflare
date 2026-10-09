// Overview view (FEAT-052): KPI tiles (projected overage, billing-period
// progress, D1 and Workers pacing, attention count), then the merged
// "Needs attention" list. Data: /api/summary v2 + /api/billing through the
// shared ApiClient. Levels mirror the starter alert thresholds (O10) so the
// screen and the pager agree on what is "red".

import { el, skeleton, statusLine } from "./dom";
import { formatPct, formatUsd, formatDateShort, formatAge, periodProgress } from "./format";
import { api, LoginExpiredError, type Billing, type ProductUsage, type Summary } from "./api";
import { collectAttention } from "./attention";
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

    // KPI tiles
    const d1Prod = findProduct(billing, "d1.rows_read");
    const wProd = findProduct(billing, "workers.requests");
    grid.append(
      kpiTile(formatUsd(billing.totalProjectedOverageUsd), "projected overage",
        billing.totalProjectedOverageUsd >= 0.01 ? (billing.totalProjectedOverageUsd >= 50 ? "critical" : "warning") : "ok"),
      kpiTile(`day ${billing.period.day} of ${billing.period.days}`, `period ends ${formatDateShort(billing.period.end)}`),
      kpiTile(d1Prod ? formatPct(pacingPct(d1Prod), 0) : "—", "D1 rows-read pacing", d1Prod ? usageLevelFromPct(pacingPct(d1Prod)) : "ok"),
      kpiTile(wProd ? formatPct(pacingPct(wProd), 0) : "—", "Workers requests pacing", wProd ? usageLevelFromPct(pacingPct(wProd)) : "ok"),
      kpiTile(String(attention.length), attention.length === 1 ? "item needs attention" : "items need attention",
        attention.some((i) => i.level === "critical") ? "critical" : attention.length ? "warning" : "ok"),
    );

    // Period progress bar
    const prog = el("div", "cf-progress");
    const fill = el("div", "cf-progress-fill");
    fill.style.width = `${periodProgress(billing.period.day, billing.period.days).elapsedPct}%`;
    prog.append(fill);

    // Needs attention list
    const attentionCard = el("section", "cf-card");
    attentionCard.append(el("h2", undefined, "Needs attention"));
    if (attention.length === 0) {
      attentionCard.append(el("p", "cf-empty cf-empty-quiet", "All clear — nothing above threshold."));
    } else {
      for (const item of attention) {
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
    }
    root.replaceChildren(statusBar(sumRes, billRes, root), grid, prog, attentionCard);
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
