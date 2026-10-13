// @vitest-environment jsdom
// UpdateModal: the centered dialog shown when a new service worker takes
// control of an already-controlled page (a deploy landed while the app was
// open). Covers the dialog contract (title, actions, Escape/backdrop
// dismiss, reload callback) and the useSwUpdateReady wiring: only a page
// that was ALREADY controlled treats controllerchange as an update — a
// first install also fires the event and must stay silent.
import { cleanup, fireEvent, render, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import UpdateModal, { useSwUpdateReady } from "./UpdateModal";

// Minimal serviceWorker stand-in: a real EventTarget with the two fields
// the hook touches. jsdom ships no serviceWorker at all.
function stubServiceWorker(controller: unknown): EventTarget & { controller: unknown } {
  const sw = new EventTarget() as EventTarget & { controller: unknown };
  sw.controller = controller;
  Object.defineProperty(navigator, "serviceWorker", {
    configurable: true,
    writable: true,
    value: sw,
  });
  return sw;
}

const realServiceWorkerDescriptor = Object.getOwnPropertyDescriptor(Navigator.prototype, "serviceWorker");

afterEach(() => {
  cleanup();
  if (realServiceWorkerDescriptor) {
    Object.defineProperty(Navigator.prototype, "serviceWorker", realServiceWorkerDescriptor);
  } else {
    // @ts-expect-error restoring the pre-stub state on engines without one
    delete navigator.serviceWorker;
  }
});

describe("UpdateModal", () => {
  it("renders the centered dialog: eyebrow, title, body, and both actions", () => {
    render(<UpdateModal onReload={() => undefined} onDismiss={() => undefined} />);
    const dialog = document.querySelector('[role="dialog"]');
    expect(dialog).toBeTruthy();
    expect(dialog?.getAttribute("aria-modal")).toBe("true");
    expect(dialog?.querySelector(".cf-modal-eyebrow")?.textContent).toBe("CosmoLabs Ops");
    expect(document.getElementById("cf-update-title")?.textContent).toBe("Update available");
    const buttons = document.querySelectorAll(".cf-modal-actions .cf-btn");
    expect(buttons).toHaveLength(2);
    expect(buttons[0].textContent).toBe("Update now");
    expect(buttons[1].textContent).toBe("Later");
  });

  it("Update now shows the in-button progress, then reloads; Later dismisses", () => {
    const onReload = vi.fn();
    const onDismiss = vi.fn();
    vi.useFakeTimers();
    try {
      render(<UpdateModal onReload={onReload} onDismiss={onDismiss} />);
      const [update, later] = Array.from(document.querySelectorAll<HTMLButtonElement>(".cf-modal-actions .cf-btn"));
      later.click();
      expect(onDismiss).toHaveBeenCalledTimes(1);
      fireEvent.click(update);
      // The button becomes the progress: disabled, sweeping fill, label.
      expect(update.disabled).toBe(true);
      expect(document.querySelector(".cf-update-progress-fill")).toBeTruthy();
      expect(update.textContent).toContain("Updating…");
      expect(onReload).not.toHaveBeenCalled();
      // The reload fires after the sweep (950ms beat).
      vi.advanceTimersByTime(1000);
      expect(onReload).toHaveBeenCalledTimes(1);
    } finally {
      vi.useRealTimers();
    }
  });

  it("Escape and backdrop click dismiss; a click inside the dialog does not", () => {
    const onDismiss = vi.fn();
    render(<UpdateModal onReload={() => undefined} onDismiss={onDismiss} />);
    fireEvent.click(document.querySelector(".cf-modal")!);
    expect(onDismiss).not.toHaveBeenCalled();
    fireEvent.click(document.querySelector(".cf-modal-backdrop")!);
    expect(onDismiss).toHaveBeenCalledTimes(1);
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onDismiss).toHaveBeenCalledTimes(2);
  });
});

describe("useSwUpdateReady", () => {
  it("flips true when a new worker takes control of an already-controlled page", async () => {
    const sw = stubServiceWorker({ state: "activated" });
    const { result } = renderHook(() => useSwUpdateReady());
    expect(result.current).toBe(false);
    sw.dispatchEvent(new Event("controllerchange"));
    await waitFor(() => {
      expect(result.current).toBe(true);
    });
  });

  it("stays silent on a first install (page was not controlled when it loaded)", async () => {
    const sw = stubServiceWorker(null);
    const { result } = renderHook(() => useSwUpdateReady());
    sw.dispatchEvent(new Event("controllerchange"));
    // handler ran, but wasControlled=false: no update surfaced
    await new Promise((resolve) => setTimeout(resolve, 10));
    expect(result.current).toBe(false);
  });

  it("no serviceWorker support: hook stays false and never throws", () => {
    const { result } = renderHook(() => useSwUpdateReady());
    expect(result.current).toBe(false);
  });
});
