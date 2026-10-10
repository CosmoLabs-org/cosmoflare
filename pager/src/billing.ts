// Billing helpers (FEAT-052): the pure layer behind the React billing view —
// meter geometry (meterPct), the grade/overage head pair (gradeLabel +
// overageHeadText) and the rolling-storage exclusion. The imperative vanilla
// renderer that used to live here was retired (TASK-pD575FN);
// pager/src/app/views/BillingView.tsx is the only render path now.

import { formatUsd } from "./format";
import type { ProductUsage } from "./api";
import { gaugeLevel, type Level } from "./gauges";

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

/**
 * The overage half of the card-head pair (operator 2026-10-10, Workers Paid
 * section): the projected cost beyond the allowance when there is one, and
 * "no overage" when there is not — the pair (grade + overage) stays complete
 * and same-size on every card instead of overage appearing only on some.
 */
export function overageHeadText(p: ProductUsage): string {
  return p.projectedOverageUsd > 0 ? `+${formatUsd(p.projectedOverageUsd)} overage` : "no overage";
}

/**
 * The grade half of the pair: the same decision gaugeLevel makes for the
 * bar's badge, worded — OK / NEAR LIMIT / OVER LIMIT — so the chip in the
 * head and the badge on the gauge can never disagree.
 */
export function gradeLabel(usedPct: number, projectedPct: number): { text: string; level: Level } {
  const { level } = gaugeLevel(usedPct, projectedPct);
  const text = level === "warning" ? "NEAR LIMIT" : level === "critical" ? "OVER LIMIT" : "OK";
  return { text, level };
}

/** Rolling-bytes metrics (BUG-pFAKDN3): kv.storage's number is a rolling
 *  cumulative, not live size — it must never drive allowance visuals
 *  (rings, over-limit cards, attention) even though the payload carries it
 *  for awareness. */
export function isRollingStorage(id: string): boolean {
  return id === "kv.storage";
}
