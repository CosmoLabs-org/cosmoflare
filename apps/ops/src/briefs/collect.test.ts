// Zone-facts collector tests (P-15): global fetch stubbed per route, KV as an
// in-memory map (the rules_quota.test.ts pattern). Covers the happy path,
// the cached() short-circuit (no refetch on a second collect), probe-error
// degradation, the hard coverage failure, and the live alert-feed filter.
import { afterEach, describe, expect, it, vi } from "vitest";
import { ALERT_FEED_KEY, type AlertFeedEntry } from "../scheduled";
import { collectZoneFacts } from "./collect";

// Minimal env: only CF_ACCOUNT_ID / CF_API_TOKEN / OPS_KV are touched
// (untyped object, `as never` at call sites — the sibling-test pattern).
const ENV = {
  CF_ACCOUNT_ID: "acct",
  CF_API_TOKEN: "tok",
  OPS_KV: {
    store: new Map<string, string>(),
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

let fetchCalls: string[] = [];

interface Route {
  match: (url: string) => boolean;
  reply: (url: string) => Response;
}

function routes(...rs: Route[]): void {
  fetchCalls = [];
  vi.stubGlobal("fetch", async (input: RequestInfo | URL): Promise<Response> => {
    const url = String(input);
    fetchCalls.push(url);
    for (const r of rs) if (r.match(url)) return r.reply(url);
    throw new Error(`unexpected fetch: ${url}`);
  });
}

const json = (body: unknown, status = 200): Response => new Response(JSON.stringify(body), { status });

function graphql(groups: { sum: { requests: number; cachedRequests: number } }[]): Route {
  return {
    match: (u) => u.includes("/client/v4/graphql"),
    reply: () => json({ data: { viewer: { zones: [{ httpRequests1dGroups: groups }] } } }),
  };
}

function zoneRoute(id: string, name: string): Route {
  return { match: (u) => u.includes("/zones?account.id"), reply: () => json({ success: true, result: [{ id, name }] }) };
}

function rulesetsRoutes(cacheRules: number): Route[] {
  const rules = Array.from({ length: cacheRules }, (_, i) => ({ description: `rule ${i}`, expression: "true", action: "cache" }));
  return [
    { match: (u) => u.includes("/pagerules"), reply: () => json({ success: true, result: [] }) },
    { match: (u) => u.includes("http_request_cache_settings/entrypoint"), reply: () => json({ success: true, result: { rules } }) },
    { match: (u) => u.includes("/rulesets/phases/"), reply: () => json({ success: true, result: { rules: [] } }) },
  ];
}

const probeRoute = (headers: Record<string, string>): Route => ({
  match: (u) => u.startsWith("https://example.com/"),
  reply: () => new Response("<html><body>hi</body></html>", { status: 200, headers }),
});

function alert(id: string, title: string): AlertFeedEntry {
  return { ts: "2026-10-15T00:00:00.000Z", ruleId: id, condition: "zone-cache-miss-pct", title, value: 63.6, threshold: 50, unit: "%", message: "m", sent: true, context: "46 misses of 139 eligible requests" };
}

describe("collectZoneFacts", () => {
  it("collects coverage, probe headers, quota, cache-rule count, and zone-filtered firing alerts", async () => {
    ENV.OPS_KV.store.set(
      ALERT_FEED_KEY,
      JSON.stringify([
        alert("hot rule/zone-1", "cache miss % · example.com"),
        alert("by name/example.com", "uncached · example.com"), // zone-key may be the zone NAME
        alert("elsewhere/zone-2", "other zone"),
        alert("telemetry-gap/d1", "Telemetry gap (d1)"),
      ]),
    );
    routes(
      graphql([
        { sum: { requests: 300, cachedRequests: 20 } },
        { sum: { requests: 200, cachedRequests: 40 } },
      ]),
      zoneRoute("zone-1", "example.com"),
      ...rulesetsRoutes(3),
      probeRoute({ "content-type": "text/html; charset=utf-8", "cache-control": "public, s-maxage=3600", "cf-cache-status": "DYNAMIC" }),
    );

    const facts = await collectZoneFacts(ENV as never, "example.com");

    expect(facts.zone).toBe("example.com");
    expect(facts.zoneId).toBe("zone-1");
    expect(facts.window).toBe("7d");
    // coverage sums across the daily groups: 500 requests, 60 cached, 12%
    expect(facts.coverage).toEqual({ requests: 500, cached: 60, hitRatePct: 12 });
    expect(facts.probe).toEqual({ contentType: "text/html; charset=utf-8", cacheControl: "public, s-maxage=3600", cfCacheStatus: "DYNAMIC" });
    // cache-rules family: used=3, authorized=true -> count exposed
    expect(facts.cacheRuleCount).toBe(3);
    expect(facts.quota?.families).toHaveLength(5);
    expect(facts.quota?.families[0]).toEqual({ family: "page-rules", label: "Page Rules", used: 0, allowance: 3, authorized: true });
    // the per-rule inventory rows are stripped — facts carry counters only
    expect("rules" in (facts.quota?.families[0] ?? {})).toBe(false);
    // only the queried zone's alerts survive the ruleId segment filter
    expect(facts.firingAlerts.map((a) => a.ruleId)).toEqual(["hot rule/zone-1", "by name/example.com"]);
  });

  it("short-circuits through cached(): a second collect refetches nothing and equals the first", async () => {
    // Distinct zone: cached()'s L1 memo is module-scope and the happy-path
    // test above already seeded example.com's entries.
    routes(
      graphql([{ sum: { requests: 100, cachedRequests: 10 } }]),
      zoneRoute("zone-short", "short.example"),
      ...rulesetsRoutes(0),
      { match: (u) => u.startsWith("https://short.example/"), reply: () => new Response("<html></html>", { headers: { "content-type": "text/html", "cache-control": "public", "cf-cache-status": "HIT" } }) },
    );

    const first = await collectZoneFacts(ENV as never, "short.example");
    const remote = (calls: string[]) => calls.filter((u) => u.includes("/client/v4/graphql") || u.startsWith("https://example.com/")).length;
    const afterFirst = remote(fetchCalls);
    expect(afterFirst).toBeGreaterThan(0);

    const second = await collectZoneFacts(ENV as never, "short.example");
    expect(remote(fetchCalls)).toBe(afterFirst); // coverage + probe served from L1
    expect(second).toEqual(first); // quota is KV-cached 5 min; alerts unchanged
  });

  it("degrades a probe failure to probe: undefined without throwing", async () => {
    routes(
      graphql([{ sum: { requests: 100, cachedRequests: 10 } }]),
      zoneRoute("zone-dead", "dead.example"),
      ...rulesetsRoutes(0),
      { match: (u) => u.startsWith("https://dead.example/"), reply: () => { throw new Error("connection refused"); } },
    );

    const facts = await collectZoneFacts(ENV as never, "dead.example");
    expect(facts.probe).toBeUndefined();
    expect(facts.coverage).toEqual({ requests: 100, cached: 10, hitRatePct: 10 });
  });

  it("treats a coverage GraphQL failure as a hard error (rejects)", async () => {
    routes(
      { match: (u) => u.includes("/client/v4/graphql"), reply: () => json({ errors: [{ message: "dataset unavailable" }] }, 400) },
      zoneRoute("zone-gf", "graphfail.example"),
      ...rulesetsRoutes(0),
      { match: (u) => u.startsWith("https://graphfail.example/"), reply: () => new Response("<html></html>") },
    );

    await expect(collectZoneFacts(ENV as never, "graphfail.example")).rejects.toThrow();
  });

  it("degrades a corrupt alert-feed KV value to an empty list", async () => {
    ENV.OPS_KV.store.set(ALERT_FEED_KEY, "{corrupt json");
    routes(
      graphql([{ sum: { requests: 100, cachedRequests: 10 } }]),
      zoneRoute("zone-1", "example.com"),
      ...rulesetsRoutes(0),
      probeRoute({ "content-type": "text/html", "cache-control": "public", "cf-cache-status": "HIT" }),
    );

    const facts = await collectZoneFacts(ENV as never, "example.com");
    expect(facts.firingAlerts).toEqual([]);
  });

  it("throws when the zone is unknown (hard failure for the endpoint's 404)", async () => {
    routes(zoneRoute("zone-1", "example.com"), ...rulesetsRoutes(0));
    await expect(collectZoneFacts(ENV as never, "nope.example")).rejects.toThrow("not found");
  });
});
