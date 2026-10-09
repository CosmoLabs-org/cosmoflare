import { describe, expect, it } from "vitest";
import { daysUntil, expiryLevel, sortDomains } from "./domains";

const NOW = Date.parse("2026-10-10T00:00:00Z");

describe("daysUntil (FEAT-p9GD588)", () => {
  it("whole days to an ISO date, floor semantics", () => {
    expect(daysUntil("2026-10-11", NOW)).toBe(1);
    expect(daysUntil("2026-11-09", NOW)).toBe(30);
    expect(daysUntil("2026-10-08", NOW)).toBe(-2);
  });
});

describe("expiryLevel", () => {
  it("critical inside 14 days (and past), warning to 30, ok beyond, null without expiry", () => {
    expect(expiryLevel(14)).toBe("critical");
    expect(expiryLevel(-3)).toBe("critical");
    expect(expiryLevel(15)).toBe("warning");
    expect(expiryLevel(30)).toBe("warning");
    expect(expiryLevel(31)).toBe("ok");
    expect(expiryLevel(null)).toBeNull();
  });
});

describe("sortDomains", () => {
  it("soonest expiry first; no-expiry domains last by name", () => {
    const got = sortDomains([
      { name: "zeta.io", expiresAt: "2027-03-01" },
      { name: "alpha.dev", expiresAt: null },
      { name: "soon.com", expiresAt: "2026-10-20" },
      { name: "beta.app", expiresAt: null },
    ]);
    expect(got.map((d) => d.name)).toEqual(["soon.com", "zeta.io", "alpha.dev", "beta.app"]);
  });
});
