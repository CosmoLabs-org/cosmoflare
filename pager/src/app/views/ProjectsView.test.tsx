// @vitest-environment jsdom
// ProjectsView (P-05b): per-project cards with the vanilla head (name +
// overage pair), driver chips, and ranked consumers (rank badges, share
// bars) — built from the injected billing fetch (the worker_profile.test.ts
// injection pattern), with projects.ts's sort/consumer helpers reused
// verbatim.
import { cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { LoginExpiredError, type Billing, type FetchResult, type ProductUsage } from "../../api";
import ProjectsView from "./ProjectsView";

afterEach(cleanup);

function product(id: string, product: string, metric: string, overageUsd: number, consumers: ProductUsage["topConsumers"]): ProductUsage {
  return {
    id,
    product,
    metric,
    unit: "ops",
    included: 100,
    used: 120,
    projected: 130,
    unitPriceUsd: 1,
    priceUnit: 1,
    projectedOverageUsd: overageUsd,
    topConsumers: consumers,
  };
}

const BILLING: Billing = {
  generatedAt: "2026-10-10T00:00:00Z",
  period: { start: "2026-10-01T00:00:00Z", end: "2026-11-01T00:00:00Z", day: 10, days: 31, source: "calendar" },
  products: [
    product("workers.requests", "Workers", "Requests", 4, [
      { name: "api-gateway", project: "ops", used: 9_100_000, share: 0.6 },
      { name: "cdn", project: "web", used: 2_300_000, share: 0.4 },
    ]),
    product("r2.storage", "R2", "Storage (average)", 2, [
      { name: "media-bucket", project: "web", used: 180, share: 0.8 },
      { name: "api-gateway", project: "ops", used: 20, share: 0.2 },
    ]),
  ],
  totalProjectedOverageUsd: 6,
  projects: [
    { project: "web", projectedOverageUsd: 2.4, drivers: ["workers.requests", "r2.storage"] },
    { project: "ops", projectedOverageUsd: 3.6, drivers: ["workers.requests", "r2.storage"] },
  ],
  pricing: { verifiedOn: "2026-10-09", sources: [] },
  errors: [],
};

function billingFetcher(billing: Billing, opts: { throwErr?: Error } = {}): Parameters<typeof ProjectsView>[0]["fetcher"] {
  return async <T,>(_endpoint: string): Promise<FetchResult<T>> => {
    if (opts.throwErr) throw opts.throwErr;
    return { data: billing as unknown as T, source: "network", ageSec: 0, demo: false };
  };
}

describe("ProjectsView (P-05b)", () => {
  it("renders one card per project, worst payer first, with the name + overage pair head", async () => {
    render(<ProjectsView fetcher={billingFetcher(BILLING)} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-projects-list")).toBeTruthy();
    });
    const cards = document.querySelectorAll(".cf-projects-list .cf-project");
    expect(cards.length).toBe(2);
    // sortProjectsByOverage: ops (3.6) before web (2.4).
    expect(cards[0].querySelector(".cf-project-name")?.textContent).toBe("ops");
    expect(cards[0].querySelector(".cf-project-overage")?.textContent).toBe("+$3.60 overage");
    expect(cards[0].querySelector(".cf-project-overage")?.classList.contains("cf-level-critical-text")).toBe(true);
    expect(cards[0].classList.contains("cf-over")).toBe(true);
    expect(cards[1].querySelector(".cf-project-name")?.textContent).toBe("web");
    // No page heading — the masthead owns titles; the card subtitle stays.
    expect(document.querySelector(".cf-projects-list h2")).toBeNull();
    expect(document.querySelector(".cf-row-detail")?.textContent).toBe(
      "Projected cost beyond allowances, by project, across Cloudflare products.",
    );
  });

  it("renders driver chips with product labels", async () => {
    render(<ProjectsView fetcher={billingFetcher(BILLING)} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-chips")).toBeTruthy();
    });
    // Both cards render their chips (2 drivers each).
    const chips = Array.from(document.querySelectorAll(".cf-chips .cf-chip")).map((c) => c.textContent);
    expect(chips).toEqual(["Workers requests", "R2 storage", "Workers requests", "R2 storage"]);
  });

  it("renders ranked consumers with rank badges, product labels and share bars", async () => {
    render(<ProjectsView fetcher={billingFetcher(BILLING)} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-consumers-ranked")).toBeTruthy();
    });
    // Both cards rank their own consumers 1..2.
    const ranks = Array.from(document.querySelectorAll(".cf-consumer-rank")).map((r) => r.textContent);
    expect(ranks).toEqual(["1", "2", "1", "2"]);
    // Within each card, biggest share first (projectConsumers); cards read
    // in list order (ops first — worst payer).
    const names = Array.from(document.querySelectorAll(".cf-consumer-name")).map((n) => n.textContent);
    expect(names).toEqual(["api-gateway", "api-gateway", "media-bucket", "cdn"]);
    const projects = Array.from(document.querySelectorAll(".cf-consumer-project")).map((n) => n.textContent);
    expect(projects).toEqual(["Workers requests", "R2 storage", "R2 storage", "Workers requests"]);
    const shares = Array.from(document.querySelectorAll(".cf-consumer-share")).map((n) => n.textContent);
    expect(shares).toEqual(["60%", "20%", "80%", "40%"]);
    const fills = Array.from(document.querySelectorAll<HTMLElement>(".cf-consumer-sharefill")).map((n) => n.style.width);
    expect(fills).toEqual(["60%", "20%", "80%", "40%"]);
  });

  it("paints the quiet empty state when no projects exist", async () => {
    render(
      <ProjectsView
        fetcher={billingFetcher({ ...BILLING, projects: [] })}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-empty-quiet")).toBeTruthy();
    });
    expect(document.querySelector(".cf-empty-quiet")?.textContent).toBe("No projects found on this account.");
  });

  it("shows the login-expired message on LoginExpiredError", async () => {
    render(<ProjectsView fetcher={billingFetcher(BILLING, { throwErr: new LoginExpiredError() })} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-login-expired")).toBeTruthy();
    });
  });

  it("offers a Retry card after a first-load failure", async () => {
    render(<ProjectsView fetcher={billingFetcher(BILLING, { throwErr: new Error("HTTP 503") })} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-card.cf-level-warning")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card h2")?.textContent).toBe("Could not load projects");
    expect(document.querySelector(".cf-card .cf-btn")?.textContent).toBe("Retry");
  });
});
