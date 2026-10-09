import { describe, expect, it, vi } from "vitest";
import worker, { billingResponse, summaryResponse } from "./index";

// Mock the cron engine module: the routing tests below stub its entry points
// instead of running the real rule evaluation (that is scheduled.test.ts's job).
const scheduledMocks = vi.hoisted(() => ({
  runScheduled: vi.fn(),
  testFire: vi.fn(),
}));
vi.mock("./scheduled", () => ({
  runScheduled: scheduledMocks.runScheduled,
  testFire: scheduledMocks.testFire,
}));

// Auth bypass switch: the real verifyAccessJwt fails closed without Access
// config, so tests that exercise post-auth routing flip this on; the default
// false keeps the forbidden tests on the real fail-closed path.
const authBypass = vi.hoisted(() => ({ value: false }));
vi.mock("./access", async (importOriginal) => {
  const mod = await importOriginal<typeof import("./access")>();
  return {
    ...mod,
    verifyAccessJwt: async (...args: Parameters<typeof mod.verifyAccessJwt>) =>
      authBypass.value ? { ok: true as const, email: "owner@example.com" } : mod.verifyAccessJwt(...args),
  };
});

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

// ---------------------------------------------------------------------------
// /api/billing, /api/subscribe, /api/test-fire and the cron handler (FEAT-052)

// Billing upstream mock: the subscriptions REST call is a live-verified 403
// with the ops token's scopes (billing.ts), so resolvePeriod falls through to
// the calendar; GraphQL returns an empty account row (no usage rows to merge).
function mockBillingUpstream(counts: { fetches: number }): typeof fetch {
  const real = globalThis.fetch;
  globalThis.fetch = (async (input: RequestInfo | URL) => {
    counts.fetches++;
    const url = String(input);
    if (url.includes("/subscriptions")) return new Response("Authentication error", { status: 403 });
    return new Response(JSON.stringify({ data: { viewer: { accounts: [{}] } } }));
  }) as typeof fetch;
  return real;
}

describe("billingResponse upstream cache", () => {
  it("loads once, serves the second call from cache, and reports cache metadata", async () => {
    const counts = { fetches: 0 };
    const real = mockBillingUpstream(counts);
    try {
      const { ctx } = fakeCtx();
      const first = await billingResponse(ctx, env as never, false);
      const body1 = (await first.json()) as { cache: { ageSec: number; stale: boolean }; period: { source: string }; products: unknown[] };
      expect(body1.cache.stale).toBe(false);
      expect(body1.period.source).toBe("calendar");
      expect(Array.isArray(body1.products)).toBe(true);
      const afterFirst = counts.fetches;
      expect(afterFirst).toBeGreaterThan(0);
      const second = await billingResponse(fakeCtx().ctx, env as never, false);
      const body2 = (await second.json()) as { cache: { ageSec: number; stale: boolean } };
      expect(body2.cache.stale).toBe(false);
      expect(body2.cache.ageSec).toBeLessThan(900);
      expect(counts.fetches).toBe(afterFirst); // zero extra upstream calls: served from L1
    } finally {
      globalThis.fetch = real;
    }
  });
});

describe("api routes behind the Access gate", () => {
  it("rejects /api/subscribe without a valid Access JWT (defense in depth)", async () => {
    const { ctx } = fakeCtx();
    const res = await worker.fetch(new Request("https://ops.example/api/subscribe", { method: "POST", body: JSON.stringify({}) }), env as never, ctx);
    expect(res.status).toBe(403);
  });
  it("rejects /api/test-fire without a valid Access JWT (defense in depth)", async () => {
    const { ctx } = fakeCtx();
    const res = await worker.fetch(new Request("https://ops.example/api/test-fire", { method: "POST" }), env as never, ctx);
    expect(res.status).toBe(403);
  });
  it("rejects /api/rules without a valid Access JWT (defense in depth)", async () => {
    const { ctx } = fakeCtx();
    const res = await worker.fetch(new Request("https://ops.example/api/rules", { method: "PUT", body: JSON.stringify({ rules: [] }) }), env as never, ctx);
    expect(res.status).toBe(403);
  });
  it("returns 404 for an unknown /api path", async () => {
    authBypass.value = true;
    try {
      const { ctx } = fakeCtx();
      const res = await worker.fetch(new Request("https://ops.example/api/nope"), env as never, ctx);
      expect(res.status).toBe(404);
    } finally {
      authBypass.value = false;
    }
  });
  it("registers a subscription (POST /api/subscribe) and serves GET count", async () => {
    authBypass.value = true;
    const store = new Map<string, string>();
    const kv = {
      // emulates KVNamespace.get(key, "json"): parses the stored string
      get: async (key: string, type?: string) =>
        type === "json" && store.has(key) ? JSON.parse(store.get(key)!) : (store.get(key) ?? null),
      put: async (key: string, value: string) => void store.set(key, value),
    } as unknown as KVNamespace;
    const subEnv = { ...env, OPS_KV: kv } as never;
    try {
      const { ctx } = fakeCtx();
      const post = await worker.fetch(
        new Request("https://ops.example/api/subscribe", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ endpoint: "https://push.example/endpoint1", keys: { p256dh: "k1", auth: "a1" } }),
        }),
        subEnv,
        ctx,
      );
      expect(post.status).toBe(201);
      expect(await post.json()).toEqual({ ok: true, count: 1 });
      const get = await worker.fetch(new Request("https://ops.example/api/subscribe"), subEnv, ctx);
      expect(await get.json()).toEqual({ count: 1 });
    } finally {
      authBypass.value = false;
    }
  });
  it("POST /api/test-fire fires the test push; GET returns 405", async () => {
    authBypass.value = true;
    scheduledMocks.testFire.mockResolvedValueOnce({ sent: 2, pruned: 0, issues: [] });
    try {
      const { ctx } = fakeCtx();
      const post = await worker.fetch(new Request("https://ops.example/api/test-fire", { method: "POST" }), env as never, ctx);
      expect(post.status).toBe(200);
      expect(await post.json()).toEqual({ sent: 2, pruned: 0, issues: [] });
      const get = await worker.fetch(new Request("https://ops.example/api/test-fire"), env as never, ctx);
      expect(get.status).toBe(405);
    } finally {
      authBypass.value = false;
    }
  });
  it("the scheduled handler delegates to runScheduled via ctx.waitUntil (no floating promise)", async () => {
    scheduledMocks.runScheduled.mockResolvedValueOnce({ fired: 1, sent: 1, pruned: 0, gaps: [], issues: [] });
    const { ctx, waits } = fakeCtx();
    const controller = { cron: "*/5 * * * *" } as unknown as ScheduledController;
    await worker.scheduled!(controller, env as never, ctx);
    expect(scheduledMocks.runScheduled).toHaveBeenCalledWith(env);
    expect(waits.length).toBe(1);
    await Promise.all(waits);
  });
});
