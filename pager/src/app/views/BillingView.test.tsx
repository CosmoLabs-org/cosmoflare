// @vitest-environment jsdom
// BillingView (P-04): the React view renders the grade+overage pair on
// every product card, excludes kv.storage (rolling bytes) with the muted
// note, and renders the ranked top consumers — against the localhost
// fixture data through the same ApiClient path production uses.
import { cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import BillingView from "./BillingView";

afterEach(cleanup);

describe("BillingView (P-04)", () => {
  it("renders a grade+overage pair head on every product card", async () => {
    render(<BillingView />);
    await waitFor(() => {
      expect(document.querySelector(".cf-product")).toBeTruthy();
    });
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
});
