// The "Needs attention" merge (FEAT-052): critical/warning rows from every
// section, critical first, each pointing at its section route. Pure data —
// no DOM — so the merge is unit-testable.

import { levelForD1, levelForMiss, levelForUncached, levelForUsage, levelForErrorPct } from "./dashboard";
import { formatCount } from "./format";
import type { Billing, Summary } from "./api";

/** One merged "needs attention" row; `route` is a RouteId string. */
export interface AttentionItem {
  level: "warning" | "critical";
  title: string;
  detail: string;
  route: string;
}

// Overage attention levels: any projected overage is worth surfacing; $50+
// is where a human should actually act today. Deliberately simple.
const OVERAGE_WARN_USD = 0.01;
const OVERAGE_CRIT_USD = 50;

export function collectAttention(summary: Summary | null, billing: Billing | null): AttentionItem[] {
  const items: AttentionItem[] = [];
  if (billing) {
    for (const p of billing.products) {
      if (p.projectedOverageUsd >= OVERAGE_WARN_USD) {
        const critical = p.projectedOverageUsd >= OVERAGE_CRIT_USD;
        items.push({
          level: critical ? "critical" : "warning",
          title: `${p.product} ${p.metric.toLowerCase()} overage`,
          detail: `projected $${p.projectedOverageUsd.toFixed(2)} beyond the ${formatCount(p.included)} allowance`,
          route: "billing",
        });
      }
      if (p.used >= p.included) {
        items.push({
          level: "critical",
          title: `${p.product} ${p.metric.toLowerCase()} past allowance`,
          detail: `${formatCount(p.used)} used of ${formatCount(p.included)} included`,
          route: "billing",
        });
      }
    }
    for (const e of billing.errors) {
      items.push({ level: "warning", title: "Telemetry gap", detail: e, route: "billing" });
      break; // one row for the notice; the billing view lists the rest
    }
  }

  if (summary) {
    for (const u of summary.usage) {
      const level = levelForUsage(u.projectedPct);
      if (level !== "ok") {
        items.push({
          level,
          title: `${u.name} pacing`,
          detail: `${u.pct.toFixed(0)}% of the period limit${u.projectedPct !== null ? `, projected ${u.projectedPct.toFixed(0)}%` : ""}`,
          route: "billing",
        });
      }
    }
    for (const w of summary.workers) {
      const level = levelForErrorPct(w.errorPct);
      if (level !== "ok") {
        items.push({
          level,
          title: `${w.script} errors`,
          detail: `${w.errorPct.toFixed(2)}% of ${w.requests.toLocaleString("en-US")} requests`,
          route: "workers",
        });
      }
    }
    for (const d of summary.d1) {
      const level = levelForD1(d.rowsRead);
      if (level !== "ok") {
        items.push({
          level,
          title: `${d.name} rows read`,
          detail: `${d.rowsRead.toLocaleString("en-US")} rows in 24h`,
          route: "d1",
        });
      }
    }
    for (const z of summary.zones) {
      const level = zoneAttentionLevel(z.uncached, z.missPct);
      if (level !== "ok") {
        items.push({
          level,
          title: `${z.zone} cache misses`,
          detail: `${z.uncached.toLocaleString("en-US")} uncached${z.missPct !== null ? ` · ${z.missPct.toFixed(1)}% miss` : ""}`,
          route: "zones",
        });
      }
    }
  }
  return items.sort((a, b) => (a.level === b.level ? 0 : a.level === "critical" ? -1 : 1));
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
