// Accessibility gate (BUG-036/037/038 audit follow-up).
//
// Two layers:
//   1. axe-core via vitest-axe (`toPassAxe`) — catches general ARIA/name/role
//      regressions on the real component trees.
//   2. Manual structural assertions for the specific audit findings that axe
//      cannot see (aria-live regions, h1 landmark structure, aria-current on
//      the active nav item, the connecting status pane).
//
// axe's `color-contrast` rule is disabled per-run: jsdom loads no stylesheet,
// so computed colors are UA defaults and the rule cannot produce signal.
// Page-level structure rules are additionally disabled for the fragment
// renders (a lone Notifications view has no landmarks by design).

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { axe } from "vitest-axe";
import { invoke } from "@tauri-apps/api/core";
import App from "../App";
import { Notifications } from "../views/Notifications";

// vitest hoists vi.mock above the imports, so App's transitive
// @tauri-apps/api/core import is mocked. Each describe sets the resolved
// value on the shared mock (endpoint vs null → "Connecting…" pane).
vi.mock("@tauri-apps/api/core", () => ({ invoke: vi.fn() }));

class FakeEventSource {
  constructor(_url: string) {}
  addEventListener() {}
  removeEventListener() {}
  close() {}
}

const FRAGMENT_RULES = {
  rules: {
    "color-contrast": { enabled: false },
    region: { enabled: false },
    "landmark-one-main": { enabled: false },
    "page-has-heading-one": { enabled: false },
  },
} as const;

const PAGE_RULES = { rules: { "color-contrast": { enabled: false } } } as const;

describe("a11y: full app (axe)", () => {
  beforeEach(() => {
    vi.mocked(invoke).mockResolvedValue({ url: "http://127.0.0.1:1", token: "t" });
    vi.stubGlobal("EventSource", FakeEventSource);
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => [] }));
  });
  afterEach(() => vi.unstubAllGlobals());

  it("has no axe violations (BUG-036 audit gate)", async () => {
    const { container } = render(<App />);
    await waitFor(() => expect(screen.getByText("Zones")).toBeInTheDocument());
    const results = await axe(container, PAGE_RULES);
    expect(results).toHaveNoViolations();
  });

  it("marks the active nav item with aria-current=page", async () => {
    render(<App />);
    await waitFor(() => expect(screen.getByText("Zones")).toBeInTheDocument());

    // Dashboard is the initial view.
    expect(screen.getByRole("button", { name: "Dashboard" })).toHaveAttribute(
      "aria-current",
      "page"
    );
    expect(screen.getByRole("button", { name: "Notifications" })).not.toHaveAttribute(
      "aria-current"
    );

    // Switching moves the current-page marker.
    fireEvent.click(screen.getByRole("button", { name: "Notifications" }));
    expect(screen.getByRole("button", { name: "Notifications" })).toHaveAttribute(
      "aria-current",
      "page"
    );
    expect(screen.getByRole("button", { name: "Dashboard" })).not.toHaveAttribute(
      "aria-current"
    );
  });

  it("renders the sidebar as a nav landmark labelled Views", async () => {
    render(<App />);
    await waitFor(() =>
      expect(screen.getByRole("navigation", { name: "Views" })).toBeInTheDocument()
    );
  });
});

describe("a11y: connecting pane (BUG-038)", () => {
  beforeEach(() => {
    // daemon_endpoint not ready yet → App stays in the connecting state.
    vi.mocked(invoke).mockResolvedValue(null);
    vi.stubGlobal("EventSource", FakeEventSource);
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => [] }));
  });
  afterEach(() => vi.unstubAllGlobals());

  it("announces the connecting state through role=status", () => {
    render(<App />);
    expect(screen.getByRole("status")).toHaveTextContent(/connecting to daemon/i);
  });
});

describe("a11y: notifications fragment (axe)", () => {
  it("has no axe violations on the fragment render", async () => {
    const { container } = render(
      <Notifications items={[{ raw: { message: "Cloudflare back online" } }]} unread={1} />
    );
    const results = await axe(container, FRAGMENT_RULES);
    expect(results).toHaveNoViolations();
  });
});

// --- BUG-047: notifications must be announced on EVERY view (WCAG 4.1.3) ---
//
// Per-view live regions unmount on tab switch, so incoming notifications are
// silent while any non-Notifications view is active. The fix is an
// always-mounted, visually-hidden polite live region at the App level plus
// the unread count in the nav button's accessible name.

describe("a11y: notification announcements (BUG-047)", () => {
  let notifListeners: Array<(e: { data: string }) => void>;
  class NotifEventSource {
    constructor(_url: string) {}
    addEventListener(channel: string, h: EventListener) {
      if (channel === "notifications")
        notifListeners.push(h as unknown as (e: { data: string }) => void);
    }
    removeEventListener() {}
    close() {}
  }
  const emit = (msg: string) =>
    notifListeners.forEach((h) => h({ data: JSON.stringify({ message: msg }) }));

  beforeEach(() => {
    notifListeners = [];
    vi.mocked(invoke).mockResolvedValue({ url: "http://127.0.0.1:1", token: "t" });
    vi.stubGlobal("EventSource", NotifEventSource);
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => [] }));
  });
  afterEach(() => vi.unstubAllGlobals());

  it("announces notifications from an always-mounted polite live region on the Dashboard view", async () => {
    render(<App />);
    await waitFor(() => expect(screen.getByText("Zones")).toBeInTheDocument());

    emit("Backups quota at 90%");
    await waitFor(() => {
      const region = document.querySelector('[aria-live="polite"].cf-live-region');
      expect(region).not.toBeNull();
      expect(region?.textContent).toContain("Backups quota at 90%");
    });
  });

  it("keeps the live region mounted while the Notifications view is active", async () => {
    render(<App />);
    await waitFor(() => expect(screen.getByText("Zones")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Notifications" }));

    emit("Cloudflare back online");
    await waitFor(() => {
      const region = document.querySelector('[aria-live="polite"].cf-live-region');
      expect(region).not.toBeNull();
      expect(region?.textContent).toContain("Cloudflare back online");
    });
  });

  it("names the nav button with the unread count once notifications arrive", async () => {
    render(<App />);
    await waitFor(() => expect(screen.getByText("Zones")).toBeInTheDocument());

    // Zero unread: plain accessible name (existing tests rely on it).
    expect(screen.getByRole("button", { name: "Notifications" })).toBeInTheDocument();

    emit("Zone record missing");
    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: /Notifications, 1 unread/i })
      ).toBeInTheDocument()
    );
  });
});
