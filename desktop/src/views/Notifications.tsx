// Real-time notifications panel. Subscribes to the daemon's SSE `notifications`
// channel, keeps a scrollable history (capped at MAX_NOTIFICATIONS, newest
// first), and shows an unread badge cleared on demand.
//
// Notification source (v1, BR-07): daemon-internal events the daemon already
// observes — cloudflare_online transitions, per-poll errors/recoveries, alert-
// rule evaluations. The outbound webhook sender is deferred to v2 (needs an
// in-process pub/sub seam). The daemon's Publish("notifications", ...) is the
// production point; this panel is the consumer.

import { useCallback, useRef, useState } from "react";
import { useDaemonSSE } from "../api/sse";
import type { DaemonEndpoint } from "../api/client";

export interface NotificationItem {
  raw: unknown;
  message?: string;
  /** Stable id for list keys (assigned by the hook). */
  id?: number;
}

/** Maximum items retained in the scrollable history. */
export const MAX_NOTIFICATIONS = 200;

/**
 * Pure reducer: prepend newest-first and cap the history. Extracted so the cap
 * is unit-testable without driving 200 SSE events through a component.
 */
export function appendNotification(
  history: NotificationItem[],
  item: NotificationItem
): NotificationItem[] {
  return [item, ...history].slice(0, MAX_NOTIFICATIONS);
}

interface NotificationsProps {
  items: NotificationItem[];
  unread: number;
  onSeen?: () => void;
}

export function Notifications({ items, unread, onSeen }: NotificationsProps) {
  return (
    <div className="cf-notifications">
      <div className="cf-notifications-header">
        <h2 className="cf-view-title">Notifications</h2>
        <span
          className="cf-unread-badge"
          data-testid="unread-badge"
          role="status"
          aria-label={`${unread} unread notification${unread === 1 ? "" : "s"}`}
        >
          {unread}
        </span>
        <button
          type="button"
          data-testid="mark-seen"
          className="cf-mark-seen"
          onClick={onSeen}
          disabled={unread === 0}
        >
          Mark read
        </button>
      </div>
      {/* role=log + aria-live=polite (WCAG 4.1.3): incoming notifications are
          announced without stealing focus. The wrapper div keeps the ul a real
          list for assistive tech. */}
      <div className="cf-notification-log" role="log" aria-live="polite">
        {items.length === 0 ? (
          <p className="cf-empty">No notifications yet.</p>
        ) : (
          <ul className="cf-notification-list">
            {items.map((it, i) => (
              <li
                key={it.id ?? i}
                data-testid={`notification-${i}`}
                className="cf-notification-item"
              >
                {it.message ?? JSON.stringify(it.raw)}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

/** Accumulate `notifications` SSE frames into a capped history + unread count. */
export function useNotifications(endpoint: DaemonEndpoint | null) {
  const [items, setItems] = useState<NotificationItem[]>([]);
  const [unread, setUnread] = useState(0);
  const counter = useRef(0);

  useDaemonSSE(endpoint, (e) => {
    if (e.channel !== "notifications") return;
    const raw = e.data;
    const maybeMessage = (raw as { message?: unknown } | null)?.message;
    const message = typeof maybeMessage === "string" ? maybeMessage : undefined;
    counter.current += 1;
    setItems((prev) => appendNotification(prev, { raw, message, id: counter.current }));
    setUnread((u) => u + 1);
  });

  const markSeen = useCallback(() => setUnread(0), []);
  return { items, unread, markSeen };
}
