import { describe, expect, it } from "vitest";
import {
  addExclude,
  removeExclude,
  thresholdHint,
  validateRuleDraft,
  D1_ROWS_READ,
  type AlertRule,
} from "./rules";

// Pure helpers only — the suite runs in node-env vitest with no DOM
// library, matching the payload.test.ts / sort.test.ts convention.

describe("validateRuleDraft", () => {
  const CONDITIONS = ["d1-rows-read", "worker-error-pct", "zone-miss-pct"];

  it("accepts a valid draft", () => {
    const rule: AlertRule = { name: "D1 reads runaway", condition: D1_ROWS_READ, threshold: 1e9, enabled: true };
    expect(validateRuleDraft(rule, CONDITIONS)).toEqual([]);
  });

  it("rejects an empty name", () => {
    const rule: AlertRule = { name: "", condition: D1_ROWS_READ, threshold: 1e9, enabled: true };
    expect(validateRuleDraft(rule, CONDITIONS)).toContain("Name is required.");
  });

  it("rejects a whitespace-only name", () => {
    const rule: AlertRule = { name: "   ", condition: D1_ROWS_READ, threshold: 1e9, enabled: true };
    expect(validateRuleDraft(rule, CONDITIONS)).toContain("Name is required.");
  });

  it("rejects an unknown condition", () => {
    const rule: AlertRule = { name: "X", condition: "no-such-condition", threshold: 1, enabled: true };
    expect(validateRuleDraft(rule, CONDITIONS)).toContain("Choose a condition from the list.");
  });

  it("rejects a negative threshold", () => {
    const rule: AlertRule = { name: "X", condition: D1_ROWS_READ, threshold: -1, enabled: true };
    expect(validateRuleDraft(rule, CONDITIONS)).toContain("Threshold must be a number of 0 or more.");
  });

  it("rejects a non-finite threshold", () => {
    const rule: AlertRule = { name: "X", condition: D1_ROWS_READ, threshold: NaN, enabled: true };
    expect(validateRuleDraft(rule, CONDITIONS)).toContain("Threshold must be a number of 0 or more.");
  });

  it("accepts a zero threshold", () => {
    const rule: AlertRule = { name: "X", condition: D1_ROWS_READ, threshold: 0, enabled: true };
    expect(validateRuleDraft(rule, CONDITIONS)).toEqual([]);
  });

  it("collects every problem, not just the first", () => {
    const rule: AlertRule = { name: "", condition: "nope", threshold: -5, enabled: true };
    expect(validateRuleDraft(rule, CONDITIONS)).toHaveLength(3);
  });
});

describe("exclude chips", () => {
  it("adds a trimmed value", () => {
    expect(addExclude([], "  cf-state  ")).toEqual(["cf-state"]);
  });

  it("skips empty input", () => {
    expect(addExclude(["a"], "")).toEqual(["a"]);
    expect(addExclude(["a"], "   ")).toEqual(["a"]);
  });

  it("skips an exact duplicate", () => {
    expect(addExclude(["cf-state"], "cf-state")).toEqual(["cf-state"]);
  });

  it("keeps case-distinct names (Cloudflare names are case-sensitive)", () => {
    expect(addExclude(["cf-state"], "CF-State")).toEqual(["cf-state", "CF-State"]);
  });

  it("does not mutate the input list", () => {
    const before = ["a"];
    addExclude(before, "b");
    expect(before).toEqual(["a"]);
  });

  it("removes every exact match (trimmed)", () => {
    expect(removeExclude(["a", "b", "a"], " a ")).toEqual(["b"]);
  });

  it("removing a missing value is a no-op", () => {
    expect(removeExclude(["a"], "zz")).toEqual(["a"]);
  });

  it("add then remove round-trips", () => {
    let list = addExclude([], " media-bucket ");
    list = addExclude(list, "backups");
    list = removeExclude(list, "media-bucket");
    expect(list).toEqual(["backups"]);
  });
});

describe("thresholdHint", () => {
  it("formats a d1-rows-read threshold as a count", () => {
    expect(thresholdHint(D1_ROWS_READ, 1e9)).toBe("1.0B");
  });

  it("formats a kv-writes threshold as a count (FB-29)", () => {
    expect(thresholdHint("kv-writes", 250_000)).toBe("250.0k");
  });

  it("returns null for other conditions", () => {
    expect(thresholdHint("worker-error-pct", 1)).toBeNull();
  });

  it("returns null for non-positive and non-finite thresholds", () => {
    expect(thresholdHint(D1_ROWS_READ, 0)).toBeNull();
    expect(thresholdHint(D1_ROWS_READ, NaN)).toBeNull();
  });
});
