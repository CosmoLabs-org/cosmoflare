// Zone profile collector tests: happy path, per-section failure isolation,
// unknown zone, and the KV-cached zone-list resolution — global fetch
// stubbed, KV stubbed as an in-memory map (the worker_profile.test.ts
// pattern).
import { afterEach, describe, expect, it, vi } from "vitest";
import { collectZoneProfile, type ZoneProfilePayload } from "./zone_profile";

const ENV = {
  CF_ACCOUNT_ID: "acct",
  CF_API_TOKEN: "tok",
  OPS_KV: {
    store: new Map<string, string>(),
    // Real KV contract: get(key, "json") returns the PARSED value.
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

// Never-enabled DNSSEC maps to null fields; the metadata fixture below is
// the /zones/{id} shape the metadata section maps.
const ZONE = {
  name: "cosmolabs.org",
  status: "active",
  plan: { name: "Free Website" },
  paused: false,
  type: "full",
  name_servers: ["a.ns.cloudflare.com", "b.ns.cloudflare.com"],
  created_on: "2021-03-01T00:00:00Z",
};

const PAYLOAD = (over: Partial<ZoneProfilePayload>): ZoneProfilePayload => ({
  generatedAt: "2026-10-14T00:00:00.000Z",
  name: "cosmolabs.org",
  zoneId: "z1",
  metadata: {
    name: "cosmolabs.org",
    status: "active",
    planName: "Free Website",
    paused: false,
    type: "full",
    nameServers: ["a.ns.cloudflare.com", "b.ns.cloudflare.com"],
    createdOn: "2021-03-01T00:00:00Z",
  },
  sslMode: "strict",
  dnssec: { status: "active", algorithm: "13" },
  errors: [],
  ...over,
});

describe("collectZoneProfile", () => {
  it("collects metadata, SSL mode, and DNSSEC (happy path, cold KV)", async () => {
    stubFetch({
      // ssl/dnssec keys must precede the bare "/zones/z1" metadata key —
      // stubFetch matches on url.includes in insertion order.
      "/zones/z1/settings/ssl": { value: "strict" },
      "/zones/z1/dnssec": { status: "active", algorithm: "13" },
      "/zones?account.id=acct": [{ id: "z1", name: "cosmolabs.org" }, { id: "z2", name: "noble.coffee" }],
      "/zones/z1": ZONE,
    });
    const got = await collectZoneProfile(ENV as never, "cosmolabs.org", new Date("2026-10-14T00:00:00Z"));
    expect(got).toEqual<ZoneProfilePayload>(PAYLOAD({}));
    // The cold pass backfills the cron:zones KV cache for later callers.
    expect(ENV.OPS_KV.store.has("cron:zones")).toBe(true);
  });

  it("resolves the id from a warm cron:zones cache without refetching /zones", async () => {
    ENV.OPS_KV.store.set(
      "cron:zones",
      JSON.stringify({ at: Date.now(), value: [{ id: "z9", name: "cosmolabs.org" }] }),
    );
    stubFetch({
      "/zones/z9/settings/ssl": { value: "strict" },
      "/zones/z9/dnssec": { status: "active", algorithm: "13" },
      "/zones/z9": { ...ZONE, plan: { name: "Warm plan" }, name_servers: ["a.ns.cloudflare.com"] },
    });
    const got = await collectZoneProfile(ENV as never, "cosmolabs.org", new Date("2026-10-14T00:00:00Z"));
    expect(got.zoneId).toBe("z9");
    expect(got.metadata?.planName).toBe("Warm plan");
  });

  it("zone list failure → one errors line, no sections, no throw", async () => {
    vi.stubGlobal("fetch", async (): Promise<Response> => {
      return new Response(JSON.stringify({ success: false, errors: [{ message: "auth fail" }] }), { status: 403 });
    });
    const got = await collectZoneProfile(ENV as never, "cosmolabs.org", new Date("2026-10-14T00:00:00Z"));
    expect(got.errors).toEqual(["zone list unavailable: auth fail"]);
    expect(got.zoneId).toBeNull();
    expect(got.metadata).toBeNull();
    expect(got.sslMode).toBeNull();
    expect(got.dnssec).toBeNull();
  });

  it("unknown zone name → 'zone not found' line, sections stay null", async () => {
    stubFetch({ "/zones?account.id=acct": [{ id: "z1", name: "cosmolabs.org" }] });
    const got = await collectZoneProfile(ENV as never, "ghost.test", new Date("2026-10-14T00:00:00Z"));
    expect(got.errors).toEqual(["zone not found: ghost.test"]);
    expect(got.zoneId).toBeNull();
    expect(got.metadata).toBeNull();
  });

  it("sections fail independently — SSL fails, metadata and DNSSEC still land", async () => {
    stubFetch({
      "/zones/z1/settings/ssl": new Response(JSON.stringify({ success: false, errors: [{ message: "missing ssl read" }] }), { status: 403 }),
      "/zones/z1/dnssec": { status: "active", algorithm: "13" },
      "/zones?account.id=acct": [{ id: "z1", name: "cosmolabs.org" }],
      "/zones/z1": ZONE,
    });
    const got = await collectZoneProfile(ENV as never, "cosmolabs.org", new Date("2026-10-14T00:00:00Z"));
    expect(got.metadata).not.toBeNull();
    expect(got.sslMode).toBeNull();
    expect(got.errors).toEqual(["SSL mode unavailable: missing ssl read"]);
    expect(got.dnssec).toEqual({ status: "active", algorithm: "13" });
  });

  it("DNSSEC result null (never enabled) → nulls carry through, no error", async () => {
    stubFetch({
      "/zones/z1/settings/ssl": { value: "flexible" },
      "/zones/z1/dnssec": { status: null, algorithm: null },
      "/zones?account.id=acct": [{ id: "z1", name: "cosmolabs.org" }],
      "/zones/z1": ZONE,
    });
    const got = await collectZoneProfile(ENV as never, "cosmolabs.org", new Date("2026-10-14T00:00:00Z"));
    expect(got.dnssec).toEqual({ status: null, algorithm: null });
    expect(got.errors).toEqual([]);
  });
});
