import { describe, expect, it } from "vitest";
import { formatCount, levelForD1, levelForMiss, levelForUncached, levelForUsage } from "./dashboard";

describe("dashboard formatting", () => {
  it("formats counts like the CLI (2.9B / 52.3M / 3.3k / 522)", () => {
    expect(formatCount(2_945_546_702)).toBe("2.9B");
    expect(formatCount(52_332_502)).toBe("52.3M");
    expect(formatCount(3289)).toBe("3.3k");
    expect(formatCount(522)).toBe("522");
  });
  it("maps values to the starter alert thresholds (O10)", () => {
    expect(levelForUncached(36_321)).toBe("critical");
    expect(levelForUncached(8_192)).toBe("warning"); // amber band at half the 10k paging threshold
    expect(levelForUncached(4_000)).toBe("ok");
    expect(levelForMiss(74)).toBe("warning");
    expect(levelForMiss(null)).toBe("ok");
    expect(levelForD1(2.9e9)).toBe("critical");
    expect(levelForD1(52e6)).toBe("ok");
    expect(levelForUsage(120)).toBe("critical");
    expect(levelForUsage(85)).toBe("warning");
    expect(levelForUsage(10)).toBe("ok");
  });
});
