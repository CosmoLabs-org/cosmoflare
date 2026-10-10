// @vitest-environment jsdom
// DurableObjectsView (P-05a): /api/durable-objects has no fixture, so in
// jsdom the fetch throws and the view paints its error state (fixture
// honesty); with an injected fetcher (the worker_profile.test.ts pattern)
// the data-present table paints with graded bars from the do.requests
// allowance — and the bars drop when the billing fetch fails.
import { cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import type { Billing, FetchResult } from "../../api";
import type { DOPayload } from "../../durable_objects";
import DurableObjectsView from "./DurableObjectsView";

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

const PAYLOAD: DOPayload = {
  generatedAt: "2026-10-10T00:00:00Z",
  namespaces: 2,
  rows: [
    { namespaceId: "a", script: "edge-cache", requests: 900_000, errors: 4_500, errorPct: 0.5 },
    { namespaceId: "b", script: "rate-limiter", requests: 120_000, errors: 0, errorPct: 0 },
  ],
  errors: [],
};

function fetcherReturning(doRes: FetchResult<DOPayload> | Error, billing?: Billing): Parameters<typeof DurableObjectsView>[0]["fetcher"] {
  return async <T,>(endpoint: string): Promise<FetchResult<T>> => {
    if (endpoint.startsWith("api/durable-objects")) {
      if (doRes instanceof Error) throw doRes;
      return doRes as unknown as FetchResult<T>;
    }
    if (endpoint.startsWith("api/billing") && billing) {
      return { data: billing as unknown as T, source: "network", ageSec: 0, demo: false };
    }
    throw new Error("HTTP 500");
  };
}

const BILLING = {
  generatedAt: "2026-10-10T00:00:00Z",
  period: { start: "", end: "", day: 10, days: 30, source: "calendar" as const },
  products: [
    { id: "do.requests", product: "Durable Objects", metric: "Requests", unit: "requests", included: 1_000_000, used: 1_020_000, projected: 1_020_000, unitPriceUsd: 0.15, priceUnit: 1_000_000, projectedOverageUsd: 0, topConsumers: [] },
  ],
  totalProjectedOverageUsd: 0,
  projects: [],
  pricing: { verifiedOn: "2026-10-01", sources: [] },
  errors: [],
} satisfies Billing;

describe("DurableObjectsView (P-05a)", () => {
  it("paints the error card when the endpoint has no fixture and fails", async () => {
    render(<DurableObjectsView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-card.cf-level-warning")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card h2")?.textContent).toBe("Could not load Durable Objects");
    // Retry offered.
    expect(document.querySelector(".cf-card .cf-btn")?.textContent).toBe("Retry");
  });

  it("paints the namespace table with graded bars from the do.requests allowance", async () => {
    render(
      <DurableObjectsView
        fetcher={fetcherReturning({ data: PAYLOAD, source: "network", ageSec: 0, demo: false }, BILLING)}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-table")).toBeTruthy();
    });
    expect(document.querySelectorAll(".cf-table tbody tr").length).toBe(2);
    expect(document.querySelectorAll(".cf-mobile-list .cf-mrow-top").length).toBe(2);
    // Requests desc → edge-cache (900k) leads.
    expect(document.querySelector(".cf-table tbody tr td")?.textContent).toContain("edge-cache");
    expect(document.querySelectorAll(".cf-mini-bar").length).toBeGreaterThan(0);
    // Subtitle carries the namespace count; no page h2.
    expect(document.querySelector(".cf-card h2")).toBeNull();
    expect(document.querySelector(".cf-card .cf-row-detail")?.textContent).toContain("2 namespaces");
  });

  it("drops the usage bars when the billing fetch fails", async () => {
    render(
      <DurableObjectsView fetcher={fetcherReturning({ data: PAYLOAD, source: "network", ageSec: 0, demo: false })} />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-table")).toBeTruthy();
    });
    expect(document.querySelectorAll(".cf-mini-bar").length).toBe(0);
  });

  it("surfaces a partial load in the subtitle when the payload carries errors", async () => {
    const partial: DOPayload = { ...PAYLOAD, errors: ["zone x telemetry missing"] };
    render(<DurableObjectsView fetcher={fetcherReturning({ data: partial, source: "network", ageSec: 0, demo: false })} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-table")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card .cf-row-detail")?.textContent).toContain("Partial load — zone x telemetry missing");
  });
});
