import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient, totalAgeSec, type CacheInfo } from "./api";
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
