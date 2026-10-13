// Worker profile collector tests: happy path, per-section failure isolation,
// and the empty-schedules shape — global fetch stubbed (the domains.test.ts
// / index.test.ts upstream-mock pattern), KV stubbed as an in-memory map.
import { afterEach, describe, expect, it, vi } from "vitest";
import { collectWorkerProfile, type WorkerProfilePayload } from "./worker_profile";

const ENV = {
  CF_ACCOUNT_ID: "acct",
  CF_API_TOKEN: "tok",
  OPS_KV: {
    store: new Map<string, string>(),
    // Real KV contract: get(key, "json") returns the PARSED value — the
    // collector's cache check depends on it.
    get: async <T>(key: string, type?: string): Promise<T | null> => {
      const raw = ENV.OPS_KV.store.get(key);
      if (raw === undefined) return null;
      return (type === "json" ? JSON.parse(raw) : raw) as T;
    },
    put: async (key: string, value: string): Promise<void> => {
      ENV.OPS_KV.store.set(key, value);
    },
  },
};

afterEach(() => {
  ENV.OPS_KV.store.clear();
  vi.unstubAllGlobals();
});

/** Stub global fetch with a path→body routing table; non-matched paths 404. */
function stubFetch(routes: Record<string, unknown>): void {
  vi.stubGlobal("fetch", async (input: RequestInfo | URL): Promise<Response> => {
    const url = String(input);
    for (const [path, body] of Object.entries(routes)) {
      if (url.includes(path)) {
        // A pre-built Response must be cloned per call — fetchRetry's
        // second attempt would otherwise hit an already-read body.
        if (body instanceof Response) return body.clone();
        return new Response(JSON.stringify({ success: true, result: body }));
      }
    }
    return new Response(JSON.stringify({ success: false, errors: [{ message: "not found" }] }), { status: 404 });
  });
}

const emptyProfile = (): WorkerProfilePayload => ({
  generatedAt: "2026-10-13T00:00:00.000Z",
  bindings: [],
  cronTriggers: [],
  routes: [],
  versions: [],
  errors: [],
});

describe("collectWorkerProfile", () => {
  it("collects bindings, cron triggers, routes, and versions (happy path)", async () => {
    stubFetch({
      "/workers/scripts/api-gateway/settings": { bindings: [{ type: "kv_namespace", name: "CACHE" }, { type: "plain_text", name: "FLAG" }] },
      "/workers/scripts/api-gateway/schedules": [{ cron: "*/5 * * * *" }, { cron: "0 3 * * *" }],
      "/workers/scripts/api-gateway/versions": [
        { id: "v1", number: 3, metadata: { created_on: "2026-10-01T00:00:00Z" } },
        { id: "v2", number: 7, metadata: { created_on: "2026-10-05T00:00:00Z" } },
        { id: "v3", number: 5, metadata: { created_on: "2026-10-03T00:00:00Z" } },
      ],
      "/zones?account.id=acct": [{ id: "z1", name: "cosmolabs.org" }, { id: "z2", name: "noble.coffee" }],
      "/zones/z1/workers/routes": [{ pattern: "cosmolabs.org/api*", script: "api-gateway" }, { pattern: "cosmolabs.org/other*", script: "other" }],
      "/zones/z2/workers/routes": [{ pattern: "noble.coffee/*", script: "edge" }],
    });
    const got = await collectWorkerProfile(ENV as never, "api-gateway", new Date("2026-10-13T00:00:00Z"));
    expect(got).toEqual<WorkerProfilePayload>({
      generatedAt: "2026-10-13T00:00:00.000Z",
      bindings: [{ type: "kv_namespace", name: "CACHE" }, { type: "plain_text", name: "FLAG" }],
      cronTriggers: ["*/5 * * * *", "0 3 * * *"],
      routes: [{ zone: "cosmolabs.org", pattern: "cosmolabs.org/api*" }],
      versions: [
        { number: 7, createdOn: "2026-10-05T00:00:00Z" },
        { number: 5, createdOn: "2026-10-03T00:00:00Z" },
        { number: 3, createdOn: "2026-10-01T00:00:00Z" },
      ],
      errors: [],
    });
  });

  it("keeps the five most recent versions and caps there", async () => {
    stubFetch({
      "/workers/scripts/api-gateway/versions": Array.from({ length: 8 }, (_, i) => ({ id: `v${i}`, number: i + 1, metadata: { created_on: "2026-10-01T00:00:00Z" } })),
      "/workers/scripts/api-gateway/settings": { bindings: [] },
      "/workers/scripts/api-gateway/schedules": [],
      "/zones?account.id=acct": [],
    });
    const got = await collectWorkerProfile(ENV as never, "api-gateway");
    expect(got.versions).toHaveLength(5);
    expect(got.versions.map((v) => v.number)).toEqual([8, 7, 6, 5, 4]);
    expect(got.errors).toEqual([]);
  });

  it("a failing section becomes an errors line while the others still fill in", async () => {
    stubFetch({
      // 403 — the realistic section failure (missing Workers Scripts
      // Read) and NOT retryable, so fetchRetry returns it immediately.
      "/workers/scripts/api-gateway/settings": new Response(JSON.stringify({ success: false, errors: [{ message: "workers.api.error" }] }), { status: 403 }),
      "/workers/scripts/api-gateway/schedules": [{ cron: "*/5 * * * *" }],
      "/workers/scripts/api-gateway/versions": [{ id: "v1", number: 1, metadata: { created_on: "2026-10-01T00:00:00Z" } }],
      "/zones?account.id=acct": [],
    });
    const got = await collectWorkerProfile(ENV as never, "api-gateway");
    expect(got.bindings).toEqual([]);
    expect(got.errors).toEqual(["bindings unavailable: workers.api.error"]);
    expect(got.cronTriggers).toEqual(["*/5 * * * *"]);
    expect(got.versions).toEqual([{ number: 1, createdOn: "2026-10-01T00:00:00Z" }]);
  });

  it("empty schedules means no cron triggers and no error", async () => {
    stubFetch({
      "/workers/scripts/api-gateway/settings": { bindings: [] },
      "/workers/scripts/api-gateway/schedules": [],
      "/workers/scripts/api-gateway/versions": [],
      "/zones?account.id=acct": [],
    });
    const got = await collectWorkerProfile(ENV as never, "api-gateway");
    expect(got).toEqual<WorkerProfilePayload>({ ...emptyProfile(), generatedAt: got.generatedAt });
    expect(got.errors).toEqual([]);
  });

  it("routes reuse the scheduled engine's cached zone list (cron:zones) and skip stale entries", async () => {
    ENV.OPS_KV.store.set("cron:zones", JSON.stringify({ at: Date.now(), value: [{ id: "z1", name: "cached.example" }] }));
    let zoneFetches = 0;
    vi.stubGlobal("fetch", async (input: RequestInfo | URL): Promise<Response> => {
      const url = String(input);
      if (url.includes("/zones?account.id=")) {
        zoneFetches++;
        return new Response(JSON.stringify({ success: true, result: [] }));
      }
      if (url.includes("/zones/z1/workers/routes")) {
        return new Response(JSON.stringify({ success: true, result: [{ pattern: "cached.example/*", script: "api-gateway" }] }));
      }
      if (url.includes("/scripts/api-gateway/")) {
        return new Response(JSON.stringify({ success: true, result: {} }));
      }
      return new Response(JSON.stringify({ success: true, result: [] }));
    });
    const got = await collectWorkerProfile(ENV as never, "api-gateway");
    expect(zoneFetches).toBe(0); // KV cache absorbed the zone list
    expect(got.routes).toEqual([{ zone: "cached.example", pattern: "cached.example/*" }]);
  });
});
