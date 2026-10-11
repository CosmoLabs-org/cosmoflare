import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient, LoginExpiredError, totalAgeSec, type CacheInfo } from "./api";
import type { RulesPayload } from "./rules";

// Node-environment tests (no DOM, no sessionStorage): every client gets an
// injected fetch mock and an in-memory storage stand-in, matching the
// constructor options used by the views in production.

function memoryStorage(): Pick<Storage, "getItem" | "setItem" | "removeItem"> {
  const map = new Map<string, string>();
  return {
    getItem: (k) => map.get(k) ?? null,
    setItem: (k, v) => void map.set(k, v),
    removeItem: (k) => void map.delete(k),
  };
}

// localStorage stand-in for the persisted last-good store (FEAT-pRXPMMY).
function localMemoryStorage(): Pick<Storage, "getItem" | "setItem" | "removeItem"> {
  const map = new Map<string, string>();
  return {
    getItem: (k) => map.get(k) ?? null,
    setItem: (k, v) => void map.set(k, v),
    removeItem: (k) => void map.delete(k),
  };
}

const RULES: RulesPayload = {
  rules: [{ name: "D1 reads", condition: "d1-rows-read", threshold: 1e9, enabled: true }],
  conditions: ["d1-rows-read"],
  starter: true,
};

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("totalAgeSec (BUG-057 defect 2)", () => {
  it("adds the server cache age to the client copy age", () => {
    const cache: CacheInfo = { ageSec: 1800, stale: false };
    expect(totalAgeSec(600, cache)).toBe(2400);
  });

  it("is the client copy age when the response carries no cache metadata", () => {
    expect(totalAgeSec(45)).toBe(45);
    expect(totalAgeSec(45, undefined)).toBe(45);
  });

  it("never reports a fresh age for hours-old data", () => {
    // 2h client copy of a 1h server-cached response: 3h, not 0.
    expect(totalAgeSec(2 * 3600, { ageSec: 3600, stale: true })).toBe(3 * 3600);
  });
});

describe("ApiClient.updateCache (BUG-057 defect 1)", () => {
  it("serves the saved rules from the cache without a network call after a PUT", async () => {
    const fetchMock = vi.fn(async () => jsonResponse(RULES));
    const client = new ApiClient({ fetchFn: fetchMock, storage: memoryStorage() });

    const first = await client.fetchJson<RulesPayload>("api/rules");
    expect(first.data.rules).toHaveLength(1);
    expect(fetchMock).toHaveBeenCalledTimes(1);

    // The save handler's cache write after a successful PUT.
    const saved: RulesPayload = {
      rules: [{ name: "Saved", condition: "d1-rows-read", threshold: 5, enabled: true }],
      conditions: RULES.conditions,
      starter: false,
    };
    client.updateCache("api/rules", saved);

    // Immediately, after 60s (route re-entry), and via a second client
    // backed by the same sessionStorage (reload): the saved set, no refetch.
    const again = await client.fetchJson<RulesPayload>("api/rules");
    expect(again.data.rules[0]?.name).toBe("Saved");
    expect(again.data.starter).toBe(false);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("survives a reload through sessionStorage", async () => {
    const storage = memoryStorage();
    const fetchMock = vi.fn(async () => jsonResponse(RULES));
    const first = new ApiClient({ fetchFn: fetchMock, storage });
    await first.fetchJson<RulesPayload>("api/rules");

    const saved: RulesPayload = { ...RULES, starter: false };
    first.updateCache("api/rules", saved);

    // "Reload": a brand-new client sharing the same sessionStorage.
    const second = new ApiClient({ fetchFn: fetchMock, storage });
    const res = await second.fetchJson<RulesPayload>("api/rules");
    expect(res.source).toBe("session");
    expect(res.data.starter).toBe(false);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("a second Save cannot PUT the stale pre-save rules back", async () => {
    const storage = memoryStorage();
    const fetchMock = vi.fn(async () => jsonResponse(RULES));
    const client = new ApiClient({ fetchFn: fetchMock, storage });
    await client.fetchJson<RulesPayload>("api/rules");

    const saved: RulesPayload = {
      rules: [{ name: "v2", condition: "d1-rows-read", threshold: 7, enabled: false }],
      conditions: RULES.conditions,
      starter: false,
    };
    client.updateCache("api/rules", saved);
    const res = await client.fetchJson<RulesPayload>("api/rules");
    // The draft set the next Save PUTs comes from the cache: it must be the
    // echo of the first save, not the pre-save rules.
    expect(res.data.rules).toEqual(saved.rules);
  });
});

describe("ApiClient.fetchJson onRevalidate (BUG-057 defect 2)", () => {
  it("renders the stale cached copy instantly, then revalidates and calls onRevalidate with fresh data", async () => {
    let clock = 1_000_000;
    const fresh: RulesPayload = { rules: [{ name: "fresh", condition: "d1-rows-read", threshold: 1, enabled: true }], conditions: RULES.conditions, starter: false };
    const fetchMock = vi.fn(async () => jsonResponse(clock === 1_000_000 ? RULES : fresh));
    const client = new ApiClient({
      fetchFn: fetchMock,
      storage: memoryStorage(),
      now: () => clock,
    });

    await client.fetchJson<RulesPayload>("api/rules");
    clock += 120_000; // past the 60s freshness window

    const onRevalidate = vi.fn();
    const res = await client.fetchJson<RulesPayload>("api/rules", { onRevalidate });
    expect(res.source).toBe("memory");
    expect(res.ageSec).toBe(120);
    expect(res.data.rules[0]?.name).toBe("D1 reads"); // instant cached paint

    // Let the background revalidation settle, then the view repaints.
    await vi.waitFor(() => expect(onRevalidate).toHaveBeenCalledTimes(1));
    expect(onRevalidate).toHaveBeenCalledWith(fresh);
  });

  it("does not call onRevalidate when the cached copy is still fresh", async () => {
    let clock = 1_000_000;
    const fetchMock = vi.fn(async () => jsonResponse(RULES));
    const client = new ApiClient({ fetchFn: fetchMock, storage: memoryStorage(), now: () => clock });
    await client.fetchJson<RulesPayload>("api/rules");
    clock += 10_000;
    const onRevalidate = vi.fn();
    const res = await client.fetchJson<RulesPayload>("api/rules", { onRevalidate });
    expect(onRevalidate).not.toHaveBeenCalled();
    expect(res.ageSec).toBe(10);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});

describe("ApiClient persisted last-good store (FEAT-pRXPMMY)", () => {
  it("serves the persisted copy from a fresh launch with source 'local' and its real age, then revalidates in the background", async () => {
    let clock = 1_000_000;
    const fresh: RulesPayload = { rules: [{ name: "fresh", condition: "d1-rows-read", threshold: 1, enabled: true }], conditions: RULES.conditions, starter: false };
    const fetchMock = vi.fn(async () => jsonResponse(clock === 1_000_000 ? RULES : fresh));
    const localStore = localMemoryStorage();

    // Last session: a successful fetch persists the copy to localStorage.
    const previous = new ApiClient({ fetchFn: fetchMock, storage: memoryStorage(), localStore, now: () => clock });
    await previous.fetchJson<RulesPayload>("api/rules");

    // Fresh app launch: new client, empty memory AND empty sessionStorage,
    // same persisted localStorage — the offline-instant-paint path.
    clock += 3_600_000; // one hour later; the persisted copy is stale
    const launched = new ApiClient({ fetchFn: fetchMock, storage: memoryStorage(), localStore, now: () => clock });
    const onRevalidate = vi.fn();
    const res = await launched.fetchJson<RulesPayload>("api/rules", { onRevalidate });

    expect(res.source).toBe("local");
    expect(res.ageSec).toBe(3600); // real persisted age, not a fake fresh 0
    expect(res.data.rules[0]?.name).toBe("D1 reads"); // last session's data paints instantly

    // The background revalidate settles and hands the view fresh data.
    await vi.waitFor(() => expect(onRevalidate).toHaveBeenCalledTimes(1));
    expect(onRevalidate).toHaveBeenCalledWith(fresh);
    // 1 initial-session fetch + 1 stale-path background revalidate.
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("prefers memory and sessionStorage over the persisted copy", async () => {
    let clock = 1_000_000;
    const fetchMock = vi.fn(async () => jsonResponse(RULES));
    const session = memoryStorage();
    const localStore = localMemoryStorage();

    const first = new ApiClient({ fetchFn: fetchMock, storage: session, localStore, now: () => clock });
    await first.fetchJson<RulesPayload>("api/rules"); // fills all three stores

    // A reload-sharing client (same sessionStorage + localStore) hits the
    // session tier, not the persisted one.
    const reloaded = new ApiClient({ fetchFn: fetchMock, storage: session, localStore, now: () => clock });
    const res = await reloaded.fetchJson<RulesPayload>("api/rules");
    expect(res.source).toBe("session");
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("localStorage quota failure is non-fatal: memory and session still hold the copy", async () => {
    let clock = 1_000_000;
    const fetchMock = vi.fn(async () => jsonResponse(RULES));
    const fullLocalStore: Pick<Storage, "getItem" | "setItem" | "removeItem"> = {
      getItem: () => null,
      setItem: () => {
        throw new Error("QuotaExceededError");
      },
      removeItem: () => undefined,
    };
    const client = new ApiClient({ fetchFn: fetchMock, storage: memoryStorage(), localStore: fullLocalStore, now: () => clock });

    // The store() write-through must swallow the quota throw...
    const res = await client.fetchJson<RulesPayload>("api/rules");
    expect(res.source).toBe("network");
    expect(fetchMock).toHaveBeenCalledTimes(1);

    // ...and the copy is still served afterwards (memory/session survive).
    clock += 10_000;
    const again = await client.fetchJson<RulesPayload>("api/rules");
    expect(again.source).toBe("memory");
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("budget logic is intact: a stale persisted copy with a spent budget is served with no network call", async () => {
    let clock = 1_000_000;
    const fetchMock = vi.fn(async () => {
      throw new Error("network down");
    });
    const localStore = localMemoryStorage();

    // Last session persists the copy.
    const good = new ApiClient({ fetchFn: async () => jsonResponse(RULES), storage: memoryStorage(), localStore, now: () => clock });
    await good.fetchJson<RulesPayload>("api/rules");

    // Fresh launch an hour later: the persisted copy is stale...
    clock = 1_060_000;
    const client = new ApiClient({ fetchFn: fetchMock, storage: memoryStorage(), localStore, now: () => clock });
    // ...and 6 failed forced refreshes burn the whole budget.
    for (let i = 0; i < 6; i++) {
      await expect(client.fetchJson<RulesPayload>("api/rules", { refresh: true })).rejects.toThrow("network down");
    }
    expect(fetchMock).toHaveBeenCalledTimes(6);

    // Plain navigation now: the persisted copy renders at its real age with
    // no 7th network attempt — the BUG-pZKJ5VB stale-fallback covers the
    // persisted tier too.
    const res = await client.fetchJson<RulesPayload>("api/rules");
    expect(res.source).toBe("local");
    expect(res.ageSec).toBe(60);
    expect(fetchMock).toHaveBeenCalledTimes(6);
  });
});

describe("ApiClient network fetch budget (BUG-pZKJ5VB)", () => {
  // Forced (?refresh) fetches always bypass the cache, so they are the
  // cleanest way to fire repeated *uncached* network attempts against one
  // endpoint inside a single client (one shared budget).

  it("rejects the 7th network attempt in 60s with a plain budget error", async () => {
    let clock = 1_000_000;
    const fetchMock = vi.fn(async () => jsonResponse(RULES));
    const client = new ApiClient({ fetchFn: fetchMock, storage: memoryStorage(), now: () => clock });

    for (let i = 0; i < 6; i++) {
      const res = await client.fetchJson<RulesPayload>("api/rules", { refresh: true });
      expect(res.source).toBe("network");
    }
    expect(fetchMock).toHaveBeenCalledTimes(6);

    let caught: unknown;
    try {
      await client.fetchJson<RulesPayload>("api/rules", { refresh: true });
    } catch (err) {
      caught = err;
    }
    expect(caught).toBeInstanceOf(Error);
    expect((caught as Error).message).toBe("fetch budget exceeded for api/rules — retry in a moment");
    // A spent budget is not an expired login.
    expect(caught).not.toBeInstanceOf(LoginExpiredError);
    expect(fetchMock).toHaveBeenCalledTimes(6); // the 7th never left
  });

  it("serves the stale cached copy without a network call when the budget is exhausted", async () => {
    let clock = 1_000_000;
    // Every network attempt fails, so nothing ever overwrites the cache and
    // each failed attempt still burns a real budget slot.
    const fetchMock = vi.fn(async () => {
      throw new Error("network down");
    });
    const client = new ApiClient({ fetchFn: fetchMock, storage: memoryStorage(), now: () => clock });

    // Cached copy present without spending budget (the post-PUT write-through).
    client.updateCache("api/rules", RULES);

    // 50s later: 6 failed forced refreshes exhaust the budget...
    clock = 1_050_000;
    for (let i = 0; i < 6; i++) {
      await expect(client.fetchJson<RulesPayload>("api/rules", { refresh: true })).rejects.toThrow("network down");
    }
    expect(fetchMock).toHaveBeenCalledTimes(6);

    // ...20s after that the copy is 70s old (past FRESH_MS) but the budget
    // attempts are only 20s into their 60s window, so plain navigation
    // serves the stale copy at its real age with no network call at all.
    clock = 1_070_000;
    const res = await client.fetchJson<RulesPayload>("api/rules");
    expect(res.source).toBe("memory");
    expect(res.ageSec).toBe(70); // real age, not a fresh 0
    expect(res.data.rules).toEqual(RULES.rules);
    expect(fetchMock).toHaveBeenCalledTimes(6); // no 7th network call
  });

  it("falls back to the sessionStorage copy when memory is empty and the budget is spent", async () => {
    let clock = 1_000_000;
    const storage = memoryStorage();
    const good = new ApiClient({ fetchFn: async () => jsonResponse(RULES), storage, now: () => clock });
    await good.fetchJson<RulesPayload>("api/rules"); // fills sessionStorage

    // A fresh client sharing that storage whose network always fails:
    // 6 failed attempts burn the budget (they are real network attempts).
    const failing = vi.fn(async () => {
      throw new Error("network down");
    });
    const client = new ApiClient({ fetchFn: failing, storage, now: () => clock });
    clock = 1_050_000;
    for (let i = 0; i < 6; i++) {
      await expect(client.fetchJson<RulesPayload>("api/rules", { refresh: true })).rejects.toThrow("network down");
    }

    clock = 1_070_000; // session copy stale, budget window still live
    const res = await client.fetchJson<RulesPayload>("api/rules");
    expect(res.source).toBe("session");
    expect(res.ageSec).toBe(70);
    expect(failing).toHaveBeenCalledTimes(6);
  });

  it("resets once the 60s window slides (61s later a fetch succeeds again)", async () => {
    let clock = 1_000_000;
    const fetchMock = vi.fn(async () => jsonResponse(RULES));
    const client = new ApiClient({ fetchFn: fetchMock, storage: memoryStorage(), now: () => clock });

    for (let i = 0; i < 6; i++) {
      await client.fetchJson<RulesPayload>("api/rules", { refresh: true });
    }
    await expect(client.fetchJson<RulesPayload>("api/rules", { refresh: true })).rejects.toThrow(
      "fetch budget exceeded",
    );

    clock += 61_000; // every attempt just aged out of the window
    const res = await client.fetchJson<RulesPayload>("api/rules", { refresh: true });
    expect(res.source).toBe("network");
    expect(fetchMock).toHaveBeenCalledTimes(7);
  });

  it("tracks budgets per endpoint independently", async () => {
    let clock = 1_000_000;
    const fetchMock = vi.fn(async () => jsonResponse(RULES));
    const client = new ApiClient({ fetchFn: fetchMock, storage: memoryStorage(), now: () => clock });

    for (let i = 0; i < 6; i++) {
      await client.fetchJson<RulesPayload>("api/summary", { refresh: true });
    }
    await expect(client.fetchJson<RulesPayload>("api/summary", { refresh: true })).rejects.toThrow(
      "fetch budget exceeded for api/summary",
    );

    // A different endpoint never touched its own budget.
    const res = await client.fetchJson<RulesPayload>("api/rules", { refresh: true });
    expect(res.source).toBe("network");
  });

  it("cache hits never consume budget", async () => {
    let clock = 1_000_000;
    const fetchMock = vi.fn(async () => jsonResponse(RULES));
    const client = new ApiClient({ fetchFn: fetchMock, storage: memoryStorage(), now: () => clock });

    // One real network attempt...
    await client.fetchJson<RulesPayload>("api/rules");
    expect(fetchMock).toHaveBeenCalledTimes(1);

    // ...then 7 instant cache hits (still inside FRESH_MS)...
    for (let i = 0; i < 7; i++) {
      const res = await client.fetchJson<RulesPayload>("api/rules");
      expect(res.source).toBe("memory");
    }
    expect(fetchMock).toHaveBeenCalledTimes(1);

    // ...and a forced network fetch is still allowed (attempt #2 of 6).
    const res = await client.fetchJson<RulesPayload>("api/rules", { refresh: true });
    expect(res.source).toBe("network");
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
