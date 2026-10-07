import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Notifications, type NotificationItem } from "../views/Notifications";

// FEAT-045 Task 8: desktop ack/snooze parity with the pager PWA. Ack/snooze is
// purely local component state — no API calls, no persistence: acked items are
// dimmed + struck through, snoozed items are hidden until the window expires.

const ITEMS: NotificationItem[] = [
  { raw: { message: "zone offline" }, severity: "critical" },
  { raw: { message: "quota at 80%" }, severity: "warning" },
];

describe("Notifications ack/snooze parity (FEAT-045)", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("starts unacked with a Snooze 10m action and no summary row", () => {
    render(<Notifications items={ITEMS} unread={2} />);
    const item = screen.getByTestId("notification-0");
    expect(item).not.toHaveAttribute("data-acked");
    expect(screen.getAllByRole("button", { name: "Acknowledge" })).toHaveLength(
      2
    );
    expect(
      screen.getAllByRole("button", { name: "Snooze 10m" })
    ).toHaveLength(2);
    expect(screen.queryByTestId("snoozed-summary")).not.toBeInTheDocument();
  });

  it("marks the clicked item acked: data-acked=true with a dimmed class", () => {
    render(<Notifications items={ITEMS} unread={2} />);
    fireEvent.click(screen.getAllByRole("button", { name: "Acknowledge" })[0]);
    const first = screen.getByTestId("notification-0");
    expect(first).toHaveAttribute("data-acked", "true");
    expect(first).toHaveClass("is-acked");
    // The sibling is untouched and stays visible.
    expect(screen.getByTestId("notification-1")).not.toHaveAttribute(
      "data-acked"
    );
    // Acked items stay in the list — dimmed, not removed.
    expect(screen.getAllByTestId(/^notification-/)).toHaveLength(2);
  });

  it("hides a snoozed item from the list and shows a Snoozed (n) summary row", () => {
    render(<Notifications items={ITEMS} unread={2} />);
    fireEvent.click(screen.getAllByRole("button", { name: "Snooze 10m" })[0]);
    // The snoozed item is gone from the list...
    expect(screen.queryByTestId("notification-0")).not.toBeInTheDocument();
    // ...its sibling is still rendered under its own index...
    expect(screen.getByTestId("notification-1")).toBeInTheDocument();
    // ...and a summary row reports how many are currently snoozed.
    expect(screen.getByTestId("snoozed-summary")).toHaveTextContent(
      "Snoozed (1)"
    );
  });

  it("reveals a snoozed item and drops the summary once 10 minutes pass", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-10-07T12:00:00Z"));
    // Host forces a re-render after the clock moves: the expiry check runs on
    // render via Date.now(), and nothing else in the tree changes by itself.
    function Host() {
      const [, setTick] = useState(0);
      return (
        <>
          <button type="button" onClick={() => setTick((t) => t + 1)}>
            tick
          </button>
          <Notifications items={ITEMS} unread={2} />
        </>
      );
    }
    render(<Host />);
    fireEvent.click(screen.getAllByRole("button", { name: "Snooze 10m" })[0]);
    expect(screen.queryByTestId("notification-0")).not.toBeInTheDocument();
    expect(screen.getByTestId("snoozed-summary")).toHaveTextContent(
      "Snoozed (1)"
    );

    vi.setSystemTime(new Date("2026-10-07T12:10:01Z"));
    fireEvent.click(screen.getByRole("button", { name: "tick" }));
    expect(screen.getByTestId("notification-0")).toBeInTheDocument();
    expect(screen.queryByTestId("snoozed-summary")).not.toBeInTheDocument();
  });
});
