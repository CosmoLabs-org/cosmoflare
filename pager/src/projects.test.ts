import { describe, expect, it } from "vitest";
import { projectConsumers, sortProjectsByOverage } from "./projects";
import type { Billing, ProductUsage } from "./api";

// FEAT-p0QKZT0: the projects view is pure presentation over /api/billing —
// these helpers carry the ordering and attribution logic.

const product = (id: string, over: Partial<ProductUsage> = {}): ProductUsage => ({
  id, product: "D1", metric: "Rows read", unit: "rows",
  included: 25e9, used: 1e9, projected: 2e9, unitPriceUsd: 0.001,
  priceUnit: 1e6, projectedOverageUsd: 0, topConsumers: [],
  ...over,
});

const billing = (projects: Billing["projects"], products: ProductUsage[]): Billing => ({
  generatedAt: "2026-10-10T00:00:00Z",
  period: { start: "2026-09-23T00:00:00Z", end: "2026-10-23T00:00:00Z", day: 17, days: 30, source: "anchor" },
  products,
  projects,
  totalProjectedOverageUsd: projects.reduce((s, p) => s + p.projectedOverageUsd, 0),
  pricing: { verifiedOn: "2026-10-09", sources: [] },
  errors: [],
});

describe("sortProjectsByOverage", () => {
  it("puts the worst payer first, ties broken by name", () => {
    const got = sortProjectsByOverage([
      { project: "mycarguide", projectedOverageUsd: 1.44 },
      { project: "api", projectedOverageUsd: 0 },
      { project: "beekey", projectedOverageUsd: 2 },
    ]);
    expect(got.map((p) => p.project)).toEqual(["beekey", "mycarguide", "api"]);
  });
  it("stable order for equal overage", () => {
    const got = sortProjectsByOverage([
      { project: "zeta", projectedOverageUsd: 0 },
      { project: "alpha", projectedOverageUsd: 0 },
    ]);
    expect(got.map((p) => p.project)).toEqual(["alpha", "zeta"]);
  });
});

describe("projectConsumers", () => {
  it("collects one project's consumers across products, biggest share first", () => {
    const b = billing(
      [{ project: "mycarguide", projectedOverageUsd: 1.9, drivers: ["kv.storage", "d1.rows_read"] }],
      [
        product("kv.storage", { unit: "GB", topConsumers: [
          { name: "cache-db", project: "mycarguide", used: 3.5, share: 0.9 },
        ] }),
        product("d1.rows_read", { topConsumers: [
          { name: "api-prod", project: "mycarguide", used: 12e9, share: 0.83 },
          { name: "other", project: "beekey", used: 1e9, share: 0.07 },
        ] }),
      ],
    );
    const got = projectConsumers(b, "mycarguide");
    expect(got).toHaveLength(2);
    expect(got[0]?.share).toBe(0.9); // biggest first
    expect(got.every((c) => c.name !== "other")).toBe(true); // other projects excluded
    expect(got[1]?.productName).toBe("D1 rows read");
  });
  it("returns empty for a project with no consumer attribution", () => {
    const b = billing([{ project: "empty", projectedOverageUsd: 0, drivers: [] }], [product("kv.reads")]);
    expect(projectConsumers(b, "empty")).toEqual([]);
  });
});
