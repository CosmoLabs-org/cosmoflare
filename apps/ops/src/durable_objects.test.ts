import { describe, expect, it, vi } from "vitest";
import { collectDurableObjects, errorPct, mergeDoRows } from "./durable_objects";

vi.mock("./summary", () => ({
  rest: vi.fn(async () => [
    { id: "ns1", script: "ccs-state-worker" },
    { id: "ns2", script: "idle-worker" },
  ]),
}));

const USAGE = {
  "ccs-state-worker": { requests: 20409, errors: 43 },
};

describe("mergeDoRows (FEAT-p3KJAC2)", () => {
  it("joins namespaces with usage; idle namespaces appear with zeros; worst first", () => {
    const rows = mergeDoRows(
      [
        { id: "ns1", script: "ccs-state-worker" },
        { id: "ns2", script: "idle-worker" },
      ],
      USAGE,
    );
    expect(rows).toHaveLength(2);
    expect(rows[0].namespaceId).toBe("ns1");
    expect(rows[0].requests).toBe(20409);
    expect(rows[0].errorPct).toBeCloseTo((43 / 20409) * 100, 6);
    expect(rows[1]).toMatchObject({ namespaceId: "ns2", script: "idle-worker", requests: 0, errors: 0, errorPct: 0 });
  });
  it("a script with usage but no namespace row keeps an empty id", () => {
    const rows = mergeDoRows([], { "ghost-worker": { requests: 5, errors: 0 } });
    expect(rows[0]).toMatchObject({ namespaceId: "", script: "ghost-worker", requests: 5 });
  });
});

describe("errorPct", () => {
  it("zero requests means zero percent, never NaN", () => {
    expect(errorPct(0, 5)).toBe(0);
    expect(errorPct(100, 1)).toBe(1);
  });
});

describe("collectDurableObjects", () => {
  it("collects namespaces + usage with soft failures", async () => {
    const orig = globalThis.fetch;
    globalThis.fetch = (async () =>
      new Response(JSON.stringify({ data: { viewer: { accounts: [{ o: [{ dimensions: { scriptName: "ccs-state-worker" }, sum: { requests: 20409, errors: 43 } }] }] } } }), { status: 200 })) as typeof fetch;
    try {
      const got = await collectDurableObjects("acct", "tok", new Date("2026-10-10T00:00:00Z"));
      expect(got.namespaces).toBe(2);
      expect(got.rows[0].requests).toBe(20409);
      expect(got.errors).toEqual([]);
    } finally {
      globalThis.fetch = orig;
    }
  });
  it("a GraphQL failure lands in errors, namespaces still counted", async () => {
    const orig = globalThis.fetch;
    globalThis.fetch = (async () => new Response(JSON.stringify({ errors: [{ message: "dataset denied" }] }), { status: 200 })) as typeof fetch;
    try {
      const got = await collectDurableObjects("acct", "tok");
      expect(got.namespaces).toBe(2);
      expect(got.errors[0]).toContain("dataset denied");
      expect(got.rows.every((r) => r.requests === 0)).toBe(true);
    } finally {
      globalThis.fetch = orig;
    }
  });
});
