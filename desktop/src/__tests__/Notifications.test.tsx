import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import {
  MAX_NOTIFICATIONS,
  Notifications,
  appendNotification,
  type NotificationItem,
  useNotifications,
} from "../views/Notifications";
import type { DaemonEndpoint } from "../api/client";

// --- pure reducer ---

describe("appendNotification", () => {
  it("prepends newest-first and caps at MAX_NOTIFICATIONS", () => {
    let hist: NotificationItem[] = [];
    hist = appendNotification(hist, { raw: { message: "a" } });
    hist = appendNotification(hist, { raw: { message: "b" } });
    expect(hist.length).toBe(2);
    expect(hist[0].raw).toEqual({ message: "b" });

    // Fill to the cap and verify it holds.
    for (let i = 0; i < MAX_NOTIFICATIONS + 5; i++) {
      hist = appendNotification(hist, { raw: { i } });
    }
    expect(hist.length).toBe(MAX_NOTIFICATIONS);
  });
});

// --- presentational view ---

describe("Notifications (view)", () => {
  it("renders the items and an unread badge", () => {
    render(
      <Notifications
        items={[
          { raw: { message: "Cloudflare back online" } },
          { raw: { message: "Quota at 80%" } },
        ]}
        unread={2}
      />
    );
    expect(screen.getAllByTestId(/^notification-/)).toHaveLength(2);
    expect(screen.getByTestId("unread-badge")).toHaveTextContent("2");
  });

  it("calls onSeen when the mark-seen button is clicked", () => {
    let seen = 0;
    render(
      <Notifications items={[{ raw: { message: "x" } }]} unread={1} onSeen={() => (seen = 1)} />
    );
    fireEvent.click(screen.getByTestId("mark-seen"));
    expect(seen).toBe(1);
  });

  // BUG-038 (WCAG 4.1.3): incoming notifications must be announced. The list
  // wrapper is a polite live log, and the unread badge announces a described
  // count instead of a bare number.
  it("exposes the notification list as a polite live log (BUG-038)", () => {
    render(<Notifications items={[{ raw: { message: "x" } }]} unread={1} />);
    const log = screen.getByRole("log");
    expect(log).toHaveAttribute("aria-live", "polite");
  });

  it("announces the unread count with a descriptive label (BUG-038)", () => {
    render(
      <Notifications
        items={[
          { raw: { message: "a" } },
          { raw: { message: "b" } },
          { raw: { message: "c" } },
        ]}
        unread={3}
      />
    );
    const badge = screen.getByRole("status");
    expect(badge).toHaveAttribute("aria-label", "3 unread notifications");
    expect(badge).toHaveTextContent("3");
  });

  it("uses singular wording for exactly one unread notification", () => {
    render(<Notifications items={[{ raw: { message: "x" } }]} unread={1} />);
    expect(screen.getByRole("status")).toHaveAttribute(
      "aria-label",
      "1 unread notification"
    );
  });
});

// --- SSE accumulation via a mocked EventSource ---

class MockEventSource {
  static instances: MockEventSource[] = [];
  url: string;
  private handlers: Record<string, Array<(e: MessageEvent) => void>> = {};
  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
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

const fakeEndpoint: DaemonEndpoint = { url: "http://127.0.0.1:1", token: "t" };

function HookHost() {
  const { items, unread, markSeen } = useNotifications(fakeEndpoint);
  return <Notifications items={items} unread={unread} onSeen={markSeen} />;
}

describe("useNotifications (SSE)", () => {
  beforeEach(() => {
    MockEventSource.instances = [];
    (globalThis as unknown as { EventSource: typeof EventSource }).EventSource =
      MockEventSource as unknown as typeof EventSource;
  });
  afterEach(() => {
    // leave EventSource in place; jsdom never had a real one anyway
  });

  it("accumulates notifications from the SSE channel and tracks unread", async () => {
    render(<HookHost />);
    await waitFor(() => expect(MockEventSource.instances.length).toBe(1));
    act(() => {
      MockEventSource.instances[0].emit("notifications", { message: "CF back online" });
      MockEventSource.instances[0].emit("notifications", { message: "Quota 80%" });
    });
    await waitFor(() => {
      expect(screen.getByTestId("unread-badge")).toHaveTextContent("2");
      expect(screen.getAllByTestId(/^notification-/)).toHaveLength(2);
    });
  });

  it("clears the unread badge on mark-seen but keeps history", async () => {
    render(<HookHost />);
    await waitFor(() => expect(MockEventSource.instances.length).toBe(1));
    act(() => MockEventSource.instances[0].emit("notifications", { message: "hi" }));
    await waitFor(() => expect(screen.getByTestId("unread-badge")).toHaveTextContent("1"));
    fireEvent.click(screen.getByTestId("mark-seen"));
    expect(screen.getByTestId("unread-badge")).toHaveTextContent("0");
    expect(screen.getAllByTestId(/^notification-/)).toHaveLength(1);
  });

  it("ignores non-notification channels", async () => {
    render(<HookHost />);
    await waitFor(() => expect(MockEventSource.instances.length).toBe(1));
    act(() => {
      MockEventSource.instances[0].emit("status", { cloudflare_online: true });
      MockEventSource.instances[0].emit("metrics", { zones: 3 });
    });
    // No notification items should appear.
    expect(screen.queryAllByTestId(/^notification-/)).toHaveLength(0);
    expect(screen.getByTestId("unread-badge")).toHaveTextContent("0");
  });
});
