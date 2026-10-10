// @vitest-environment jsdom
// DomainsView (P-05b): sorted rows with expiry attention coloring and the
// "external" handling, the in-component profile drill-down (tap a row →
// Registration/DNS/Security/Renewal sections; "All domains" returns), and
// the /api/domains/detail enrichment (the historical "Previous nameservers
// (before Cloudflare)" label) — with every fetch injected
// (the worker_profile.test.ts pattern).
import { cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { FetchResult } from "../../api";
import type { DomainDetailPayload, DomainRecord, DomainsPayload } from "../../domains";
import DomainsView from "./DomainsView";

afterEach(cleanup);

const DAY = 86_400_000;
const iso = (daysFromNow: number): string => new Date(Date.now() + daysFromNow * DAY).toISOString();

function domain(partial: Partial<DomainRecord> & { id: string; name: string }): DomainRecord {
  return {
    status: "active",
    paused: false,
    expiresAt: null,
    plan: "Pro",
    type: "full",
    createdOn: "2024-01-15T00:00:00Z",
    modifiedOn: "2026-09-01T00:00:00Z",
    nameServers: ["ada.ns.cloudflare.com", "bob.ns.cloudflare.com"],
    ...partial,
  };
}

const DOMAINS: DomainsPayload = {
  generatedAt: "2026-10-10T00:00:00Z",
  domains: [
    domain({ id: "z3", name: "later.org", expiresAt: iso(90) }),
    domain({ id: "z1", name: "soon.dev", expiresAt: iso(10) }),
    domain({ id: "z4", name: "outside.example", expiresAt: null }),
    domain({ id: "z2", name: "warn.net", expiresAt: iso(20), status: "pending" }),
  ],
  errors: [],
};

const DETAIL: DomainDetailPayload = {
  zone: {
    ...domain({ id: "z1", name: "soon.dev", expiresAt: iso(10) }),
    originalNameServers: ["ns1.dreamhost.com", "ns2.dreamhost.com"],
    activatedOn: "2024-01-16T00:00:00Z",
    ownerType: "organization",
    dnssecStatus: "active",
    sslMode: "strict",
  },
  errors: [],
};

/** Stub fetcher: answers api/domains and the detail/summary endpoints; the
 *  summary deliberately fails to prove enrichment is soft. */
function domainsFetcher(opts: { detail?: DomainDetailPayload; failFirstLoad?: Error } = {}): Parameters<typeof DomainsView>[0]["fetcher"] {
  return async <T,>(endpoint: string): Promise<FetchResult<T>> => {
    if (opts.failFirstLoad) throw opts.failFirstLoad;
    if (endpoint === "api/domains") return { data: DOMAINS as unknown as T, source: "network", ageSec: 0, demo: false };
    if (endpoint.startsWith("api/domains/detail")) {
      if (opts.detail) return { data: opts.detail as unknown as T, source: "network", ageSec: 0, demo: false };
      throw new Error("HTTP 503");
    }
    throw new Error("HTTP 404");
  };
}

const text = (sel: string): string[] => Array.from(document.querySelectorAll(sel)).map((n) => n.textContent);

describe("DomainsView (P-05b)", () => {
  it("renders sorted rows with expiry attention coloring, external last", async () => {
    render(<DomainsView fetcher={domainsFetcher()} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-domain-row")).toBeTruthy();
    });
    // sortDomains: soonest expiry first, no-expiry domains last by name.
    const names = text(".cf-domain-name");
    expect(names).toEqual(["soon.dev", "warn.net", "later.org", "outside.example"]);
    // Expiry levels: ≤14d critical, ≤30d warning, else ok, null carries none.
    const expiry = document.querySelectorAll(".cf-domain-expiry");
    expect(expiry[0].classList.contains("cf-level-critical-text")).toBe(true);
    expect(expiry[0].textContent).toMatch(/^\d+d · /);
    expect(expiry[1].classList.contains("cf-level-warning-text")).toBe(true);
    expect(expiry[2].classList.contains("cf-level-ok-text")).toBe(true);
    expect(expiry[3].classList.contains("cf-level-critical-text")).toBe(false);
    expect(expiry[3].textContent).toBe("external");
    // The pending (non-active) zone carries its muted badge.
    expect(text(".cf-domain-badges .cf-badge")).toEqual(["pending"]);
    // Subtitle counts the expiring domains; no page heading.
    expect(document.querySelector(".cf-row-detail")?.textContent).toBe("4 domains · 2 expiring within 30 days");
    expect(document.querySelector(".cf-card h2")).toBeNull();
  });

  it("drills into the profile on row tap and returns on 'All domains'", async () => {
    render(<DomainsView fetcher={domainsFetcher()} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-domain-row")).toBeTruthy();
    });
    const rows = document.querySelectorAll<HTMLButtonElement>(".cf-domain-row");
    rows[1].click(); // warn.net
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    expect(document.querySelector(".cf-profile-title")?.textContent).toBe("warn.net");
    expect(document.querySelector(".cf-profile-eyebrow")?.textContent).toBe("Domain profile");
    // The four sections, in the vanilla order.
    const sections = text(".cf-profile-section-title");
    expect(sections).toEqual(["Renewal", "Registration", "DNS", "Security"]);
    // Back control returns to the list.
    const back = document.querySelector<HTMLButtonElement>(".cf-profile-back");
    expect(back?.textContent).toBe("All domains");
    back!.click();
    await waitFor(() => {
      expect(document.querySelector(".cf-domain-row")).toBeTruthy();
    });
    expect(document.querySelector(".cf-profile")).toBeNull();
  });

  it("enriches the profile from api/domains/detail — failures leave the base profile", async () => {
    render(<DomainsView fetcher={domainsFetcher({ detail: DETAIL })} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-domain-row")).toBeTruthy();
    });
    document.querySelectorAll<HTMLButtonElement>(".cf-domain-row")[0].click(); // soon.dev
    await waitFor(() => {
      expect(document.querySelector(".cf-profile-title")?.textContent).toBe("soon.dev");
    });
    // The historical label, verbatim.
    await waitFor(() => {
      expect(text(".cf-domain-field-label")).toContain("Previous nameservers (before Cloudflare)");
    });
    const labels = text(".cf-domain-field-label");
    const values = text(".cf-domain-field-value");
    const idx = labels.indexOf("Previous nameservers (before Cloudflare)");
    expect(values[idx]).toBe("ns1.dreamhost.com\nns2.dreamhost.com");
    expect(labels).toContain("DNSSEC");
    expect(labels).toContain("SSL mode");
    expect(labels).toContain("Activated");
    expect(labels).toContain("Owner");
    // Traffic stays absent — the summary fetch failed (soft join).
    expect(labels).not.toContain("Traffic (24h)");
  });

  it("keeps the base profile when the detail fetch fails", async () => {
    render(<DomainsView fetcher={domainsFetcher()} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-domain-row")).toBeTruthy();
    });
    document.querySelectorAll<HTMLButtonElement>(".cf-domain-row")[0].click();
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    // Let any in-flight enrichment settle.
    await new Promise((r) => setTimeout(r, 20));
    expect(text(".cf-domain-field-label")).not.toContain("Previous nameservers (before Cloudflare)");
    // The base sections still paint.
    expect(text(".cf-profile-section-title")).toEqual(["Renewal", "Registration", "DNS", "Security"]);
  });

  it("paints the quiet empty state when no domains exist", async () => {
    const empty: DomainsPayload = { generatedAt: "2026-10-10T00:00:00Z", domains: [], errors: [] };
    const fetcher: Parameters<typeof DomainsView>[0]["fetcher"] = async <T,>(endpoint: string) => {
      if (endpoint === "api/domains") return { data: empty as unknown as T, source: "network", ageSec: 0, demo: false };
      throw new Error("HTTP 404");
    };
    render(<DomainsView fetcher={fetcher} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-empty-quiet")).toBeTruthy();
    });
    expect(document.querySelector(".cf-empty-quiet")?.textContent).toBe("No domains found on this account.");
  });
});
