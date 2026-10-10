// Overview helpers (FEAT-052): the pure layer behind the React overview —
// level thresholds (levelFor*), KPI pacing math (pacingPct/pacingSublabel),
// ring-gauge prop mapping (ringPropsFor) and period labels. The imperative
// vanilla renderer that used to live here was retired (TASK-pD575FN);
// pager/src/app/views/OverviewView.tsx is the only render path now.

import { formatAmount, formatCount, formatPct, formatUsd, formatDateShort, periodProgress } from "./format";
import type { BillingPeriod, ProductUsage } from "./api";
import { productLabel, type RingGaugeProps } from "./gauges";

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
/** The overage dollars under a ring (item-6, operator 2026-10-10): "+$2.00"
 * when the product projects past its allowance, null when it does not — a
 * clean ring stays clean; a costing ring says so at a glance. */
export function ringOverageText(p: ProductUsage): string | null {
  return p.projectedOverageUsd > 0 ? `+${formatUsd(p.projectedOverageUsd)}` : null;
}

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

// usageLevelFromPct colors the pacing KPI tiles with the same bands as
// levelForUsage (the projected % of limit from the summary rows).
export function usageLevelFromPct(pct: number): Level {
  return levelForUsage(pct);
}
