// D1 profile collector tests: happy path (list lookup + metadata GET),
// unknown-name, list failure, and metadata 404 — global fetch stubbed (the
// worker_profile.test.ts upstream-mock pattern), KV stubbed as a map.
import { afterEach, describe, expect, it, vi } from "vitest";
import { collectD1Profile, type D1ProfilePayload } from "./d1_profile";

const ENV = {
  CF_ACCOUNT_ID: "acct",
  CF_API_TOKEN: "tok",
  OPS_KV: {
    store: new Map<string, string>(),
    get: async <T,>(key: string, type?: string): Promise<T | null> => {
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

/** Stub global fetch with a path-to-body routing table; non-matched paths 404. */
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

describe("collectD1Profile", () => {
  it("resolves the name via the database list and returns the metadata", async () => {
    stubFetch({
      // Most-specific path first: the stub matches by substring in
      // insertion order, so the bare list path must not shadow the uuid one.
      "/d1/database/u1": {
        uuid: "u1",
        name: "prod-db",
        created_at: "2026-01-05T00:00:00Z",
        version: "production",
        running_in_region: "WNAM",
        num_tables: 8,
        file_size: 12345678,
        read_replication: { mode: "auto", parts: 2 },
      },
      "/d1/database": [{ uuid: "u1", name: "prod-db" }, { uuid: "u2", name: "other" }],
    });
    const got = await collectD1Profile(ENV as never, "prod-db", {}, new Date("2026-10-14T00:00:00Z"));
    expect(got).toEqual<D1ProfilePayload>({
      generatedAt: "2026-10-14T00:00:00.000Z",
      metadata: {
        name: "prod-db",
        createdAt: "2026-01-05T00:00:00Z",
        version: "production",
        region: "WNAM",
        numTables: 8,
        fileSize: 12345678,
        readReplication: { mode: "auto", parts: 2 },
      },
      errors: [],
    });
  });

  it("an unknown name yields an errors line and no metadata", async () => {
    stubFetch({
      "/d1/database": [{ uuid: "u1", name: "prod-db" }],
      "/d1/database/u1": { name: "prod-db" },
    });
    const got = await collectD1Profile(ENV as never, "nope");
    expect(got.metadata).toBeNull();
    expect(got.errors).toEqual(["metadata unavailable: no database named nope"]);
  });

  it("a failing list lookup degrades to an errors line", async () => {
    stubFetch({
      "/d1/database": new Response(JSON.stringify({ success: false, errors: [{ message: "d1.api.error" }] }), { status: 403 }),
    });
    const got = await collectD1Profile(ENV as never, "prod-db");
    expect(got.metadata).toBeNull();
    expect(got.errors).toEqual(["metadata unavailable: d1.api.error"]);
  });

  it("a failing metadata GET degrades to an errors line", async () => {
    stubFetch({
      "/d1/database/u1": new Response(JSON.stringify({ success: false, errors: [{ message: "d1.not_found" }] }), { status: 404 }),
      "/d1/database": [{ uuid: "u1", name: "prod-db" }],
    });
    const got = await collectD1Profile(ENV as never, "prod-db");
    expect(got.metadata).toBeNull();
    expect(got.errors).toEqual(["metadata unavailable: d1.not_found"]);
  });

  it("an injected d1List dep replaces the REST call (index.ts cached list)", async () => {
    stubFetch({
      "/d1/database": [],
    });
    const got = await collectD1Profile(ENV as never, "prod-db", {
      d1List: async () => [{ uuid: "u9", name: "prod-db" }],
    });
    expect(got.metadata).not.toBeNull();
    expect(got.errors).toEqual([]);
  });
});
