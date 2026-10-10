// @vitest-environment jsdom
// WorkersView (P-05a): the React table family paints the fixture Workers
// rows (jsdom on localhost → the ApiClient falls back to fixtures), whole
// rows tap through to the profile route, the name cell is a link, graded
// usage bars ride on the cached billing allowance, and the sort/filter/
// collapse controls work — the same behaviors tables.ts's renderWorkers
// offered through the bridge.
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import WorkersView from "./WorkersView";

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
  window.location.hash = "";
});

describe("WorkersView (P-05a)", () => {
  it("paints the fixture workers into the sortable table + mobile list", async () => {
    render(<WorkersView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-table")).toBeTruthy();
    });
    // 3 fixture workers, under the collapse cap of 10.
    const bodyRows = document.querySelectorAll(".cf-table tbody tr");
    expect(bodyRows.length).toBe(3);
    // The mobile stacked list mirrors the same rows.
    expect(document.querySelectorAll(".cf-mobile-list .cf-mrow-top").length).toBe(3);
    // Six sortable header buttons; the initial sort (requests, desc) is
    // announced via aria-sort with the active glyph.
    expect(document.querySelectorAll(".cf-sort-btn").length).toBe(6);
    expect(document.querySelector('.cf-table th[aria-sort="descending"]')).toBeTruthy();
    expect(document.querySelector(".cf-sort-icon.is-active")).toBeTruthy();
    // No "(24h)" h2 — the masthead owns the title; the card subtitle stays.
    expect(document.querySelector(".cf-card h2")).toBeNull();
    expect(document.querySelector(".cf-card .cf-row-detail")?.textContent).toBe("(24h)");
  });

  it("links the name cell and the whole row to the worker profile route", async () => {
    render(<WorkersView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-table tbody tr")).toBeTruthy();
    });
    const link = document.querySelector<HTMLAnchorElement>(".cf-table tbody .cf-link");
    expect(link?.getAttribute("href")).toContain("#/worker/api-gateway");
    // Whole-row tap-through: every body row carries cf-row-link.
    expect(document.querySelectorAll(".cf-table tbody tr.cf-row-link").length).toBe(3);
    const firstRow = document.querySelector<HTMLTableRowElement>(".cf-table tbody tr");
    expect(firstRow?.textContent).toContain("api-gateway");
    fireEvent.click(firstRow!);
    expect(window.location.hash).toBe("#/worker/api-gateway");
  });

  it("re-sorts from the header buttons and persists the preference", async () => {
    render(<WorkersView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-table tbody tr")).toBeTruthy();
    });
    // Fixture errors: api-gateway 620, ccs-state 1800, cdn 0 — a numeric
    // header defaults to descending, so ccs-state leads.
    fireEvent.click(screen.getByRole("button", { name: /Errors/ }));
    const firstScript = document.querySelector(".cf-table tbody tr td")?.textContent;
    expect(firstScript).toContain("ccs-state");
    expect(document.querySelector('.cf-table th[aria-sort="descending"]')?.textContent).toContain("Errors");
    // The preference round-trips through storage (FEAT-pRC6EDA).
    expect(JSON.parse(sortPrefStore.map.get("cf-sort:workers")!)).toEqual({ key: "errors", dir: "desc" });
  });

  it("filters rows case-insensitively across the columns", async () => {
    render(<WorkersView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-table tbody tr")).toBeTruthy();
    });
    const filter = document.querySelector<HTMLInputElement>(".cf-filter");
    fireEvent.input(filter!, { target: { value: "cdn" } });
    expect(document.querySelectorAll(".cf-table tbody tr").length).toBe(1);
    expect(document.querySelector(".cf-table tbody")?.textContent).toContain("cdn");
  });

  it("shows graded usage bars from the cached billing allowance", async () => {
    render(<WorkersView />);
    // The billing fixture carries workers.requests included=10M, so every
    // row gets a mini usage bar (table + mobile mirror).
    await waitFor(() => {
      expect(document.querySelectorAll(".cf-mini-bar").length).toBeGreaterThan(0);
    });
  });
});
