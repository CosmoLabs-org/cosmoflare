import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { vapidSubject } from "./webpush";

// Mock the push library at the module boundary: buildPushPayload returns a
// fixed RequestInit-shaped object and records the payload string.
const builtRequests: { url: string; init: RequestInit }[] = [];
vi.mock("@block65/webcrypto-web-push", () => ({
  buildPushPayload: async (message: { data: string }, sub: { endpoint: string }) => {
    builtRequests.push({ url: sub.endpoint, init: { method: "POST", headers: { authorization: "vapid t=x", ttl: "3600", urgency: "normal" }, body: new TextEncoder().encode(message.data) } });
    return { method: "POST", headers: { authorization: "vapid t=x", ttl: "3600", urgency: "normal" }, body: new TextEncoder().encode(message.data) };
  },
}));

import { sendPushes, VapidConfig } from "./webpush";
import type { StoredSubscription } from "./subscriptions";

// subFor builds a stored subscription.
function subFor(endpoint: string): StoredSubscription {
  return { endpoint, p256dh: "pk", auth: "ak" };
}

// vapidP builds a valid VAPID config.
function vapidP(overrides: Partial<VapidConfig> = {}): VapidConfig {
  return { VAPID_PUBLIC_KEY: "pub", VAPID_PRIVATE_KEY: "priv", VAPID_SUBJECT: "mailto:ops@example.com", ...overrides };
}

// payloadP builds a minimal pager payload.
function payloadP(): unknown {
  return { id: "x", severity: "info", service: "cloudflare", title: "t", detail: "d", fired_at: "2026-10-09T00:00:00Z" };
}

let realFetch: typeof fetch;

beforeEach(() => {
  builtRequests.length = 0;
  realFetch = globalThis.fetch;
});

afterEach(() => {
  globalThis.fetch = realFetch;
});

describe("vapidSubject", () => {
  it("adds a single mailto: prefix when absent", () => {
    expect(vapidSubject("ops@example.com").subject).toBe("mailto:ops@example.com");
  });
  it("keeps a single prefix as-is", () => {
    expect(vapidSubject("mailto:ops@example.com").subject).toBe("mailto:ops@example.com");
  });
  it("flags a double prefix (the mailto:mailto: bug class)", () => {
    const out = vapidSubject("mailto:mailto:ops@example.com");
    expect(out.error).toContain("exactly once");
  });
});

describe("sendPushes", () => {
  it("delivers to every subscription and counts sends", async () => {
    globalThis.fetch = (async () => new Response(null, { status: 201 })) as typeof fetch;
    const out = await sendPushes(vapidP(), [subFor("https://p/1"), subFor("https://p/2")], payloadP());
    expect(out).toEqual({ sent: 2, pruned: 0, issues: [] });
    expect(builtRequests.map((r) => r.url)).toEqual(["https://p/1", "https://p/2"]);
    // the envelope is the JSON payload string
    const envelope = new TextDecoder().decode(builtRequests[0].init.body as Uint8Array);
    expect(JSON.parse(envelope)).toEqual(payloadP());
  });
  it("prunes on 404 and 410, keeps delivering to the rest", async () => {
    globalThis.fetch = (async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/gone")) return new Response(null, { status: 410 });
      if (url.endsWith("/gone2")) return new Response(null, { status: 404 });
      return new Response(null, { status: 201 });
    }) as typeof fetch;
    const out = await sendPushes(vapidP(), [subFor("https://p/gone"), subFor("https://p/ok"), subFor("https://p/gone2")], payloadP());
    expect(out).toEqual({ sent: 1, pruned: 2, issues: [] });
  });
  it("records status + reason for other non-2xx without aborting", async () => {
    globalThis.fetch = (async () => new Response(JSON.stringify({ reason: "BadJwtToken" }), { status: 403 })) as typeof fetch;
    const out = await sendPushes(vapidP(), [subFor("https://p/1")], payloadP());
    expect(out.sent).toBe(0);
    expect(out.issues[0]).toContain("403");
    expect(out.issues[0]).toContain("BadJwtToken");
  });
  it("transport errors become issues, remaining subs still attempted", async () => {
    globalThis.fetch = (async (input: RequestInfo | URL) => {
      if (String(input).endsWith("/1")) throw new Error("boom");
      return new Response(null, { status: 201 });
    }) as typeof fetch;
    const out = await sendPushes(vapidP(), [subFor("https://p/1"), subFor("https://p/2")], payloadP());
    expect(out.sent).toBe(1);
    expect(out.issues[0]).toContain("boom");
  });
  it("an unusable VAPID_SUBJECT fails fast with an issue and zero fetches", async () => {
    const fetchSpy = (async () => new Response(null, { status: 201 })) as typeof fetch;
    globalThis.fetch = fetchSpy;
    const out = await sendPushes(vapidP({ VAPID_SUBJECT: "mailto:mailto:x@y" }), [subFor("https://p/1")], payloadP());
    expect(out.sent).toBe(0);
    expect(out.issues[0]).toContain("exactly once");
  });
});
