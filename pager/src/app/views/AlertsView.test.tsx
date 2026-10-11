// @vitest-environment jsdom
// AlertsView as the Notifications center: the feed rows render newest first
// with their value lines from an injected fetcher, clicking a row expands
// the inline detail (numbers table + the plain-language explanation for that
// condition + the muted delivery footer), the unread dot marks entries
// newer than the localStorage "pager-alerts-seen" timestamp and the render
// marks everything read (quota-safe), and the empty/error states follow the
// house patterns. The IndexedDB store module is mocked — the push-history
// section is covered by store.test.ts.
import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { FetchResult } from "../../api";
import type { AlertFeed, AlertFeedEntry } from "../../alerts";
import AlertsView from "./AlertsView";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  window.localStorage.clear();
});

vi.mock("../../store", () => ({
  listAlerts: async () => [],
  acknowledge: async () => undefined,
  snooze: async () => undefined,
}));

const NOW = Date.parse("2026-10-12T12:00:00Z");
// The view computes relative times against Date.now(); pin the clock so the
// assertions are deterministic.
vi.spyOn(Date, "now").mockImplementation(() => NOW + 14 * 60 * 1000);

function entry(partial: Partial<AlertFeedEntry>): AlertFeedEntry {
  return {
    ts: new Date(NOW + 5 * 60 * 1000).toISOString(),
    ruleId: "uncached requests/hot.example",
    condition: "zone-uncached-requests",
    title: "uncached requests",
    value: 12000,
    threshold: 10000,
    unit: "requests",
    message: "zone-uncached-requests hot.example [uncached requests]: observed 12000 meets threshold 10000 (requests)",
    sent: true,
    ...partial,
  };
}

function feedFetcher(data: AlertFeed): Parameters<typeof AlertsView>[0]["fetcher"] {
  return async <T,>(endpoint: string): Promise<FetchResult<T>> => {
    if (endpoint === "api/alerts/feed") return { data: data as unknown as T, source: "network", ageSec: 0, demo: false };
    throw new Error("HTTP 404");
  };
}

const SEEN_KEY = "pager-alerts-seen";

describe("AlertsView — Notifications center", () => {
  it("renders feed rows newest first with their value lines from the stubbed fetcher", async () => {
    const older = entry({ ts: new Date(NOW - 2 * 60 * 60 * 1000).toISOString(), title: "older fire", value: 250000, threshold: 100000, unit: "writes", condition: "kv-writes", ruleId: "kvday/SESSIONS" });
    const newer = entry({});
    render(<AlertsView fetcher={feedFetcher({ feed: [newer, older] })} />);
    await waitFor(() => {
      expect(document.querySelectorAll(".cf-feed-row")).toHaveLength(2);
    });
    const rows = document.querySelectorAll(".cf-feed-row");
    // newest first: the given order is preserved (worker serves newest first)
    expect(rows[0].querySelector(".cf-feed-title")?.textContent).toBe("uncached requests");
    expect(rows[0].querySelector(".cf-feed-time")?.textContent).toBe("14m ago");
    expect(rows[0].querySelector(".cf-feed-value")?.textContent).toBe("12000 / 10000 requests");
    expect(rows[1].querySelector(".cf-feed-title")?.textContent).toBe("older fire");
    expect(rows[1].querySelector(".cf-feed-time")?.textContent).toBe("2h ago");
    expect(rows[1].querySelector(".cf-feed-value")?.textContent).toBe("250000 / 100000 writes");
  });

  it("clicking a row expands the detail: numbers table, the condition explanation, and the delivery footer", async () => {
    const kvFire = entry({ condition: "kv-writes", title: "kvday", value: 250000, threshold: 100000, unit: "writes", ruleId: "kvday/SESSIONS", sent: false });
    render(<AlertsView fetcher={feedFetcher({ feed: [kvFire] })} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-feed-row")).toBeTruthy();
    });
    expect(document.querySelector(".cf-feed-detail")).toBeNull();
    fireEvent.click(document.querySelector<HTMLButtonElement>(".cf-feed-row")!);
    await waitFor(() => {
      expect(document.querySelector(".cf-feed-detail")).toBeTruthy();
    });
    // aria-expanded reflects the toggle
    expect(document.querySelector(".cf-feed-row")?.getAttribute("aria-expanded")).toBe("true");
    // numbers table: value, threshold, unit, fired at
    const cells = document.querySelectorAll(".cf-feed-table td");
    expect(Array.from(cells).map((c) => c.textContent)).toEqual(["250000", "100000", "writes", kvFire.ts]);
    // the plain-language explanation for the kv-writes condition
    const text = document.querySelector(".cf-feed-explanation")?.textContent ?? "";
    expect(text).toContain("KV writes");
    expect(text).toContain("namespace");
    // muted footer: rule id + push-not-delivered (sent:false entry)
    expect(document.querySelector(".cf-feed-footer")?.textContent).toContain("kvday/SESSIONS");
    expect(document.querySelector(".cf-feed-footer")?.textContent).toContain("push not delivered");
    // clicking again collapses
    fireEvent.click(document.querySelector<HTMLButtonElement>(".cf-feed-row")!);
    expect(document.querySelector(".cf-feed-detail")).toBeNull();
  });

  it("marks unread entries newer than the seen timestamp and marks all read on render", async () => {
    window.localStorage.setItem(SEEN_KEY, String(NOW)); // seen up to NOW
    const read = entry({ ts: new Date(NOW - 60 * 60 * 1000).toISOString(), ruleId: "old/z" });
    const fresh = entry({}); // NOW + 5m: newer than the seen timestamp
    render(<AlertsView fetcher={feedFetcher({ feed: [fresh, read] })} />);
    await waitFor(() => {
      expect(document.querySelectorAll(".cf-feed-row")).toHaveLength(2);
    });
    const dots = document.querySelectorAll(".cf-feed-dot");
    expect(dots).toHaveLength(1); // only the newer-than-seen entry
    // mark-all-read: the seen timestamp advanced past both entries
    expect(Number(window.localStorage.getItem(SEEN_KEY))).toBeGreaterThan(NOW);
  });

  it("paints the quiet empty state when nothing has fired yet", async () => {
    render(<AlertsView fetcher={feedFetcher({ feed: [] })} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-empty-quiet")).toBeTruthy();
    });
    expect(document.querySelector(".cf-empty-quiet")?.textContent).toBe(
      "Nothing has fired yet — alerts land here the moment a rule trips.",
    );
    expect(document.querySelector(".cf-feed-row")).toBeNull();
  });

  it("shows the Retry card when the feed fetch fails and Retry recovers", async () => {
    let fail = true;
    const fetcher: Parameters<typeof AlertsView>[0]["fetcher"] = async <T,>(endpoint: string): Promise<FetchResult<T>> => {
      if (endpoint !== "api/alerts/feed") throw new Error("HTTP 404");
      if (fail) throw new Error("HTTP 503");
      return { data: { feed: [entry({})] } as unknown as T, source: "network", ageSec: 0, demo: false };
    };
    render(<AlertsView fetcher={fetcher} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-card.cf-level-warning")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card-title")?.textContent).toBe("Could not load the alert feed");
    fail = false;
    document.querySelector<HTMLButtonElement>(".cf-card .cf-btn")!.click();
    await waitFor(() => {
      expect(document.querySelector(".cf-feed-row")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card.cf-level-warning")).toBeNull();
  });
});
