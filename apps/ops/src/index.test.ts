import { describe, expect, it } from "vitest";
import worker, { summaryResponse } from "./index";

const env = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "tok", VAPID_PUBLIC_KEY: "BPUBLICKEY", ACCESS_TEAM_DOMAIN: "", ACCESS_AUD: "", ASSETS: { fetch: async () => new Response("asset") } };

function fakeCtx(): { ctx: ExecutionContext; waits: Promise<unknown>[] } {
  const waits: Promise<unknown>[] = [];
  const ctx = {
    waitUntil: (p: Promise<unknown>): void => {
      waits.push(p);
    },
    passThroughOnException: (): void => {},
  } as unknown as ExecutionContext;
  return { ctx, waits };
}

describe("worker routing and access guard", () => {
  it("rejects /api/* without the Cloudflare Access assertion header (defense in depth)", async () => {
    const { ctx } = fakeCtx();
    const res = await worker.fetch(new Request("https://ops.example/api/vapid-public-key"), env as never, ctx);
    expect(res.status).toBe(403);
  });
  it("rejects a forged assertion header (fails closed without verifiable Access config)", async () => {
    const { ctx } = fakeCtx();
    const res = await worker.fetch(new Request("https://ops.example/api/vapid-public-key", { headers: { "Cf-Access-Jwt-Assertion": "x" } }), env as never, ctx);
    expect(res.status).toBe(403);
  });
  it("serves static assets for non-API paths", async () => {
    const { ctx } = fakeCtx();
    const res = await worker.fetch(new Request("https://ops.example/"), env as never, ctx);
    expect(await res.text()).toBe("asset");
  });
});

// Full upstream mock: GraphQL (per-script, monthly usage, D1, zones) + REST lists.
function mockUpstream(counts: { fetches: number }): typeof fetch {
  const real = globalThis.fetch;
  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
    counts.fetches++;
    const url = String(input);
    const body = init?.body ? String(init.body) : "";
    if (url.endsWith("/graphql")) {
      if (body.includes("scriptName")) {
        return new Response(JSON.stringify({ data: { viewer: { accounts: [{ w: [{ dimensions: { scriptName: "rafa-api" }, sum: { requests: 695, errors: 2 }, quantiles: { cpuTimeP50: 315, cpuTimeP99: 827 } }] }] } } }));
      }
      if (body.includes("databaseId")) {
        return new Response(JSON.stringify({ data: { viewer: { accounts: [{ g: [{ dimensions: { databaseId: "db1" }, sum: { rowsRead: 100, readQueries: 10, rowsWritten: 4 } }] }] } } }));
      }
      if (body.includes("httpRequestsAdaptiveGroups")) {
        return new Response(JSON.stringify({ data: { viewer: { zones: [{ zoneTag: "z1", httpRequestsAdaptiveGroups: [{ count: 50, dimensions: { cacheStatus: "hit" } }, { count: 10, dimensions: { cacheStatus: "dynamic" } }] }] } } }));
      }
      if (body.includes("d1AnalyticsAdaptiveGroups")) {
        return new Response(JSON.stringify({ data: { viewer: { accounts: [{ d: [{ sum: { rowsRead: 25 } }] }] } } }));
      }
      return new Response(JSON.stringify({ data: { viewer: { accounts: [{ w: [{ sum: { requests: 5_000_000 } }] }] } } }));
    }
    if (url.includes("/zones?")) return new Response(JSON.stringify({ success: true, result: [{ id: "z1", name: "zone.example" }], result_info: { total_pages: 1 } }));
    if (url.includes("/d1/database")) return new Response(JSON.stringify({ success: true, result: [{ uuid: "db1", name: "my-db" }], result_info: { total_pages: 1 } }));
    return new Response(JSON.stringify({ success: true, result: [], result_info: { total_pages: 1 } }));
  }) as typeof fetch;
  return real;
}

describe("summaryResponse upstream cache", () => {
  it("loads once, serves the second call from cache, and reports cache metadata", async () => {
    const counts = { fetches: 0 };
    const real = mockUpstream(counts);
    const { ctx } = fakeCtx();
    try {
      const first = await summaryResponse(ctx, env as never, false);
      const body1 = (await first.json()) as { cache: { ageSec: number; stale: boolean }; workers: { script: string; cpuP50Ms: number }[] };
      expect(body1.cache.stale).toBe(false);
      expect(body1.workers[0].script).toBe("rafa-api");
      expect(body1.workers[0].cpuP50Ms).toBe(0.315); // µs → ms
      const afterFirst = counts.fetches;
      expect(afterFirst).toBeGreaterThan(0);
      const second = await summaryResponse(ctx, env as never, false);
      const body2 = (await second.json()) as { cache: { ageSec: number; stale: boolean }; d1: { name: string; databaseId: string; rowsWritten: number }[]; zones: { byStatus: Record<string, number> }[] };
      expect(body2.cache.stale).toBe(false);
      expect(body2.cache.ageSec).toBeLessThan(300);
      expect(counts.fetches).toBe(afterFirst); // zero extra upstream calls
      expect(body2.d1[0]).toMatchObject({ name: "my-db", databaseId: "db1", rowsWritten: 4 });
      expect(body2.zones[0].byStatus).toEqual({ hit: 50, dynamic: 10 });
    } finally {
      globalThis.fetch = real;
    }
  });
  it("keeps serving the cached summary when the upstream goes down (stale-while-revalidate window)", async () => {
    const counts = { fetches: 0 };
    const real = mockUpstream(counts);
    const { ctx, waits } = fakeCtx();
    try {
      const first = await summaryResponse(ctx, env as never, false);
      const firstBody = (await first.json()) as { cache: { stale: boolean } };
      expect(firstBody.cache.stale).toBe(false);
      // Upstream dies; the L2/L1 path in this test still has a fresh L1 entry,
      // so the second call must not touch the dead upstream.
      globalThis.fetch = (async () => {
        throw new Error("upstream down");
      }) as typeof fetch;
      const second = await summaryResponse(ctx, env as never, false);
      const body = (await second.json()) as { cache: { stale: boolean }; workers: { script: string }[] };
      expect(body.cache.stale).toBe(false);
      expect(body.workers.length).toBeGreaterThan(0);
      await Promise.all(waits);
    } finally {
      globalThis.fetch = real;
    }
  });
});
