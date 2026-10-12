// @vitest-environment jsdom
// OverviewView (P-04): the React view paints KPI values from the localhost
// fixture data (the same ApiClient path production uses), the attention
// rows are tappable disclosures, and every paint feeds the bottom status
// strip. jsdom on localhost → the ApiClient falls back to fixtures, exactly
// like the App shell smoke test. The first-load skeleton and the Retry/
// recover error state spy the same api.fetchJson seam.
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api, type Billing, type FetchResult, type Summary } from "../../api";
import { makeFixtures } from "../../fixtures";
import OverviewView from "./OverviewView";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

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

  it("renders the skeleton until both first fetches resolve", async () => {
    installStatusStrip();
    const fx = makeFixtures(new Date());
    const gates: Array<(r: FetchResult<Summary | Billing>) => void> = [];
    vi.spyOn(api, "fetchJson").mockImplementation(
      <T,>() =>
        new Promise<FetchResult<T>>((res) => {
          gates.push(res as unknown as (r: FetchResult<Summary | Billing>) => void);
        }),
    );
    render(<OverviewView />);
    expect(document.querySelector(".cf-loading")).toBeTruthy();
    expect(document.querySelectorAll(".cf-skeleton").length).toBe(4);
    expect(document.querySelector(".cf-kpis")).toBeNull();
    // Promise.all fires api/summary first, then api/billing.
    gates[0]({ data: fx.summary, source: "network", ageSec: 0, demo: false });
    gates[1]({ data: fx.billing, source: "network", ageSec: 0, demo: false });
    await waitFor(() => {
      expect(document.querySelector(".cf-kpis")).toBeTruthy();
    });
    expect(document.querySelector(".cf-loading")).toBeNull();
  });

  it("shows the Retry card on a failed first load and Retry recovers", async () => {
    installStatusStrip();
    const fx = makeFixtures(new Date());
    let fail = true;
    vi.spyOn(api, "fetchJson").mockImplementation(async <T,>(endpoint: string): Promise<FetchResult<T>> => {
      if (fail) throw new Error("HTTP 503");
      const data = endpoint.startsWith("api/billing") ? fx.billing : fx.summary;
      return { data: data as unknown as T, source: "network", ageSec: 0, demo: false };
    });
    render(<OverviewView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-card.cf-level-warning")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card h2")?.textContent).toBe("Could not load the overview");
    expect(document.querySelector(".cf-card .cf-btn")?.textContent).toBe("Retry");
    fail = false;
    document.querySelector<HTMLButtonElement>(".cf-card .cf-btn")!.click();
    await waitFor(() => {
      expect(document.querySelector(".cf-kpis")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card.cf-level-warning")).toBeNull();
  });
});

// FEAT-059: the Email Routing card renders only when the /api/email
// collector returned zones — an errors-only payload (the token lacks
// Email Routing Read) or a failed fetch must leave the overview unchanged.
describe("OverviewView email card (FEAT-059)", () => {
  const fx = makeFixtures(new Date());

  function mockEndpoints(email: unknown | null, fail = false): void {
    vi.spyOn(api, "fetchJson").mockImplementation(async <T,>(endpoint: string): Promise<FetchResult<T>> => {
      if (fail || endpoint === "api/email" && email === null) throw new Error("HTTP 503");
      const data = endpoint === "api/billing" ? fx.billing : endpoint === "api/email" ? email : fx.summary;
      return { data: data as unknown as T, source: "network", ageSec: 0, demo: false };
    });
  }

  it("renders the tile + per-zone rows when the collector returns zones", async () => {
    installStatusStrip();
    mockEndpoints({
      generatedAt: "2026-10-11T12:00:00.000Z",
      zones: [
        { zone: "cosmolabs.org", enabled: true, name: "cosmolabs.org" },
        { zone: "noble.coffee", enabled: false },
      ],
      errors: [],
    });
    render(<OverviewView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-kpis")).toBeTruthy();
    });
    const card = document.querySelector('section[aria-label="Email Routing"]');
    expect(card).toBeTruthy();
    expect(card?.querySelector("h2")?.textContent).toBe("Email Routing");
    // The summary tile: enabled zones over total.
    expect(card?.querySelector(".cf-kpi-value")?.textContent).toBe("1/2");
    // One row per zone, the enabled/disabled chip as the data face.
    const rows = card?.querySelectorAll(".cf-consumers-list .cf-consumer");
    expect(rows?.length).toBe(2);
    expect(rows?.[0].querySelector(".cf-consumer-name")?.textContent).toBe("cosmolabs.org");
    expect(rows?.[0].querySelector(".cf-chip")?.textContent).toBe("enabled");
    expect(rows?.[1].querySelector(".cf-consumer-name")?.textContent).toBe("noble.coffee");
    expect(rows?.[1].querySelector(".cf-chip")?.textContent).toBe("disabled");
  });

  it("stays absent when the collector returns an errors-only payload", async () => {
    installStatusStrip();
    mockEndpoints({ generatedAt: "2026-10-11T12:00:00.000Z", zones: [], errors: ["email routing unavailable for cosmolabs.org: HTTP 403"] });
    render(<OverviewView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-kpis")).toBeTruthy();
    });
    expect(document.querySelector('section[aria-label="Email Routing"]')).toBeNull();
  });

  it("stays absent when the /api/email fetch fails", async () => {
    installStatusStrip();
    // email=null makes ONLY api/email throw (see the helper); summary and
    // billing must still resolve so the view renders at all.
    mockEndpoints(null);
    render(<OverviewView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-kpis")).toBeTruthy();
    });
    expect(document.querySelector('section[aria-label="Email Routing"]')).toBeNull();
  });
});
