import { describe, expect, it } from "vitest";
import {
  COOLDOWN_MS,
  CONDITIONS,
  evaluate,
  formatValue,
  humanCount,
  pruneState,
  STATE_TTL_MS,
  type FireState,
  type Rule,
  type ZoneObs,
} from "./rules";

// ruleP builds a rule with defaults so test intent stays visible.
function ruleP(overrides: Partial<Rule>): Rule {
  return { name: "r", condition: "zone-uncached-requests", threshold: 1, enabled: true, ...overrides };
}

// stateP builds empty cooldown memory.
function stateP(): FireState {
  return { lastFired: {}, lastValue: {}, fires: {} };
}

// zoneObsP builds a zone observation row.
function zoneObsP(overrides: Partial<ZoneObs>): ZoneObs {
  return { zone: "a.example", uncached: 0, eligible: 500, missPct: 10, ...overrides };
}

describe("humanCount / formatValue (Go format.go + evaluator.go parity)", () => {
  it("renders row counts as 2.9B / 52.3M / 3.3k", () => {
    expect(humanCount(2_945_546_702)).toBe("2.9B");
    expect(humanCount(52_340_000)).toBe("52.3M");
    expect(humanCount(3_300)).toBe("3.3k");
    expect(humanCount(120)).toBe("120");
  });
  it("formats percents one decimal and requests plain", () => {
    expect(formatValue(33.333333, "%")).toBe("33.3");
    expect(formatValue(960, "requests")).toBe("960");
    expect(formatValue(1e9, "rows")).toBe("1.0B");
  });
});

describe("conditionValues thresholds", () => {
  it("zone-uncached-requests fires and names the zone; no floor on volume", () => {
    const zones = [zoneObsP({ zone: "hot.example", uncached: 12_345 })];
    const fires = evaluate([ruleP({ name: "uncached", threshold: 10000 })], zones, [], {}, stateP(), 1000);
    expect(fires).toHaveLength(1);
    expect(fires[0].ruleName).toBe("uncached");
    expect(fires[0].scopeName).toBe("hot.example");
    expect(fires[0].alertId).toBe("uncached/hot.example");
    expect(fires[0].message).toBe(
      "zone-uncached-requests hot.example [uncached]: observed 12345 meets threshold 10000 (requests)",
    );
  });
  it("zone-cache-miss-pct skips zones under the 100-eligible floor (missPct null)", () => {
    const zones = [zoneObsP({ zone: "tiny.example", eligible: 3, missPct: null })];
    const fires = evaluate([ruleP({ name: "misspct", condition: "zone-cache-miss-pct", threshold: 50 })], zones, [], {}, stateP(), 1000);
    expect(fires).toHaveLength(0);
  });
  it("d1-rows-read fires and names the database", () => {
    const d1 = [{ name: "big-db", rowsRead: 2_000_000_000 }];
    const fires = evaluate([ruleP({ name: "d1", condition: "d1-rows-read", threshold: 1e9 })], [], d1, {}, stateP(), 1000);
    expect(fires).toHaveLength(1);
    expect(fires[0].scopeName).toBe("big-db");
    expect(fires[0].message).toContain("observed 2.0B meets threshold 1.0B (rows)");
  });
  it("an unknown condition is skipped silently (registered-but-unimplemented parity)", () => {
    expect(evaluate([ruleP({ condition: "error-rate", threshold: 1 })], [zoneObsP({ uncached: 5000 })], [], {}, stateP(), 1000)).toHaveLength(0);
  });
});

describe("excludes", () => {
  it("case-insensitive exact match on name", () => {
    const zones = [zoneObsP({ zone: "Hot.Example", uncached: 99999 })];
    const fires = evaluate([ruleP({ exclude: ["hot.example"] })], zones, [], {}, stateP(), 1000);
    expect(fires).toHaveLength(0);
  });
  it("a non-excluded zone still fires", () => {
    const zones = [zoneObsP({ zone: "hot.example", uncached: 99999 }), zoneObsP({ zone: "keep.example", uncached: 99999 })];
    const fires = evaluate([ruleP({ exclude: ["HOT.EXAMPLE"] })], zones, [], {}, stateP(), 1000);
    expect(fires.map((f) => f.scopeName)).toEqual(["keep.example"]);
  });
});

describe("cooldown and escalation", () => {
  const rule = ruleP({ name: "uncached", threshold: 10000 });
  const hot = [zoneObsP({ zone: "hot.example", uncached: 12000 })];

  it("second run within the 1h cooldown is suppressed", () => {
    const st = stateP();
    expect(evaluate([rule], hot, [], {}, st, 0)).toHaveLength(1);
    expect(evaluate([rule], hot, [], {}, st, COOLDOWN_MS - 1)).toHaveLength(0);
  });
  it("re-pages after the cooldown elapses", () => {
    const st = stateP();
    evaluate([rule], hot, [], {}, st, 0);
    expect(evaluate([rule], hot, [], {}, st, COOLDOWN_MS)).toHaveLength(1);
  });
  it("escalates when the value is ≥2× the last paged value, with the escalation prefix", () => {
    const st = stateP();
    evaluate([rule], hot, [], {}, st, 0); // last value 12000
    const worse = [zoneObsP({ zone: "hot.example", uncached: 24000 })];
    const fires = evaluate([rule], worse, [], {}, st, COOLDOWN_MS - 1);
    expect(fires).toHaveLength(1);
    expect(fires[0].message.startsWith("escalated (≥2× last page): ")).toBe(true);
  });
  it("a 1.99× value inside the cooldown stays suppressed", () => {
    const st = stateP();
    evaluate([rule], hot, [], {}, st, 0);
    const fires = evaluate([rule], [zoneObsP({ zone: "hot.example", uncached: 23_999 })], [], {}, st, COOLDOWN_MS - 1);
    expect(fires).toHaveLength(0);
  });
  it("two offenders under one rule fire independently (separate cooldown keys)", () => {
    const st = stateP();
    const both = [zoneObsP({ zone: "a.example", uncached: 11000 }), zoneObsP({ zone: "b.example", uncached: 11000 })];
    expect(evaluate([rule], both, [], {}, st, 0)).toHaveLength(2);
    expect(evaluate([rule], both, [], {}, st, COOLDOWN_MS - 1)).toHaveLength(0);
    // only a.example doubled
    const onlyADoubled = [zoneObsP({ zone: "a.example", uncached: 22000 }), zoneObsP({ zone: "b.example", uncached: 11000 })];
    expect(evaluate([rule], onlyADoubled, [], {}, st, COOLDOWN_MS - 1)).toHaveLength(1);
  });
  it("CONDITIONS units match the Go registry", () => {
    expect(CONDITIONS["zone-uncached-requests"].unit).toBe("requests");
    expect(CONDITIONS["zone-cache-miss-pct"].unit).toBe("%");
    expect(CONDITIONS["d1-rows-read"].unit).toBe("rows");
  });
});

describe("pruneState (bounded fire-state)", () => {
  it("drops lastFired/lastValue/fires entries whose lastFired is older than 24h", () => {
    const now = 10 * STATE_TTL_MS;
    const st: FireState = {
      lastFired: { "old/x": now - STATE_TTL_MS - 1, "fresh/y": now - 1000 },
      lastValue: { "old/x": 5, "fresh/y": 6 },
      fires: { "old/x": 3, "fresh/y": 1 },
    };
    pruneState(st, now);
    expect(st.lastFired).toEqual({ "fresh/y": now - 1000 });
    expect(st.lastValue).toEqual({ "fresh/y": 6 });
    expect(st.fires).toEqual({ "fresh/y": 1 });
  });
  it("keeps an entry exactly at the 24h boundary (older-than is strict)", () => {
    const now = 5 * STATE_TTL_MS;
    const st: FireState = { lastFired: { "edge/z": now - STATE_TTL_MS }, lastValue: { "edge/z": 1 }, fires: { "edge/z": 1 } };
    pruneState(st, now);
    expect(st.lastFired).toEqual({ "edge/z": now - STATE_TTL_MS });
  });
  it("leaves empty state untouched", () => {
    const st = stateP();
    pruneState(st, 1000);
    expect(st).toEqual({ lastFired: {}, lastValue: {}, fires: {} });
  });
});

describe("telemetry gaps", () => {
  it("pages once per dataset per hour, only when an enabled rule depends on it", () => {
    const st = stateP();
    const d1Rule = ruleP({ name: "d1", condition: "d1-rows-read", threshold: 1e9 });
    const gaps = { zone: "zones list HTTP 500" };
    // d1 rule does not depend on zone data: no page
    expect(evaluate([d1Rule], [], [], gaps, st, 0)).toHaveLength(0);
    // a zone rule depends on it: page
    const zRule = ruleP({ name: "uncached", threshold: 10000 });
    const gapFires = evaluate([zRule], [], [], gaps, st, 0);
    expect(gapFires).toHaveLength(1);
    expect(gapFires[0].alertId).toBe("telemetry-gap/zone");
    expect(gapFires[0].message).toContain("telemetry gap: zone analytics unavailable");
    // suppressed within the hour
    expect(evaluate([zRule], [], [], gaps, st, COOLDOWN_MS - 1)).toHaveLength(0);
    // pages again after the hour
    expect(evaluate([zRule], [], [], gaps, st, COOLDOWN_MS)).toHaveLength(1);
  });
});
