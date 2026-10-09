import { describe, expect, it } from "vitest";
import {
  ariaLabelFor,
  arcLens,
  capTopRings,
  clampPct,
  fmtGaugePct,
  gaugeLevel,
  labelLines,
  levelFor,
  limitLabel,
  productLabel,
  ringCenterValue,
  ringUsedLine,
  scaleFitPct,
  sortByProjectedDesc,
  todayDotPos,
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

describe("productLabel", () => {
  it("maps the Workers Paid billing ids to proper product names", () => {
    expect(productLabel("workers.requests", "Workers", "Requests")).toBe("Workers requests");
    expect(productLabel("workers.cpu_time", "Workers", "CPU time")).toBe("Workers CPU time");
    expect(productLabel("r2.class_a", "R2", "Class A")).toBe("R2 Class A ops");
    expect(productLabel("r2.class_b", "R2", "Class B")).toBe("R2 Class B ops");
    expect(productLabel("durable_objects.requests", "Durable Objects", "Requests")).toBe("Durable Objects requests");
    expect(productLabel("durable_objects.duration", "Durable Objects", "Duration")).toBe("Durable Objects duration");
    expect(productLabel("kv.reads", "KV", "Reads")).toBe("KV reads");
    expect(productLabel("kv.writes", "KV", "Writes")).toBe("KV writes");
    expect(productLabel("kv.storage", "KV", "Storage")).toBe("KV storage");
    expect(productLabel("d1.rows_read", "D1", "Rows read")).toBe("D1 rows read");
    expect(productLabel("d1.rows_written", "D1", "Rows written")).toBe("D1 rows written");
    expect(productLabel("d1.storage", "D1", "Storage")).toBe("D1 storage");
    expect(productLabel("r2.storage", "R2", "Storage (average)")).toBe("R2 storage");
  });
  it("falls back to 'product metric' for unknown ids (never an empty label)", () => {
    expect(productLabel("queuing.messages", "Queues", "Messages")).toBe("Queues messages");
  });
});

describe("capTopRings", () => {
  const ids = (n: number): { id: string }[] => Array.from({ length: n }, (_, i) => ({ id: `p${i}` }));
  it("keeps the 8 closest to their limit and counts the hidden rest", () => {
    const { shown, hiddenCount } = capTopRings(ids(13));
    expect(shown.map((p) => p.id)).toEqual(["p0", "p1", "p2", "p3", "p4", "p5", "p6", "p7"]);
    expect(hiddenCount).toBe(5);
  });
  it("exactly 8 products hide nothing", () => {
    const { shown, hiddenCount } = capTopRings(ids(8));
    expect(shown).toHaveLength(8);
    expect(hiddenCount).toBe(0);
  });
  it("fewer than 8 products hide nothing", () => {
    const { shown, hiddenCount } = capTopRings(ids(3));
    expect(shown).toHaveLength(3);
    expect(hiddenCount).toBe(0);
  });
});

describe("ring center text", () => {
  it("the big value is the projected %, not the used %", () => {
    expect(ringCenterValue(106)).toBe("106%");
    expect(ringCenterValue(7.25)).toBe("7.3%");
    expect(ringCenterValue(214)).toBe("214%");
  });
  it("the muted second line states the used share so far", () => {
    expect(ringUsedLine(30.7)).toBe("30.7% used so far");
    expect(ringUsedLine(4)).toBe("4% used so far");
  });
});

describe("todayDotPos", () => {
  const cx = 60;
  const cy = 60;
  const r = 50;
  it("sits on the track: 0% at 12 o'clock, 25% right, 50% bottom, back at top for 100%", () => {
    expect(todayDotPos(0, cx, cy, r)).toEqual({ x: 60, y: 10 });
    expect(todayDotPos(25, cx, cy, r).x).toBeCloseTo(110);
    expect(todayDotPos(25, cx, cy, r).y).toBeCloseTo(60);
    expect(todayDotPos(50, cx, cy, r)).toEqual({ x: 60, y: 110 });
    const full = todayDotPos(100, cx, cy, r);
    expect(full.x).toBeCloseTo(60);
    expect(full.y).toBeCloseTo(10);
  });
  it("clamps out-of-range positions onto the ring", () => {
    expect(todayDotPos(-10, cx, cy, r)).toEqual({ x: 60, y: 10 });
    const over = todayDotPos(140, cx, cy, r);
    expect(over.x).toBeCloseTo(60);
    expect(over.y).toBeCloseTo(10);
  });
  it("a half-period day lands at the 6 o'clock angle (day/days on the ring)", () => {
    const { x, y } = todayDotPos((17 / 30) * 100, cx, cy, r);
    expect(y).toBeGreaterThan(cy); // past the horizontal midline → lower half
    expect(x).toBeLessThan(cx); // 17/30 is just past 6 o'clock → left half
  });
});

describe("labelLines", () => {
  it("short labels stay one line", () => {
    expect(labelLines("D1 rows read")).toEqual(["D1 rows read"]);
  });
  it("long labels split into two balanced lines on a word boundary", () => {
    expect(labelLines("Durable Objects duration")).toEqual(["Durable Objects", "duration"]);
    expect(labelLines("Durable Objects requests")).toEqual(["Durable Objects", "requests"]);
  });
  it("a single long word never splits", () => {
    expect(labelLines("Supercalifragilisticexpialidocious")).toEqual(["Supercalifragilisticexpialidocious"]);
  });
});
