import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  AUDIT_CAP,
  appendAudit,
  auditResponse,
  handleAttach,
  handleDetach,
  handleZoneRemove,
  NO_WRITE_TOKEN,
  parseProtectedHosts,
  readAudit,
  type AuditEntry,
  type LifecycleEnv,
} from "./lifecycle";

// The cache module is spied (not replaced): invalidate() calls are recorded
// for the cache-invalidation assertions while cached() keeps its real
// behavior (the zone handlers read the cached zones list through it).
const cacheMocks = vi.hoisted(() => ({ invalidate: vi.fn() }));
vi.mock("./cache", async (importOriginal) => {
  const mod = await importOriginal<typeof import("./cache")>();
  return { ...mod, invalidate: (...args: Parameters<typeof mod.invalidate>) => cacheMocks.invalidate(...args) };
});

// stubLifecycleFetch — the attachments.test.ts stub pattern, extended with
// the request (method + body) so mutation assertions can inspect the wire.
interface Route {
  match: (url: string, req: Request) => boolean;
  body: unknown;
  status?: number;
}
function stubLifecycleFetch(routes: Route[]): { requests: { url: string; method: string; body: unknown }[] } {
  const requests: { url: string; method: string; body: unknown }[] = [];
  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
    const req = new Request(input, init);
    requests.push({ url: req.url, method: req.method, body: req.body ? await req.json() : null });
    const route = routes.find((r) => r.match(req.url, req));
    if (!route) throw new Error(`unexpected fetch: ${req.method} ${req.url}`);
    return new Response(JSON.stringify(route.body), { status: route.status ?? 200 });
  }) as typeof fetch;
  return { requests };
}

// lifecycleKV — in-memory KVNamespace stand-in (rules-api.test.ts pattern)
// with the store exposed so tests can assert the audit bytes.
function lifecycleKV(): { kv: KVNamespace; store: Map<string, string> } {
  const store = new Map<string, string>();
  const kv = {
    get: async (key: string) => store.get(key) ?? null,
    put: async (key: string, value: string) => void store.set(key, value),
    delete: async (key: string) => void store.delete(key),
  } as unknown as KVNamespace;
  return { kv, store };
}

function fakeCtx(): ExecutionContext {
  return { waitUntil: () => undefined, passThroughOnException: () => undefined } as unknown as ExecutionContext;
}

function post(path: string, body: unknown): Request {
  return new Request(`https://ops.example${path}`, { method: "POST", body: JSON.stringify(body) });
}

function baseEnv(kv: KVNamespace, over: Partial<LifecycleEnv> = {}): LifecycleEnv {
  return {
    CF_ACCOUNT_ID: "acct",
    CF_API_TOKEN: "readtok",
    CF_API_TOKEN_WRITE: "writetok",
    OPS_KV: kv,
    PROTECTED_HOSTS: "cosmolabs.org,noble.coffee,cosmoflare.dev",
    ...over,
  };
}

// Zone-list stub: the zone handlers resolve server-side truth from the
// cached() zones list (GET /zones?account.id=… via rest(), paginated).
const ZONES_OK = { match: (u: string) => u.includes("/zones?account.id="), body: { success: true, result: [
  { id: "z1", name: "noble.coffee", status: "active" },
  { id: "z2", name: "example.dev", status: "active" },
], result_info: { total_pages: 1 } } };

async function auditOf(store: Map<string, string>): Promise<AuditEntry[]> {
  return JSON.parse(store.get("audit") ?? "[]") as AuditEntry[];
}

beforeEach(() => {
  cacheMocks.invalidate.mockClear();
});

describe("parseProtectedHosts", () => {
  it("splits, trims and drops empties; undefined yields an empty list", () => {
    expect(parseProtectedHosts(" a.org ,b.org,, ")).toEqual(["a.org", "b.org"]);
    expect(parseProtectedHosts(undefined)).toEqual([]);
  });
});

describe("POST /api/domains/detach (FEAT-pEWA6N6 phase 2)", () => {
  it("503s without a write token and still audits the attempt", async () => {
    const { kv, store } = lifecycleKV();
    const res = await handleDetach(post("/api/domains/detach", { kind: "pages", project: "noble-web", name: "staging.noble.coffee" }), baseEnv(kv, { CF_API_TOKEN_WRITE: undefined }), fakeCtx(), "owner@example.com");
    expect(res.status).toBe(503);
    expect((await res.json()) as { error: string }).toEqual({ error: NO_WRITE_TOKEN });
    const audit = await auditOf(store);
    expect(audit).toHaveLength(1);
    expect(audit[0]).toMatchObject({ action: "detach", target: "staging.noble.coffee", operator: "owner@example.com", ok: false, error: NO_WRITE_TOKEN });
  });

  it("detaches a Pages domain by project+hostname, audits, and invalidates the read caches", async () => {
    const { kv, store } = lifecycleKV();
    const { requests } = stubLifecycleFetch([
      { match: (u, req) => u.endsWith("/accounts/acct/pages/projects/noble-web/domains/staging.noble.coffee") && req.method === "DELETE", body: { success: true, result: null } },
    ]);
    const res = await handleDetach(post("/api/domains/detach", { kind: "pages", project: "noble-web", name: "staging.noble.coffee" }), baseEnv(kv), fakeCtx(), "owner@example.com");
    expect(res.status).toBe(200);
    expect((await res.json()) as { ok: boolean }).toMatchObject({ ok: true });
    expect(requests).toHaveLength(1);
    expect(requests[0].url).toBe("https://api.cloudflare.com/client/v4/accounts/acct/pages/projects/noble-web/domains/staging.noble.coffee");
    expect(requests[0].method).toBe("DELETE");
    const audit = await auditOf(store);
    expect(audit).toEqual([expect.objectContaining({ action: "detach", target: "noble-web/staging.noble.coffee", operator: "owner@example.com", ok: true })]);
    expect(cacheMocks.invalidate).toHaveBeenCalledTimes(2);
    expect(cacheMocks.invalidate).toHaveBeenCalledWith(expect.anything(), "attachments");
    expect(cacheMocks.invalidate).toHaveBeenCalledWith(expect.anything(), "domains");
  });

  it("surfaces a Cloudflare error body verbatim (HTTP 400 stub) and audits the failure", async () => {
    const { kv, store } = lifecycleKV();
    stubLifecycleFetch([
      { match: (u) => u.includes("/pages/projects/noble-web/domains/"), body: { success: false, errors: [{ message: "routing conflict: domain is in use" }] }, status: 400 },
    ]);
    const res = await handleDetach(post("/api/domains/detach", { kind: "pages", project: "noble-web", name: "staging.noble.coffee" }), baseEnv(kv), fakeCtx(), "owner@example.com");
    expect(res.status).toBe(502);
    expect((await res.json()) as { error: string }).toEqual({ error: "routing conflict: domain is in use" });
    const audit = await auditOf(store);
    expect(audit[0]).toMatchObject({ action: "detach", ok: false, error: "routing conflict: domain is in use" });
    expect(cacheMocks.invalidate).not.toHaveBeenCalled();
  });

  it("detaches a Workers custom domain BY domainId, not hostname", async () => {
    const { kv } = lifecycleKV();
    const { requests } = stubLifecycleFetch([
      { match: (u, req) => u.endsWith("/accounts/acct/workers/domains/dmn_123") && req.method === "DELETE", body: { success: true, result: null } },
    ]);
    const res = await handleDetach(post("/api/domains/detach", { kind: "workers", name: "api.example.dev", domainId: "dmn_123" }), baseEnv(kv), fakeCtx(), "owner@example.com");
    expect(res.status).toBe(200);
    expect(requests).toHaveLength(1);
    expect(requests[0].url).toBe("https://api.cloudflare.com/client/v4/accounts/acct/workers/domains/dmn_123");
  });

  it("rejects a workers detach without domainId and an unknown kind", async () => {
    const { kv } = lifecycleKV();
    const noId = await handleDetach(post("/api/domains/detach", { kind: "workers", name: "api.example.dev" }), baseEnv(kv), fakeCtx(), "op");
    expect(noId.status).toBe(400);
    const badKind = await handleDetach(post("/api/domains/detach", { kind: "zones" }), baseEnv(kv), fakeCtx(), "op");
    expect(badKind.status).toBe(400);
  });
});

describe("POST /api/domains/attach", () => {
  it("re-attaches a Pages domain with body {name}", async () => {
    const { kv } = lifecycleKV();
    const { requests } = stubLifecycleFetch([
      { match: (u, req) => u.endsWith("/accounts/acct/pages/projects/noble-web/domains") && req.method === "POST", body: { success: true, result: { name: "staging.noble.coffee" } } },
    ]);
    const res = await handleAttach(post("/api/domains/attach", { kind: "pages", project: "noble-web", name: "staging.noble.coffee" }), baseEnv(kv), fakeCtx(), "owner@example.com");
    expect(res.status).toBe(200);
    expect(requests[0].body).toEqual({ name: "staging.noble.coffee" });
    expect(cacheMocks.invalidate).toHaveBeenCalledWith(expect.anything(), "attachments");
  });

  it("re-attaches a Workers domain with {hostname, service, zone_id} resolved from the cached zones list", async () => {
    const { kv } = lifecycleKV();
    const { requests } = stubLifecycleFetch([
      ZONES_OK,
      { match: (u, req) => u.endsWith("/accounts/acct/workers/domains") && req.method === "PUT", body: { success: true, result: null } },
    ]);
    const res = await handleAttach(post("/api/domains/attach", { kind: "workers", name: "api.example.dev", service: "noble-api" }), baseEnv(kv), fakeCtx(), "owner@example.com");
    expect(res.status).toBe(200);
    expect(requests.some((r) => r.url.includes("/zones?account.id="))).toBe(true);
    const put = requests.find((r) => r.method === "PUT");
    expect(put?.body).toEqual({ hostname: "api.example.dev", service: "noble-api", zone_id: "z2" });
  });

  it("refuses a Workers attach whose hostname has no zone in the account", async () => {
    const { kv, store } = lifecycleKV();
    stubLifecycleFetch([ZONES_OK]);
    const res = await handleAttach(post("/api/domains/attach", { kind: "workers", name: "api.other.org", service: "x" }), baseEnv(kv), fakeCtx(), "owner@example.com");
    expect(res.status).toBe(400);
    expect(((await res.json()) as { error: string }).error).toContain("no zone found for api.other.org");
    expect((await auditOf(store))[0]).toMatchObject({ action: "attach", ok: false });
  });
});

describe("POST /api/zones/remove", () => {
  it("refuses a typed-name mismatch with 403 and never calls the zone DELETE", async () => {
    const { kv, store } = lifecycleKV();
    const { requests } = stubLifecycleFetch([ZONES_OK]);
    const res = await handleZoneRemove(post("/api/zones/remove", { zoneId: "z1", typedName: "noble.coffe", override: false }), baseEnv(kv), fakeCtx(), "owner@example.com");
    expect(res.status).toBe(403);
    expect(((await res.json()) as { error: string }).error).toContain('type "noble.coffee" exactly');
    expect(requests.filter((r) => r.method === "DELETE")).toHaveLength(0);
    expect((await auditOf(store))[0]).toMatchObject({ action: "zone-remove", target: "noble.coffee", ok: false });
  });

  it("refuses a protected host even with a matching typed name while the override flag is off", async () => {
    const { kv } = lifecycleKV();
    const { requests } = stubLifecycleFetch([ZONES_OK]);
    // override:true alone is not enough — ZONE_REMOVE_OVERRIDE is unset.
    const res = await handleZoneRemove(post("/api/zones/remove", { zoneId: "z1", typedName: "noble.coffee", override: true }), baseEnv(kv), fakeCtx(), "owner@example.com");
    expect(res.status).toBe(403);
    expect(((await res.json()) as { error: string }).error).toContain("protected host");
    expect(requests.filter((r) => r.method === "DELETE")).toHaveLength(0);
  });

  it("removes a non-protected zone on an exact typed name and invalidates the zone caches", async () => {
    const { kv } = lifecycleKV();
    const { requests } = stubLifecycleFetch([
      ZONES_OK,
      { match: (u, req) => u.endsWith("/zones/z2") && req.method === "DELETE", body: { success: true, result: null } },
    ]);
    const res = await handleZoneRemove(post("/api/zones/remove", { zoneId: "z2", typedName: "example.dev", override: false }), baseEnv(kv), fakeCtx(), "owner@example.com");
    expect(res.status).toBe(200);
    expect(requests.filter((r) => r.method === "DELETE").map((r) => r.url)).toEqual(["https://api.cloudflare.com/client/v4/zones/z2"]);
    for (const key of ["zones-list", "domains", "attachments"]) {
      expect(cacheMocks.invalidate).toHaveBeenCalledWith(expect.anything(), key);
    }
  });

  it("lets a protected zone through only with override:true AND ZONE_REMOVE_OVERRIDE=1", async () => {
    const { kv } = lifecycleKV();
    const { requests } = stubLifecycleFetch([
      ZONES_OK,
      { match: (u, req) => u.endsWith("/zones/z1") && req.method === "DELETE", body: { success: true, result: null } },
    ]);
    const stillRefused = await handleZoneRemove(post("/api/zones/remove", { zoneId: "z1", typedName: "noble.coffee", override: false }), baseEnv(kv, { ZONE_REMOVE_OVERRIDE: "1" }), fakeCtx(), "owner@example.com");
    expect(stillRefused.status).toBe(403);
    const ok = await handleZoneRemove(post("/api/zones/remove", { zoneId: "z1", typedName: "noble.coffee", override: true }), baseEnv(kv, { ZONE_REMOVE_OVERRIDE: "1" }), fakeCtx(), "owner@example.com");
    expect(ok.status).toBe(200);
    expect(requests.filter((r) => r.method === "DELETE").map((r) => r.url)).toEqual(["https://api.cloudflare.com/client/v4/zones/z1"]);
  });

  it("404s when the zoneId is not in the cached zone list", async () => {
    const { kv } = lifecycleKV();
    stubLifecycleFetch([ZONES_OK]);
    const res = await handleZoneRemove(post("/api/zones/remove", { zoneId: "zz", typedName: "whatever", override: false }), baseEnv(kv), fakeCtx(), "owner@example.com");
    expect(res.status).toBe(404);
  });
});

describe("audit log (KV 'audit')", () => {
  it("appends newest-first and caps the array at the 200 newest entries", async () => {
    const { kv, store } = lifecycleKV();
    for (let i = 0; i < AUDIT_CAP + 5; i++) {
      await appendAudit(baseEnv(kv), { ts: new Date(2026, 0, 1, 0, 0, i).toISOString(), action: "detach", target: `t${i}`, operator: "op", ok: true });
    }
    const audit = await auditOf(store);
    expect(audit).toHaveLength(AUDIT_CAP);
    expect(audit[0].target).toBe(`t${AUDIT_CAP + 4}`); // newest first
    expect(audit[AUDIT_CAP - 1].target).toBe("t5"); // oldest surviving
    expect(await readAudit(baseEnv(kv))).toEqual(audit); // readAudit parses the same bytes
  });

  it("GET /api/audit returns the list newest-first", async () => {
    const { kv } = lifecycleKV();
    await appendAudit(baseEnv(kv), { ts: "2026-10-12T00:00:00Z", action: "detach", target: "old", operator: "op", ok: true });
    await appendAudit(baseEnv(kv), { ts: "2026-10-12T01:00:00Z", action: "attach", target: "new", operator: "op", ok: false, error: "boom" });
    const res = await auditResponse(baseEnv(kv));
    expect(res.status).toBe(200);
    expect((await res.json()) as { audit: AuditEntry[] }).toEqual({
      audit: [
        expect.objectContaining({ target: "new", ok: false, error: "boom" }),
        expect.objectContaining({ target: "old", ok: true }),
      ],
    });
  });

  it("is inert without OPS_KV (mutations still work, audit reads empty)", async () => {
    await appendAudit({ CF_ACCOUNT_ID: "a", CF_API_TOKEN: "t" }, { ts: "x", action: "detach", target: "t", operator: "op", ok: true });
    const res = await auditResponse({ CF_ACCOUNT_ID: "a", CF_API_TOKEN: "t" });
    expect((await res.json()) as { audit: unknown[] }).toEqual({ audit: [] });
  });
});
