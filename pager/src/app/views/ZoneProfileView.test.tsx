// @vitest-environment jsdom
// ZoneProfileView tests: the drill-in mirrors WorkerProfileView — back
// control, stats faces, section cards render only with data, and a failed
// side fetch leaves the analytics card untouched (the worker_profile view
// test's injection pattern).
import { cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { LoginExpiredError, type FetchResult, type Summary, type ZoneProfilePayload, type ZoneRow } from "../../api";
import ZoneProfileView from "./ZoneProfileView";

afterEach(cleanup);

const ZONEROW: ZoneRow = { zone: "cosmolabs.org", total: 4_200_000, uncached: 61_234, missPct: 1.46, byStatus: { "200": 4_100_000 } };

const SUMMARY = (zones: ZoneRow[]): Summary => ({
  generatedAt: "2026-10-14T00:00:00Z",
  windowHours: 24,
  usage: [],
  d1: [],
  zones,
  workers: [],
  errors: [],
});

/** Endpoint-aware fetcher: maps "api/summary" and "api/zone-profile?…"
 *  to injected payloads (the WorkerProfileView.test.tsx pattern). */
function multiFetcher(endpoints: Record<string, unknown>): Parameters<typeof ZoneProfileView>[0]["fetcher"] {
  return async <T,>(endpoint: string): Promise<FetchResult<T>> => {
    for (const [prefix, body] of Object.entries(endpoints)) {
      if (endpoint === prefix || endpoint.startsWith(`${prefix}?`)) {
        return { data: body as T, source: "network", ageSec: 0, demo: false };
      }
    }
    throw new Error(`unexpected endpoint ${endpoint}`);
  };
}

const PROFILE: ZoneProfilePayload = {
  generatedAt: "2026-10-14T00:00:00.000Z",
  name: "cosmolabs.org",
  zoneId: "z1",
  metadata: {
    name: "cosmolabs.org",
    status: "active",
    planName: "Free Website",
    paused: false,
    type: "full",
    nameServers: ["a.ns.cloudflare.com", "b.ns.cloudflare.com"],
    createdOn: "2021-03-01T00:00:00Z",
  },
  sslMode: "strict",
  dnssec: { status: "active", algorithm: "13" },
  errors: [],
};

describe("ZoneProfileView", () => {
  it("renders the 24h stats plus the metadata, SSL, and DNSSEC cards", async () => {
    render(
      <ZoneProfileView
        name="cosmolabs.org"
        fetcher={multiFetcher({ "api/summary": SUMMARY([ZONEROW]), "api/zone-profile": PROFILE })}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    // Back control to the Zones section.
    const back = document.querySelector<HTMLAnchorElement>(".cf-profile-back");
    expect(back?.getAttribute("href")).toBe("#/zones");
    expect(back?.textContent).toBe("All zones");
    // Stats: the same numbers the Zones table shows.
    const values = Array.from(document.querySelectorAll(".cf-stat-value")).map((v) => v.textContent);
    expect(values).toEqual(["4.2M", "61.2k", "1.46%"]);
    // The three configuration cards.
    for (const title of ["Zone metadata", "TLS/SSL", "DNSSEC"]) {
      const h2 = Array.from(document.querySelectorAll(".cf-card h2")).find((el) => el.textContent === title);
      expect(h2, title).toBeTruthy();
    }
    // Metadata fields: plan, status, created.
    const text = document.body.textContent ?? "";
    expect(text).toContain("Free Website");
    expect(text).toContain("active");
    expect(text).toContain("2021-03-01");
    // SSL mode + DNSSEC status/algorithm rows.
    expect(text).toContain("strict");
    expect(text).toContain("Algorithm13");
  });

  it("renders no metadata/SSL/DNSSEC cards when the payload carries none", async () => {
    render(
      <ZoneProfileView
        name="cosmolabs.org"
        fetcher={multiFetcher({ "api/summary": SUMMARY([ZONEROW]), "api/zone-profile": { ...PROFILE, metadata: null, sslMode: null, dnssec: null } })}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    const headings = Array.from(document.querySelectorAll(".cf-card h2")).map((el) => el.textContent);
    for (const title of ["Zone metadata", "TLS/SSL", "DNSSEC", "Collection notes"]) {
      expect(headings).not.toContain(title);
    }
    // The analytics card still renders (some card is present).
    expect(headings.length).toBeGreaterThan(0);
  });

  it("a failed side fetch leaves the analytics card untouched (no section cards)", async () => {
    const fetcher: Parameters<typeof ZoneProfileView>[0]["fetcher"] = async <T,>(endpoint: string): Promise<FetchResult<T>> => {
      if (endpoint === "api/summary") {
        return { data: SUMMARY([ZONEROW]) as unknown as T, source: "network", ageSec: 0, demo: false };
      }
      throw new Error("HTTP 500");
    };
    // A KNOWN zone: the analytics rows render, only the side fetch fails.
    render(<ZoneProfileView name="cosmolabs.org" fetcher={fetcher} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-stat-value")).toBeTruthy();
    });
    const headings = Array.from(document.querySelectorAll(".cf-card h2")).map((el) => el.textContent);
    for (const title of ["Zone metadata", "TLS/SSL", "DNSSEC"]) {
      expect(headings).not.toContain(title);
    }
  });

  it("shows the quiet not-found state for a zone missing from the summary", async () => {
    render(
      <ZoneProfileView
        name="ghost.test"
        fetcher={multiFetcher({ "api/summary": SUMMARY([]), "api/zone-profile": PROFILE })}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card h2")?.textContent).toBe("Zone not found in the 24h window");
    expect(document.querySelector(".cf-empty-quiet")?.textContent).toBe("No 24h analytics row for ghost.test.");
  });

  it("shows the login-expired message on LoginExpiredError", async () => {
    const fetcher: Parameters<typeof ZoneProfileView>[0]["fetcher"] = async <T,>(): Promise<FetchResult<T>> => {
      throw new LoginExpiredError();
    };
    render(<ZoneProfileView name="cosmolabs.org" fetcher={fetcher} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-login-expired")).toBeTruthy();
    });
    expect(document.querySelector(".cf-login-expired")?.textContent).toBe(
      "your login expired — close and reopen the app to sign in again",
    );
  });

  it("offers a Retry card after a first-load failure", async () => {
    const fetcher: Parameters<typeof ZoneProfileView>[0]["fetcher"] = async <T,>(): Promise<FetchResult<T>> => {
      throw new Error("HTTP 503");
    };
    render(<ZoneProfileView name="cosmolabs.org" fetcher={fetcher} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-card.cf-level-warning")).toBeTruthy();
    });
    expect(document.querySelector(".cf-row-detail")?.textContent).toBe("HTTP 503");
    expect(document.querySelector(".cf-card .cf-btn")?.textContent).toBe("Retry");
  });

  it("renders the skeleton while the first fetch is in flight", async () => {
    let release!: (r: FetchResult<Summary>) => void;
    const gate = new Promise<FetchResult<Summary>>((res) => {
      release = res;
    });
    const fetcher: Parameters<typeof ZoneProfileView>[0]["fetcher"] = async <T,>(): Promise<FetchResult<T>> =>
      gate as unknown as Promise<FetchResult<T>>;
    render(<ZoneProfileView name="cosmolabs.org" fetcher={fetcher} />);
    expect(document.querySelector(".cf-loading")).toBeTruthy();
    expect(document.querySelectorAll(".cf-skeleton").length).toBe(3);
    expect(document.querySelector(".cf-profile")).toBeNull();
    release({ data: SUMMARY([]), source: "network", ageSec: 0, demo: false });
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    expect(document.querySelector(".cf-loading")).toBeNull();
  });

  it("renders collection notes for per-section errors", async () => {
    render(
      <ZoneProfileView
        name="cosmolabs.org"
        fetcher={multiFetcher({ "api/summary": SUMMARY([ZONEROW]), "api/zone-profile": { ...PROFILE, sslMode: null, errors: ["SSL mode unavailable: missing ssl read"] } })}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector(".cf-profile")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card h2")?.textContent === "Collection notes" ||
      Array.from(document.querySelectorAll(".cf-card h2")).some((el) => el.textContent === "Collection notes")).toBe(true);
    expect(document.body.textContent).toContain("SSL mode unavailable: missing ssl read");
  });
});
