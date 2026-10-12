import { describe, expect, it } from "vitest";
import { gradeLabel, overageHeadText,
  projectedMonthlyTotal,
  projectedOverageTotal,
  WORKERS_PAID_BASE_USD,
  WORKERS_PAID_PLAN,
} from "./billing";
import type { ProductUsage } from "./api";

// Operator 2026-10-10 (Workers Paid section): every product card head shows
// BOTH its grade and its overage, always, at the same size — the pair reads
// as one system instead of overage appearing only on some cards. IMP-002's
// "one status per card" stays: the pair IS the status; no second wording.

const base: ProductUsage = {
  id: "d1.rows_read", product: "D1", metric: "Rows read", unit: "rows",
  included: 25e9, used: 14.4e9, projected: 25.5e9, unitPriceUsd: 0.001,
  priceUnit: 1e6, projectedOverageUsd: 0, topConsumers: [],
};

describe("overageHeadText (grade+overage pair, operator 2026-10-10)", () => {
  it("shows the projected overage cost when there is one", () => {
    expect(overageHeadText({ ...base, projectedOverageUsd: 1.444 })).toBe("+$1.44 overage");
  });
  it("emits 'no overage' for a zero-overage product — the pair stays complete", () => {
    expect(overageHeadText(base)).toBe("no overage");
  });
  it("never emits a second status wording like 'within allowance'", () => {
    expect(overageHeadText(base)).not.toContain("within allowance");
  });
});

describe("gradeLabel (grade half of the pair)", () => {
  it("OK below the warning band", () => {
    expect(gradeLabel(50, 60)).toEqual({ text: "OK", level: "ok" });
  });
  it("NEAR LIMIT from the warning band (projected ≥75%)", () => {
    expect(gradeLabel(70, 80)).toEqual({ text: "NEAR LIMIT", level: "warning" });
  });
  it("OVER LIMIT once used or projected passes 100%", () => {
    expect(gradeLabel(95, 105)).toEqual({ text: "OVER LIMIT", level: "critical" });
    expect(gradeLabel(101, 101)).toEqual({ text: "OVER LIMIT", level: "critical" });
  });
  it("matches the gauge's own level decision (the badge and the chip agree)", () => {
    expect(gradeLabel(0, 76).level).toBe("warning");
    expect(gradeLabel(0, 91).level).toBe("critical");
  });
});

describe("monthly cost totals (operator 2026-10-12)", () => {
  const over = (usd: number): ProductUsage => ({ projectedOverageUsd: usd } as ProductUsage);
  it("sums only positive projected overages", () => {
    expect(projectedOverageTotal([over(1.44), over(0), over(0.5)])).toBeCloseTo(1.94, 9);
  });
  it("monthly total = Workers Paid base + overage", () => {
    expect(projectedMonthlyTotal([over(1.44)])).toBeCloseTo(6.44, 9);
    expect(projectedMonthlyTotal([])).toBe(WORKERS_PAID_BASE_USD);
    expect(WORKERS_PAID_PLAN).toBe("Workers Paid");
  });
});
