// @vitest-environment jsdom
// ZonesView (P-05a): the fixture zones paint with the stacked byStatus
// bars (reuse of tables.ts's byStatusStack through the DOM slot), the
// aggregated shared legend, and the zone attention row classes.
import { cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import ZonesView from "./ZonesView";

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

describe("ZonesView (P-05a)", () => {
  it("paints the fixture zones with stacked byStatus bars and the shared legend", async () => {
    render(<ZonesView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-table")).toBeTruthy();
    });
    // Two fixture zones (both total > 0).
    expect(document.querySelectorAll(".cf-table tbody tr").length).toBe(2);
    // Stacked status bars render segments per zone (desktop cells + mobile rows).
    expect(document.querySelectorAll(".cf-zonestack .cf-seg").length).toBeGreaterThan(0);
    expect(document.querySelectorAll(".cf-zonestack .cf-stack-legend").length).toBeGreaterThan(0);
    // One aggregated legend for phones, shown under the filter/sort bar.
    expect(document.querySelectorAll(".cf-shared-legend").length).toBe(1);
    expect(document.querySelector(".cf-shared-legend")?.textContent).toContain("hit");
    expect(document.querySelector(".cf-card h2")).toBeNull();
    expect(document.querySelector(".cf-card .cf-row-detail")?.textContent).toBe("(24h)");
  });
});
