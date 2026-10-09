import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { parsePayload } from "../../../pager/src/payload";
import { handleSubscribe, SUBS_KEY } from "./subscriptions";
import { runScheduled, starterRules, testFire, RULES_KEY, STATE_KEY } from "./scheduled";

// Mock the push library at the module boundary and capture every envelope.
const { envelopes } = vi.hoisted(() => ({ envelopes: [] as { url: string; data: string }[] }));
vi.mock("@block65/webcrypto-web-push", () => ({
  buildPushPayload: async (message: { data: string }, sub: { endpoint: string }) => {
    envelopes.push({ url: sub.endpoint, data: message.data });
    return {
      method: "POST",
      headers: { authorization: "vapid t=x", ttl: "3600", urgency: "normal" },
      body: new TextEncoder().encode(message.data),
    };
  },
}));

// FakeKV exposes put-call tracking for write-only-when-changed assertions.
interface FakeKV extends KVNamespace {
  puts: string[];
}

// fakeKV builds an in-memory KVNamespace stand-in with put-call tracking.
function fakeKV(): FakeKV {
  const store = new Map<string, unknown>();
  const puts: string[] = [];
  return {
    puts,
    get: async (key: string, type?: string) => (type === "json" ? (store.get(key) ?? null) : (store.get(key) ?? null)),
    put: async (key: string, value: string) => {
      puts.push(key);
      store.set(key, JSON.parse(value));
    },
  } as unknown as FakeKV;
}

// envP builds a minimal OpsEnv over a fake KV.
function envP(kv: KVNamespace): OpsEnvLike {
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

// OpsEnvLike mirrors OpsEnv without importing the type (it would drag the
// workers-types KVNamespace into an unused-position error; structural pass).
interface OpsEnvLike {
  CF_ACCOUNT_ID: string;
  CF_API_TOKEN: string;
  VAPID_PUBLIC_KEY: string;
  VAPID_PRIVATE_KEY: string;
  VAPID_SUBJECT: string;
  OPS_KV: KVNamespace;
  ACCESS_TEAM_DOMAIN: string;
  ACCESS_AUD: string;
  ASSETS: Fetcher;
}

// routeFetch builds a fetch stub that answers the Cloudflare REST lists, the
// two GraphQL datasets, and push endpoints (201).
function routeFetch(opts: { zoneGqlFail?: boolean; d1GqlFail?: boolean; pushStatus?: number } = {}): { calls: string[] } {
  const calls: string[] = [];
  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    calls.push(url);
    if (!url.startsWith("https://api.cloudflare.com")) {
      return new Response(null, { status: opts.pushStatus ?? 201 }); // push endpoint
    }
    if (url.includes("/graphql")) {
      const query = String((init?.body as string) ?? "");
      const fail = query.includes("d1Analytics") ? opts.d1GqlFail : opts.zoneGqlFail;
      if (fail) return new Response(JSON.stringify({ errors: [{ message: "dataset down" }] }), { status: 200 });
      if (query.includes("d1Analytics")) {
        const body = {
          data: {
            viewer: {
              accounts: [
                {
                  g: [
                    {
                      sum: { rowsRead: 2_000_000_000, readQueries: 100 },
                      dimensions: { databaseId: "db-1" },
                    },
                  ],
                },
              ],
            },
          },
        };
        return new Response(JSON.stringify(body), { status: 200 });
      }
      const body = {
        data: {
          viewer: {
            zones: [
              {
                zoneTag: "z-1",
                httpRequestsAdaptiveGroups: [
                  { count: 12000, dimensions: { cacheStatus: "dynamic" } },
                  { count: 500, dimensions: { cacheStatus: "hit" } },
                ],
              },
            ],
          },
        },
      };
      return new Response(JSON.stringify(body), { status: 200 });
    }
    if (url.includes("/zones?")) {
      return new Response(JSON.stringify({ success: true, result: [{ id: "z-1", name: "hot.example", status: "active" }] }), { status: 200 });
    }
    if (url.includes("/d1/database")) {
      return new Response(JSON.stringify({ success: true, result: [{ uuid: "db-1", name: "big-db" }] }), { status: 200 });
    }
    return new Response(JSON.stringify({ success: false, errors: [{ message: "unexpected " + url }] }), { status: 500 });
  }) as typeof fetch;
  return { calls };
}

const NOW = new Date("2026-10-09T12:00:00Z");
const SUB = { endpoint: "https://push.example/1", keys: { p256dh: "pk", auth: "ak" } };

// seedSubs registers one device directly in KV (endpoint must be https).
async function seedSubs(kv: FakeKV): Promise<void> {
  await kv.put(SUBS_KEY, JSON.stringify([{ endpoint: "https://push.example/1", p256dh: "pk", auth: "ak" }]));
}

let realFetch: typeof fetch;

beforeEach(() => {
  envelopes.length = 0;
  realFetch = globalThis.fetch;
});

afterEach(() => {
  globalThis.fetch = realFetch;
});

describe("runScheduled", () => {
  it("makes zero upstream calls when no rule is enabled", async () => {
    const kv = fakeKV();
    await kv.put(RULES_KEY, JSON.stringify([{ name: "off", condition: "zone-uncached-requests", threshold: 1, enabled: false }]));
    const { calls } = routeFetch();
    const out = await runScheduled(envP(kv), NOW);
    expect(calls).toHaveLength(0);
    expect(out).toEqual({ fired: 0, sent: 0, pruned: 0, gaps: [], issues: [] });
  });
  it("uses starter rules when KV 'rules' is absent; threshold fire names the zone and validates against the pager schema", async () => {
    const kv = fakeKV();
    await seedSubs(kv);
    const { calls } = routeFetch();
    const out = await runScheduled(envP(kv), NOW);
    // starters: uncached 10000 (12k dynamic fires), miss-pct 50 (0% missed of
    // 500 eligible → no fire), d1 1e9 (2B fires)
    expect(out.fired).toBe(2);
    expect(out.sent).toBe(2); // one subscription, two fires
    // zone list REST + 1 zone GraphQL batch + d1 list REST + 1 d1 GraphQL + 2 pushes
    expect(calls.filter((c) => c.includes("api.cloudflare.com")).length).toBe(4);
    // all envelopes pass the pager wire validation
    for (const e of envelopes) {
      const payload = parsePayload(JSON.parse(e.data));
      expect(payload).not.toBeNull();
      expect(payload!.severity).toBe("info");
      expect(payload!.service).toBe("cloudflare");
    }
    // the uncached fire names the zone in the detail
    const uncached = envelopes.map((e) => JSON.parse(e.data)).find((p) => p.title === "uncached requests");
    expect(uncached.detail).toContain("hot.example");
    expect(uncached.id).toBe("uncached requests/hot.example");
    // fire-state persisted exactly once
    expect(kv.puts.filter((k) => k === STATE_KEY).length).toBe(1);
    // list caches written
    expect(kv.puts).toContain("zones");
    expect(kv.puts).toContain("d1-list");
  });
  it("caches the zone/D1 name lists in KV for 1h (no REST list calls on the second run)", async () => {
    const kv = fakeKV();
    const { calls } = routeFetch();
    await runScheduled(envP(kv), NOW);
    const listCallsFirstRun = calls.filter((c) => c.includes("/zones?") || c.includes("/d1/database")).length;
    expect(listCallsFirstRun).toBe(2);
    envelopes.length = 0;
    calls.length = 0;
    await runScheduled(envP(kv), NOW);
    expect(calls.filter((c) => c.includes("/zones?") || c.includes("/d1/database")).length).toBe(0);
    // GraphQL still runs every cycle
    expect(calls.filter((c) => c.includes("/graphql")).length).toBe(2);
  });
  it("cooldown suppresses a second run within 1h (no pushes, no state rewrite)", async () => {
    const kv = fakeKV();
    routeFetch();
    await runScheduled(envP(kv), NOW);
    const putsAfterFirst = kv.puts.filter((k) => k === STATE_KEY).length;
    envelopes.length = 0;
    const out = await runScheduled(envP(kv), NOW);
    expect(out.fired).toBe(0);
    expect(out.sent).toBe(0);
    expect(envelopes).toHaveLength(0);
    expect(kv.puts.filter((k) => k === STATE_KEY).length).toBe(putsAfterFirst);
  });
  it("pages a telemetry gap once per hour when the zone dataset fails, then stays quiet", async () => {
    const kv = fakeKV();
    await seedSubs(kv);
    routeFetch({ zoneGqlFail: true });
    const first = await runScheduled(envP(kv), NOW);
    // the d1 rule still fires on real data (2B rows) alongside the zone gap page
    expect(first.fired).toBe(2);
    expect(first.gaps[0]).toContain("zone: dataset down");
    expect(first.sent).toBe(2);
    const gapPayload = envelopes.map((e) => JSON.parse(e.data)).find((p) => p.title === "telemetry-gap");
    expect(gapPayload.id).toBe("telemetry-gap/zone");
    // within the hour: quiet
    envelopes.length = 0;
    const second = await runScheduled(envP(kv), NOW);
    expect(second.fired).toBe(0);
    expect(second.sent).toBe(0);
    expect(envelopes).toHaveLength(0);
    // after the hour: the gap re-pages and the d1 rule re-fires (cooldowns expired)
    const later = new Date(NOW.getTime() + 61 * 60 * 1000);
    const third = await runScheduled(envP(kv), later);
    expect(third.fired).toBe(2);
  });
  it("a failed D1 name list is an issue, not a gap (rows keep IDs as names)", async () => {
    const kv = fakeKV();
    await seedSubs(kv);
    // Starters include the d1 rule; break only the d1 database list endpoint.
    const { calls } = routeFetch();
    const inner = globalThis.fetch;
    globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes("/d1/database")) return new Response(JSON.stringify({ success: false, errors: [{ message: "d1 list down" }] }), { status: 500 });
      return inner(input as never, init);
    }) as typeof fetch;
    const out = await runScheduled(envP(kv), NOW);
    expect(out.gaps).toEqual([]); // d1 GraphQL itself succeeded
    expect(out.issues.some((i) => i.includes("d1 name list failed"))).toBe(true);
    // rows still evaluated under the database ID name
    const d1Fire = envelopes.map((e) => JSON.parse(e.data)).find((p) => p.title === "d1 rows read");
    expect(d1Fire.detail).toContain("db-1");
    expect(calls.length).toBeGreaterThan(0);
  });
});

describe("testFire", () => {
  it("sends the canned payload matching the CLI's --test-fire, valid against the pager schema", async () => {
    const kv = fakeKV();
    routeFetch();
    await handleSubscribe(new Request("https://ops/api/subscribe", { method: "POST", body: JSON.stringify(SUB) }), envP(kv));
    const out = await testFire(envP(kv));
    expect(out.sent).toBe(1);
    const payload = parsePayload(JSON.parse(envelopes[0].data));
    expect(payload!.id).toBe("test-fire");
    expect(payload!.title).toBe("Cosmoflare pager test");
    expect(payload!.detail).toContain("alerts watch --test-fire");
  });
  it("does nothing with zero subscriptions", async () => {
    const kv = fakeKV();
    const { calls } = routeFetch();
    const out = await testFire(envP(kv));
    expect(out).toEqual({ sent: 0, pruned: 0, issues: [] });
    expect(calls).toHaveLength(0);
  });
});

// starterRules sanity: the O10 trio with the documented thresholds.
describe("starterRules", () => {
  it("seeds uncached 10000, miss-pct 50, d1 rows 1e9, all enabled", () => {
    expect(starterRules()).toEqual([
      { name: "uncached requests", condition: "zone-uncached-requests", threshold: 10000, enabled: true },
      { name: "cache miss %", condition: "zone-cache-miss-pct", threshold: 50, enabled: true },
      { name: "d1 rows read", condition: "d1-rows-read", threshold: 1e9, enabled: true },
    ]);
  });
});
