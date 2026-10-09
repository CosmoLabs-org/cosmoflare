import { describe, expect, it } from "vitest";
import { collectAttention, capAttention, zoneAttentionLevel } from "./attention";
import type { Billing, Summary } from "./api";

// emptySummary / billingWith: the two API shapes with every list empty, so
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

type ProductUsageLite = {
  id: string; product: string; metric: string; unit: string;
  included: number; used: number, projected: number; unitPriceUsd: number;
  priceUnit: number; projectedOverageUsd: number; topConsumers: never[];
};

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

  it("merges overage and past-allowance into ONE row per product", () => {
    // 4 GB used of 1 GB included with a projected $1.44 overage — exactly the
    // screenshot's duplicate pair — must become a single merged row.
    const p = product(1.44, 4, 1);
    p.product = "KV";
    p.metric = "Stored data";
    p.unit = "GB";
    const items = collectAttention(emptySummary(), billingWith([p]));
    expect(items).toHaveLength(1);
    expect(items[0].title).toBe("KV stored data");
    expect(items[0].detail).toBe("4.0 GB of 1.0 GB included · +$1.44 projected");
    expect(items[0].level).toBe("critical"); // used > included
  });

  it("keeps one merged row per product when two products are over", () => {
    const kv = product(1.44, 4, 1);
    kv.product = "KV"; kv.metric = "Stored data"; kv.unit = "GB";
    const d1 = product(0.5, 12_000_000, 10_000_000);
    d1.product = "D1"; d1.metric = "Rows read"; d1.unit = "rows";
    const items = collectAttention(emptySummary(), billingWith([kv, d1]));
    const titles = items.map((i) => i.title);
    expect(titles).toEqual(["KV stored data", "D1 rows read"]);
  });

  it("marks a product with headroom as warning when only the projection overruns", () => {
    // used 214 of 10 GB, already past allowance → critical even at $0.
    const past = product(0, 214, 10);
    past.unit = "GB";
    past.product = "R2"; past.metric = "Storage (average)";
    // used 8 of 10 GB, projected overage → warning.
    const projected = product(1, 8_000_000, 10_000_000);
    const items = collectAttention(emptySummary(), billingWith([projected, past]));
    const byTitle = new Map(items.map((i) => [i.title, i.level]));
    expect(byTitle.get("R2 storage (average)")).toBe("critical");
    expect(byTitle.get("Workers requests")).toBe("warning");
  });

  it("flags a product past allowance as critical even with $0 overage", () => {
    const items = collectAttention(emptySummary(), billingWith([product(0, 12_000_000, 10_000_000)]));
    expect(items).toHaveLength(1);
    expect(items[0].level).toBe("critical");
    expect(items[0].detail).toBe("12.0M requests of 10.0M requests included");
  });

  it("formats GB amounts with one decimal", () => {
    const p = product(0, 3.9, 1);
    p.unit = "GB";
    const items = collectAttention(emptySummary(), billingWith([p]));
    expect(items[0].detail).toBe("3.9 GB of 1.0 GB included");
  });

  it("sorts critical before warning, then by projected USD desc", () => {
    const w1 = product(1);        // warning, $1
    const w5 = product(5);        // warning, $5
    const c1 = product(0.5, 20, 10); // critical (past), $0.50
    const items = collectAttention(emptySummary(), billingWith([w1, w5, c1]));
    expect(items.map((i) => i.level)).toEqual(["critical", "warning", "warning"]);
    expect(items[1].overageUsd).toBe(5);
    expect(items[2].overageUsd).toBe(1);
  });

  it("links each item to its section route", () => {
    const summary = emptySummary();
    summary.workers.push({ script: "api", requests: 100, errors: 10, errorPct: 10, cpuP50Ms: null, cpuP99Ms: null });
    summary.d1.push({ name: "db", rowsRead: 2e9, rowsWritten: 0, readQueries: 0, rowsPerQuery: 0 });
    const items = collectAttention(summary, billingWith([]));
    expect(items.find((i) => i.title === "api errors")?.route).toBe("workers");
    expect(items.find((i) => i.title === "db rows read")?.route).toBe("d1");
  });

  it("prefers billing over the summary for pacing rows (one number per metric)", () => {
    const summary = emptySummary();
    summary.usage.push({ name: "Workers requests", used: 18_400_000, limit: 10_000_000, pct: 184, projectedPct: 380 });
    // Billing loaded: the summary usage row must NOT become a second row.
    const withBilling = collectAttention(summary, billingWith([product(1)]));
    expect(withBilling.filter((i) => i.title.includes("pacing"))).toHaveLength(0);
    expect(withBilling.map((i) => i.title)).toEqual(["Workers requests"]);
    // Billing failed (null): the summary fallback carries the pacing row.
    const fallback = collectAttention(summary, null);
    expect(fallback).toHaveLength(1);
    expect(fallback[0].title).toBe("Workers requests pacing");
  });

  it("formats raw counts compactly in summary rows", () => {
    const summary = emptySummary();
    summary.d1.push({ name: "big", rowsRead: 412_000_000, rowsWritten: 0, readQueries: 0, rowsPerQuery: 0 });
    summary.workers.push({ script: "api", requests: 412_000_000, errors: 10_000, errorPct: 5, cpuP50Ms: null, cpuP99Ms: null });
    summary.zones.push({ zone: "example.com", total: 412_000_000, uncached: 412_000, missPct: 8.8 });
    const items = collectAttention(summary, billingWith([]));
    expect(items.find((i) => i.title === "big rows read")?.detail).toBe("412.0M rows in 24h");
    expect(items.find((i) => i.title === "api errors")?.detail).toBe("5.0% of 412.0M requests");
    expect(items.find((i) => i.title === "example.com cache misses")?.detail).toBe("412.0k uncached · 8.8% miss");
  });

  it("surfaces billing telemetry gaps once", () => {
    const items = collectAttention(emptySummary(), billingWith([], ["d1 storage: timeout"]));
    expect(items.filter((i) => i.title === "Telemetry gap")).toHaveLength(1);
  });
});

describe("capAttention", () => {
  it("shows the first 8 rows and reports the rest", () => {
    const items = Array.from({ length: 10 }, (_, n) => ({
      level: "warning" as const, title: `t${n}`, detail: "", route: "billing" as const,
      overageUsd: 10 - n, magnitude: 0,
    }));
    const { shown, extra } = capAttention(items);
    expect(shown).toHaveLength(8);
    expect(extra).toBe(2);
  });
  it("returns everything untouched under the cap", () => {
    const items = Array.from({ length: 3 }, (_, n) => ({
      level: "warning" as const, title: `t${n}`, detail: "", route: "billing" as const,
      overageUsd: 0, magnitude: 0,
    }));
    expect(capAttention(items)).toEqual({ shown: items, extra: 0 });
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
