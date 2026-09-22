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

/** FEAT-041: severity triage level carried by notification payloads. */
export type NotificationSeverity = "info" | "warning" | "critical";

const SEVERITIES: readonly NotificationSeverity[] = ["info", "warning", "critical"];

export interface NotificationItem {
  raw: unknown;
  message?: string;
  /**
   * Visual severity (FEAT-041): drives the left-border color and the severity
   * dot/label. Optional because older daemon payloads omit it; anything unset
   * renders as "info". Ingest raw SSE frames through {@link parseSeverity}.
   */
  severity?: NotificationSeverity;
  /** Stable id for list keys (assigned by the hook). */
  id?: number;
}

/** Maximum items retained in the scrollable history. */
export const MAX_NOTIFICATIONS = 200;

/**
 * Defensive severity extraction: anything missing, null, or outside the known
 * set falls back to "info" — an old daemon streaming pre-FEAT-041 payloads
 * must render identically to before, never crash the panel.
 */
export function parseSeverity(value: unknown): NotificationSeverity {
  return typeof value === "string" && (SEVERITIES as readonly string[]).includes(value)
    ? (value as NotificationSeverity)
    : "info";
}

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

/** Normalize an item's severity, defaulting pre-FEAT-041 items to "info". */
function severityOf(it: NotificationItem): NotificationSeverity {
  return it.severity ?? "info";
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
            {items.map((it, i) => {
              const severity = severityOf(it);
              return (
                <li
                  key={it.id ?? i}
                  data-testid={`notification-${i}`}
                  data-severity={severity}
                  className={`cf-notification-item is-${severity}`}
                >
                  <span className="cf-notification-severity">
                    <span className="cf-severity-mark" aria-hidden />
                    <span className="cf-severity-label">{severity}</span>
                  </span>
                  <span className="cf-notification-message">
                    {it.message ?? JSON.stringify(it.raw)}
                  </span>
                </li>
              );
            })}
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
    // FEAT-041: severity is optional on the wire — parseSeverity defaults
    // pre-FEAT-041 payloads to "info" so they render unchanged.
    const maybeSeverity = (raw as { severity?: unknown } | null)?.severity;
    const severity = parseSeverity(maybeSeverity);
    counter.current += 1;
    setItems((prev) =>
      appendNotification(prev, { raw, message, severity, id: counter.current })
    );
    setUnread((u) => u + 1);
  });

  const markSeen = useCallback(() => setUnread(0), []);
  return { items, unread, markSeen };
}
