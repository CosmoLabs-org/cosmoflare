// @vitest-environment jsdom
// D1View (P-05a): the fixture D1 databases paint rows-read-desc first,
// graded usage bars ride the d1.rows_read allowance, and the level row
// classes follow dashboard.ts levelForD1.
import { cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
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
});
