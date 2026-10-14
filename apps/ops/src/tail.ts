// Worker live tail (worker profile expansion, 2026-10-14): mints a tail
// WebSocket server-side (POST /accounts/{account}/workers/scripts/{script}/tails
// — the auth probe on 2026-10-14 confirmed the ops token mints successfully)
// and streams the invocation events to the pager as NDJSON over a chunked
// Response. HARD RULE (paid Workers plan, operator directive): every fetch
// path carries an enforced budget — a per-isolate concurrency cap of ONE
// active tail, a 30 s duration cutoff, and a 200-event cutoff. Cache
// absorption is not a cap: nothing here is cached, every stream pays the
// mint POST and holds one live WebSocket for at most 30 s.
// Degrades gracefully: a mint failure is a JSON error (the pager renders the
// note), a quiet worker simply produces no event lines before the cutoff.

import { fetchRetry } from "./retry";
import type { OpsEnv } from "./scheduled";

const API = "https://api.cloudflare.com/client/v4";

/** Event cutoff — at most this many invocation events per stream. */
export const MAX_TAIL_EVENTS = 200;
/** Duration cutoff — the stream (and the upstream WebSocket) dies after this. */
export const MAX_TAIL_DURATION_MS = 30_000;
/** Per-isolate concurrency cap: ONE live tail per isolate, ever. */
export const MAX_CONCURRENT_TAILS = 1;

// The per-isolate slot is module state — it survives across requests in the
// same isolate, which is exactly what the operator's budget rule needs. A
// second concurrent request is refused 429 without any upstream call.
let activeTails = 0;

/** Test hook: release every held slot (tests always call this in afterEach). */
export function resetTailBudgetForTests(): void {
  activeTails = 0;
}

interface TailWsEvent {
  addEventListener(type: "message", cb: (e: { data: unknown }) => void): void;
  addEventListener(type: "close", cb: () => void): void;
  addEventListener(type: "error", cb: () => void): void;
  addEventListener(type: "open", cb: () => void): void;
  close(): void;
  // accept() is the runtime's way to adopt a client socket; present on both
  // server and outbound client sockets in the Workers runtime.
  accept?(): void;
}

export interface TailHandlerOpts {
  /** Overrides for tests — production uses the exported constants. */
  maxEvents?: number;
  durationMs?: number;
}

interface MintResult {
  id: string;
  url: string;
}

/** Mint a tail WebSocket URL (POST tails). Throws on !success or transport
 *  error; the caller turns that into a JSON error before streaming starts. */
async function mintTailUrl(env: OpsEnv, script: string): Promise<MintResult> {
  const scriptParam = encodeURIComponent(script);
  const res = await fetchRetry(`${API}/accounts/${env.CF_ACCOUNT_ID}/workers/scripts/${scriptParam}/tails`, {
    method: "POST",
    headers: { Authorization: `Bearer ${env.CF_API_TOKEN}` },
  });
  const body = (await res.json()) as { success?: boolean; result?: MintResult; errors?: { message: string }[] };
  if (!res.ok || body.success !== true || !body.result?.url) {
    throw new Error(body.errors?.[0]?.message ?? `HTTP ${res.status}`);
  }
  return body.result;
}

function jsonError(error: string, status: number): Response {
  return new Response(JSON.stringify({ error }), {
    status,
    headers: { "Content-Type": "application/json", "Cache-Control": "private, no-store" },
  });
}

/**
 * GET /api/worker-tail?name=<script> — one bounded live-tail stream.
 * Responses are NDJSON: {"type":"open",...}, {"type":"event",...} per
 * invocation, then {"type":"end","reason":...}. Every terminal path (event
 * cutoff, duration cutoff, upstream close/error, client disconnect) emits
 * exactly one end line and releases the concurrency slot.
 */
export async function handleWorkerTail(request: Request, env: OpsEnv, name: string | null, opts: TailHandlerOpts = {}): Promise<Response> {
  if (!name) return jsonError("missing name", 400);

  // Budget gate BEFORE any upstream call (operator hard rule): the per-
  // isolate cap refuses without spending a mint POST.
  if (activeTails >= MAX_CONCURRENT_TAILS) {
    return jsonError("another live tail is already streaming — try again when it ends", 429);
  }
  activeTails += 1;

  // Mint first: a failure here never opens the stream — the caller gets a
  // plain JSON error with the upstream message (404 script gone, 403 scope,
  // transport) and the slot is released immediately.
  let mint: MintResult;
  try {
    mint = await mintTailUrl(env, name);
  } catch (err) {
    activeTails -= 1;
    const message = err instanceof Error ? err.message : String(err);
    // The two upstream statuses worth preserving: 404 (script gone) and
    // 403 (token scope) — anything else is a 502 with the message.
    const status = message.includes("does not exist") ? 404 : 502;
    return jsonError(`tail unavailable: ${message}`, status);
  }

  const maxEvents = opts.maxEvents ?? MAX_TAIL_EVENTS;
  const durationMs = opts.durationMs ?? MAX_TAIL_DURATION_MS;

  const ws: TailWsEvent = new WebSocket(mint.url);
  try {
    ws.accept?.();
  } catch {
    // accept() is optional on some runtimes; absence is fine.
  }

  let events = 0;
  // `ended` = terminal path taken (no more event writes, slot released);
  // `controllerClosed` = the ReadableStream controller is closed (writeLine
  // refuses past it). The end line must be written BEFORE ended flips, so the
  // two flags stay separate — one flag suppressed the end line (test-caught).
  let ended = false;
  let controllerClosed = false;
  let durationTimer: ReturnType<typeof setTimeout> | undefined;
  let wsClosed = false;

  const closeWs = (): void => {
    if (wsClosed) return;
    wsClosed = true;
    try {
      ws.close();
    } catch {
      // already closing/closed — nothing to do
    }
  };

  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      const encoder = new TextEncoder();

      const writeLine = (line: string): void => {
        if (controllerClosed) return;
        try {
          controller.enqueue(encoder.encode(`${line}\n`));
        } catch {
          // The client vanished between the abort listener and this enqueue —
          // stop writing; cancel()/abort handles the cleanup.
          controllerClosed = true;
        }
      };

      const end = (reason: string, upstreamError?: string): void => {
        if (ended) return;
        // Write the end line first, then flip: writeLine must still be able
        // to enqueue (the write-before-flag order is the test-caught bug).
        writeLine(JSON.stringify({ type: "end", reason, error: upstreamError }));
        ended = true;
        if (durationTimer !== undefined) clearTimeout(durationTimer);
        closeWs();
        // Release the per-isolate slot exactly once, on every terminal path.
        activeTails -= 1;
        try {
          controller.close();
        } catch {
          // close() after an enqueue failure — nothing to do.
        } finally {
          controllerClosed = true;
        }
      };

      const abortHandler = (): void => end("client-disconnect");
      request.signal.addEventListener("abort", abortHandler);

      writeLine(JSON.stringify({ type: "open", tailId: mint.id }));

      // Duration cutoff: the hard 30 s ceiling on the whole stream.
      durationTimer = setTimeout(() => end("duration-cutoff"), durationMs);

      ws.addEventListener("message", (e) => {
        if (ended) return;
        let parsed: unknown;
        try {
          parsed = JSON.parse(typeof e.data === "string" ? e.data : String(e.data));
        } catch {
          // A non-JSON frame is skipped, not fatal — tails send JSON events.
          return;
        }
        events += 1;
        writeLine(JSON.stringify({ type: "event", event: parsed }));
        if (events >= maxEvents) end("event-cutoff");
      });
      ws.addEventListener("close", () => end("ws-closed"));
      ws.addEventListener("error", () => end("ws-error", "the tail WebSocket connection failed"));
    },
    // cancel() is the client walking away mid-stream (fetch body cancelled):
    // same cleanup as an abort — one end, one slot release, ws closed. If
    // end() already ran, the slot is released there and cancel() must not
    // release it twice.
    cancel(): void {
      if (ended) return;
      ended = true;
      if (durationTimer !== undefined) clearTimeout(durationTimer);
      closeWs();
      activeTails -= 1;
    },
  });

  return new Response(stream, {
    headers: {
      "Content-Type": "application/x-ndjson",
      "Cache-Control": "private, no-store",
    },
  });
}
