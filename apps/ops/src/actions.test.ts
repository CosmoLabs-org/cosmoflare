// handlePurge (operator action buttons, 2026-10-12): the first one-tap
// write-path action beyond domain lifecycle. Contract mirrors lifecycle's
// handlers: write-token gate → audited mutation with the WRITE token →
// the CF API's error text surfaces verbatim.
import { beforeEach, describe, expect, it } from "vitest";
import { handlePurge, handleDnsToggle } from "./actions";
import { readAudit, type LifecycleEnv } from "./lifecycle";

interface Route {
  match: (url: string, req: Request) => boolean;
  body: unknown;
  status?: number;
}

function stubFetch(routes: Route[]): { requests: { url: string; method: string; body: unknown }[] } {
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

function fakeKV(): KVNamespace {
  const store = new Map<string, string>();
  return {
    get: async (key: string) => store.get(key) ?? null,
    put: async (key: string, value: string) => void store.set(key, value),
    delete: async (key: string) => void store.delete(key),
  } as unknown as KVNamespace;
}

function fakeCtx(): ExecutionContext {
  return { waitUntil: () => undefined, passThroughOnException: () => undefined } as unknown as ExecutionContext;
}

function post(body: unknown): Request {
  return new Request("https://ops.example/api/zones/purge", { method: "POST", body: JSON.stringify(body) });
}

function postToggle(body: unknown): Request {
  return new Request("https://ops.example/api/dns/toggle", { method: "POST", body: JSON.stringify(body) });
}

const OPERATOR = "gab@cosmolabs.org";

const ZONES_OK = {
  match: (u: string) => u.includes("/zones?account.id="),
  body: { success: true, result: [{ id: "z1", name: "noble.coffee", status: "active" }, { id: "z2", name: "example.dev", status: "active" }, { id: "z3", name: "cosmolabs.org", status: "active" }], result_info: { total_pages: 1 } },
};

beforeEach(() => {
  globalThis.fetch = ((_input: RequestInfo | URL, _init?: RequestInit) => Promise.resolve(new Response("{}", { status: 200 }))) as typeof fetch;
});

describe("handlePurge", () => {
  it("purges by zone NAME through the cached zones list, with the WRITE token, and audits ok", async () => {
    const kv = fakeKV();
    const env: LifecycleEnv = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "readtok", CF_API_TOKEN_WRITE: "writetok", OPS_KV: kv } as LifecycleEnv;
    const { requests } = stubFetch([
      ZONES_OK,
      { match: (u, r) => u.endsWith("/purge_cache") && r.method === "POST", body: { success: true } },
    ]);
    const res = await handlePurge(post({ zone: "noble.coffee" }), env, fakeCtx(), OPERATOR);
    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({ zone: "noble.coffee", purged: true });
    const purge = requests.find((r) => r.url.endsWith("/purge_cache"));
    expect(purge?.url).toContain("/zones/z1/purge_cache");
    expect(purge?.body).toEqual({ purge_everything: true });
    const audit = await readAudit(env);
    expect(audit[0]).toMatchObject({ action: "purge", target: "noble.coffee", operator: OPERATOR, ok: true });
  });

  it("accepts a raw zone id too", async () => {
    const env: LifecycleEnv = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "r", CF_API_TOKEN_WRITE: "w", OPS_KV: fakeKV() } as LifecycleEnv;
    const { requests } = stubFetch([
      ZONES_OK,
      { match: (u) => u.endsWith("/purge_cache"), body: { success: true } },
    ]);
    const res = await handlePurge(post({ zone: "z2" }), env, fakeCtx(), OPERATOR);
    expect(res.status).toBe(200);
    expect(requests.find((r) => r.url.endsWith("/purge_cache"))?.url).toContain("/zones/z2/");
  });

  it("503 + audit when no write token is configured", async () => {
    const kv = fakeKV();
    const env: LifecycleEnv = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "r", OPS_KV: kv } as unknown as LifecycleEnv;
    const res = await handlePurge(post({ zone: "noble.coffee" }), env, fakeCtx(), OPERATOR);
    expect(res.status).toBe(503);
    const audit = await readAudit(env);
    expect(audit[0]).toMatchObject({ action: "purge", ok: false, error: expect.stringContaining("CF_API_TOKEN_WRITE") });
  });

  it("surfaces the CF error verbatim with the permission hint and audits the failure", async () => {
    const kv = fakeKV();
    const env: LifecycleEnv = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "r", CF_API_TOKEN_WRITE: "w", OPS_KV: kv } as LifecycleEnv;
    stubFetch([
      ZONES_OK,
      { match: (u) => u.endsWith("/purge_cache"), body: { success: false, errors: [{ message: "not entitled to purge" }] } },
    ]);
    const res = await handlePurge(post({ zone: "noble.coffee" }), env, fakeCtx(), OPERATOR);
    expect(res.status).toBe(502);
    const body = (await res.json()) as { error: string; hint: string };
    expect(body.error).toBe("not entitled to purge");
    expect(body.hint).toContain("Cache Purge");
    const audit = await readAudit(env);
    expect(audit[0]).toMatchObject({ action: "purge", ok: false, error: "not entitled to purge" });
  });

  it("400 without a zone, 404 for an unknown zone", async () => {
    const env: LifecycleEnv = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "r", CF_API_TOKEN_WRITE: "w", OPS_KV: fakeKV() } as LifecycleEnv;
    const bad = await handlePurge(post({}), env, fakeCtx(), OPERATOR);
    expect(bad.status).toBe(400);
    stubFetch([ZONES_OK]);
    const missing = await handlePurge(post({ zone: "nope.example" }), env, fakeCtx(), OPERATOR);
    expect(missing.status).toBe(404);
  });
});

describe("handleDnsToggle", () => {
  const DNS_PATCH = { match: (u: string, r: Request) => u.includes("/dns_records/") && r.method === "PATCH", body: { success: true } };

  it("pauses a record on a normal zone with the WRITE token and audits the pause", async () => {
    const kv = fakeKV();
    const env: LifecycleEnv = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "r", CF_API_TOKEN_WRITE: "w", OPS_KV: kv, PROTECTED_HOSTS: "cosmolabs.org" } as unknown as LifecycleEnv;
    const { requests } = stubFetch([ZONES_OK, DNS_PATCH]);
    const res = await handleDnsToggle(postToggle({ zone: "example.dev", record: "rec1", name: "api.example.dev", paused: true }), env, fakeCtx(), OPERATOR);
    expect(res.status).toBe(200);
    expect(await res.json()).toMatchObject({ zone: "example.dev", record: "rec1", paused: true });
    const patch = requests.find((r) => r.method === "PATCH");
    expect(patch?.url).toContain("/zones/z2/dns_records/rec1");
    expect(patch?.body).toEqual({ paused: true });
    const audit = await readAudit(env);
    expect(audit[0]).toMatchObject({ action: "dns-toggle", target: "example.dev: api.example.dev → paused", ok: true });
  });

  it("pausing a record on a PROTECTED host refuses 403 without the operator override", async () => {
    const kv = fakeKV();
    const env: LifecycleEnv = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "r", CF_API_TOKEN_WRITE: "w", OPS_KV: kv, PROTECTED_HOSTS: "cosmolabs.org" } as unknown as LifecycleEnv;
    const { requests } = stubFetch([ZONES_OK, DNS_PATCH]);
    const res = await handleDnsToggle(postToggle({ zone: "cosmolabs.org", record: "rec1", paused: true }), env, fakeCtx(), OPERATOR);
    expect(res.status).toBe(403);
    expect(((await res.json()) as { error: string }).error).toContain("protected host");
    expect(requests.find((r) => r.method === "PATCH")).toBeUndefined(); // never reached CF
    const audit = await readAudit(env);
    expect(audit[0]).toMatchObject({ action: "dns-toggle", ok: false });
  });

  it("RE-ENABLING a record on a protected host is always allowed (restores service)", async () => {
    const env: LifecycleEnv = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "r", CF_API_TOKEN_WRITE: "w", OPS_KV: fakeKV(), PROTECTED_HOSTS: "cosmolabs.org" } as unknown as LifecycleEnv;
    const { requests } = stubFetch([ZONES_OK, DNS_PATCH]);
    const res = await handleDnsToggle(postToggle({ zone: "cosmolabs.org", record: "rec1", paused: false }), env, fakeCtx(), OPERATOR);
    expect(res.status).toBe(200);
    expect(requests.find((r) => r.method === "PATCH")?.body).toEqual({ paused: false });
  });

  it("503 without a write token; 400 on a malformed body; CF errors surface verbatim", async () => {
    const noToken: LifecycleEnv = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "r", OPS_KV: fakeKV() } as unknown as LifecycleEnv;
    const gated = await handleDnsToggle(postToggle({ zone: "example.dev", record: "rec1", paused: true }), noToken, fakeCtx(), OPERATOR);
    expect(gated.status).toBe(503);
    const env: LifecycleEnv = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "r", CF_API_TOKEN_WRITE: "w", OPS_KV: fakeKV() } as LifecycleEnv;
    const malformed = await handleDnsToggle(postToggle({ zone: "example.dev", record: "rec1" }), env, fakeCtx(), OPERATOR);
    expect(malformed.status).toBe(400);
    stubFetch([ZONES_OK, { match: (u, r) => u.includes("/dns_records/") && r.method === "PATCH", body: { success: false, errors: [{ message: "dns edit denied" }] } }]);
    const failed = await handleDnsToggle(postToggle({ zone: "example.dev", record: "rec1", paused: true }), env, fakeCtx(), OPERATOR);
    expect(failed.status).toBe(502);
    const failedBody = (await failed.json()) as { error: string; hint: string };
    expect(failedBody.error).toBe("dns edit denied");
    expect(failedBody.hint).toContain("DNS > Edit");
  });
});
