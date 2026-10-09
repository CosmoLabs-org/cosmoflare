import { describe, expect, it } from "vitest";
import { collectAttention, zoneAttentionLevel } from "./attention";
import type { Billing, Summary } from "./api";

// emptySummary / emptyBilling: the two API shapes with every list empty, so
// tests add only the rows under test.
function emptySummary(): Summary {
  return {
    generatedAt: "2026-10-09T00:00:00Z",
    windowHours: 24,
    usage: [],
    d1: [],
    zones: [],
    workers: [],
    errors: [],
  };
}

function product(overage: number, used: number = 0, included: number = 1): ProductUsageLite {
  return {
    id: "x",
    product: "Workers",
    metric: "Requests",
    unit: "requests",
    included,
    used,
    projected: used,
    unitPriceUsd: 0.3,
    priceUnit: 1e6,
    projectedOverageUsd: overage,
    topConsumers: [],
  };
}

type ProductUsageLite = {
  id: string; product: string; metric: string; unit: string;
  included: number; used: number; projected: number; unitPriceUsd: number;
  priceUnit: number; projectedOverageUsd: number; topConsumers: never[];
};

function billingWith(products: ProductUsageLite[], errors: string[] = []): Billing {
  return {
    generatedAt: "2026-10-09T00:00:00Z",
    period: { start: "2026-10-01T00:00:00Z", end: "2026-11-01T00:00:00Z", day: 9, days: 31, source: "calendar" },
    products: products as ProductUsageLite[],
    totalProjectedOverageUsd: products.reduce((s, p) => s + p.projectedOverageUsd, 0),
    projects: [],
    pricing: { verifiedOn: "2026-10-09", sources: [] },
    errors,
  } as unknown as Billing;
}

describe("collectAttention", () => {
  it("returns nothing when everything is clean", () => {
    const items = collectAttention(emptySummary(), billingWith([]));
    expect(items).toEqual([]);
  });

  it("sorts critical before warning", () => {
    const items = collectAttention(emptySummary(), billingWith([
      product(1),        // warning ($1 overage)
      product(75),       // critical ($75 overage)
    ]));
    expect(items.map((i) => i.level)).toEqual(["critical", "warning"]);
  });

  it("links each item to its section route", () => {
    const summary = emptySummary();
    summary.workers.push({ script: "api", requests: 100, errors: 10, errorPct: 10, cpuP50Ms: null, cpuP99Ms: null });
    summary.d1.push({ name: "db", rowsRead: 2e9, rowsWritten: 0, readQueries: 0, rowsPerQuery: 0 });
    const items = collectAttention(summary, billingWith([]));
    expect(items.find((i) => i.title === "api errors")?.route).toBe("workers");
    expect(items.find((i) => i.title === "db rows read")?.route).toBe("d1");
  });

  it("flags a product past allowance as critical even with $0 overage", () => {
    const items = collectAttention(emptySummary(), billingWith([product(0, 12_000_000, 10_000_000)]));
    expect(items.some((i) => i.level === "critical" && i.title.includes("past allowance"))).toBe(true);
  });

  it("surfaces billing telemetry gaps once", () => {
    const items = collectAttention(emptySummary(), billingWith([], ["d1 storage: timeout"]));
    expect(items.filter((i) => i.title === "Telemetry gap")).toHaveLength(1);
  });
});

describe("zoneAttentionLevel", () => {
  it("takes the worst of uncached and miss-rate levels", () => {
    expect(zoneAttentionLevel(36_321, null)).toBe("critical");
    expect(zoneAttentionLevel(8_192, null)).toBe("warning");
    expect(zoneAttentionLevel(100, 74)).toBe("warning");
    expect(zoneAttentionLevel(100, 10)).toBe("ok");
  });
});
