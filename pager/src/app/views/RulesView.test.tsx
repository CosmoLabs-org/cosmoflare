// @vitest-environment jsdom
// RulesView (P-06): draft add, client validation (per-card highlight +
// action-bar error), the exclude chips (add dedupes, remove filters), the
// PUT payload shape, the server echo adoption (a second Save must not PUT
// stale rules), the starter note clearing on save, and the 400
// RulesHttpError path with the index highlight — GETs injected through the
// fetcher, the PUT observed on a stubbed global fetch.
import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { FetchResult } from "../../api";
import type { AlertRule, RulesPayload } from "../../rules";
import RulesView from "./RulesView";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

const CONDITIONS = ["d1-rows-read", "kv-writes", "cpu-p99"];

function payload(rules: AlertRule[], starter = false): RulesPayload {
  return { rules, conditions: CONDITIONS, starter };
}

function rule(partial: Partial<AlertRule>): AlertRule {
  return { name: "", condition: CONDITIONS[0], threshold: 0, exclude: [], enabled: true, ...partial };
}

/** Fetcher answering only api/rules with the given payload. */
function rulesFetcher(data: RulesPayload): Parameters<typeof RulesView>[0]["fetcher"] {
  return async <T,>(endpoint: string): Promise<FetchResult<T>> => {
    if (endpoint === "api/rules") return { data: data as unknown as T, source: "network", ageSec: 0, demo: false };
    throw new Error("HTTP 404");
  };
}

/** PUT stub: records the last body; replies with the echo (or a 400). A
 *  plain response lookalike — jsdom has no Response constructor. */
function stubPut(opts: { status?: number; body?: unknown } = {}): { lastBody: () => unknown } {
  const calls: unknown[] = [];
  vi.stubGlobal("fetch", async (_input: unknown, init?: RequestInit) => {
    calls.push(JSON.parse(String(init?.body)));
    const status = opts.status ?? 200;
    const body = opts.status === 400 ? opts.body : { rules: opts.body ?? undefined };
    return {
      ok: status >= 200 && status < 300,
      status,
      type: "basic",
      json: async () => body,
    } as unknown as Response;
  });
  return { lastBody: () => calls[calls.length - 1] };
}

const input = (label: string): HTMLInputElement =>
  document.querySelector<HTMLInputElement>(`[aria-label="${label}"]`)!;

const inputByValue = (label: string, value: string): void => {
  fireEvent.change(input(label), { target: { value } });
};

describe("RulesView (P-06)", () => {
  it("renders one card per rule and adds a draft on 'Add rule'", async () => {
    render(<RulesView fetcher={rulesFetcher(payload([rule({ name: "kv blast" })]))} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-rule")).toBeTruthy();
    });
    expect(document.querySelectorAll(".cf-rule")).toHaveLength(1);
    expect(input("Rule 1 name").value).toBe("kv blast");

    document.querySelector<HTMLButtonElement>(".cf-rules-actions .cf-btn")!.click();
    await waitFor(() => {
      expect(document.querySelectorAll(".cf-rule")).toHaveLength(2);
    });
    // The new draft starts empty (its own name input).
    expect(input("Rule 2 name").value).toBe("");
  });

  it("client validation: saving an empty draft highlights the card and shows the action-bar error, no PUT", async () => {
    const put = stubPut();
    render(<RulesView fetcher={rulesFetcher(payload([rule({ name: "  " })]))} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-rule")).toBeTruthy();
    });
    document.querySelector<HTMLButtonElement>(".cf-rules-actions .cf-btn-primary")!.click();
    await waitFor(() => {
      expect(document.querySelector(".cf-save-error")).toBeTruthy();
    });
    expect(document.querySelector(".cf-save-error")?.textContent).toBe("Rule 1: Name is required.");
    expect(document.querySelector(".cf-rule")?.classList.contains("cf-rule-invalid")).toBe(true);
    expect(put.lastBody()).toBeUndefined(); // nothing was PUT
  });

  it("Save PUTs the whole draft set in the wire shape and adopts the echo", async () => {
    const echoRules = [rule({ name: "kv blast (saved)", threshold: 5000, exclude: ["cf-state"], enabled: false })];
    const put = stubPut({ body: echoRules });
    render(<RulesView fetcher={rulesFetcher(payload([rule({ name: "kv blast" })]))} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-rule")).toBeTruthy();
    });
    // Edit the threshold and add an exclude chip before saving.
    inputByValue("Rule 1 threshold", "1500000000");
    inputByValue("Rule 1 add exclude", "cf-trace");
    document.querySelector<HTMLButtonElement>(".cf-field-row .cf-btn")!.click();
    await waitFor(() => {
      expect(document.querySelector(".cf-chip-value")?.textContent).toBe("cf-trace");
    });

    document.querySelector<HTMLButtonElement>(".cf-rules-actions .cf-btn-primary")!.click();
    await waitFor(() => {
      expect(document.querySelector(".cf-save-status")?.textContent).toBe("Saved");
    });
    // PUT payload shape: { rules: [...] } with the edited draft, verbatim.
    const body = put.lastBody() as { rules: AlertRule[] };
    expect(body.rules).toEqual([
      { name: "kv blast", condition: "d1-rows-read", threshold: 1500000000, exclude: ["cf-trace"], enabled: true },
    ]);
    // Echo adopted: the inputs now show the server's stored copy.
    await waitFor(() => {
      expect(input("Rule 1 name").value).toBe("kv blast (saved)");
    });
    expect(input("Rule 1 threshold").value).toBe("5000");
    expect(document.querySelector(".cf-alert-list")).toBeNull();
    expect(document.querySelector(".cf-chip-value")?.textContent).toBe("cf-state");
    expect(document.querySelector(".cf-toggle")?.textContent).toBe("Off");
  });

  it("a duplicate exclude chip is not added and remove kills it", async () => {
    render(<RulesView fetcher={rulesFetcher(payload([rule({ exclude: ["cf-state"] })]))} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-chip-value")?.textContent).toBe("cf-state");
    });
    // Duplicate: ignored (still one chip).
    inputByValue("Rule 1 add exclude", " cf-state ");
    document.querySelector<HTMLButtonElement>(".cf-field-row .cf-btn")!.click();
    expect(document.querySelectorAll(".cf-chip")).toHaveLength(1);
    // Remove it.
    document.querySelector<HTMLButtonElement>(".cf-chip-remove")!.click();
    await waitFor(() => {
      expect(document.querySelectorAll(".cf-chip")).toHaveLength(0);
    });
  });

  it("the starter note shows for the starter set and clears on save", async () => {
    const echoRules = [rule({ name: "kept" })];
    stubPut({ body: echoRules });
    render(<RulesView fetcher={rulesFetcher(payload([rule({ name: "kept" })]), true)} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-rule-note")).toBeTruthy();
    });
    expect(document.querySelector(".cf-rule-note")?.textContent).toBe("Using starter rules — save to customise");
    document.querySelector<HTMLButtonElement>(".cf-rules-actions .cf-btn-primary")!.click();
    await waitFor(() => {
      expect(document.querySelector(".cf-save-status")?.textContent).toBe("Saved");
    });
    expect(document.querySelector(".cf-rule-note")).toBeNull();
  });

  it("a 400 carries RulesHttpError with the index: 'Rule N:' label and the offending card highlighted", async () => {
    stubPut({ status: 400, body: { error: "threshold out of range", index: 1 } });
    render(
      <RulesView
        fetcher={rulesFetcher(payload([rule({ name: "one" }), rule({ name: "two", condition: "kv-writes" })]))}
      />,
    );
    await waitFor(() => {
      expect(document.querySelectorAll(".cf-rule")).toHaveLength(2);
    });
    document.querySelector<HTMLButtonElement>(".cf-rules-actions .cf-btn-primary")!.click();
    await waitFor(() => {
      expect(document.querySelector(".cf-save-error")).toBeTruthy();
    });
    expect(document.querySelector(".cf-save-error")?.textContent).toBe("Rule 2: threshold out of range");
    const cards = document.querySelectorAll(".cf-rule");
    expect(cards[0].classList.contains("cf-rule-invalid")).toBe(false);
    expect(cards[1].classList.contains("cf-rule-invalid")).toBe(true);
  });

  it("the D1 rows-read threshold gets the formatted count hint", async () => {
    render(<RulesView fetcher={rulesFetcher(payload([rule({ name: "d1", threshold: 1_000_000_000 })]))} />);
    await waitFor(() => {
      expect(document.querySelector(".cf-rule")).toBeTruthy();
    });
    const hint = document.querySelector(".cf-rule-hint")!;
    expect(hint.hidden).toBe(false);
    expect(hint.textContent).toBe("1.0B");
  });
});
