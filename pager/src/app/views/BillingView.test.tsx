// @vitest-environment jsdom
// BillingView (P-04): the React view renders the grade+overage pair on
// every product card, excludes kv.storage (rolling bytes) with the muted
// note, and renders the ranked top consumers — against the localhost
// fixture data through the same ApiClient path production uses. The
// first-load skeleton and the Retry/recover error state spy the same
// api.fetchJson seam.
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api, type Billing, type FetchResult } from "../../api";
import { makeFixtures } from "../../fixtures";
import BillingView from "./BillingView";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

/** The fixture billing payload, as fetchJson would hand it to the view. */
function fixtureBilling(): Billing {
  return makeFixtures(new Date()).billing;
}

describe("BillingView (P-04)", () => {
  it("renders a grade+overage pair head on every product card", async () => {
    render(<BillingView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-product")).toBeTruthy();
    });
    // Cost lockup (operator 2026-10-13): the month on ONE line — gray base,
    // plus, overage, month total — no equals sign, no column layout.
    expect(document.querySelector(".cf-cost-line-base")?.textContent).toBe("$5.00");
    const overage = document.querySelector(".cf-cost-line-over")?.textContent ?? "";
    expect(/^\$[\d,.]+$/.test(overage)).toBe(true);
    expect(document.querySelector(".cf-cost-line-total")?.textContent).toMatch(/^\$[\d,.]+$/);
    expect(document.querySelector(".cf-cost-line-month")?.textContent).toBe("this month");
    expect(document.querySelector(".cf-cost-op")).toBeNull();
    const cards = document.querySelectorAll(".cf-product");
    expect(cards.length).toBe(4); // all fixture products but kv.storage
    for (const card of Array.from(cards)) {
      const grade = card.querySelector(".cf-product-grade")?.textContent;
      const overage = card.querySelector(".cf-product-overage")?.textContent;
      expect(["OK", "NEAR LIMIT", "OVER LIMIT"]).toContain(grade);
      expect(overage === "no overage" || /^\+\$[\d,.]+ overage$/.test(overage ?? "")).toBe(true);
    }
    // Bar gauges inside the cards, with the detail text row.
    expect(document.querySelectorAll(".cf-gauge").length).toBe(4);
    expect(document.querySelectorAll(".cf-gauge-detail").length).toBe(4);
  });

  it("excludes kv.storage from the allowance cards with the muted note", async () => {
    render(<BillingView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-product")).toBeTruthy();
    });
    const names = Array.from(document.querySelectorAll(".cf-product-name")).map((n) => n.textContent);
    expect(names).not.toContain("KV storage");
    // The muted rolling-bytes note stays (BUG-pFAKDN3).
    expect(document.querySelector(".cf-products .cf-empty-quiet")?.textContent).toContain("KV storage hidden");
  });

  it("renders ranked top consumers with rank, name and share", async () => {
    render(<BillingView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-consumer")).toBeTruthy();
    });
    const consumers = document.querySelectorAll(".cf-consumers-ranked .cf-consumer");
    expect(consumers.length).toBeGreaterThan(0);
    // Rank numbers start at 1 and the share bar + pct render.
    expect(consumers[0].querySelector(".cf-consumer-rank")?.textContent).toBe("1");
    expect(consumers[0].querySelector(".cf-consumer-name")?.textContent?.length).toBeGreaterThan(0);
    expect(consumers[0].querySelector(".cf-consumer-share")?.textContent).toMatch(/%$/);
    // Projects table and pricing footnote still paint.
    expect(document.querySelector(".cf-table")).toBeTruthy();
    expect(document.querySelector(".cf-pricing-note")?.textContent).toContain("Prices verified");
    // Telemetry gaps card (fixture billing carries one error).
    expect(document.querySelector(".cf-card.cf-level-warning")?.textContent).toContain("Telemetry gaps");
  });

  it("collapses the cost line to the base alone when no overage is projected (critique P2)", async () => {
    const calm = fixtureBilling();
    calm.products = calm.products.map((p) => ({ ...p, projectedOverageUsd: 0 }));
    calm.totalProjectedOverageUsd = 0;
    vi.spyOn(api, "fetchJson").mockImplementation(async <T,>(): Promise<FetchResult<T>> => ({
      data: calm as unknown as T,
      source: "network",
      ageSec: 0,
      demo: false,
    }));
    render(<BillingView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-product")).toBeTruthy();
    });
    // Zero overage: the base carries the month — no "+ $0.00" noise, no
    // duplicated total.
    expect(document.querySelector(".cf-cost-line-base")?.textContent).toBe("$5.00");
    expect(document.querySelector(".cf-cost-line-over")).toBeNull();
    expect(document.querySelector(".cf-cost-line-total")).toBeNull();
    expect(document.querySelector(".cf-cost-line-month")?.textContent).toBe("this month");
  });

  it("renders the skeleton while the first fetch is in flight", async () => {
    let release!: (r: FetchResult<Billing>) => void;
    vi.spyOn(api, "fetchJson").mockImplementation(
      () =>
        new Promise<FetchResult<Billing>>((res) => {
          release = res;
        }),
    );
    render(<BillingView />);
    expect(document.querySelector(".cf-loading")).toBeTruthy();
    expect(document.querySelectorAll(".cf-skeleton").length).toBe(4);
    expect(document.querySelector(".cf-product")).toBeNull();
    release({ data: fixtureBilling(), source: "network", ageSec: 0, demo: false });
    await waitFor(() => {
      expect(document.querySelector(".cf-product")).toBeTruthy();
    });
    expect(document.querySelector(".cf-loading")).toBeNull();
  });

  it("shows the Retry card on a failed first load and Retry recovers", async () => {
    let fail = true;
    vi.spyOn(api, "fetchJson").mockImplementation(async <T,>(): Promise<FetchResult<T>> => {
      if (fail) throw new Error("HTTP 503");
      return { data: fixtureBilling() as unknown as T, source: "network", ageSec: 0, demo: false };
    });
    render(<BillingView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-card.cf-level-warning")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card h2")?.textContent).toBe("Could not load billing");
    expect(document.querySelector(".cf-card .cf-btn")?.textContent).toBe("Retry");
    fail = false;
    document.querySelector<HTMLButtonElement>(".cf-card .cf-btn")!.click();
    await waitFor(() => {
      expect(document.querySelector(".cf-product")).toBeTruthy();
    });
    // The error card is gone (any remaining cf-level-warning card is the
    // fixture's telemetry-gaps notice, not the retry state).
    expect(screen.queryByText("Could not load billing")).toBeNull();
  });
});
