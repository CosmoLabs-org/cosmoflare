import { describe, expect, it } from "vitest";
import { handleSubscribe, MAX_SUBS, validateSubscription, type StoredSubscription } from "./subscriptions";
import type { OpsEnv } from "./scheduled";

// fakeKV builds an in-memory KVNamespace stand-in (get/put with "json").
function fakeKV(): KVNamespace {
  const store = new Map<string, unknown>();
  return {
    get: async (key: string, type?: string) => (type === "json" ? (store.get(key) ?? null) : store.get(key) ?? null),
    put: async (key: string, value: string) => {
      store.set(key, JSON.parse(value));
    },
  } as unknown as KVNamespace;
}

// envP builds a minimal OpsEnv for handler tests.
function envP(kv: KVNamespace = fakeKV()): OpsEnv {
  return {
    CF_ACCOUNT_ID: "acct",
    CF_API_TOKEN: "tok",
    VAPID_PUBLIC_KEY: "pub",
    VAPID_PRIVATE_KEY: "priv",
    VAPID_SUBJECT: "mailto:ops@example.com",
    OPS_KV: kv,
    ACCESS_TEAM_DOMAIN: "team.cloudflareaccess.com",
    ACCESS_AUD: "aud",
    ASSETS: {} as Fetcher,
  };
}

// subP builds a valid browser PushSubscription body.
function subP(endpoint: string): unknown {
  return { endpoint, keys: { p256dh: "p256dh-key", auth: "auth-key" } };
}

describe("validateSubscription", () => {
  it("accepts the browser shape and stores flat", () => {
    const out = validateSubscription(subP("https://push.example/abc"));
    expect("sub" in out && out.sub).toEqual({ endpoint: "https://push.example/abc", p256dh: "p256dh-key", auth: "auth-key" });
  });
  it("rejects http endpoints and missing keys", () => {
    expect("error" in validateSubscription({ endpoint: "http://push.example/x", keys: { p256dh: "a", auth: "b" } })).toBe(true);
    expect("error" in validateSubscription({ endpoint: "https://push.example/x", keys: { p256dh: "a" } })).toBe(true);
    expect("error" in validateSubscription({})).toBe(true);
    expect("error" in validateSubscription("nope")).toBe(true);
  });
});

describe("handleSubscribe", () => {
  it("POST validates, stores, dedupes by endpoint (replace) and caps at 10", async () => {
    const env = envP();
    expect((await handleSubscribe(new Request("https://ops/api/subscribe", { method: "POST", body: JSON.stringify(subP("https://p/1")) }), env)).status).toBe(201);
    // duplicate endpoint: replaces, count stays 1
    const dup = new Request("https://ops/api/subscribe", { method: "POST", body: JSON.stringify({ endpoint: "https://p/1", keys: { p256dh: "new", auth: "keys" } }) });
    const dupRes = await handleSubscribe(dup, env);
    expect(dupRes.status).toBe(201);
    expect(await (await handleSubscribe(new Request("https://ops/api/subscribe"), env)).json()).toEqual({ count: 1 });
    // fill to the cap
    for (let i = 2; i <= MAX_SUBS; i++) {
      await handleSubscribe(new Request("https://ops/api/subscribe", { method: "POST", body: JSON.stringify(subP(`https://p/${i}`)) }), env);
    }
    expect(await (await handleSubscribe(new Request("https://ops/api/subscribe"), env)).json()).toEqual({ count: MAX_SUBS });
    // one more distinct endpoint → 409
    const over = await handleSubscribe(new Request("https://ops/api/subscribe", { method: "POST", body: JSON.stringify(subP("https://p/11")) }), env);
    expect(over.status).toBe(409);
  });
  it("DELETE removes by endpoint", async () => {
    const env = envP();
    await handleSubscribe(new Request("https://ops/api/subscribe", { method: "POST", body: JSON.stringify(subP("https://p/1")) }), env);
    const res = await handleSubscribe(new Request("https://ops/api/subscribe", { method: "DELETE", body: JSON.stringify({ endpoint: "https://p/1" }) }), env);
    expect(await res.json()).toEqual({ ok: true, count: 0 });
  });
  it("DELETE with a missing endpoint is a 400", async () => {
    const env = envP();
    const res = await handleSubscribe(new Request("https://ops/api/subscribe", { method: "DELETE", body: JSON.stringify({}) }), env);
    expect(res.status).toBe(400);
  });
  it("non-JSON POST body is a 400", async () => {
    const env = envP();
    const res = await handleSubscribe(new Request("https://ops/api/subscribe", { method: "POST", body: "not json" }), env);
    expect(res.status).toBe(400);
  });
  it("GET returns the count and other methods are 405", async () => {
    const env = envP();
    expect(await (await handleSubscribe(new Request("https://ops/api/subscribe"), env)).json()).toEqual({ count: 0 });
    const res = await handleSubscribe(new Request("https://ops/api/subscribe", { method: "PUT" }), env);
    expect(res.status).toBe(405);
  });
  it("stored subs survive a fresh env over the same KV", async () => {
    const kv = fakeKV();
    const env = envP(kv);
    await handleSubscribe(new Request("https://ops/api/subscribe", { method: "POST", body: JSON.stringify(subP("https://p/1")) }), env);
    const list = (await kv.get("subs", "json")) as StoredSubscription[] | null;
    expect(list ?? []).toHaveLength(1);
    expect(list![0]).toEqual({ endpoint: "https://p/1", p256dh: "p256dh-key", auth: "auth-key" });
  });
});
