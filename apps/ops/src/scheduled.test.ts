import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { parsePayload } from "../../../pager/src/payload";
import { handleSubscribe, SUBS_KEY } from "./subscriptions";
import { runScheduled, starterRules, testFire, RULES_KEY, STATE_KEY, ALERT_FEED_KEY } from "./scheduled";

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
  /** Every put call with its options (expirationTtl assertions, BUG-058). */
  putCalls: { key: string; value: string; options?: { expirationTtl?: number } }[];
}

// fakeKV builds an in-memory KVNamespace stand-in with put-call tracking.
function fakeKV(): FakeKV {
  const store = new Map<string, unknown>();
  const puts: string[] = [];
  const putCalls: { key: string; value: string; options?: { expirationTtl?: number } }[] = [];
  return {
    puts,
    putCalls,
    get: async (key: string, type?: string) => (type === "json" ? (store.get(key) ?? null) : (store.get(key) ?? null)),
    put: async (key: string, value: string, options?: { expirationTtl?: number }) => {
      puts.push(key);
      putCalls.push({ key, value, options });
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
function routeFetch(opts: { zoneGqlFail?: boolean; d1GqlFail?: boolean; kvGqlFail?: boolean; pushStatus?: number } = {}): { calls: string[] } {
  const calls: string[] = [];
  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    calls.push(url);
    if (!url.startsWith("https://api.cloudflare.com")) {
      return new Response(null, { status: opts.pushStatus ?? 201 }); // push endpoint
    }
    if (url.includes("/graphql")) {
      const query = String((init?.body as string) ?? "");
      const fail = query.includes("d1Analytics") ? opts.d1GqlFail : query.includes("kvOperations") ? opts.kvGqlFail : opts.zoneGqlFail;
      if (fail) return new Response(JSON.stringify({ errors: [{ message: "dataset down" }] }), { status: 200 });
      if (query.includes("kvOperations")) {
        const body = {
          data: {
            viewer: {
              accounts: [
                {
                  g: [
                    { sum: { requests: 150_000 }, dimensions: { namespaceId: "ns-1", actionType: "write" } },
                    { sum: { requests: 900_000 }, dimensions: { namespaceId: "ns-1", actionType: "read" } },
                  ],
                },
              ],
            },
          },
        };
        return new Response(JSON.stringify(body), { status: 200 });
      }
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
    if (url.includes("/storage/kv/namespaces")) {
      return new Response(JSON.stringify({ success: true, result: [{ id: "ns-1", title: "SESSIONS" }] }), { status: 200 });
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
  it("makes zero fetch calls when no device is paired, even with enabled rules (FEAT-052)", async () => {
    const kv = fakeKV(); // rules fall back to starters (all enabled); no subs seeded
    const { calls } = routeFetch();
    const out = await runScheduled(envP(kv), NOW);
    expect(calls).toHaveLength(0); // no telemetry, no pushes
    expect(out).toEqual({ fired: 0, sent: 0, pruned: 0, gaps: [], issues: [] });
    expect(kv.puts).not.toContain(STATE_KEY); // no state churn either
  });
  it("kv-writes rule: fetches only kv telemetry, counts writes only, pages under the condition headline (FB-29)", async () => {
    const kv = fakeKV();
    await kv.put(RULES_KEY, JSON.stringify([{ name: "kvday", condition: "kv-writes", threshold: 100_000, enabled: true }]));
    await seedSubs(kv);
    const { calls } = routeFetch();
    const out = await runScheduled(envP(kv), NOW);
    expect(out.fired).toBe(1);
    expect(out.sent).toBe(1);
    // namespace list REST + kv GraphQL = 2 upstream calls; zone/d1 untouched
    expect(calls.filter((c) => c.includes("api.cloudflare.com")).length).toBe(2);
    expect(calls.some((c) => c.includes("/storage/kv/namespaces"))).toBe(true);
    expect(kv.puts).toContain("cron:kv-list");
    const payload = parsePayload(JSON.parse(envelopes[0].data));
    expect(payload).not.toBeNull();
    // headline names the condition and the namespace, not the raw rule slug
    expect(payload!.title).toBe("KV writes · SESSIONS");
    // human body with the numbers: 150k writes counted, 900k reads excluded
    expect(payload!.detail).toBe("150000 writes in 24h — threshold 100000");
  });
  it("a kv telemetry gap pages when the kv GraphQL fails and a kv rule is enabled", async () => {
    const kv = fakeKV();
    await kv.put(RULES_KEY, JSON.stringify([{ name: "kvday", condition: "kv-writes", threshold: 100_000, enabled: true }]));
    await seedSubs(kv);
    routeFetch({ kvGqlFail: true });
    const out = await runScheduled(envP(kv), NOW);
    expect(out.gaps.some((g) => g.startsWith("kv:"))).toBe(true);
    expect(out.fired).toBe(1); // the telemetry-gap fire
    expect(out.sent).toBe(1);
  });
  it("starter rules, two fires: ONE digest push listing both, validating against the pager schema", async () => {
    const kv = fakeKV();
    await seedSubs(kv);
    const { calls } = routeFetch();
    const out = await runScheduled(envP(kv), NOW);
    // starters: uncached 10000 (12k dynamic fires), miss-pct 50 (0% missed of
    // 500 eligible → no fire), d1 1e9 (2B fires)
    expect(out.fired).toBe(2);
    expect(out.sent).toBe(1); // ONE push per run reaches the one device
    // zone list REST + 1 zone GraphQL batch + d1 list REST + 1 d1 GraphQL + 1 push
    expect(calls.filter((c) => c.includes("api.cloudflare.com")).length).toBe(4);
    // one envelope: the digest
    expect(envelopes).toHaveLength(1);
    const payload = parsePayload(JSON.parse(envelopes[0].data));
    expect(payload).not.toBeNull();
    expect(payload!.severity).toBe("info");
    expect(payload!.service).toBe("cloudflare");
    expect(payload!.title).toBe("2 alerts fired");
    // each fire is one digest line: condition headline · scope — numbers
    expect(payload!.detail).toContain("Uncached requests · hot.example — 12000 requests in 24h — threshold 10000");
    expect(payload!.detail).toContain("D1 rows read · big-db — 2.0B rows in 24h — threshold 1.0B");
    // fire-state persisted exactly once
    expect(kv.puts.filter((k) => k === STATE_KEY).length).toBe(1);
    // list caches written under the cron-prefixed keys (BUG-058: no collision
    // with cache.ts's API-side KV keys)
    expect(kv.puts).toContain("cron:zones");
    expect(kv.puts).toContain("cron:d1-list");
    expect(kv.puts).not.toContain("zones");
    expect(kv.puts).not.toContain("d1-list");
  });
  it("writes the cron list caches with expirationTtl 86400 and never on cache.ts's API keys (BUG-058)", async () => {
    const kv = fakeKV();
    await seedSubs(kv);
    routeFetch();
    await runScheduled(envP(kv), NOW);
    for (const key of ["cron:zones", "cron:d1-list"]) {
      const call = kv.putCalls.find((c) => c.key === key);
      expect(call).toBeDefined();
      expect(call!.options?.expirationTtl).toBe(86400);
    }
    // cache.ts's API-side keys (index.ts LIST_TTL: "zones-list", "d1-list")
    // stay untouched by the cron path — a {at, value} cron write over those
    // keys parsed as storedAt=undefined → NaN age → fresh forever.
    for (const call of kv.putCalls) {
      expect(["zones-list", "d1-list", "summary", "billing", "zones"]).not.toContain(call.key);
    }
  });
  it("caches the zone/D1 name lists in KV for 1h (no REST list calls on the second run)", async () => {
    const kv = fakeKV();
    await seedSubs(kv); // telemetry (and its list calls) only run with a paired device
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
    await seedSubs(kv); // no paired device now short-circuits before telemetry
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
  it("a ≥2× jump inside the cooldown still pages, at warning severity (escalation stays immediate)", async () => {
    const kv = fakeKV();
    await kv.put(RULES_KEY, JSON.stringify([{ name: "kvday", condition: "kv-writes", threshold: 100_000, enabled: true }]));
    await seedSubs(kv);
    routeFetch();
    await runScheduled(envP(kv), NOW); // pages at 150k
    envelopes.length = 0;
    // same 24h window now reports ≥2× the last paged value
    const inner = globalThis.fetch;
    globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes("/graphql") && String(init?.body ?? "").includes("kvOperations")) {
        const body = { data: { viewer: { accounts: [{ g: [{ sum: { requests: 400_000 }, dimensions: { namespaceId: "ns-1", actionType: "write" } }] }] } } };
        return new Response(JSON.stringify(body), { status: 200 });
      }
      return inner(input as never, init);
    }) as typeof fetch;
    const out = await runScheduled(envP(kv), new Date(NOW.getTime() + 30 * 60 * 1000)); // 30min: inside the 6h cooldown
    expect(out.fired).toBe(1); // the escalation fire
    expect(out.sent).toBe(1);
    const payload = parsePayload(JSON.parse(envelopes[0].data));
    expect(payload!.severity).toBe("warning");
    expect(payload!.title).toBe("KV writes · SESSIONS");
    expect(payload!.detail).toContain("400000 writes in 24h");
  });
  it("pages a telemetry gap once per 6h cooldown when the zone dataset fails, then stays quiet", async () => {
    const kv = fakeKV();
    await seedSubs(kv);
    routeFetch({ zoneGqlFail: true });
    const first = await runScheduled(envP(kv), NOW);
    // the d1 rule still fires on real data (2B rows) alongside the zone gap page
    expect(first.fired).toBe(2);
    expect(first.gaps[0]).toContain("zone: dataset down");
    expect(first.sent).toBe(1); // one digest push
    const digest = JSON.parse(envelopes[0].data);
    expect(digest.title).toBe("2 alerts fired");
    expect(digest.detail).toContain("Telemetry gap (zone)");
    // within the cooldown: quiet
    envelopes.length = 0;
    const second = await runScheduled(envP(kv), NOW);
    expect(second.fired).toBe(0);
    expect(second.sent).toBe(0);
    expect(envelopes).toHaveLength(0);
    // the old 1h cooldown re-paged standing breaches hourly (operator
    // 2026-10-12): +61min must ALSO stay quiet under the 6h cooldown
    const at61m = await runScheduled(envP(kv), new Date(NOW.getTime() + 61 * 60 * 1000));
    expect(at61m.fired).toBe(0);
    // after the 6h cooldown: the gap re-pages and the d1 rule re-fires
    const later = new Date(NOW.getTime() + 6 * 60 * 60 * 1000 + 60 * 1000);
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
    // rows still evaluated under the database ID name — the digest line names db-1
    const digest = JSON.parse(envelopes[0].data);
    expect(digest.detail).toContain("D1 rows read · db-1");
    expect(calls.length).toBeGreaterThan(0);
  });
  it("no paired device: zero upstream calls (the pre-pairing cron state), no throw", async () => {
    const kv = fakeKV();
    const { calls } = routeFetch();
    const out = await runScheduled(envP(kv), NOW);
    expect(calls).toHaveLength(0);
    expect(out).toEqual({ fired: 0, sent: 0, pruned: 0, gaps: [], issues: [] });
  });
  it("fires with subs but VAPID unset: the delivery problem lands in issues and fire-state still persists", async () => {
    const kv = fakeKV();
    await seedSubs(kv);
    routeFetch();
    // wrangler delivers unset vars as undefined at runtime; mirror that here.
    const env = { ...envP(kv), VAPID_PUBLIC_KEY: undefined, VAPID_PRIVATE_KEY: undefined } as unknown as OpsEnvLike;
    const out = await runScheduled(env, NOW);
    expect(out.fired).toBe(2); // rules still evaluate and fire
    expect(out.sent).toBe(0);
    expect(out.issues).toContain("VAPID_PRIVATE_KEY is not set");
    expect(envelopes).toHaveLength(0); // nothing was deliverable
    // the crash used to skip this write, re-evaluating the same fires every run
    expect(kv.puts.filter((k) => k === STATE_KEY).length).toBe(1);
  });
  it("a 410 push prunes the dead endpoint from KV 'subs' in one write, keeping the live one", async () => {
    const kv = fakeKV();
    await kv.put(SUBS_KEY, JSON.stringify([
      { endpoint: "https://push.example/1", p256dh: "pk", auth: "ak" },
      { endpoint: "https://push.example/2", p256dh: "pk", auth: "ak" },
    ]));
    routeFetch();
    const inner = globalThis.fetch;
    globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input) === "https://push.example/2") return new Response(null, { status: 410 });
      return inner(input as never, init);
    }) as typeof fetch;
    const subsWritesBefore = kv.puts.filter((k) => k === SUBS_KEY).length;
    const out = await runScheduled(envP(kv), NOW);
    expect(out.pruned).toBe(1); // ONE push per run hits the dead endpoint once
    expect(out.sent).toBe(1); // the live device got the digest
    // one read-modify-write for the whole run removed only the dead endpoint
    const kept = (await kv.get(SUBS_KEY, "json")) as { endpoint: string }[];
    expect(kept.map((s) => s.endpoint)).toEqual(["https://push.example/1"]);
    expect(kv.puts.filter((k) => k === SUBS_KEY).length).toBe(subsWritesBefore + 1);
  });
  it("a fired rule appends a delivered alert-feed entry (Notifications center)", async () => {
    const kv = fakeKV();
    await kv.put(RULES_KEY, JSON.stringify([{ name: "kvday", condition: "kv-writes", threshold: 100_000, enabled: true }]));
    await seedSubs(kv);
    routeFetch();
    const out = await runScheduled(envP(kv), NOW);
    expect(out.fired).toBe(1);
    expect(out.sent).toBe(1);
    const feed = (await kv.get(ALERT_FEED_KEY, "json")) as Array<{
      ts: string; ruleId: string; condition: string; title: string;
      value: number; threshold: number; unit: string; message: string; sent: boolean;
    }>;
    expect(feed).toHaveLength(1);
    const entry = feed[0];
    expect(entry.ts).toBe(NOW.toISOString());
    expect(entry.ruleId.startsWith("kvday/")).toBe(true);
    expect(entry.condition).toBe("kv-writes");
    expect(entry.title).toBe("kvday");
    expect(entry.value).toBe(150_000);
    expect(entry.threshold).toBe(100_000);
    expect(entry.unit).toBe("writes");
    expect(entry.message).toContain("SESSIONS");
    expect(entry.sent).toBe(true);
  });
  it("caps the alert-feed at 100 entries, newest first", async () => {
    const kv = fakeKV();
    await kv.put(RULES_KEY, JSON.stringify([{ name: "kvday", condition: "kv-writes", threshold: 100_000, enabled: true }]));
    await seedSubs(kv);
    routeFetch();
    const stale = Array.from({ length: 100 }, (_, i) => ({
      ts: new Date(NOW.getTime() - (i + 1) * 60_000).toISOString(),
      ruleId: `old/${i}`, condition: "kv-writes", title: "old", value: 1, threshold: 1,
      unit: "writes", message: "stale", sent: true,
    }));
    await kv.put(ALERT_FEED_KEY, JSON.stringify(stale));
    const out = await runScheduled(envP(kv), NOW);
    expect(out.fired).toBe(1);
    const feed = (await kv.get(ALERT_FEED_KEY, "json")) as { ruleId: string; ts: string }[];
    expect(feed).toHaveLength(100); // 101 total → capped at MAX_FEED
    expect(feed[0].ruleId.startsWith("kvday/")).toBe(true); // newest first
    expect(feed[0].ts).toBe(NOW.toISOString());
    expect(feed[1].ruleId).toBe("old/0"); // the previous newest follows
  });
  it("records sent:false in the alert-feed when push delivery fails", async () => {
    const kv = fakeKV();
    await kv.put(RULES_KEY, JSON.stringify([{ name: "kvday", condition: "kv-writes", threshold: 100_000, enabled: true }]));
    await seedSubs(kv);
    routeFetch({ pushStatus: 500 });
    const out = await runScheduled(envP(kv), NOW);
    expect(out.fired).toBe(1);
    expect(out.sent).toBe(0);
    expect(out.issues.length).toBeGreaterThan(0); // the delivery problem is surfaced
    // the fire still lands in the history, honestly marked not delivered
    const feed = (await kv.get(ALERT_FEED_KEY, "json")) as { ruleId: string; sent: boolean }[];
    expect(feed).toHaveLength(1);
    expect(feed[0].ruleId.startsWith("kvday/")).toBe(true);
    expect(feed[0].sent).toBe(false);
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
    expect(payload!.title).toBe("Test alert · Cosmoflare");
    expect(payload!.detail).toContain("Pairing works");
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
