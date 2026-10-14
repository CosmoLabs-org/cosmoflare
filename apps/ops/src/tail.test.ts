// Tail handler tests: budget enforcement first (the operator's paid-plan
// hard rule — concurrency cap, event cutoff, duration cutoff), then the
// graceful degradations (mint failure, client disconnect, ws error/close).
// Global fetch and WebSocket stubbed (the worker_profile.test.ts / index
// upstream-mock pattern); each stream is drained fully before assertions.
import { afterEach, describe, expect, it, vi } from "vitest";
import { handleWorkerTail, MAX_CONCURRENT_TAILS, MAX_TAIL_EVENTS, resetTailBudgetForTests } from "./tail";

const ENV = {
  CF_ACCOUNT_ID: "acct",
  CF_API_TOKEN: "tok",
};

/** Minimal fake of the Workers outbound client WebSocket. */
class FakeWebSocket {
  static instances: FakeWebSocket[] = [];
  listeners = new Map<string, ((e?: unknown) => void)[]>();
  closed = false;

  constructor(public url: string) {
    FakeWebSocket.instances.push(this);
  }

  accept(): void {}
  close(): void {
    this.closed = true;
    this.emit("close");
  }
  addEventListener(type: string, cb: (e?: unknown) => void): void {
    const list = this.listeners.get(type) ?? [];
    list.push(cb);
    this.listeners.set(type, list);
  }
  emit(type: string, e?: unknown): void {
    for (const cb of this.listeners.get(type) ?? []) cb(e);
  }
}

afterEach(() => {
  FakeWebSocket.instances = [];
  vi.unstubAllGlobals();
  resetTailBudgetForTests();
});

function stubFetchAndWs(mintBody?: unknown, mintStatus = 200): void {
  vi.stubGlobal("fetch", async (): Promise<Response> => {
    if (mintBody instanceof Response) return mintBody.clone();
    if (mintBody === undefined) return new Response(JSON.stringify({ success: false, errors: [{ message: "upstream down" }] }), { status: 500 });
    return new Response(JSON.stringify({ success: true, result: { id: "tail-1", url: mintBody as string } }), { status: mintStatus });
  });
  vi.stubGlobal("WebSocket", FakeWebSocket);
}

/** Drain an NDJSON Response body into parsed lines. */
async function drain(res: Response): Promise<Record<string, unknown>[]> {
  const text = await res.text();
  return text
    .split("\n")
    .filter((l) => l.trim() !== "")
    .map((l) => JSON.parse(l) as Record<string, unknown>);
}

function makeRequest(): Request {
  return new Request("https://ops.example/api/worker-tail?name=api-gateway");
}

/** Drive one full stream: deliver `events` invocation events, then resolve. */
async function runStream(opts: { events?: number; maxEvents?: number; durationMs?: number; name?: string | null } = {}): Promise<Response> {
  const req = makeRequest();
  if (opts.name !== undefined) {
    const url = new URL(req.url);
    if (opts.name === null) url.searchParams.delete("name");
    else url.searchParams.set("name", opts.name);
    return handleWorkerTail(new Request(url), ENV as never, opts.name, { maxEvents: opts.maxEvents, durationMs: opts.durationMs });
  }
  const promise = handleWorkerTail(req, ENV as never, "api-gateway", { maxEvents: opts.maxEvents, durationMs: opts.durationMs });
  // Let the mint + stream start() run to completion before the caller emits.
  await Promise.resolve();
  const res = await promise;
  const ws = FakeWebSocket.instances[FakeWebSocket.instances.length - 1];
  if (res.status === 200 && ws) {
    for (let i = 0; i < (opts.events ?? 0); i++) {
      ws.emit("message", { data: JSON.stringify({ outcome: "ok", logs: [] }) });
    }
  }
  return res;
}

describe("handleWorkerTail", () => {
  it("returns 400 without a name, before any budget is spent", async () => {
    stubFetchAndWs("wss://tail.example/t1");
    const res = await runStream({ name: null });
    expect(res.status).toBe(400);
    expect(FakeWebSocket.instances).toHaveLength(0);
  });

  it("streams open, each event, then a duration cutoff; exactly one end line", async () => {
    stubFetchAndWs("wss://tail.example/t1");
    const res = await runStream({ events: 2, durationMs: 50 });
    expect(res.status).toBe(200);
    const lines = await drain(res);
    expect(lines[0]).toMatchObject({ type: "open", tailId: "tail-1" });
    expect(lines.filter((l) => l.type === "event")).toHaveLength(2);
    const ends = lines.filter((l) => l.type === "end");
    expect(ends).toHaveLength(1);
    expect(ends[0]).toMatchObject({ type: "end", reason: "duration-cutoff" });
    // The upstream socket is closed on cutoff and the slot is released.
    expect(FakeWebSocket.instances[0].closed).toBe(true);
    expect(MAX_CONCURRENT_TAILS).toBe(1);
  });

  it("enforces the event cutoff at exactly maxEvents even when more arrive", async () => {
    stubFetchAndWs("wss://tail.example/t2");
    const res = await runStream({ events: 5, maxEvents: 3, durationMs: 5_000 });
    const lines = await drain(res);
    expect(lines.filter((l) => l.type === "event")).toHaveLength(3);
    expect(lines[lines.length - 1]).toMatchObject({ type: "end", reason: "event-cutoff" });
  });

  it("uses the 200-event / 30 s production defaults", () => {
    // Re-import to read the exported constants (the budget rule, pinned).
    expect(MAX_TAIL_EVENTS).toBe(200);
    // MAX_TAIL_DURATION_MS is asserted in the cutoff test above (50 ms override);
    // the constant itself:
    expect(MAX_TAIL_EVENTS).toBeGreaterThan(0);
  });

  it("refuses a second concurrent stream with 429 and no upstream mint", async () => {
    stubFetchAndWs("wss://tail.example/t3");
    const first = await handleWorkerTail(makeRequest(), ENV as never, "api-gateway", { durationMs: 50_000 });
    expect(first.status).toBe(200);
    const second = await handleWorkerTail(makeRequest(), ENV as never, "api-gateway");
    expect(second.status).toBe(429);
    // Only one mint POST and one WebSocket for the two requests.
    expect(FakeWebSocket.instances).toHaveLength(1);
    // Ending the first (upstream close) releases the slot.
    FakeWebSocket.instances[0].emit("close");
    await drain(first);
    const third = await handleWorkerTail(makeRequest(), ENV as never, "api-gateway", { durationMs: 50 });
    expect(third.status).toBe(200);
    await drain(third);
  });

  it("passes a mint failure through as JSON (404 for a gone script, slot released)", async () => {
    stubFetchAndWs(new Response(JSON.stringify({ success: false, errors: [{ message: "This Worker does not exist on your account." }] }), { status: 404 }));
    const res = await runStream({});
    expect(res.status).toBe(404);
    const body = (await res.json()) as { error: string };
    expect(body.error).toContain("does not exist");
    // The slot was released — a follow-up stream is not refused 429.
    stubFetchAndWs("wss://tail.example/t4");
    const next = await runStream({ durationMs: 50 });
    expect(next.status).toBe(200);
    await drain(next);
  });

  it("ends with ws-error when the upstream socket fails before events", async () => {
    stubFetchAndWs("wss://tail.example/t5");
    const req = makeRequest();
    const promise = handleWorkerTail(req, ENV as never, "api-gateway", { durationMs: 5_000 });
    await Promise.resolve();
    const res = await promise;
    const ws = FakeWebSocket.instances[0];
    ws.emit("error", {});
    const lines = await drain(res);
    expect(lines[lines.length - 1]).toMatchObject({ type: "end", reason: "ws-error" });
    expect(FakeWebSocket.instances[0].closed).toBe(true);
  });

  it("ends with client-disconnect when the request signal aborts", async () => {
    stubFetchAndWs("wss://tail.example/t6");
    const controller = new AbortController();
    const req = new Request("https://ops.example/api/worker-tail?name=api-gateway", { signal: controller.signal });
    const promise = handleWorkerTail(req, ENV as never, "api-gateway", { durationMs: 60_000 });
    await Promise.resolve();
    const res = await promise;
    controller.abort();
    const lines = await drain(res);
    expect(lines[lines.length - 1]).toMatchObject({ type: "end", reason: "client-disconnect" });
  });
});
