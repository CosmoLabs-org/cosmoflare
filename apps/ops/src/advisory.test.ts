// Route tests for GET /api/advisory (P-17, FEAT-pC1EV32). The collector is
// mocked at the module boundary (briefs/collect owns its own suite) — these
// tests pin the endpoint contract: Access gate, query-param handling, the
// envelope shape, the zero-findings null brief, and the 502 partial path.
// compileBrief/renderAgentBrief/TEMPLATES stay REAL, so the composition the
// pager consumes is exercised end-to-end against fixture facts.

import { afterEach, describe, expect, it, vi } from "vitest";
import worker from "./index";
import type { ZoneFacts } from "./briefs/types";

// Switchable collector stub: each test sets `next` to fixture facts or a
// throwing function; the default fixture is a low-cache-hit + firing-alert
// zone (a brief that MUST render non-empty).
const collect = vi.hoisted(() => ({
  next: null as null | ((env: unknown, zone: string) => Promise<ZoneFacts>),
}));
vi.mock("./briefs/collect", () => ({
  collectZoneFacts: (env: unknown, zone: string) => {
    if (!collect.next) throw new Error("test bug: no collector fixture set");
    return collect.next(env, zone);
  },
}));

// Same Access bypass switch index.test.ts uses: default false keeps the
// rejection tests on the real fail-closed path.
const authBypass = vi.hoisted(() => ({ value: false }));
vi.mock("./access", async (importOriginal) => {
  const mod = await importOriginal<typeof import("./access")>();
  return {
    ...mod,
    verifyAccessJwt: async (...args: Parameters<typeof mod.verifyAccessJwt>) =>
      authBypass.value ? { ok: true as const, email: "owner@example.com" } : mod.verifyAccessJwt(...args),
  };
});

afterEach(() => {
  authBypass.value = false;
  collect.next = null;
});

const env = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "tok", VAPID_PUBLIC_KEY: "BPUBLICKEY", ACCESS_TEAM_DOMAIN: "", ACCESS_AUD: "", ASSETS: { fetch: async () => new Response("asset") } };

function fakeCtx(): ExecutionContext {
  return { waitUntil: () => undefined, passThroughOnException: () => undefined } as unknown as ExecutionContext;
}

// Minimal upstream: only the zones list the route's name→id resolution
// reads (the shared "zones-list" cached key — never a per-advisory fetch).
function mockZones(): () => void {
  const real = globalThis.fetch;
  globalThis.fetch = (async (input: RequestInfo | URL) => {
    const url = String(input);
    if (url.includes("/zones?")) {
      return new Response(JSON.stringify({ success: true, result: [{ id: "z1", name: "zone.example" }], result_info: { total_pages: 1 } }));
    }
    return new Response(JSON.stringify({ success: true, result: [], result_info: { total_pages: 1 } }));
  }) as typeof fetch;
  return () => {
    globalThis.fetch = real;
  };
}

function firingFacts(): ZoneFacts {
  return {
    zone: "zone.example",
    zoneId: "z1",
    window: "7d",
    coverage: { requests: 10_000, cached: 100, hitRatePct: 1 },
    firingAlerts: [],
  };
}

function cleanFacts(): ZoneFacts {
  return { zone: "zone.example", zoneId: "z1", window: "7d", firingAlerts: [] };
}

describe("GET /api/advisory", () => {
  it("rejects /api/advisory without a valid Access JWT (defense in depth)", async () => {
    const restore = mockZones();
    try {
      const res = await worker.fetch(new Request("https://ops.example/api/advisory?zone=zone.example"), env as never, fakeCtx());
      expect(res.status).toBe(403);
    } finally {
      restore();
    }
  });

  it("rejects non-GET methods with 405", async () => {
    authBypass.value = true;
    const restore = mockZones();
    try {
      const res = await worker.fetch(new Request("https://ops.example/api/advisory?zone=zone.example", { method: "POST" }), env as never, fakeCtx());
      expect(res.status).toBe(405);
      expect(await res.json()).toMatchObject({ error: expect.stringContaining("method") });
    } finally {
      restore();
    }
  });

  it("missing zone param is a 400", async () => {
    authBypass.value = true;
    const restore = mockZones();
    try {
      const res = await worker.fetch(new Request("https://ops.example/api/advisory"), env as never, fakeCtx());
      expect(res.status).toBe(400);
      expect(await res.json()).toMatchObject({ error: expect.stringContaining("zone") });
    } finally {
      restore();
    }
  });

  it("unknown zone is a 404", async () => {
    authBypass.value = true;
    const restore = mockZones();
    try {
      const res = await worker.fetch(new Request("https://ops.example/api/advisory?zone=nope.example"), env as never, fakeCtx());
      expect(res.status).toBe(404);
      expect(await res.json()).toMatchObject({ error: expect.stringContaining("nope.example") });
    } finally {
      restore();
    }
  });

  it("returns the full envelope with the rendered brief for a firing zone", async () => {
    authBypass.value = true;
    collect.next = async () => firingFacts();
    const restore = mockZones();
    try {
      const res = await worker.fetch(new Request("https://ops.example/api/advisory?zone=zone.example"), env as never, fakeCtx());
      expect(res.status).toBe(200);
      expect(res.headers.get("Cache-Control")).toBe("private, no-store");

      const body = (await res.json()) as {
        zone: string; zoneId: string; generatedAt: string; facts: ZoneFacts;
        brief: { findings: unknown[] } | null; rendered: { agentBrief: string };
      };
      expect(body.zone).toBe("zone.example");
      expect(body.zoneId).toBe("z1");
      expect(Number.isNaN(Date.parse(body.generatedAt))).toBe(false);
      expect(body.facts.coverage?.hitRatePct).toBe(1);
      expect(body.brief).not.toBeNull();
      expect(body.brief!.findings.length).toBeGreaterThan(0);
      // The rendered text is the exact renderer output for the compiled brief.
      expect(body.rendered.agentBrief.startsWith("# zone.example — Cloudflare optimization directive")).toBe(true);
      expect(body.rendered.agentBrief.endsWith("Zone zone.example (z1) — measured 7d by CosmoLabs Ops.")).toBe(true);
      expect(body.rendered.agentBrief).toContain("## Problem");
      expect(body.rendered.agentBrief).toContain("## Actions");
      expect(body.rendered.agentBrief).toContain("## Verify");
    } finally {
      restore();
    }
  });

  it("zero findings yield brief: null and an empty agentBrief", async () => {
    authBypass.value = true;
    collect.next = async () => cleanFacts();
    const restore = mockZones();
    try {
      const res = await worker.fetch(new Request("https://ops.example/api/advisory?zone=zone.example"), env as never, fakeCtx());
      expect(res.status).toBe(200);
      const body = (await res.json()) as { brief: unknown; rendered: { agentBrief: string } };
      expect(body.brief).toBeNull();
      expect(body.rendered.agentBrief).toBe("");
    } finally {
      restore();
    }
  });

  it("a collector hard failure is a 502 with partial: true — never a fabricated brief", async () => {
    authBypass.value = true;
    collect.next = async () => {
      throw new Error("coverage analytics down");
    };
    const restore = mockZones();
    try {
      const res = await worker.fetch(new Request("https://ops.example/api/advisory?zone=zone.example"), env as never, fakeCtx());
      expect(res.status).toBe(502);
      const body = (await res.json()) as { error: string; partial: boolean };
      expect(body.partial).toBe(true);
      expect(body.error).toContain("coverage analytics down");
    } finally {
      restore();
    }
  });
});
