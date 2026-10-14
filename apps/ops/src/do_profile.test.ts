import { afterEach, describe, expect, it, vi } from "vitest";
import { collectDoProfile } from "./do_profile";

// The namespaces REST list collectDurableObjects reads (vi.mock hoists this
// above the imports, the durable_objects.test.ts pattern).
vi.mock("./summary", () => ({
  rest: vi.fn(async () => [
    { id: "ns1", script: "ccs-state-worker" },
    { id: "ns2", script: "idle-worker" },
  ]),
}));

afterEach(() => {
  vi.unstubAllGlobals();
});

/** GraphQL response body for the 24h invocations query. */
function graphqlBody(rows: { scriptName: string; requests: number; errors: number }[], gqlErrors: string[] = []): string {
  return JSON.stringify({
    data: {
      viewer: {
        accounts: [{
          o: rows.map((r) => ({ dimensions: { scriptName: r.scriptName }, sum: { requests: r.requests, errors: r.errors } })),
        }],
      },
    },
    errors: gqlErrors.length > 0 ? gqlErrors.map((message) => ({ message })) : undefined,
  });
}

/** Stub the GraphQL fetch for the WHOLE test — the collector awaits the
 *  (mocked) namespaces list first, so a microtask-restore would unstub
 *  fetch before the GraphQL call fires and hit the real network. */
function mockFetchOnce(body: string): void {
  vi.stubGlobal("fetch", (async () => new Response(body, { status: 200 })) as typeof fetch);
}

describe("collectDoProfile", () => {
  it("returns the namespace's 24h analytics row with its REST namespace ID", async () => {
    mockFetchOnce(graphqlBody([{ scriptName: "ccs-state-worker", requests: 20409, errors: 43 }]));
    const got = await collectDoProfile("acct", "tok", "ccs-state-worker", new Date("2026-10-10T00:00:00Z"));
    expect(got.found).toBe(true);
    expect(got.name).toBe("ccs-state-worker");
    expect(got.requests).toBe(20409);
    expect(got.errors).toBe(43);
    expect(got.errorPct).toBeCloseTo((43 / 20409) * 100, 6);
    expect(got.namespaceId).toBe("ns1");
    expect(got.collectionErrors).toEqual([]);
  });

  it("an unknown namespace is found:false with zeros — quiet, not an error", async () => {
    mockFetchOnce(graphqlBody([{ scriptName: "ccs-state-worker", requests: 5, errors: 0 }]));
    const got = await collectDoProfile("acct", "tok", "no-such-do", new Date("2026-10-10T00:00:00Z"));
    expect(got.found).toBe(false);
    expect(got.requests).toBe(0);
    expect(got.errors).toBe(0);
    expect(got.namespaceId).toBeNull();
    expect(got.collectionErrors).toEqual([]);
  });

  it("partial GraphQL success stays quiet — the row lands, warnings drop", async () => {
    // collectDurableObjects' shipped contract: a GraphQL errors[] alongside
    // returned rows is a partial success — rows win, no error line. Only a
    // rowless response with errors surfaces the line.
    mockFetchOnce(graphqlBody([{ scriptName: "idle-worker", requests: 7, errors: 1 }], ["dataset denied"]));
    const got = await collectDoProfile("acct", "tok", "idle-worker", new Date("2026-10-10T00:00:00Z"));
    expect(got.found).toBe(true);
    expect(got.requests).toBe(7);
    expect(got.errors).toBe(1);
    expect(got.namespaceId).toBe("ns2");
    expect(got.collectionErrors).toEqual([]);
  });
});
