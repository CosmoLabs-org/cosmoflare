import { describe, expect, it } from "vitest";
import {
  formatCount,
  formatUsd,
  formatPct,
  formatAge,
  formatDateShort,
  periodProgress,
  secondsUntil,
} from "./format";

describe("formatCount", () => {
  it("abbreviates at the k/M/B boundaries", () => {
    expect(formatCount(0)).toBe("0");
    expect(formatCount(999)).toBe("999");
    expect(formatCount(1_000)).toBe("1.0k");
    expect(formatCount(3_289)).toBe("3.3k");
    expect(formatCount(52_332_502)).toBe("52.3M");
  });
});

describe("formatUsd", () => {
  it("always renders two decimals with thousands separators", () => {
    expect(formatUsd(0)).toBe("$0.00");
    expect(formatUsd(4.2)).toBe("$4.20");
    expect(formatUsd(1234.5)).toBe("$1,234.50");
  });
});

describe("formatPct", () => {
  it("honors the digits argument", () => {
    expect(formatPct(74.24, 0)).toBe("74%");
    expect(formatPct(74.24)).toBe("74.2%");
    expect(formatPct(8.875, 2)).toBe("8.88%");
  });
});

describe("formatAge", () => {
  it("buckets ages into human units", () => {
    expect(formatAge(0)).toBe("just now");
    expect(formatAge(30)).toBe("just now");
    expect(formatAge(60)).toBe("1m ago");
    expect(formatAge(3_600)).toBe("1h ago");
    expect(formatAge(90_000)).toBe("1d ago");
  });
});

describe("formatDateShort", () => {
  it("renders an RFC3339 date as 'Oct 31'", () => {
    expect(formatDateShort("2026-10-31T00:00:00Z")).toMatch(/Oct 31/);
  });
  it("returns the input unchanged when unparseable", () => {
    expect(formatDateShort("not-a-date")).toBe("not-a-date");
  });
});

describe("periodProgress", () => {
  it("computes elapsed share and days left", () => {
    expect(periodProgress(10, 31)).toEqual({ elapsedPct: (10 / 31) * 100, daysLeft: 21 });
  });
  it("clamps elapsed to 100% and days-left to 0", () => {
    expect(periodProgress(40, 31).elapsedPct).toBe(100);
    expect(periodProgress(40, 31).daysLeft).toBe(0);
    expect(periodProgress(0, 31).elapsedPct).toBe(0);
    expect(periodProgress(31, 31).daysLeft).toBe(0);
  });
  it("clamps negative days to zero before computing", () => {
    expect(periodProgress(-2, 31).elapsedPct).toBe(0);
    expect(periodProgress(-2, 31).daysLeft).toBe(31);
  });
});

describe("secondsUntil", () => {
  it("counts down to an RFC3339 end", () => {
    const now = Date.parse("2026-10-09T12:00:00Z");
    expect(secondsUntil("2026-10-09T13:00:00Z", now)).toBe(3600);
  });
  it("is never negative and treats unparseable ends as 0", () => {
    const now = Date.parse("2026-10-09T12:00:00Z");
    expect(secondsUntil("2026-10-09T11:00:00Z", now)).toBe(0);
    expect(secondsUntil("garbage", now)).toBe(0);
  });
});
