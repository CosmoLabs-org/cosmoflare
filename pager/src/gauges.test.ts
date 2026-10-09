import { describe, expect, it } from "vitest";
import {
  ariaLabelFor,
  arcLens,
  clampPct,
  fmtGaugePct,
  gaugeLevel,
  levelFor,
  limitLabel,
  scaleFitPct,
  sortByProjectedDesc,
} from "./gauges";

describe("levelFor", () => {
  it("bands the projected % of allowance at 75/90", () => {
    expect(levelFor(0)).toBe("ok");
    expect(levelFor(74.9)).toBe("ok");
    expect(levelFor(75)).toBe("warning");
    expect(levelFor(89.9)).toBe("warning");
    expect(levelFor(90)).toBe("critical");
    expect(levelFor(150)).toBe("critical");
  });
});

describe("limitLabel", () => {
  it("says Over limit past 100%, Near limit below", () => {
    expect(limitLabel(150)).toBe("Over limit");
    expect(limitLabel(100)).toBe("Near limit");
    expect(limitLabel(80)).toBe("Near limit");
  });
});

describe("gaugeLevel", () => {
  it("runs the bands on the projection", () => {
    expect(gaugeLevel(4, 7)).toEqual({ level: "ok", overLimit: false, limitText: null });
    expect(gaugeLevel(40, 80)).toEqual({ level: "warning", overLimit: false, limitText: "Near limit" });
    expect(gaugeLevel(40, 95)).toEqual({ level: "critical", overLimit: false, limitText: "Near limit" });
  });
  it("flips to Over limit when used or projected passes 100%", () => {
    expect(gaugeLevel(120, 40)).toEqual({ level: "critical", overLimit: true, limitText: "Over limit" });
    expect(gaugeLevel(40, 150)).toEqual({ level: "critical", overLimit: true, limitText: "Over limit" });
  });
});

describe("arcLens", () => {
  const r = 10;
  const C = 2 * Math.PI * r;
  it("0% draws nothing", () => {
    const g = arcLens(0, 0, r);
    expect(g.usedLen).toBe(0);
    expect(g.projectedLen).toBe(0);
    expect(g.overflow).toBe(false);
  });
  it("50% draws half the circumference", () => {
    const g = arcLens(50, 75, r);
    expect(g.usedLen).toBeCloseTo(C / 2);
    expect(g.projectedLen).toBeCloseTo((3 * C) / 4);
    expect(g.overflow).toBe(false);
  });
  it("100% draws the full ring without the notch", () => {
    const g = arcLens(100, 100, r);
    expect(g.usedLen).toBeCloseTo(C);
    expect(g.projectedLen).toBeCloseTo(C);
    expect(g.overflow).toBe(false);
  });
  it("150% caps the arc and flags the overflow", () => {
    const g = arcLens(150, 0, r);
    expect(g.usedLen).toBeCloseTo(C);
    expect(g.overflow).toBe(true);
  });
  it("a projected overflow flags too, and negatives clamp to 0", () => {
    const g = arcLens(40, 120, r);
    expect(g.projectedLen).toBeCloseTo(C);
    expect(g.overflow).toBe(true);
    expect(arcLens(-10, -5, r).usedLen).toBe(0);
  });
});

describe("clampPct", () => {
  it("clamps into 0..100", () => {
    expect(clampPct(-5)).toBe(0);
    expect(clampPct(42)).toBe(42);
    expect(clampPct(184)).toBe(100);
  });
});

describe("fmtGaugePct", () => {
  it("one decimal below 100, integer at 100+", () => {
    expect(fmtGaugePct(4)).toBe("4");
    expect(fmtGaugePct(7.25)).toBe("7.3");
    expect(fmtGaugePct(184.56)).toBe("185");
    expect(fmtGaugePct(99.96)).toBe("100");
  });
});

describe("ariaLabelFor", () => {
  it("matches the task's example shape", () => {
    expect(ariaLabelFor({ label: "Workers requests", usedPct: 4, projectedPct: 7, sublabel: "10M requests" })).toBe(
      "Workers requests: 4% used, projected 7% of 10M requests",
    );
  });
  it("falls back to 'allowance' without a sublabel", () => {
    expect(ariaLabelFor({ label: "R2 storage", usedPct: 214, projectedPct: 214 })).toBe(
      "R2 storage: 214% used, projected 214% of allowance",
    );
  });
});

describe("sortByProjectedDesc", () => {
  it("orders worst-first, ties broken by id", () => {
    const items = [
      { id: "a", projected: 500, included: 1000 },
      { id: "c", projected: 100, included: 1000 },
      { id: "b", projected: 900, included: 1000 },
    ];
    expect(sortByProjectedDesc(items).map((p) => p.id)).toEqual(["b", "a", "c"]);
  });
  it("treats a zero allowance as 0% (never a divide-by-zero)", () => {
    const items = [
      { id: "z", projected: 500, included: 0 },
      { id: "a", projected: 1, included: 1000 },
    ];
    expect(sortByProjectedDesc(items).map((p) => p.id)).toEqual(["a", "z"]);
  });
});

describe("scaleFitPct", () => {
  it("fits values against a scale; zero scale is safe", () => {
    expect(scaleFitPct(50, 200)).toBe(25);
    expect(scaleFitPct(200, 200)).toBe(100);
    expect(scaleFitPct(10, 0)).toBe(0);
  });
});
