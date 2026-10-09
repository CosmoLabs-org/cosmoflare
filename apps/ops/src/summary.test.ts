import { describe, expect, it } from "vitest";
import { monthPacing, summarizeD1, summarizeZones } from "./summary";

describe("summarizeZones", () => {
  it("sums cacheStatus groups per zone, computes uncached volume and miss % with the 100-eligible floor", () => {
    const rows = summarizeZones(
      [
        { zoneTag: "z1", groups: [
          { count: 939, cacheStatus: "dynamic" }, { count: 21, cacheStatus: "bypass" },
          { count: 61, cacheStatus: "hit" }, { count: 83, cacheStatus: "miss" },
          { count: 4, cacheStatus: "expired" }, { count: 236, cacheStatus: "revalidated" },
          { count: 197, cacheStatus: "none" },
        ] },
        { zoneTag: "z2", groups: [{ count: 40, cacheStatus: "dynamic" }, { count: 10, cacheStatus: "hit" }] },
      ],
      { z1: "churches.app", z2: "tiny.example", z3: "idle.example" },
    );
    const churches = rows.find((r) => r.zone === "churches.app")!;
    expect(churches.total).toBe(1541);
    expect(churches.uncached).toBe(960); // dynamic + bypass
    expect(churches.missPct).toBeCloseTo((100 * 87) / 384, 9); // (miss+expired)/eligible
    const tiny = rows.find((r) => r.zone === "tiny.example")!;
    expect(tiny.missPct).toBeNull(); // 10 eligible < 100 floor
    expect(rows.find((r) => r.zone === "idle.example")!.total).toBe(0); // listed zone with no traffic
    expect(rows[0].zone).toBe("churches.app"); // sorted by uncached desc
  });
});

describe("summarizeD1", () => {
  it("sums per database, names from the list (ID fallback), rows per query, sorted by rows read", () => {
    const rows = summarizeD1(
      [
        { databaseId: "db-small", rowsRead: 1000, readQueries: 10 },
        { databaseId: "db-big", rowsRead: 2_945_546_702, readQueries: 73_151 },
        { databaseId: "db-small", rowsRead: 500, readQueries: 5 },
      ],
      { "db-big": "mycarguide-db" },
    );
    expect(rows.map((r) => r.name)).toEqual(["mycarguide-db", "db-small"]);
    expect(rows[1].rowsRead).toBe(1500);
    expect(rows[0].rowsPerQuery).toBe(Math.round(2_945_546_702 / 73_151));
  });
});

describe("monthPacing", () => {
  it("projects linearly to the end of the calendar month (UTC)", () => {
    // Day 10 of a 30-day month at 00:00 → 9 days elapsed; 40% used → projected 133.3%
    const p = monthPacing(40, 100, new Date(Date.UTC(2026, 10, 10, 0, 0, 0)));
    expect(p.pct).toBe(40);
    expect(p.projectedPct).toBeCloseTo((40 * 30) / 9, 6);
  });
  it("returns null projection on the first instant of the month", () => {
    expect(monthPacing(1, 100, new Date(Date.UTC(2026, 10, 1))).projectedPct).toBeNull();
  });
});
