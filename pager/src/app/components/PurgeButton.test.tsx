// @vitest-environment jsdom
// PurgeButton → LeverButton: the purge lever still two-step confirms, fires
// onPurge with the zone, flashes "Purged", and surfaces rejections.
import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import PurgeButton from "./PurgeButton";

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

function button(): HTMLButtonElement {
  return document.querySelector<HTMLButtonElement>(".cf-purge .cf-lever")!;
}

describe("PurgeButton", () => {
  it("idle → armed (danger, Confirm purge) → fires onPurge with the zone → done flash", async () => {
    const onPurge = vi.fn().mockResolvedValue(undefined);
    render(<PurgeButton zone="noble.coffee" onPurge={onPurge} />);
    expect(button().textContent).toBe("Purge cache");
    fireEvent.click(button());
    expect(button().textContent).toBe("Confirm purge");
    expect(button().classList.contains("is-armed")).toBe(true);
    expect(onPurge).not.toHaveBeenCalled(); // armed, not fired
    fireEvent.click(button());
    await waitFor(() => {
      expect(button().textContent).toBe("Purged");
    });
    expect(onPurge).toHaveBeenCalledWith("noble.coffee");
  });

  it("a rejected purge shows the error and a Retry label", async () => {
    const onPurge = vi.fn().mockRejectedValue(new Error("write token not configured"));
    render(<PurgeButton zone="example.dev" onPurge={onPurge} />);
    fireEvent.click(button());
    fireEvent.click(button());
    await waitFor(() => {
      expect(document.querySelector(".cf-lever-error")?.textContent).toContain("write token not configured");
    });
    expect(button().textContent).toBe("Retry purge cache");
    expect(document.querySelector(".cf-lever-error")?.getAttribute("role")).toBe("alert");
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
