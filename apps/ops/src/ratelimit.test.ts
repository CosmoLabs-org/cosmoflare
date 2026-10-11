// Cosmoflare Ops rate-limit guard tests (FEAT-pT0AFDM / BUG-pZKJ5VB): the
// pure, clock-injectable core — per-IP budget, window slide, per-IP
// independence, and the 2-window prune that bounds the map. No fake timers:
// every test drives the injected `nowMs` directly, in the billing.test.ts
// style (plain describe/it/expect, no setup/teardown).

import { describe, expect, it } from "vitest";
import { MAX_REQUESTS, WINDOW_MS, allowed, type Bucket } from "./ratelimit";

const fresh = (): Map<string, Bucket> => new Map();

describe("rate limit core (allowed)", () => {
  it("allows MAX_REQUESTS requests in one window and rejects the next with retryAfterSec in (0, 60]", () => {
    const map = fresh();
    for (let i = 0; i < MAX_REQUESTS; i++) {
      expect(allowed(map, "1.2.3.4", i).ok).toBe(true); // 0..119 ms — inside the window
    }
    const verdict = allowed(map, "1.2.3.4", 120);
    expect(verdict.ok).toBe(false);
    expect(verdict.retryAfterSec).toBeGreaterThan(0);
    expect(verdict.retryAfterSec).toBeLessThanOrEqual(WINDOW_MS / 1000);
  });

  it("slides the window: advancing the clock 61 s resets the count", () => {
    const map = fresh();
    for (let i = 0; i < MAX_REQUESTS; i++) allowed(map, "1.2.3.4", 0);
    expect(allowed(map, "1.2.3.4", 1).ok).toBe(false); // budget exhausted
    expect(allowed(map, "1.2.3.4", 61_000).ok).toBe(true); // fresh window
  });

  it("gives two IPs independent budgets", () => {
    const map = fresh();
    for (let i = 0; i < MAX_REQUESTS; i++) allowed(map, "1.1.1.1", 0);
    expect(allowed(map, "1.1.1.1", 1).ok).toBe(false);
    expect(allowed(map, "2.2.2.2", 1).ok).toBe(true);
  });

  it("prunes entries older than two windows so the map stays bounded", () => {
    const map = fresh();
    for (let i = 0; i < 500; i++) allowed(map, `10.0.0.${i % 256}.${Math.floor(i / 256)}`, 0);
    expect(map.size).toBe(500);
    // One fresh check 121 s later sweeps every stale bucket (2 windows = 120 s).
    expect(allowed(map, "3.3.3.3", 121_000).ok).toBe(true);
    expect(map.size).toBe(1);
  });
});
