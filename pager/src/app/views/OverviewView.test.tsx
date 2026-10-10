// @vitest-environment jsdom
// OverviewView (P-04): the React view paints KPI values from the localhost
// fixture data (the same ApiClient path production uses), the attention
// rows are tappable disclosures, and every paint feeds the bottom status
// strip. jsdom on localhost → the ApiClient falls back to fixtures, exactly
// like the App shell smoke test.
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import OverviewView from "./OverviewView";

afterEach(cleanup);

// The status strip writes into #cf-status-age/#cf-status-period, which live
// in the App shell's footer; the view test provides them.
function installStatusStrip(): void {
  document.body.insertAdjacentHTML(
    "beforeend",
    '<span id="cf-status-age"></span><span id="cf-status-period" class="cf-status-period" hidden></span>',
  );
}

describe("OverviewView (P-04)", () => {
  it("paints the plan line, KPI grid and fixture-driven values", async () => {
    installStatusStrip();
    render(<OverviewView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-kpis")).toBeTruthy();
    });
    // Plan line (the vanilla view's constant).
    expect(screen.getByText("Workers Paid · $5.00/mo")).toBeTruthy();
    // Five KPI tiles, one per paintOverview tile.
    expect(document.querySelectorAll(".cf-kpis .cf-kpi").length).toBe(5);
    // The period tile carries its own progress bar.
    expect(document.querySelector(".cf-kpis .cf-progress .cf-progress-fill")).toBeTruthy();
    // Rings card: every non-rolling fixture product renders a ring
    // (4 fixture products, none rolling — under the 8-ring cap).
    expect(document.querySelectorAll(".cf-ring-svg").length).toBe(4);
    // Rings carry the center value + label band structure.
    expect(document.querySelector(".cf-ring-value")).toBeTruthy();
    expect(document.querySelector(".cf-ring-label")).toBeTruthy();
  });

  it("attention rows toggle their disclosure and keep the section link", async () => {
    installStatusStrip();
    render(<OverviewView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-attention-toggle")).toBeTruthy();
    });
    const toggle = document.querySelector<HTMLButtonElement>(".cf-attention-toggle");
    expect(toggle?.getAttribute("aria-expanded")).toBe("false");
    // Fixture account has over-limit products (workers.requests,
    // r2.storage), so a critical row exists and its mark is "!".
    expect(document.querySelector(".cf-attention.cf-level-critical")).toBeTruthy();
    // Detail hidden until tapped.
    const detail = document.querySelector<HTMLElement>(".cf-attention-detail");
    expect(detail?.hidden).toBe(true);
    fireEvent.click(toggle!);
    expect(toggle?.getAttribute("aria-expanded")).toBe("true");
    expect(detail?.hidden).toBe(false);
    // The full detail text and the secondary "Open section →" link.
    expect(detail?.querySelector(".cf-attention-text")?.textContent?.length).toBeGreaterThan(0);
    expect(detail?.querySelector<HTMLAnchorElement>(".cf-attention-open")?.href).toContain("#/billing");
  });

  it("feeds the bottom status strip on paint (age line + period)", async () => {
    installStatusStrip();
    render(<OverviewView />);
    const age = document.getElementById("cf-status-age");
    const period = document.getElementById("cf-status-period");
    await waitFor(() => {
      expect(age?.textContent).toContain("Updated");
    });
    expect(period?.hidden).toBe(false);
    expect(period?.textContent).toMatch(/^Day \d+ of \d+$/);
  });
});
