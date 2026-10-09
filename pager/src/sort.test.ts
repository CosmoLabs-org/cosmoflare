import { describe, expect, it } from "vitest";
import { cmpRows, cmpText, type SortDir } from "./sort";

describe("cmpRows", () => {
  const rows = [
    { n: 5 },
    { n: null },
    { n: 1 },
    { n: undefined },
    { n: 3 },
  ];

  it("sorts numbers by direction", () => {
    const asc = [...rows].sort(cmpRows((r) => r.n, "asc")).map((r) => r.n);
    expect(asc).toEqual([1, 3, 5, null, undefined]);
    const desc = [...rows].sort(cmpRows((r) => r.n, "desc")).map((r) => r.n);
    expect(desc).toEqual([5, 3, 1, null, undefined]);
  });

  it("sorts nulls last in both directions", () => {
    const desc = [...rows].sort(cmpRows((r) => r.n, "desc")).map((r) => r.n);
    expect(desc.slice(-2)).toEqual([null, undefined]);
  });
});

describe("cmpText", () => {
  it("sorts strings by direction, locale-aware", () => {
    const rows = ["banana", "Apple", "cherry"];
    expect([...rows].sort(cmpText((s) => s, "asc"))).toEqual(["Apple", "banana", "cherry"]);
    expect([...rows].sort(cmpText((s) => s, "desc"))).toEqual(["cherry", "banana", "Apple"]);
  });
});

describe("SortDir", () => {
  it("is only asc or desc", () => {
    const dirs: SortDir[] = ["asc", "desc"];
    expect(dirs).toHaveLength(2);
  });
});
