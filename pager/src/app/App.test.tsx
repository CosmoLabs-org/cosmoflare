// @vitest-environment jsdom
// React shell smoke (P-03): the App mounts, paints the shell chrome (brand,
// quick nav, status strip), and the imperative bridge renders a view — on
// jsdom's localhost the ApiClient falls back to fixtures, so the Overview
// paints fixture data through the same path production uses.
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import App from "./App";

afterEach(cleanup);

describe("App shell (P-03)", () => {
  it("mounts the chrome: brand lockup, quick nav sections, status strip", () => {
    render(<App />);
    expect(screen.getAllByText("CosmoLabs Ops").length).toBeGreaterThan(0);
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
