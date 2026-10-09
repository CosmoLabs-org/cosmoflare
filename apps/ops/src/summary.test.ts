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
      { z1: "site-a.example", z2: "tiny.example", z3: "idle.example" },
    );
    const siteA = rows.find((r) => r.zone === "site-a.example")!;
    expect(siteA.total).toBe(1541);
    expect(siteA.uncached).toBe(960); // dynamic + bypass
    expect(siteA.missPct).toBeCloseTo((100 * 87) / 384, 9); // (miss+expired)/eligible
    const tiny = rows.find((r) => r.zone === "tiny.example")!;
    expect(tiny.missPct).toBeNull(); // 10 eligible < 100 floor
    expect(rows.find((r) => r.zone === "idle.example")!.total).toBe(0); // listed zone with no traffic
    expect(rows[0].zone).toBe("site-a.example"); // sorted by uncached desc
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
      { "db-big": "big-db" },
    );
    expect(rows.map((r) => r.name)).toEqual(["big-db", "db-small"]);
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

describe("collectSummary failure isolation", () => {
  it("keeps the Workers usage row when the D1 dataset errors (additive, like the CLI)", async () => {
    const realFetch = globalThis.fetch;
    globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const body = init?.body ? String(init.body) : "";
      if (url.endsWith("/graphql")) {
        if (body.includes("d1AnalyticsAdaptiveGroups")) return new Response(JSON.stringify({ errors: [{ message: "d1 dataset unavailable" }] }));
        if (body.includes("workersInvocationsAdaptive")) return new Response(JSON.stringify({ data: { viewer: { accounts: [{ w: [{ sum: { requests: 5_000_000 } }] }] } } }));
        return new Response(JSON.stringify({ data: { viewer: { zones: [] } } }));
      }
      return new Response(JSON.stringify({ success: true, result: [], result_info: { total_pages: 1 } }));
    }) as typeof fetch;
    try {
      const { collectSummary } = await import("./summary");
      const s = await collectSummary("acct", "tok", new Date(Date.UTC(2026, 9, 16)));
      expect(s.usage.map((u) => u.id)).toEqual(["workers.requests_monthly"]);
      expect(s.usage[0].used).toBe(5_000_000);
      expect(s.errors.some((e) => e.includes("d1"))).toBe(true);
    } finally {
      globalThis.fetch = realFetch;
    }
  });
});
