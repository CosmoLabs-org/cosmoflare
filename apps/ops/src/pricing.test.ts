import { describe, expect, it } from "vitest";
import { PRICES, PRICING_SOURCES, priceFor } from "./pricing";

describe("PRICES", () => {
  it("every row has an https source URL, a verification date, and positive unit math", () => {
    for (const p of PRICES) {
      expect(p.sourceUrl).toMatch(/^https:\/\/developers\.cloudflare\.com\//);
      expect(p.verifiedOn).toBe("2026-10-09");
      expect(p.priceUnit).toBeGreaterThan(0);
      expect(p.included).toBeGreaterThan(0);
      expect(p.unitPriceUsd).toBeGreaterThan(0);
    }
  });

  it("covers every product id in the GET /api/billing contract", () => {
    expect(PRICES.map((p) => p.id).sort()).toEqual(
      [
        "workers.requests",
        "workers.cpu_ms",
        "d1.rows_read",
        "d1.rows_written",
        "d1.storage",
        "r2.storage",
        "r2.class_a",
        "r2.class_b",
        "kv.reads",
        "kv.writes",
        "kv.storage",
        "do.requests",
        "do.duration",
      ].sort(),
    );
  });

  it("exposes one pricing source per product family, all https", () => {
    const families = new Set(PRICES.map((p) => p.id.split(".")[0]));
    expect(new Set(PRICING_SOURCES.map((s) => s.product))).toEqual(families);
    for (const s of PRICING_SOURCES) expect(s.url).toMatch(/^https:\/\//);
  });

  it("priceFor returns the row for a known id and throws for an unknown one", () => {
    expect(priceFor("d1.rows_read").included).toBe(25_000_000_000);
    expect(() => priceFor("nope.nope")).toThrow(/no verified price/);
  });
});
