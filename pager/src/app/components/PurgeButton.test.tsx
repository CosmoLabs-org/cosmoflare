// @vitest-environment jsdom
// PurgeButton: the per-zone cache purge action. Two-step confirm — the
// first click arms (danger styling + "Confirm purge"), the second fires
// onPurge; a rejection surfaces the error text and offers Retry.
import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import PurgeButton from "./PurgeButton";

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

function button(): HTMLButtonElement {
  return document.querySelector<HTMLButtonElement>(".cf-purge .cf-btn")!;
}

describe("PurgeButton", () => {
  it("idle → armed (danger, Confirm purge) → fires onPurge with the zone → done", async () => {
    const onPurge = vi.fn().mockResolvedValue(undefined);
    render(<PurgeButton zone="noble.coffee" onPurge={onPurge} />);
    expect(button().textContent).toBe("Purge cache");
    fireEvent.click(button());
    expect(button().textContent).toBe("Confirm purge");
    expect(button().classList.contains("cf-btn-danger")).toBe(true);
    expect(onPurge).not.toHaveBeenCalled(); // armed, not fired
    fireEvent.click(button());
    await waitFor(() => {
      expect(button().textContent).toBe("Purged");
    });
    expect(onPurge).toHaveBeenCalledWith("noble.coffee");
    expect(button().disabled).toBe(false);
  });

  it("a rejected purge shows the error and a Retry label", async () => {
    const onPurge = vi.fn().mockRejectedValue(new Error("write token not configured"));
    render(<PurgeButton zone="example.dev" onPurge={onPurge} />);
    fireEvent.click(button());
    fireEvent.click(button());
    await waitFor(() => {
      expect(document.querySelector(".cf-purge-error")?.textContent).toContain("write token not configured");
    });
    expect(button().textContent).toBe("Retry purge");
    expect(document.querySelector(".cf-purge-error")?.getAttribute("role")).toBe("alert");
  });

  it("armed state resets by itself after 4s without a second click", async () => {
    vi.useFakeTimers();
    const onPurge = vi.fn();
    const { rerender } = render(<PurgeButton zone="example.dev" onPurge={onPurge} />);
    fireEvent.click(button());
    expect(button().textContent).toBe("Confirm purge");
    vi.advanceTimersByTime(4100);
    rerender(<PurgeButton zone="example.dev" onPurge={onPurge} />);
    expect(button().textContent).toBe("Purge cache"); // disarmed, nothing fired
    expect(onPurge).not.toHaveBeenCalled();
  });
});
