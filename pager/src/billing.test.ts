import { describe, expect, it } from "vitest";
import { overageHeadText } from "./billing";
import type { ProductUsage } from "./api";

// IMP-002: one status per billing card — the gauge badge is the status; the
// card head carries only a non-zero projected overage.

const base: ProductUsage = {
  id: "d1.rows_read", product: "D1", metric: "Rows read", unit: "rows",
  included: 25e9, used: 14.4e9, projected: 25.5e9, unitPriceUsd: 0.001,
  priceUnit: 1e6, projectedOverageUsd: 0, topConsumers: [],
};

describe("overageHeadText (IMP-002)", () => {
  it("shows the projected overage cost when there is one", () => {
    expect(overageHeadText({ ...base, projectedOverageUsd: 1.444 })).toBe("+$1.44 overage");
  });
  it("emits nothing for a zero-overage product — the badge is the only status", () => {
    expect(overageHeadText(base)).toBeNull();
  });
  it("never emits a second status wording like 'within allowance'", () => {
    expect(overageHeadText(base) ?? "").not.toContain("within allowance");
  });
});
