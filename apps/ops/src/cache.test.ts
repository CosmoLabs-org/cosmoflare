import { describe, expect, it } from "vitest";
import { cached } from "./cache";

// Module-scope L1 persists across tests, so every test uses its own key.

function fakeCtx(): { ctx: ExecutionContext; waits: Promise<unknown>[] } {
  const waits: Promise<unknown>[] = [];
  const ctx = {
    waitUntil: (p: Promise<unknown>): void => {
      waits.push(p);
    },
    passThroughOnException: (): void => {},
  } as unknown as ExecutionContext;
  return { ctx, waits };
}

// Map-backed fake KV (only what cache.ts uses).
function mapKV(initial?: Record<string, string>): { kv: KVNamespace; store: Map<string, string> } {
  const store = new Map(Object.entries(initial ?? {}));
  const kv = {
    get: async (k: string): Promise<string | null> => store.get(k) ?? null,
    put: async (k: string, v: string): Promise<void> => {
      store.set(k, v);
    },
  } as unknown as KVNamespace;
  return { kv, store };
}

describe("cached", () => {
  it("serves a fresh hit without reloading", async () => {
    const { ctx } = fakeCtx();
    let loads = 0;
    const opts = { ttlSec: 60, staleSec: 300 };
    const first = await cached(ctx, {}, "fresh-hit", opts, async () => {
      loads++;
      return "v1";
    });
    expect(first.value).toBe("v1");
    expect(first.stale).toBe(false);
    const second = await cached(ctx, {}, "fresh-hit", opts, async () => {
      loads++;
      return "v2";
    });
    expect(second.value).toBe("v1"); // fresh: served from L1, load not called
    expect(loads).toBe(1);
  });

  it("serves a stale entry and reloads in the background via ctx.waitUntil", async () => {
    const { ctx, waits } = fakeCtx();
    let clock = 1_000_000;
    let loads = 0;
    const opts = { ttlSec: 1, staleSec: 10, now: () => clock };
    const seed = await cached(ctx, {}, "swr", opts, async () => {
      loads++;
      return "v1";
    });
    expect(seed.stale).toBe(false);
    clock += 2 * 1000; // past the 1 s fresh TTL, inside the 10 s stale window
    const stale = await cached(ctx, {}, "swr", opts, async () => {
      loads++;
      return "v2";
    });
    expect(stale.value).toBe("v1"); // old value served immediately
    expect(stale.stale).toBe(true);
    await Promise.all(waits); // background refresh completes
    const after = await cached(ctx, {}, "swr", opts, async () => {
      loads++;
      return "v3";
    });
    expect(after.value).toBe("v2"); // background reload replaced the value
    expect(loads).toBe(2); // exactly one background reload happened
    expect(after.stale).toBe(false);
    expect(loads).toBe(2);
  });

  it("shares one load between two concurrent misses (single-flight)", async () => {
    const { ctx } = fakeCtx();
    let loads = 0;
    const opts = { ttlSec: 60, staleSec: 300 };
    const [a, b] = await Promise.all([
      cached(ctx, {}, "single-flight", opts, async () => {
        loads++;
        await new Promise((r) => setTimeout(r, 5));
        return "v1";
      }),
      cached(ctx, {}, "single-flight", opts, async () => {
        loads++;
        return "v2";
      }),
    ]);
    expect(a.value).toBe("v1");
    expect(b.value).toBe("v1");
    expect(loads).toBe(1);
  });

  it("keeps the stale value when the background refresh fails", async () => {
    const { ctx, waits } = fakeCtx();
    let clock = 1_000_000;
    const opts = { ttlSec: 1, staleSec: 10, now: () => clock };
    await cached(ctx, {}, "failed-refresh", opts, async () => "v1");
    clock += 2 * 1000; // stale
    const stale = await cached(ctx, {}, "failed-refresh", opts, async () => {
      throw new Error("upstream down");
    });
    expect(stale.value).toBe("v1"); // stale served, refresh kicked but failed
    expect(stale.stale).toBe(true);
    await Promise.all(waits); // must not reject out of waitUntil
    const again = await cached(ctx, {}, "failed-refresh", opts, async () => {
      throw new Error("upstream down");
    });
    expect(again.value).toBe("v1"); // failed refresh kept the stale value
    expect(again.stale).toBe(true);
  });

  it("allows only one forced reload per 60 s per key (refresh-storm guard)", async () => {
    const { ctx } = fakeCtx();
    let clock = 2_000_000;
    let loads = 0;
    const opts = { ttlSec: 60, staleSec: 300, now: () => clock };
    // First force: allowed, reloads (and starts the 60 s guard window).
    const f1 = await cached(ctx, {}, "force-guard", { ...opts, force: true }, async () => {
      loads++;
      return "v1";
    });
    expect(f1.value).toBe("v1");
    expect(f1.stale).toBe(false);
    clock += 30_000; // inside the 60 s guard window
    const f2 = await cached(ctx, {}, "force-guard", { ...opts, force: true }, async () => {
      loads++;
      return "v2";
    });
    expect(f2.value).toBe("v1"); // guard: returns the current value
    expect(loads).toBe(1);
    clock += 61_000; // guard window passed
    const f3 = await cached(ctx, {}, "force-guard", { ...opts, force: true }, async () => {
      loads++;
      return "v2";
    });
    expect(f3.value).toBe("v2");
    expect(loads).toBe(2);
  });

  it("reads through to a KV-backed L2 and writes back after a load", async () => {
    const { ctx } = fakeCtx();
    const { kv, store } = mapKV();
    let loads = 0;
    const opts = { ttlSec: 60, staleSec: 300, now: () => 5_000_000 };
    const miss = await cached(ctx, { OPS_KV: kv }, "kv-roundtrip", opts, async () => {
      loads++;
      return { n: 42 };
    });
    expect(miss.value).toEqual({ n: 42 });
    expect(loads).toBe(1);
    expect(store.has("kv-roundtrip")).toBe(true); // write-back happened
    // Prefill L2 directly, then read through on a key L1 has never seen:
    // the entry comes from KV without an upstream load.
    store.set("kv-prefilled", JSON.stringify({ storedAt: 5_000_000, value: "from-kv" }));
    const hit = await cached(ctx, { OPS_KV: kv }, "kv-prefilled", opts, async () => {
      loads++;
      return "unused";
    });
    expect(hit.value).toBe("from-kv");
    expect(loads).toBe(1); // L2 hit: no upstream load
  });

  it("cold start: L2 entry past ttl but inside stale serves stale:true and revalidates in the background", async () => {
    const { ctx, waits } = fakeCtx();
    const { kv, store } = mapKV();
    const t0 = 6_000_000;
    let loads = 0;
    const opts = { ttlSec: 60, staleSec: 300, now: () => t0 };
    // Cold L1: entry only in KV, aged 120 s (past the 60 s fresh TTL,
    // inside the 300 s stale window).
    store.set("l2-stale", JSON.stringify({ storedAt: t0 - 120_000, value: "from-kv-old" }));
    const res = await cached(ctx, { OPS_KV: kv }, "l2-stale", opts, async () => {
      loads++;
      return "from-load";
    });
    expect(res.value).toBe("from-kv-old"); // served immediately from L2
    expect(res.stale).toBe(true);
    await Promise.all(waits); // background revalidation completes
    expect(loads).toBe(1); // exactly one background load; no awaited load
    const after = await cached(ctx, { OPS_KV: kv }, "l2-stale", opts, async () => {
      loads++;
      return "unused";
    });
    expect(after.value).toBe("from-load"); // background reload replaced it
    expect(after.stale).toBe(false);
  });

  it("cold start: L2 entry older than the stale window is discarded and a real load is awaited", async () => {
    const { ctx } = fakeCtx();
    const { kv, store } = mapKV();
    const t0 = 7_000_000;
    let loads = 0;
    const opts = { ttlSec: 60, staleSec: 300, now: () => t0 };
    // Aged 400 s: past the 300 s stale window even though KV kept it.
    store.set("l2-expired", JSON.stringify({ storedAt: t0 - 400_000, value: "ancient" }));
    const res = await cached(ctx, { OPS_KV: kv }, "l2-expired", opts, async () => {
      loads++;
      return "from-load";
    });
    expect(res.value).toBe("from-load"); // awaited load, not the ancient value
    expect(res.stale).toBe(false);
    expect(loads).toBe(1);
  });

  it("a forced reload that fails falls back to the existing value marked stale", async () => {
    const { ctx } = fakeCtx();
    const t0 = 8_000_000;
    let clock = t0;
    const opts = { ttlSec: 60, staleSec: 300, now: () => clock };
    await cached(ctx, {}, "force-fallback", opts, async () => "v1");
    clock += 1_000;
    const forced = await cached(ctx, {}, "force-fallback", { ...opts, force: true }, async () => {
      throw new Error("upstream down");
    });
    expect(forced.value).toBe("v1"); // old value kept, not lost
    expect(forced.stale).toBe(true);
  });

  it("a forced reload that fails with no value at all rejects", async () => {
    const { ctx } = fakeCtx();
    const opts = { ttlSec: 60, staleSec: 300, now: () => 9_000_000 };
    await expect(
      cached(ctx, {}, "force-no-fallback", { ...opts, force: true }, async () => {
        throw new Error("upstream down");
      }),
    ).rejects.toThrow("upstream down");
  });
});
