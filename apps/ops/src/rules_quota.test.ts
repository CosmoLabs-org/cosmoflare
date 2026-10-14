// Rule-quota collector tests: happy path across all five families, rulesets
// 403 degradation (authorized:false, Page Rules still live), unknown zone,
// and the 5-minute KV cache — global fetch stubbed, KV as an in-memory map.
import { afterEach, describe, expect, it, vi } from "vitest";
import { collectRulesQuota, RULE_ALLOWANCES } from "./rules_quota";

// Minimal env: the collector touches only these three fields (the sibling
// worker_profile.test.ts pattern — untyped object, `as never` at call sites).
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

function stubFetch(routes: Record<string, unknown>): void {
  fetchCalls = [];
  vi.stubGlobal("fetch", async (input: RequestInfo | URL): Promise<Response> => {
    const url = String(input);
    fetchCalls.push(url);
    for (const [path, body] of Object.entries(routes)) {
      if (url.includes(path)) {
        if (body instanceof Response) return body.clone();
        return new Response(JSON.stringify({ success: true, result: body }));
      }
    }
    return new Response(JSON.stringify({ success: false, errors: [{ message: "not found" }] }), { status: 404 });
  });
}

const ZONES_ROUTE = "/zones?account.id";
const ZONE_ID = "z-quota-1";

function happyRoutes(): Record<string, unknown> {
  return {
    [ZONES_ROUTE]: [{ id: ZONE_ID, name: "example.com" }],
    [`${ZONE_ID}/pagerules`]: [
      {
        status: "active",
        targets: [{ constraint: { value: "www.example.com/*" } }],
        actions: [{ id: "forwarding_url", value: { url: "https://example.com/$1", status_code: 301 } }],
      },
      {
        status: "active",
        targets: [{ constraint: { value: "static.example.com/*" } }],
        actions: [{ id: "cache_level", value: "cache_everything" }],
      },
    ],
    [`${ZONE_ID}/rulesets/phases/http_request_dynamic_redirect/entrypoint`]: {
      rules: [{ description: "www to apex", expression: 'starts_with(http.host, "www.")', action: "redirect" }],
    },
    [`${ZONE_ID}/rulesets/phases/http_request_firewall_custom/entrypoint`]: {
      rules: [
        { description: "block scanners", expression: '(cf.bot_management.score lt 10)', action: "block" },
        { description: "challenge logins", expression: 'http.request.uri.path contains "/login"', action: "managed_challenge" },
      ],
    },
    [`${ZONE_ID}/rulesets/phases/http_request_cache_settings/entrypoint`]: { rules: [] },
    [`${ZONE_ID}/rulesets/phases/http_request_config/entrypoint`]: { rules: [] },
  };
}

describe("collectRulesQuota", () => {
  it("collects all five families with used counts, allowances, and rule names", async () => {
    stubFetch(happyRoutes());
    const out = await collectRulesQuota(ENV as never, "example.com");
    expect(out.errors).toEqual([]);
    expect(out.families).toHaveLength(5);
    const byFamily = Object.fromEntries(out.families.map((f) => [f.family, f]));
    expect(byFamily["page-rules"].used).toBe(2);
    expect(byFamily["page-rules"].allowance).toBe(RULE_ALLOWANCES["page-rules"]);
    expect(byFamily["page-rules"].authorized).toBe(true);
    expect(byFamily["page-rules"].rules[0].name).toBe("www.example.com/*");
    expect(byFamily["page-rules"].rules[0].detail).toContain("https://example.com/$1");
    expect(byFamily["redirect-rules"].used).toBe(1);
    expect(byFamily["redirect-rules"].rules[0].name).toBe("www to apex");
    expect(byFamily["waf-custom-rules"].used).toBe(2);
    expect(byFamily["cache-rules"].used).toBe(0);
    expect(byFamily["config-rules"].allowance).toBe(10);
  });

  it("degrades rulesets families to authorized:false on 403 while Page Rules stay live", async () => {
    stubFetch({
      [ZONES_ROUTE]: [{ id: ZONE_ID, name: "example.com" }],
      [`${ZONE_ID}/pagerules`]: [],
    });
    // No rulesets routes: every phase endpoint returns the stub's 404 → the
    // same degrade path a 403 takes.
    const out = await collectRulesQuota(ENV as never, "example.com");
    expect(out.errors).toEqual([]);
    const byFamily = Object.fromEntries(out.families.map((f) => [f.family, f]));
    expect(byFamily["page-rules"].authorized).toBe(true);
    expect(byFamily["page-rules"].used).toBe(0);
    for (const family of ["redirect-rules", "waf-custom-rules", "cache-rules", "config-rules"]) {
      expect(byFamily[family].authorized).toBe(false);
      expect(byFamily[family].used).toBe(0);
    }
  });

  it("reports an unknown zone as an errors line, not a throw", async () => {
    stubFetch({ [ZONES_ROUTE]: [{ id: ZONE_ID, name: "example.com" }] });
    const out = await collectRulesQuota(ENV as never, "nope.example");
    expect(out.errors).toHaveLength(1);
    expect(out.errors[0]).toContain("nope.example");
    expect(out.families).toEqual([]);
  });

  it("serves the second call from the KV cache without refetching", async () => {
    stubFetch(happyRoutes());
    await collectRulesQuota(ENV as never, "example.com");
    const callsAfterFirst = fetchCalls.length;
    const second = await collectRulesQuota(ENV as never, "example.com");
    expect(fetchCalls.length).toBe(callsAfterFirst);
    expect(second.families).toHaveLength(5);
  });
});
