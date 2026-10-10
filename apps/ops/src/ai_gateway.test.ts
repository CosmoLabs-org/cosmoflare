import { afterEach, describe, expect, it, vi } from "vitest";
import { collectAIGateways, PROBE_KEY } from "./ai_gateway";


// BR-02 (docs/brainstorming/2026-10-10-ai-gateway-monitoring.md): list
// gateways, aggregate aiGatewayRequestsAdaptiveGroups by gateway × model ×
// provider over 24h (+ MTD totals), probe the undocumented token/cost fields
// once per day through KV, degrade to counts-plus-note when absent.

const realFetch = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = realFetch;
  vi.restoreAllMocks();
});

function fakeKV(): { get(k: string): Promise<string | null>; put(k: string, v: string): Promise<void>; puts: string[] } {
  const map = new Map<string, string>();
  return {
    puts: [],
    async get(key: string) {
      return map.get(key) ?? null;
    },
    async put(key: string, value: string) {
      map.set(key, value);
      (this as { puts: string[] }).puts.push(key);
    },
  };
}

interface Call {
  url: string;
  body?: string;
}

function stubAI(opts: {
  gateways?: { id: string; name: string }[];
  graphql?: (query: string) => unknown;
} = {}): { calls: Call[] } {
  const calls: Call[] = [];
  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    calls.push({ url, body: init?.body ? String(init.body) : undefined });
    if (url.includes("/ai-gateway/gateways")) {
      return new Response(JSON.stringify({ success: true, result: opts.gateways ?? [{ id: "gw-1", name: "mycarguide" }] }), { status: 200 });
    }
    if (url.includes("/graphql")) {
      const q = String(init?.body ?? "");
      const data = opts.graphql ? opts.graphql(q) : { viewer: { accounts: [{ g: [] }] } };
      return new Response(JSON.stringify({ data }), { status: 200 });
    }
    return new Response(JSON.stringify({ success: false }), { status: 500 });
  }) as typeof fetch;
  return { calls };
}

const PROBE_QUERY_MARKERS = ["tokensIn", "cost"];

describe("collectAIGateways (BR-02)", () => {
  it("aggregates per gateway × model × provider with token/cost fields when the probe succeeds", async () => {
    const kv = fakeKV();
    stubAI({
      graphql: () => ({
        viewer: {
          accounts: [
            {
              g: [
                { count: 120, sum: { tokensIn: 1000, tokensOut: 200, cost: 0.0042 }, dimensions: { gateway: "gw-1", model: "@cf/meta/llama-3.1-8b-instruct", provider: "workers-ai" } },
                { count: 30, sum: { tokensIn: 400, tokensOut: 90, cost: 0.0011 }, dimensions: { gateway: "gw-1", model: "deepseek-chat", provider: "deepseek" } },
              ],
            },
          ],
        },
      }),
    });
    const out = await collectAIGateways("acct", "tok", new Date("2026-10-11T12:00:00Z"), kv);
    expect(out.gateways).toHaveLength(1);
    const gw = out.gateways[0];
    expect(gw.id).toBe("gw-1");
    expect(gw.name).toBe("mycarguide");
    expect(gw.requests24h).toBe(150);
    expect(gw.models).toHaveLength(2);
    const llama = gw.models.find((m) => m.model.includes("llama"));
    expect(llama?.requests).toBe(120);
    expect(llama?.tokensIn).toBe(1000);
    expect(llama?.tokensOut).toBe(200);
    expect(llama?.costUsd).toBeCloseTo(0.0042);
    expect(out.costBasis).toBe("gateway-estimate");
    expect(out.errors).toHaveLength(0);
    // The probe result persists for the day.
    expect(kv.puts).toContain(PROBE_KEY);
  });

  it("degrades to counts when the undocumented fields are absent, with an explicit gap note", async () => {
    const kv = fakeKV();
    const { calls } = stubAI({
      graphql: (q) => {
        if (PROBE_QUERY_MARKERS.some((m) => q.includes(m))) {
          // Field error shape: GraphQL answers 200 with errors for unknown fields.
          return { __probeError: true };
        }
        return { viewer: { accounts: [{ g: [{ count: 7, dimensions: { gateway: "gw-1", model: "m", provider: "p" } }] }] } };
      },
    });
    // The probe-error shape needs a GraphQL errors array; emulate by having
    // the stub return an errors body instead. Simpler: patch below.
    void calls;
    globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const q = String(init?.body ?? "");
      if (url.includes("/graphql")) {
        if (PROBE_QUERY_MARKERS.some((m) => q.includes(m))) {
          return new Response(JSON.stringify({ errors: [{ message: "unknown field" }] }), { status: 200 });
        }
        return new Response(JSON.stringify({ data: { viewer: { accounts: [{ g: [{ count: 7, dimensions: { gateway: "gw-1", model: "m", provider: "p" } }] }] } } }), { status: 200 });
      }
      return new Response(JSON.stringify({ success: true, result: [{ id: "gw-1", name: "n" }] }), { status: 200 });
    }) as typeof fetch;

    const out = await collectAIGateways("acct", "tok", new Date("2026-10-11T12:00:00Z"), kv);
    expect(out.gateways[0].models[0].requests).toBe(7);
    expect(out.gateways[0].models[0].tokensIn).toBeUndefined();
    expect(out.gateways[0].models[0].costUsd).toBeUndefined();
    expect(out.notes.some((n) => n.includes("token/cost"))).toBe(true);
  });

  it("a cached negative probe skips the probing query for the day", async () => {
    const kv = fakeKV();
    await kv.put(PROBE_KEY, JSON.stringify({ tokens: false, cost: false, checkedAt: new Date("2026-10-11T11:00:00Z").toISOString() }));
    const { calls } = stubAI({
      graphql: () => ({ viewer: { accounts: [{ g: [{ count: 2, dimensions: { gateway: "gw-1", model: "m", provider: "p" } }] }] } }),
    });
    const out = await collectAIGateways("acct", "tok", new Date("2026-10-11T12:00:00Z"), kv);
    const graphqlBodies = calls.filter((c) => c.url.includes("/graphql")).map((c) => c.body ?? "");
    expect(graphqlBodies.every((b) => !b.includes("tokensIn"))).toBe(true);
    expect(out.gateways[0].models[0].requests).toBe(2);
  });

  it("a gateway-list failure surfaces as an error row, not a throw", async () => {
    const kv = fakeKV();
    globalThis.fetch = (async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/ai-gateway/gateways")) {
        return new Response(JSON.stringify({ success: false, errors: [{ message: "nope" }] }), { status: 403 });
      }
      return new Response(JSON.stringify({ data: { viewer: { accounts: [{ g: [] }] } } }), { status: 200 });
    }) as typeof fetch;
    const out = await collectAIGateways("acct", "tok", new Date(), kv);
    expect(out.gateways).toHaveLength(0);
    expect(out.errors.some((e) => e.includes("gateway"))).toBe(true);
  });
});
