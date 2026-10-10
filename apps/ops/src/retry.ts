// Bounded retry for the Ops Worker's upstream Cloudflare calls (FEAT-061,
// mirroring pkg/cosmoflare/retry_transport.go): 2 retries, exponential
// backoff with jitter, 429 always (the API did not process the request),
// 5xx only for idempotent methods (a 5xx may have processed a POST).
// Push delivery (webpush.ts) intentionally does NOT use this — the push
// service owns delivery semantics.

export interface FetchRetryOptions {
  retries?: number;
  baseDelayMs?: number;
  maxDelayMs?: number;
  /** Injectable jitter source (0..1) — deterministic tests. */
  rand?: () => number;
}

const IDEMPOTENT = new Set(["GET", "HEAD", "OPTIONS", "PUT", "DELETE"]);

function sleep(ms: number, signal?: AbortSignal | null): Promise<void> {
  return new Promise((resolve, reject) => {
    const t = setTimeout(() => resolve(), ms);
    signal?.addEventListener("abort", () => {
      clearTimeout(t);
      reject(signal.reason ?? new Error("aborted"));
    }, { once: true });
  });
}

/**
 * fetch with the shared bounded-retry policy. String/undefined bodies are
 * replayed verbatim on retry; a streaming body makes the call one-shot
 * (it cannot be rewound).
 */
export async function fetchRetry(url: string, init: RequestInit = {}, opts: FetchRetryOptions = {}): Promise<Response> {
  const retries = opts.retries ?? 2;
  const base = opts.baseDelayMs ?? 250;
  const max = opts.maxDelayMs ?? 5000;
  const rand = opts.rand ?? Math.random;
  const method = (init.method ?? "GET").toUpperCase();
  const oneShot = init.body instanceof ReadableStream; // not replayable

  for (let attempt = 0; ; attempt++) {
    const res = await fetch(url, init);
    const retryable = res.status === 429 || (res.status >= 500 && IDEMPOTENT.has(method));
    if (!retryable || oneShot || attempt >= retries) return res;

    // Free the failed response's stream before waiting.
    await res.body?.cancel().catch(() => undefined);
    const delay = Math.min(max, base * 2 ** attempt) * (0.75 + 0.25 * rand());
    await sleep(delay, init.signal);
  }
}
