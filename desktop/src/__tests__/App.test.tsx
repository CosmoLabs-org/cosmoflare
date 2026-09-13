// Regression guard: App must provide a QueryClientProvider so the Dashboard's
// useQuery calls don't throw "No QueryClient set" at runtime. The Dashboard's
// own tests pass because they wrap in a provider; this exercises the real App
// tree end-to-end with the Tauri/fetch/EventSource seams stubbed.

import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import App from "../App";

// vitest hoists vi.mock above the imports, so App's transitive
// @tauri-apps/api/core import is mocked. Resolve immediately so App stops
// polling and mounts the Dashboard.
vi.mock("@tauri-apps/api/core", () => ({
  invoke: vi.fn().mockResolvedValue({ url: "http://127.0.0.1:1", token: "t" }),
}));

class FakeEventSource {
  constructor(public url: string) {}
  addEventListener() {}
  removeEventListener() {}
  close() {}
}

describe("App", () => {
  beforeEach(() => {
    vi.stubGlobal("EventSource", FakeEventSource);
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => [] }));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("mounts the Dashboard view without a QueryClient error", async () => {
    // If the provider were missing, this render would throw when the
    // Dashboard's useQuery fires after the endpoint resolves.
    render(<App />);
    await waitFor(() => {
      // "Zones" is a ServiceCard label that only renders once the Dashboard
      // view mounts with a live client.
      expect(screen.getByText("Zones")).toBeInTheDocument();
    });
  });
});

// Regression guard for the notifications-reset-on-tab-switch bug: the
// useNotifications hook must live at the App level, not inside the conditionally
// rendered panel. If it lived in the panel, leaving the Notifications view would
// unmount the hook and wipe the history + unread count.

class DispatchEventSource {
  static instances: DispatchEventSource[] = [];
  url: string;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  private handlers: Record<string, Array<(e: MessageEvent) => void>> = {};
  constructor(url: string) {
    this.url = url;
    DispatchEventSource.instances.push(this);
  }
  addEventListener(type: string, h: (e: MessageEvent) => void) {
    (this.handlers[type] ??= []).push(h);
  }
  removeEventListener() {}
  close() {}
  emit(type: string, data: unknown) {
    const evt = { data: JSON.stringify(data) } as MessageEvent;
    (this.handlers[type] ?? []).forEach((h) => h(evt));
  }
}

describe("App notifications persistence (#8)", () => {
  beforeEach(() => {
    DispatchEventSource.instances = [];
    vi.stubGlobal("EventSource", DispatchEventSource);
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => [] }));
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("keeps notification history + unread when leaving and returning to the tab", async () => {
    render(<App />);
    await waitFor(() => expect(DispatchEventSource.instances.length).toBeGreaterThan(0));

    // Daemon pushes a notification while the Dashboard tab is showing.
    act(() => {
      DispatchEventSource.instances.forEach((es) =>
        es.emit("notifications", { message: "CF back online" })
      );
    });

    // Switch to Notifications → present, unread = 1. With unread > 0 the
    // nav button's accessible name includes the count (BUG-047), so match
    // by regex rather than the exact string.
    fireEvent.click(screen.getByRole("button", { name: /Notifications/ }));
    await waitFor(() => {
      expect(screen.getByTestId("unread-badge")).toHaveTextContent("1");
      // The always-mounted live region (BUG-047) renders the newest message
      // too, so the text legitimately appears twice — assert presence.
      expect(screen.getAllByText("CF back online").length).toBeGreaterThan(0);
    });

    // Leave to Dashboard, then return to Notifications.
    fireEvent.click(screen.getByRole("button", { name: "Dashboard" }));
    await waitFor(() => expect(screen.getByText("Zones")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /Notifications/ }));

    // History + unread survived the round-trip.
    await waitFor(() => {
      expect(screen.getByTestId("unread-badge")).toHaveTextContent("1");
      // The always-mounted live region (BUG-047) renders the newest message
      // too, so the text legitimately appears twice — assert presence.
      expect(screen.getAllByText("CF back online").length).toBeGreaterThan(0);
    });
  });
});
