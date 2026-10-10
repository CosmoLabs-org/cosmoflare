// Cosmoflare Ops rules API tests (FEAT-052): GET starter fallback, PUT
// validation matrix, and the 405 path. The Access gate itself is index.ts's
// job (covered there); these hit handleRules directly.

import { describe, expect, it } from "vitest";
import { handleRules, MAX_EXCLUDES, MAX_RULES } from "./rules-api";
import { RULES_KEY, starterRules } from "./scheduled";
import type { Rule } from "./rules";

// fakeKV is the same in-memory KVNamespace stand-in used across the suite.
function fakeKV(): KVNamespace {
  const store = new Map<string, unknown>();
  return {
    get: async (key: string, type?: string) => (type === "json" && store.has(key) ? JSON.parse(store.get(key) as string) : (store.get(key) ?? null)),
    put: async (key: string, value: string) => void store.set(key, value),
  } as unknown as KVNamespace;
}

// envP builds a minimal OpsEnv over a fake KV (only what handleRules needs).
function envP(kv: KVNamespace): never {
  return { OPS_KV: kv } as never;
}

// rule builds a valid rule with overrides.
function rule(name: string, over: Partial<Rule> = {}): Rule {
  return { name, condition: "zone-uncached-requests", threshold: 100, enabled: true, ...over };
}

// putRules wraps handleRules PUT with a JSON body.
function putRules(kv: KVNamespace, body: unknown): Promise<Response> {
  return handleRules(new Request("https://ops.example/api/rules", { method: "PUT", body: JSON.stringify(body) }), envP(kv));
}

describe("GET /api/rules", () => {
  it("returns the starter rules with starter=true when KV is empty", async () => {
    const res = await handleRules(new Request("https://ops.example/api/rules"), envP(fakeKV()));
    expect(res.status).toBe(200);
    expect(res.headers.get("Content-Type")).toBe("application/json");
    expect(res.headers.get("Cache-Control")).toBe("private, no-store");
    const body = (await res.json()) as { rules: Rule[]; conditions: string[]; starter: boolean };
    expect(body.rules).toEqual(starterRules());
    expect(body.starter).toBe(true);
    expect(body.conditions).toEqual(["zone-uncached-requests", "zone-cache-miss-pct", "d1-rows-read", "kv-writes"]);
  });
  it("returns the stored rules with starter=false when KV holds a set", async () => {
    const kv = fakeKV();
    const stored = [rule("mine")];
    await kv.put(RULES_KEY, JSON.stringify(stored));
    const res = await handleRules(new Request("https://ops.example/api/rules"), envP(kv));
    const body = (await res.json()) as { rules: Rule[]; starter: boolean };
    expect(body.rules).toEqual(stored);
    expect(body.starter).toBe(false);
  });
});

describe("PUT /api/rules", () => {
  it("validates and persists a valid set, returning the normalized rules", async () => {
    const kv = fakeKV();
    const res = await putRules(kv, {
      rules: [
        rule("  padded  ", { exclude: ["  zone.example  ", "", "drop.example"] }),
        rule("d1 big", { condition: "d1-rows-read", threshold: 1e9 }),
      ],
    });
    expect(res.status).toBe(200);
    const body = (await res.json()) as { rules: Rule[] };
    expect(body.rules).toEqual([
      { name: "padded", condition: "zone-uncached-requests", threshold: 100, enabled: true, exclude: ["zone.example", "drop.example"] },
      { name: "d1 big", condition: "d1-rows-read", threshold: 1e9, enabled: true },
    ]);
    expect(await kv.get(RULES_KEY, "json")).toEqual(body.rules); // round-trip through KV
  });
  it("rejects an unknown condition with its index", async () => {
    const kv = fakeKV();
    const res = await putRules(kv, { rules: [rule("ok"), rule("bad", { condition: "zone-nope" })] });
    expect(res.status).toBe(400);
    expect(await res.json()).toEqual({ error: expect.stringContaining("unknown condition"), index: 1 });
    expect(await kv.get(RULES_KEY, "json")).toBeNull(); // nothing persisted
  });
  it("rejects duplicate names case-insensitively", async () => {
    const kv = fakeKV();
    const res = await putRules(kv, { rules: [rule("Alert"), rule("alert")] });
    expect(res.status).toBe(400);
    expect(await res.json()).toEqual({ error: expect.stringContaining("duplicate"), index: 1 });
  });
  it("rejects more than 20 rules", async () => {
    const kv = fakeKV();
    const res = await putRules(kv, { rules: Array.from({ length: MAX_RULES + 1 }, (_, i) => rule(`r${i}`)) });
    expect(res.status).toBe(400);
    expect(await res.json()).toEqual({ error: expect.stringContaining("too many rules"), index: -1 });
  });
  it("rejects a non-finite threshold", async () => {
    const kv = fakeKV();
    const res = await putRules(kv, { rules: [rule("bad", { threshold: Number.NaN })] });
    expect(res.status).toBe(400);
    expect((await res.json()) as { index: number }).toMatchObject({ index: 0 });
  });
  it("rejects a negative threshold", async () => {
    const res = await putRules(fakeKV(), { rules: [rule("bad", { threshold: -1 })] });
    expect(res.status).toBe(400);
  });
  it("rejects non-boolean enabled", async () => {
    const res = await putRules(fakeKV(), { rules: [{ ...rule("bad"), enabled: "yes" }] });
    expect(res.status).toBe(400);
  });
  it("rejects an exclude list over the cap after trimming/dropping empties", async () => {
    const kv = fakeKV();
    const exclude = [...Array.from({ length: MAX_EXCLUDES }, (_, i) => `zone${i}.example`), "  extra.example  "];
    const res = await putRules(kv, { rules: [rule("bad", { exclude })] });
    expect(res.status).toBe(400);
  });
  it("rejects a non-array rules body", async () => {
    const res = await putRules(fakeKV(), { rules: "nope" });
    expect(res.status).toBe(400);
    expect((await res.json()) as { index: number }).toMatchObject({ index: -1 });
  });
  it("rejects an invalid JSON body", async () => {
    const res = await handleRules(
      new Request("https://ops.example/api/rules", { method: "PUT", body: "{not json" }),
      envP(fakeKV()),
    );
    expect(res.status).toBe(400);
  });
});

describe("other methods", () => {
  it("returns 405 on DELETE", async () => {
    const res = await handleRules(new Request("https://ops.example/api/rules", { method: "DELETE" }), envP(fakeKV()));
    expect(res.status).toBe(405);
    expect(res.headers.get("Cache-Control")).toBe("private, no-store");
  });
});
