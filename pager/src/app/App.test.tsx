// @vitest-environment jsdom
// React shell smoke (P-03): the App mounts, paints the shell chrome (brand,
// quick nav, status strip), and the imperative bridge renders a view — on
// jsdom's localhost the ApiClient falls back to fixtures, so the Overview
// paints fixture data through the same path production uses.
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import { api, type FetchResult } from "../api";
import { ROUTES } from "../routes";
import type { DomainsPayload } from "../domains";

afterEach(cleanup);

const MOCK_DOMAINS: DomainsPayload = {
  generatedAt: "2026-10-11T00:00:00.000Z",
  domains: [
    { id: "d1", name: "cosmolabs.org", status: "active", paused: false, expiresAt: "2027-01-01T00:00:00.000Z" },
    { id: "d2", name: "cosmoflare.dev", status: "active", paused: false, expiresAt: "2027-02-01T00:00:00.000Z" },
    { id: "d3", name: "example.test", status: "active", paused: false, expiresAt: null },
  ],
  errors: [],
};

describe("App shell (P-03)", () => {
  it("mounts the chrome: brand lockup (with live section name), quick nav, status strip", () => {
    render(<App />);
    expect(screen.getAllByText(/CosmoLabs Ops/).length).toBeGreaterThan(0);
    expect(screen.getAllByText("Cosmoflare").length).toBeGreaterThan(0);
    expect(screen.getAllByRole("navigation", { name: "Sections" }).length).toBeGreaterThanOrEqual(2);
    expect(screen.getByRole("button", { name: "Refresh data" })).toBeTruthy();
    expect(screen.getByRole("status")).toBeTruthy();
  });

  it("quick nav carries the section links (pairing stays drawer-only)", () => {
    render(<App />);
    const quick = document.querySelector(".cf-quicknav");
    expect(quick?.textContent).toContain("Overview");
    expect(quick?.textContent).toContain("Billing");
    expect(quick?.textContent).toContain("Workers");
    expect(quick?.textContent).toContain("Durable Objects");
    expect(quick?.textContent).not.toContain("Pairing");
  });

  it("the bridge renders the Overview view from fixtures", async () => {
    render(<App />);
    // Fixture-driven overview paints its KPI section; the exact labels come
    // from makeFixtures, so assert on stable chrome the view always draws.
    await waitFor(
      () => {
        expect(document.querySelector(".cf-main .cf-view")?.childNodes.length).toBeGreaterThan(0);
      },
      { timeout: 3000 },
    );
  });
});

describe("App section identity (2026-10-11 shell pass)", () => {
  it("nav anchors carry data-route attributes (quick nav drops pairing)", () => {
    render(<App />);
    const sidebarLinks = document.querySelectorAll<HTMLAnchorElement>(".cf-navlink[data-route]");
    const quickLinks = document.querySelectorAll<HTMLAnchorElement>(".cf-quicknav-link[data-route]");
    // Sidebar/drawer variant: one link per route, each tagged with its route.
    expect(sidebarLinks.length).toBe(ROUTES.length);
    for (const a of sidebarLinks) {
      expect(ROUTES).toContain(a.getAttribute("data-route"));
      expect(a.getAttribute("href")).toBe(`#/${a.getAttribute("data-route")}`);
    }
    // Quick variant: same tagging minus the drawer-only pairing route.
    expect(quickLinks.length).toBe(ROUTES.length - 1);
    expect(document.querySelector('.cf-quicknav-link[data-route="domains"]')).toBeTruthy();
    expect(document.querySelector('.cf-quicknav-link[data-route="pairing"]')).toBeNull();
  });

  it("masthead renders the section label plus count when the counts fetch resolves", async () => {
    // The counts hook fetches api/domains among others; fixtures do not
    // answer that endpoint, so pin it with a spy and delegate the rest to
    // the real client (fixture fallback), mirroring the shell's mock-free
    // pattern elsewhere in this file.
    const original = api.fetchJson.bind(api);
    const spy = vi
      .spyOn(api, "fetchJson")
      .mockImplementation(((endpoint: string, opts?: { refresh?: boolean }) => {
        if (endpoint.startsWith("api/domains")) {
          return Promise.resolve<FetchResult<DomainsPayload>>({
            data: MOCK_DOMAINS,
            source: "fixture",
            ageSec: 0,
            demo: true,
          });
        }
        return original(endpoint, opts) as Promise<FetchResult<never>>;
      }) as typeof api.fetchJson);
    try {
      window.location.hash = "#/domains";
      render(<App />);
      const title = document.querySelector(".cf-sectionhead-title");
      await waitFor(
        () => {
          expect(title?.textContent).toContain("Domains");
          expect(document.querySelector(".cf-sectionhead-count")).toBeTruthy();
        },
        { timeout: 3000 },
      );
      // The masthead count mirrors the mocked domains payload length.
      expect(document.querySelector(".cf-sectionhead-count")?.textContent).toBe(
        ` (${MOCK_DOMAINS.domains.length})`,
      );
      expect(title?.textContent).toBe(`Domains (${MOCK_DOMAINS.domains.length})`);
    } finally {
      spy.mockRestore();
      window.location.hash = "";
    }
  });
});

describe("Bottom tab bar (operator 2026-10-12)", () => {
  /** The five primary sections + More, in render order. */
  const TAB_ROUTES = ["overview", "billing", "workers", "domains", "alerts"];

  it("renders exactly the 5 primary tabs + More, each tagged with data-route", () => {
    render(<App />);
    const tabs = document.querySelectorAll(".cf-bottomtabs [data-route]");
    expect(Array.from(tabs).map((el) => el.getAttribute("data-route"))).toEqual(TAB_ROUTES);
    // More is a button (opens the drawer), not a route anchor — it carries a
    // distinct marker-free identity as the only <button> in the bar.
    const more = document.querySelector(".cf-bottomtabs button");
    expect(more?.textContent).toContain("More");
    expect(document.querySelector(".cf-bottomtabs")).toBeTruthy();
  });

  it("marks the active route with aria-current='page'", () => {
    window.location.hash = "#/billing";
    render(<App />);
    const active = document.querySelector<HTMLAnchorElement>('.cf-bottomtab[data-route="billing"]');
    expect(active?.getAttribute("aria-current")).toBe("page");
    expect(document.querySelector('.cf-bottomtab[data-route="overview"]')?.getAttribute("aria-current")).toBeNull();
    expect(document.querySelector('.cf-bottomtab[data-route="alerts"]')?.getAttribute("aria-current")).toBeNull();
    window.location.hash = "";
  });

  it("the More button toggles the drawer with the FULL section list", () => {
    render(<App />);
    expect(document.getElementById("cf-drawer")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "More sections" }));
    const drawer = document.getElementById("cf-drawer");
    expect(drawer).toBeTruthy();
    // The full list includes drawer-only routes the tabs don't carry.
    expect(drawer?.textContent).toContain("Pairing");
    expect(drawer?.textContent).toContain("Durable Objects");
  });
});

describe("Install banner mount (FEAT-pXNQ5GW)", () => {
  /** jsdom has no matchMedia — stub it before render (per the component's
   *  own tests; the rest of this file doesn't need it). The stub carries
   *  the listener API framer-motion's useReducedMotion expects. */
  function stubAppMatchMedia(matches: boolean): void {
    (window as unknown as { matchMedia?: unknown }).matchMedia = vi.fn(
      (query: string) => ({
        matches,
        media: query,
        addEventListener: () => undefined,
        removeEventListener: () => undefined,
        addListener: () => undefined,
        removeListener: () => undefined,
      }),
    );
  }

  /** jsdom exposes no localStorage — Map stand-in (ZonesView.test.tsx
   *  pattern) so dismissal persistence is observable. */
  const bannerStore: { map: Map<string, string>; clear(): void } = (() => {
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

  it("does not render the banner when standalone", () => {
    stubAppMatchMedia(true);
    const { container } = render(<App />);
    expect(container.querySelector("aside[aria-label='Install app']")).toBeNull();
  });

  it("renders the banner when not standalone, dismissible on click", async () => {
    stubAppMatchMedia(false);
    bannerStore.clear();
    const { container } = render(<App />);
    // jsdom is not iOS and fires no beforeinstallprompt, so the banner only
    // appears once the (stubbed) event arrives.
    expect(container.querySelector("aside[aria-label='Install app']")).toBeNull();
    window.dispatchEvent(new Event("beforeinstallprompt"));
    await waitFor(() =>
      expect(container.querySelector("aside[aria-label='Install app']")).toBeTruthy(),
    );
    fireEvent.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(container.querySelector("aside[aria-label='Install app']")).toBeNull();
    expect(window.localStorage.getItem("pager-install-hint-dismissed")).toBe("1");
  });
});
