import { afterEach, describe, expect, it, vi } from "vitest";
import { fetchRetry } from "./retry";

// FEAT-061 worker-side policy: mirrors the Go RetryTransport contract.
// Real (tiny) delays keep the loop honest; the jitter source is pinned.

const realFetch = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = realFetch;
  vi.restoreAllMocks();
});

function stubFetch(statuses: number[]): { calls: number } {
  const state = { calls: 0 };
  let i = 0;
  globalThis.fetch = (async () => {
    state.calls++;
    const status = i < statuses.length ? statuses[i] : statuses[statuses.length - 1];
    i++;
    return new Response(status === 429 || status >= 500 ? "busy" : '{"ok":true}', { status });
  }) as typeof fetch;
  return state;
}

const fast = { baseDelayMs: 1, maxDelayMs: 4, rand: () => 0 };

describe("fetchRetry (FEAT-061)", () => {
  it("retries 429 to success, bounded at 2 retries", async () => {
    const s = stubFetch([429, 429, 200]);
    const res = await fetchRetry("https://api.example.com/x", {}, fast);
    expect(res.status).toBe(200);
    expect(s.calls).toBe(3);
  });

  it("exhausted 429s surface the last response (never more than 3 attempts)", async () => {
    const s = stubFetch([429]);
    const res = await fetchRetry("https://api.example.com/x", {}, fast);
    expect(res.status).toBe(429);
    expect(s.calls).toBe(3);
  });

  it("5xx retries idempotent methods only", async () => {
    const sGet = stubFetch([500, 200]);
    const get = await fetchRetry("https://api.example.com/x", { method: "GET" }, fast);
    expect(get.status).toBe(200);
    expect(sGet.calls).toBe(2);

    const sPost = stubFetch([500, 200]);
    const post = await fetchRetry("https://api.example.com/x", { method: "POST", body: "{}" }, fast);
    expect(post.status).toBe(500);
    expect(sPost.calls).toBe(1);
  });

  it("429 retries a POST (the API did not process it) with the body replayed", async () => {
    const bodies: string[] = [];
    let call = 0;
    globalThis.fetch = (async (_u: RequestInfo | URL, init?: RequestInit) => {
      call++;
      bodies.push(String(init?.body ?? ""));
      return new Response(call === 1 ? "busy" : '{"ok":true}', { status: call === 1 ? 429 : 201 });
    }) as typeof fetch;
    const res = await fetchRetry("https://api.example.com/x", { method: "POST", body: '{"name":"x"}' }, fast);
    expect(res.status).toBe(201);
    expect(bodies).toEqual(['{"name":"x"}', '{"name":"x"}']);
  });

  it("success on first try makes no waits", async () => {
    const s = stubFetch([200]);
    const res = await fetchRetry("https://api.example.com/x", {}, fast);
    expect(res.status).toBe(200);
    expect(s.calls).toBe(1);
  });

  it("a streaming body is one-shot (cannot be rewound)", async () => {
    const s = stubFetch([429, 200]);
    const stream = new ReadableStream({ start(c) { c.enqueue(new Uint8Array([1])); c.close(); } });
    const res = await fetchRetry("https://api.example.com/x", { method: "POST", body: stream }, fast);
    expect(res.status).toBe(429);
    expect(s.calls).toBe(1);
  });
});
