import { afterEach, describe, expect, it, vi } from "vitest";
import { collectEmailMetrics } from "./email";

// FEAT-059 (ROAD-107): per-zone Email Routing status. The endpoint is
// operator-gated (verified live 2026-10-10: code 10000 without Email
// Routing Read), so the collector must soft-fail into per-zone error
// strings — never a throw, never a failed page.

const realFetch = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = realFetch;
  vi.restoreAllMocks();
});

interface Call {
  url: string;
}

function stubEmail(opts: {
  zones?: { id: string; name: string }[];
  routing?: (zoneId: string) => { status: number; body: unknown };
} = {}): { calls: Call[] } {
  const calls: Call[] = [];
  globalThis.fetch = (async (input: RequestInfo | URL) => {
    const url = String(input);
    calls.push({ url });
    if (url.includes("/zones?account.id=")) {
      return new Response(JSON.stringify({ success: true, result: opts.zones ?? [{ id: "z-1", name: "cosmolabs.org" }, { id: "z-2", name: "noble.coffee" }] }), { status: 200 });
    }
    if (url.includes("/email/routing")) {
      const zoneId = url.match(/\/zones\/([^/]+)\/email\/routing/)?.[1] ?? "";
      const r = opts.routing
        ? opts.routing(zoneId)
        : { status: 200, body: { success: true, result: { enabled: true, name: "cosmolabs.org" } } };
      return new Response(JSON.stringify(r.body), { status: r.status });
    }
    return new Response(JSON.stringify({ success: false }), { status: 500 });
  }) as typeof fetch;
  return { calls };
}

describe("collectEmailMetrics (FEAT-059)", () => {
  it("returns one row per zone with enabled + name on success", async () => {
    stubEmail({
      routing: (zoneId) =>
        zoneId === "z-1"
          ? { status: 200, body: { success: true, result: { enabled: true, name: "cosmolabs.org" } } }
          : { status: 200, body: { success: true, result: { enabled: false } } },
    });
    const out = await collectEmailMetrics("acct", "tok", new Date("2026-10-11T12:00:00Z"));
    expect(out.errors).toEqual([]);
    expect(out.generatedAt).toBe("2026-10-11T12:00:00.000Z");
    // Sorted by zone name regardless of collection order.
    expect(out.zones.map((z) => z.zone)).toEqual(["cosmolabs.org", "noble.coffee"]);
    expect(out.zones[0]).toEqual({ zone: "cosmolabs.org", enabled: true, name: "cosmolabs.org" });
    // No name in the routing settings → the field stays absent.
    expect(out.zones[1]).toEqual({ zone: "noble.coffee", enabled: false });
  });

  it("soft-fails a gated zone into a per-zone error string (code 10000 shape)", async () => {
    stubEmail({
      routing: () => ({ status: 403, body: { success: false, errors: [{ code: 10000, message: "Authentication error" }] } }),
    });
    const out = await collectEmailMetrics("acct", "tok", new Date("2026-10-11T12:00:00Z"));
    expect(out.zones).toEqual([]);
    expect(out.errors).toHaveLength(2);
    expect(out.errors[0]).toBe("email routing unavailable for cosmolabs.org: HTTP 403");
    expect(out.errors[1]).toBe("email routing unavailable for noble.coffee: HTTP 403");
  });

  it("mixes successes and per-zone failures without throwing", async () => {
    stubEmail({
      routing: (zoneId) =>
        zoneId === "z-1"
          ? { status: 200, body: { success: true, result: { enabled: true, name: "cosmolabs.org" } } }
          : { status: 403, body: { success: false, errors: [{ code: 10000, message: "Authentication error" }] } },
    });
    const out = await collectEmailMetrics("acct", "tok", new Date("2026-10-11T12:00:00Z"));
    expect(out.zones).toEqual([{ zone: "cosmolabs.org", enabled: true, name: "cosmolabs.org" }]);
    expect(out.errors).toEqual(["email routing unavailable for noble.coffee: HTTP 403"]);
  });

  it("degrades a failed zone list to a single error string, no routing calls", async () => {
    const stub = stubEmail({ zones: undefined });
    // Override the zone-list answer to the gated 403 shape.
    globalThis.fetch = (async (input: RequestInfo | URL) => {
      const url = String(input);
      stub.calls.push({ url });
      return new Response(JSON.stringify({ success: false, errors: [{ code: 10000, message: "Authentication error" }] }), { status: 403 });
    }) as typeof fetch;
    const out = await collectEmailMetrics("acct", "tok", new Date("2026-10-11T12:00:00Z"));
    expect(out.zones).toEqual([]);
    // rest() surfaces the API's first error message (code 10000's text).
    expect(out.errors).toEqual(["zone list unavailable: Authentication error"]);
    expect(stub.calls.filter((c) => c.url.includes("/email/routing"))).toHaveLength(0);
  });
});
