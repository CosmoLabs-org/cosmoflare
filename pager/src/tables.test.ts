import { describe, expect, it } from "vitest";
import { readSortPref, writeSortPref } from "./tables";

// FEAT-pRC6EDA: the sort preference round-trips through storage; corrupt or
// missing entries fall back to the caller's defaults.

function memStorage(): Pick<Storage, "getItem" | "setItem"> & { map: Map<string, string> } {
  const map = new Map<string, string>();
  return {
    map,
    getItem: (k: string) => map.get(k) ?? null,
    setItem: (k: string, v: string) => void map.set(k, v),
  };
}

describe("sort preferences", () => {
  it("round-trips a chosen sort", () => {
    const s = memStorage();
    writeSortPref("workers", "cpuP99", "desc", s);
    expect(readSortPref("workers", s)).toEqual({ key: "cpuP99", dir: "desc" });
  });
  it("returns undefined for a missing preference", () => {
    expect(readSortPref("d1", memStorage())).toBeUndefined();
  });
  it("ignores corrupt or partial entries", () => {
    const s = memStorage();
    s.map.set("cf-sort:zones", "{not json");
    expect(readSortPref("zones", s)).toBeUndefined();
    s.map.set("cf-sort:zones", JSON.stringify({ key: "x", dir: "sideways" }));
    expect(readSortPref("zones", s)).toBeUndefined();
  });
});
