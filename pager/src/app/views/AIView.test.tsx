// @vitest-environment jsdom
// AIView (BR-03, ROAD-112 phase 2): per-gateway model usage cards with the
// ranked model rows, token lines when the probe found them, "est."-labeled
// costs, the probe notes as quiet lines and the honest empty state — with
// the fetch injected (the worker_profile.test.ts pattern).
import { cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { LoginExpiredError, type FetchResult } from "../../api";
import type { AIGatewaysPayload } from "../../ai";
import AIView from "./AIView";

afterEach(cleanup);

function fetcherReturning(payload: AIGatewaysPayload, opts: { throwErr?: Error } = {}): Parameters<typeof AIView>[0]["fetcher"] {
  return async <T,>(_endpoint: string): Promise<FetchResult<T>> => {
    if (opts.throwErr) throw opts.throwErr;
    return { data: payload as unknown as T, source: "network", ageSec: 0, demo: false };
  };
}

const PAYLOAD: AIGatewaysPayload = {
  generatedAt: "2026-10-10T00:00:00Z",
  gateways: [
    {
      id: "gw-1",
      name: "proj-alpha",
      requests24h: 12345,
      requestsMtd: 987654,
      models: [
        { model: "@cf/meta/llama-3-8b-instruct", provider: "workers-ai", requests: 9200, tokensIn: 12200, tokensOut: 3400, costUsd: 0.42 },
        { model: "deepseek-chat", provider: "deepseek", requests: 3145 },
      ],
    },
    {
      id: "gw-2",
      name: "proj-beta",
      requests24h: 12,
      requestsMtd: 3400,
      models: [],
    },
  ],
  costBasis: "gateway-estimate",
  notes: [],
  errors: [],
};

describe("AIView (BR-03)", () => {
  it("renders gateway name + 24h count + MTD subtitle and ranked model rows with tokens", async () => {
    render(<AIView fetcher={fetcherReturning(PAYLOAD)} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-consumers-ranked")).toBeTruthy();
    });
    // Two gateway cards, in payload order.
    const names = Array.from(document.querySelectorAll(".cf-product-name")).map((n) => n.textContent);
    expect(names).toEqual(["proj-alpha", "proj-beta"]);
    // 24h count in the head, formatted; MTD as the muted subtitle.
    const head = document.querySelectorAll(".cf-card .cf-product-status")[0];
    expect(head?.textContent).toBe("12.3k");
    expect(document.querySelector(".cf-card .cf-row-detail")?.textContent).toBe("MTD 987.7k requests");
    // Ranked rows: rank badges 1..2, model + provider, requests right.
    const ranks = Array.from(document.querySelectorAll(".cf-consumer-rank")).map((r) => r.textContent);
    expect(ranks).toEqual(["1", "2"]);
    const modelNames = Array.from(document.querySelectorAll(".cf-consumer-name")).map((n) => n.childNodes[0]?.textContent);
    expect(modelNames).toEqual(["@cf/meta/llama-3-8b-instruct", "deepseek-chat"]);
    // Tokens line when present ("12.2k in / 3.4k out"), absent otherwise.
    const who = document.querySelectorAll(".cf-consumer-who");
    const lines = Array.from(who[0].querySelectorAll(".cf-consumer-project")).map((n) => n.textContent);
    expect(lines).toEqual(["workers-ai", "12.2k in / 3.4k out"]);
    const betaLines = Array.from(who[1].querySelectorAll(".cf-consumer-project")).map((n) => n.textContent);
    expect(betaLines).toEqual(["deepseek"]);
    // Requests counts right (proj-beta has no 24h model rows at all).
    const shares = Array.from(document.querySelectorAll(".cf-consumer-share")).map((n) => n.textContent);
    expect(shares).toEqual(["9.2k", "3.1k"]);
    // Gateway card with no 24h model traffic stays quiet, not empty-broken.
    expect(document.querySelectorAll(".cf-empty-quiet").length).toBeGreaterThan(0);
  });

  it("shows a muted provider chip next to the model name when providers are mixed", async () => {
    render(<AIView fetcher={fetcherReturning(PAYLOAD)} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-consumers-ranked")).toBeTruthy();
    });
    const chips = Array.from(document.querySelectorAll(".cf-consumer-name .cf-chip")).map((c) => c.textContent);
    expect(chips).toEqual(["workers-ai", "deepseek"]);
  });

  it("renders the zero-token probe-off shape: no token lines, notes as quiet lines", async () => {
    const probeOff: AIGatewaysPayload = {
      generatedAt: "2026-10-10T00:00:00Z",
      gateways: [
        {
          id: "gw-1",
          name: "proj-alpha",
          requests24h: 9200,
          requestsMtd: 12000,
          models: [{ model: "@cf/meta/llama-3-8b-instruct", provider: "workers-ai", requests: 9200 }],
        },
      ],
      costBasis: "gateway-estimate",
      notes: ["token/cost fields unavailable on the analytics dataset — counts only (probe 2026-10-10)"],
      errors: [],
    };
    render(<AIView fetcher={fetcherReturning(probeOff)} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-consumers-ranked")).toBeTruthy();
    });
    expect(document.body.textContent).not.toContain("in /");
    expect(document.body.textContent).not.toContain("est.");
    const quiet = Array.from(document.querySelectorAll(".cf-empty-quiet")).map((n) => n.textContent);
    expect(quiet.some((t) => t?.startsWith("token/cost fields unavailable"))).toBe(true);
  });

  it("labels every dollar figure with the muted est. suffix", async () => {
    render(<AIView fetcher={fetcherReturning(PAYLOAD)} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-consumers-ranked")).toBeTruthy();
    });
    expect(document.body.textContent).toContain("$0.42");
    expect(document.body.textContent).toContain("est.");
    // The est. suffix is its own muted span after the value.
    const detail = Array.from(document.querySelectorAll(".cf-row-detail")).find((n) => n.textContent?.includes("$0.42"));
    expect(detail).toBeTruthy();
    expect(detail?.querySelector(".cf-row-detail")?.textContent).toBe("est.");
  });

  it("renders the honest empty state for a zero-gateway account", async () => {
    const empty: AIGatewaysPayload = { ...PAYLOAD, gateways: [] };
    render(<AIView fetcher={fetcherReturning(empty)} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-empty-quiet")?.textContent).toContain("No AI Gateways on this account yet");
    });
    expect(document.querySelector(".cf-consumers-ranked")).toBeNull();
  });

  it("shows the login-expired message on LoginExpiredError", async () => {
    render(<AIView fetcher={fetcherReturning(PAYLOAD, { throwErr: new LoginExpiredError() })} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-login-expired")).toBeTruthy();
    });
  });

  it("offers a Retry card after a first-load failure", async () => {
    render(<AIView fetcher={fetcherReturning(PAYLOAD, { throwErr: new Error("HTTP 503") })} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-card.cf-level-warning")).toBeTruthy();
    });
    expect(document.querySelector(".cf-card .cf-btn")?.textContent).toBe("Retry");
  });
});
