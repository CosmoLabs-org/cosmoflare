// The "Needs attention" merge (FEAT-052): critical/warning rows from every
// section, critical first, each pointing at its section route. Pure data —
// no DOM — so the merge is unit-testable.

import { levelForD1, levelForMiss, levelForUncached, levelForUsage, levelForErrorPct } from "./dashboard";
import { formatAmount, formatCount, formatPct, formatUsd } from "./format";
import type { RouteId } from "./routes";
import type { Billing, Summary } from "./api";

/** One merged "needs attention" row; `route` is the section route to open. */
export interface AttentionItem {
  level: "warning" | "critical";
  title: string;
  detail: string;
  route: RouteId;
  /** Projected overage USD — secondary sort key, descending. */
  overageUsd: number;
  /** Usage magnitude (ratio, percent, or raw count) — final sort key, descending. */
  magnitude: number;
}

// Overage attention: any projected overage is worth surfacing as a warning.
// Severity is usage-based — consuming the included allowance is critical —
// so the level comes from used-vs-included, not the dollar amount.
const OVERAGE_WARN_USD = 0.01;

// capAttention: the Overview shows the worst rows; the rest collapse behind
// a "Show all N" link to Billing. The full list still lives in the array.
export const ATTENTION_CAP = 8;

export function capAttention(
  items: AttentionItem[],
  cap: number = ATTENTION_CAP,
): { shown: AttentionItem[]; extra: number } {
  return { shown: items.slice(0, cap), extra: Math.max(0, items.length - cap) };
}

export function collectAttention(summary: Summary | null, billing: Billing | null): AttentionItem[] {
  const items: AttentionItem[] = [];
  if (billing) {
    // ONE row per product: usage-vs-allowance and the projected overage are
    // two facts about the same product, so they share a row — the old
    // "past allowance" + "overage" duplicate pair is gone. Critical when the
    // included allowance is already consumed; warning for projected overage.
    for (const p of billing.products) {
      const over = p.projectedOverageUsd >= OVERAGE_WARN_USD;
      const past = p.used >= p.included;
      if (!over && !past) continue;
      let detail = `${formatAmount(p.used, p.unit)} of ${formatAmount(p.included, p.unit)} included`;
      if (over) detail += ` · +${formatUsd(p.projectedOverageUsd)} projected`;
      items.push({
        level: past ? "critical" : "warning",
        title: `${p.product} ${p.metric.toLowerCase()}`,
        detail,
        route: "billing",
        overageUsd: p.projectedOverageUsd,
        magnitude: p.included > 0 ? p.used / p.included : Number.POSITIVE_INFINITY,
      });
    }
    for (const e of billing.errors) {
      items.push({ level: "warning", title: "Telemetry gap", detail: e, route: "billing", overageUsd: 0, magnitude: 0 });
      break; // one row for the notice; the billing view lists the rest
    }
  }

  if (summary) {
    // Pacing rows come from /api/billing when it loaded — the Overview must
    // show one number per metric, never the summary's guess next to billing's
    // projection. Summary usage is the fallback for when billing failed.
    if (!billing) {
      for (const u of summary.usage) {
        const level = levelForUsage(u.projectedPct);
        if (level !== "ok") {
          items.push({
            level,
            title: `${u.name} pacing`,
            detail: `${u.pct.toFixed(0)}% of the period limit${u.projectedPct !== null ? `, projected ${u.projectedPct.toFixed(0)}%` : ""}`,
            route: "billing",
            overageUsd: 0,
            magnitude: u.projectedPct ?? u.pct,
          });
        }
      }
    }
    for (const w of summary.workers) {
      const level = levelForErrorPct(w.errorPct);
      if (level !== "ok") {
        items.push({
          level,
          title: `${w.script} errors`,
          detail: `${formatPct(w.errorPct)} of ${formatCount(w.requests)} requests`,
          route: "workers",
          overageUsd: 0,
          magnitude: w.errorPct,
        });
      }
    }
    for (const d of summary.d1) {
      const level = levelForD1(d.rowsRead);
      if (level !== "ok") {
        items.push({
          level,
          title: `${d.name} rows read`,
          detail: `${formatCount(d.rowsRead)} rows in 24h`,
          route: "d1",
          overageUsd: 0,
          magnitude: d.rowsRead,
        });
      }
    }
    for (const z of summary.zones) {
      const level = zoneAttentionLevel(z.uncached, z.missPct);
      if (level !== "ok") {
        items.push({
          level,
          title: `${z.zone} cache misses`,
          detail: `${formatCount(z.uncached)} uncached${z.missPct !== null ? ` · ${formatPct(z.missPct)} miss` : ""}`,
          route: "zones",
          overageUsd: 0,
          magnitude: z.uncached,
        });
      }
    }
  }
  // critical first, then by projected USD desc, then by magnitude desc.
  const rank = (l: "warning" | "critical"): 0 | 1 => (l === "critical" ? 0 : 1);
  return items.sort((a, b) =>
    rank(a.level) - rank(b.level) ||
    b.overageUsd - a.overageUsd ||
    b.magnitude - a.magnitude);
}

// Worst of the uncached-count and miss-rate levels for a zone — the same
// merge the dashboard row render uses.
export function zoneAttentionLevel(uncached: number, missPct: number | null): "ok" | "warning" | "critical" {
  const a = levelForUncached(uncached);
  const b = levelForMiss(missPct);
  if (a === "critical" || b === "critical") return "critical";
  if (a === "warning" || b === "warning") return "warning";
  return "ok";
}
