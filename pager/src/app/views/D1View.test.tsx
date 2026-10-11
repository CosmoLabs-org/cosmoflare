// @vitest-environment jsdom
// D1View (P-05a): the fixture D1 databases paint rows-read-desc first,
// graded usage bars ride the d1.rows_read allowance, and the level row
// classes follow dashboard.ts levelForD1. The first-load skeleton, the
// quiet empty state and the Retry/recover error state run against an
// injected fetcher (the worker_profile.test.ts pattern).
import { cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import type { FetchResult, Summary } from "../../api";
import D1View from "./D1View";

// jsdom (as vitest configures it) exposes no localStorage; the sort
// preference helpers default to window.localStorage, so install a Map
// stand-in before any view mounts.
const sortPrefStore: { map: Map<string, string>; clear(): void } = (() => {
  const store = {
    map: new Map<string, string>(),
    clear(): void {
      store.map.clear();
    },
    getItem(key: string): string | null {
      return store.map.get(key) ?? null;
    },
    setItem(key: string, value: string): void {
      store.map.set(key, value);
    },
    removeItem(key: string): void {
      store.map.delete(key);
    },
  };
  Object.defineProperty(window, "localStorage", { value: store, configurable: true, writable: true });
  return store;
})();

afterEach(cleanup);
beforeEach(() => {
  sortPrefStore.clear();
});

const EMPTY_SUMMARY: Summary = {
  generatedAt: "2026-10-10T00:00:00Z",
  windowHours: 24,
  usage: [],
  d1: [],
  zones: [],
  workers: [],
  errors: [],
};

/** Injected fetcher: answers api/summary with the given payload; the
 *  billing side-fetch (usage-bar allowance) soft-fails like production. */
function summaryFetcher(summary: Summary, opts: { throwErr?: Error } = {}): Parameters<typeof D1View>[0]["fetcher"] {
  return async <T,>(endpoint: string): Promise<FetchResult<T>> => {
    if (opts.throwErr) throw opts.throwErr;
    if (endpoint === "api/summary") return { data: summary as unknown as T, source: "network", ageSec: 0, demo: false };
    throw new Error("HTTP 404");
  };
}

/** Skeleton probe: the api/summary fetch hangs until go() releases it. */
function gatedSummaryFetcher(summary: Summary): { fetcher: Parameters<typeof D1View>[0]["fetcher"]; go: () => void } {
  let release!: (r: FetchResult<Summary>) => void;
  const gate = new Promise<FetchResult<Summary>>((res) => {
    release = res;
  });
  const fetcher: Parameters<typeof D1View>[0]["fetcher"] = async <T,>(endpoint: string): Promise<FetchResult<T>> => {
    if (endpoint !== "api/summary") throw new Error("HTTP 404");
    return gate as unknown as Promise<FetchResult<T>>;
  };
  return {
    fetcher,
    go: () => release({ data: summary, source: "network", ageSec: 0, demo: false }),
  };
}

/** Fails every api/summary call while armed; after setFail(false) it answers. */
function recoveringSummaryFetcher(summary: Summary): { fetcher: Parameters<typeof D1View>[0]["fetcher"]; setFail: (fail: boolean) => void } {
  let fail = true;
  const fetcher: Parameters<typeof D1View>[0]["fetcher"] = async <T,>(endpoint: string): Promise<FetchResult<T>> => {
    if (endpoint !== "api/summary") throw new Error("HTTP 404");
    if (fail) throw new Error("HTTP 503");
    return { data: summary as unknown as T, source: "network", ageSec: 0, demo: false };
  };
  return { fetcher, setFail: (next) => {
    fail = next;
  } };
}

describe("D1View (P-05a)", () => {
  it("paints the fixture databases sorted by rows read desc", async () => {
    render(<D1View />);
    await waitFor(() => {
      expect(document.querySelector(".cf-table")).toBeTruthy();
    });
    const scripts = Array.from(document.querySelectorAll(".cf-table tbody tr td:first-child")).map(
      (td) => td.textContent,
    );
    // Fixture rowsRead: cf-state 412M, domains 180M, usage-ledger 47M,
    // cosmokit-registry 1.1M — desc order.
    expect(scripts).toEqual(["cf-state", "domains", "usage-ledger", "cosmokit-registry"]);
    expect(document.querySelectorAll(".cf-mobile-list .cf-mrow-top").length).toBe(4);
    expect(document.querySelector('.cf-table th[aria-sort="descending"]')?.textContent).toContain("Rows read");
  });

  it("renders graded usage bars and the (24h) subtitle, no page h2", async () => {
    render(<D1View />);
    await waitFor(() => {
      expect(document.querySelector(".cf-mini-bar")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card h2")).toBeNull();
    expect(document.querySelector(".cf-card .cf-row-detail")?.textContent).toBe("(24h)");
  });

  it("renders the skeleton while the first fetch is in flight", async () => {
    const gated = gatedSummaryFetcher(EMPTY_SUMMARY);
    render(<D1View fetcher={gated.fetcher} />);
    expect(document.querySelector(".cf-loading")).toBeTruthy();
    expect(document.querySelectorAll(".cf-skeleton").length).toBe(4);
    expect(document.querySelector(".cf-card")).toBeNull();
    gated.go();
    await waitFor(() => {
      expect(document.querySelector(".cf-card")).toBeTruthy();
    });
    expect(document.querySelector(".cf-loading")).toBeNull();
  });

  it("paints the quiet empty state when no D1 databases have activity", async () => {
    render(<D1View fetcher={summaryFetcher(EMPTY_SUMMARY)} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-empty-quiet")).toBeTruthy();
    });
    expect(document.querySelector(".cf-empty-quiet")?.textContent).toBe("No D1 activity in the window.");
    expect(document.querySelector(".cf-table")).toBeNull();
  });

  it("shows the Retry card on a failed first load and Retry recovers", async () => {
    const recovering = recoveringSummaryFetcher(EMPTY_SUMMARY);
    render(<D1View fetcher={recovering.fetcher} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-card.cf-level-warning")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card h2")?.textContent).toBe("Could not load the D1 list");
    expect(document.querySelector(".cf-card .cf-btn")?.textContent).toBe("Retry");
    recovering.setFail(false);
    document.querySelector<HTMLButtonElement>(".cf-card .cf-btn")!.click();
    await waitFor(() => {
      expect(document.querySelector(".cf-empty-quiet")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card.cf-level-warning")).toBeNull();
  });
});
