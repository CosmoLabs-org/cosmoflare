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

/**
 * FEAT-045: snooze window offered by the "Snooze 10m" action. The item stays
 * hidden until `snoozed_at + SNOOZE_MINUTES` passes, then re-appears on the
 * next render with no summary-row entry.
 */
export const SNOOZE_MINUTES = 10;

/** Stable per-item key for the local ack/snooze maps (mirrors the list key). */
function itemKey(it: NotificationItem, i: number): string {
  return String(it.id ?? i);
}

export function Notifications({ items, unread, onSeen }: NotificationsProps) {
  // FEAT-045: ack/snooze parity with the pager PWA, purely local — the
  // desktop panel has no ack/snooze API to call, so these maps live only in
  // component state (keyed by the same id ?? index the list keys use).
  const [acked, setAcked] = useState<Record<string, boolean>>({});
  const [snoozedUntil, setSnoozedUntil] = useState<Record<string, number>>({});

  // An item is snoozed while its expiry lies in the future. Evaluated at
  // render time: once the wall clock passes the timestamp, the next render
  // (any state/prop change) reveals the item again.
  const now = Date.now();
  const snoozedKeys = new Set(
    Object.keys(snoozedUntil).filter((k) => (snoozedUntil[k] ?? 0) > now)
  );
  const snoozedCount = items.filter((it, i) =>
    snoozedKeys.has(itemKey(it, i))
  ).length;

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
          <>
            <ul className="cf-notification-list">
              {items.map((it, i) => {
                const key = itemKey(it, i);
                // Snoozed items are hidden until their expiry passes; the
                // testid keeps the original items index so identities stay
                // stable across snooze/un-snooze transitions.
                if (snoozedKeys.has(key)) return null;
                const severity = severityOf(it);
                const isAcked = acked[key] === true;
                return (
                  <li
                    key={key}
                    data-testid={`notification-${i}`}
                    data-severity={severity}
                    data-acked={isAcked ? "true" : undefined}
                    className={`cf-notification-item is-${severity}${
                      isAcked ? " is-acked" : ""
                    }`}
                  >
                    <span className="cf-notification-severity">
                      <span className="cf-severity-mark" aria-hidden />
                      <span className="cf-severity-label">{severity}</span>
                    </span>
                    <span className="cf-notification-message">
                      {it.message ?? JSON.stringify(it.raw)}
                    </span>
                    {/* FEAT-045: local triage actions, mirroring the pager
                        PWA. Acknowledge dims + strikes the item; Snooze hides
                        it for SNOOZE_MINUTES. No network, no persistence. */}
                    <span className="cf-notification-actions">
                      <button
                        type="button"
                        data-testid={`ack-${i}`}
                        className="cf-ack-btn"
                        onClick={() =>
                          setAcked((prev) => ({ ...prev, [key]: true }))
                        }
                      >
                        Acknowledge
                      </button>
                      <button
                        type="button"
                        data-testid={`snooze-${i}`}
                        className="cf-snooze-btn"
                        onClick={() =>
                          setSnoozedUntil((prev) => ({
                            ...prev,
                            [key]: Date.now() + SNOOZE_MINUTES * 60_000,
                          }))
                        }
                      >
                        Snooze 10m
                      </button>
                    </span>
                  </li>
                );
              })}
            </ul>
            {snoozedCount > 0 && (
              <p
                className="cf-snoozed-summary"
                data-testid="snoozed-summary"
                role="status"
                aria-label={`${snoozedCount} snoozed notification${
                  snoozedCount === 1 ? "" : "s"
                }`}
              >
                Snoozed ({snoozedCount})
              </p>
            )}
          </>
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
