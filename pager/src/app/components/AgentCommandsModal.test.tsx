// @vitest-environment jsdom
// AgentCommandsModal (P-18, FEAT-pC1EV32): the copy-paste dispatch surface
// for a zone's advisory brief. Covers the full specced flow — open →
// loading shimmer → brief text → Copy writes the exact agentBrief →
// checkmark crossfade → 1.5s revert → Esc/overlay close — plus the
// zero-findings (disabled Copy) and error → Retry paths.
import { act, cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import AgentCommandsModal from "./AgentCommandsModal";

const BRIEF = `# zone.example — Cloudflare optimization directive

## Problem
- [critical] low-cache-hit: 7d cache hit rate 0.5% — 50 of 10000 requests served from edge cache

## Actions
1. Raise the edge cache hit rate above the 5% floor.

## Verify
- cosmoflare efficiency --json --window 7

## Context
Zone zone.example (z1) — measured 7d by CosmoLabs Ops.`;

function envelope(overrides: Partial<{ brief: unknown; agentBrief: string }> = {}): unknown {
  return {
    zone: "zone.example",
    zoneId: "z1",
    generatedAt: "2026-10-15T00:00:00.000Z",
    facts: { window: "7d" },
    brief: overrides.brief ?? { findings: [{ kind: "low-cache-hit", severity: "critical", evidence: "…" }] },
    rendered: { agentBrief: overrides.agentBrief ?? BRIEF },
  };
}

function stubFetch(handler: (url: string) => Promise<unknown>): { calls: string[] } {
  const calls: string[] = [];
  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input);
    calls.push(url);
    const body = await handler(url);
    return new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });
  }));
  return { calls };
}

function stubClipboard(): ReturnType<typeof vi.fn> {
  const writeText = vi.fn().mockResolvedValue(undefined);
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
  return writeText;
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

describe("AgentCommandsModal", () => {
  it("shimmers while loading, then renders the measured brief with header and subtitle", async () => {
    let release!: (v: unknown) => void;
    const gate = new Promise<unknown>((resolve) => (release = resolve));
    stubFetch(() => gate);
    render(<AgentCommandsModal zone="zone.example" onClose={() => undefined} />);

    // Loading shimmer before the advisory resolves.
    expect(document.querySelectorAll(".cf-skeleton").length).toBeGreaterThan(0);
    expect(document.querySelector(".cf-agent-brief")).toBeNull();

    release(envelope());
    await waitFor(() => expect(document.querySelector(".cf-agent-brief")).toBeTruthy());
    expect(document.getElementById("cf-agent-title")?.textContent).toBe("Commands to AI Agent · zone.example");
    expect(document.querySelector(".cf-modal-body")?.textContent).toBe("measured 7d · CosmoLabs Ops");
    expect(document.querySelector(".cf-agent-brief")?.textContent).toBe(BRIEF);
  });

  it("requests the encoded zone path fresh on open", async () => {
    const { calls } = stubFetch(async () => envelope());
    render(<AgentCommandsModal zone="two words.example" onClose={() => undefined} />);
    await waitFor(() => expect(calls[0]).toBe("/api/advisory?zone=two%20words.example"));
    expect(calls).toHaveLength(1);
  });

  it("Copy writes the exact agentBrief, flips to the checkmark, and reverts after 1.5s", async () => {
    stubFetch(async () => envelope());
    const writeText = stubClipboard();
    render(<AgentCommandsModal zone="zone.example" onClose={() => undefined} />);
    await waitFor(() => expect(document.querySelector(".cf-agent-brief")).toBeTruthy());

    const copy = document.querySelector<HTMLButtonElement>(".cf-agent-copy")!;
    expect(copy.textContent).toContain("Copy");

    // Fake timers BEFORE the click: the component schedules its 1.5s
    // revert with the clock that exists at copy time — arming fakes after
    // would leave the real timer running and the revert unobservable.
    vi.useFakeTimers();
    fireEvent.click(copy);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0); // flush the clipboard promise → setCopied(true)
    });
    expect(writeText).toHaveBeenCalledTimes(1);
    expect(writeText).toHaveBeenCalledWith(BRIEF);

    // The crossfade is CSS opacity over stacked faces: the check face is
    // "on" after the copy, and the label face reverts after 1.5s.
    const face = copy.querySelector<HTMLElement>(".cf-agent-face:not(.cf-agent-check)")!;
    const check = copy.querySelector<HTMLElement>(".cf-agent-check")!;
    expect(check.classList.contains("is-on")).toBe(true);
    expect(face.classList.contains("is-off")).toBe(true);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1600);
    });
    expect(face.classList.contains("is-on")).toBe(true);
    expect(check.classList.contains("is-off")).toBe(true);
  });

  it("zero findings: honest empty state and a disabled Copy", async () => {
    stubFetch(async () => envelope({ brief: null, agentBrief: "" }));
    render(<AgentCommandsModal zone="zone.example" onClose={() => undefined} />);
    // The subtitle and the empty-state line are both .cf-modal-body —
    // the empty-state is the one that carries the dispatch verdict.
    await waitFor(() =>
      expect(
        Array.from(document.querySelectorAll(".cf-modal-body")).some((el) => el.textContent?.includes("Nothing to dispatch")),
      ).toBe(true),
    );
    expect(document.querySelector(".cf-agent-brief")).toBeNull();
    const copy = document.querySelector<HTMLButtonElement>(".cf-agent-copy")!;
    expect(copy.disabled).toBe(true);
  });

  it("a failed fetch shows the error with Retry, and Retry refetches", async () => {
    let attempt = 0;
    stubFetch(async () => {
      attempt++;
      if (attempt === 1) throw new Error("HTTP 502");
      return envelope();
    });
    render(<AgentCommandsModal zone="zone.example" onClose={() => undefined} />);

    await waitFor(() =>
      expect(document.querySelector(".cf-modal-body")?.textContent).toContain("Could not load the advisory brief: HTTP 502"),
    );
    const buttons = Array.from(document.querySelectorAll<HTMLButtonElement>(".cf-modal-actions .cf-btn"));
    expect(buttons.map((b) => b.textContent)).toEqual(["Retry", "Close"]);
    buttons[0].click();
    await waitFor(() => expect(document.querySelector(".cf-agent-brief")).toBeTruthy());
  });

  it("Escape and overlay click close; a click inside the dialog does not", async () => {
    stubFetch(async () => envelope());
    const onClose = vi.fn();
    render(<AgentCommandsModal zone="zone.example" onClose={onClose} />);
    await waitFor(() => expect(document.querySelector(".cf-agent-brief")).toBeTruthy());

    fireEvent.click(document.querySelector(".cf-modal")!);
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.click(document.querySelector(".cf-modal-backdrop")!);
    expect(onClose).toHaveBeenCalledTimes(1);
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(2);
  });
});
