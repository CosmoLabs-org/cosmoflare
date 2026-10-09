import { describe, expect, it } from "vitest";
import {
  formatCount,
  levelForD1,
  levelForMiss,
  levelForUncached,
  levelForUsage,
  pacingLabel,
  periodEndsLabel,
} from "./dashboard";
import type { ProductUsage } from "./api";

describe("dashboard formatting", () => {
  it("formats counts like the CLI (2.9B / 52.3M / 3.3k / 522)", () => {
    expect(formatCount(2_945_546_702)).toBe("2.9B");
    expect(formatCount(52_332_502)).toBe("52.3M");
    expect(formatCount(3289)).toBe("3.3k");
    expect(formatCount(522)).toBe("522");
  });
  it("maps values to the starter alert thresholds (O10)", () => {
    expect(levelForUncached(36_321)).toBe("critical");
    expect(levelForUncached(8_192)).toBe("warning"); // amber band at half the 10k paging threshold
    expect(levelForUncached(4_000)).toBe("ok");
    expect(levelForMiss(74)).toBe("warning");
    expect(levelForMiss(null)).toBe("ok");
    expect(levelForD1(2.9e9)).toBe("critical");
    expect(levelForD1(52e6)).toBe("ok");
    expect(levelForUsage(120)).toBe("critical");
    expect(levelForUsage(85)).toBe("warning");
    expect(levelForUsage(10)).toBe("ok");
  });
});

describe("pacingLabel", () => {
  it("states the metric, that it is a projection, and the allowance", () => {
    const d1: ProductUsage = {
      id: "d1.rows_read", product: "D1", metric: "Rows read", unit: "rows",
      included: 25e9, used: 640e6, projected: 27e9, unitPriceUsd: 0.001,
      priceUnit: 1e6, projectedOverageUsd: 2, topConsumers: [],
    };
    expect(pacingLabel(d1)).toBe("D1 rows read · projected % of 25.0B rows");
  });
});

describe("periodEndsLabel", () => {
  it("marks a calendar-sourced period as an assumption", () => {
    const label = periodEndsLabel("2026-11-01T00:00:00Z", "calendar");
    expect(label).toMatch(/Nov 1/);
    expect(label).toContain("(calendar month)");
  });
  it("leaves subscription-sourced periods unmarked", () => {
    const label = periodEndsLabel("2026-11-01T00:00:00Z", "subscription");
    expect(label).toMatch(/Nov 1/);
    expect(label).not.toContain("(calendar month)");
  });
});
