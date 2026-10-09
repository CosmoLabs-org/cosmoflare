// Real-time notifications panel. Subscribes to the daemon's SSE `notifications`
// channel, keeps a scrollable history (capped at MAX_NOTIFICATIONS, newest
// first), and shows an unread badge cleared on demand.
//
// FEAT-055 wave C: the presentation layer is rebuilt on the shadcn/ui primitives
// from wave A (Card rows, ghost Buttons, ScrollArea) — dark-first, no
// left-border accents, no tile outlines, ease-out transitions only. Every
// load-bearing contract survives: the cf-notification-item / is-<severity> /
// is-acked classes, the data-testid / role / aria-label surface, the local
// ack + snooze logic (FEAT-045), and the useNotifications hook signature.
//
// Notification source (v1, BR-07): daemon-internal events the daemon already
// observes — cloudflare_online transitions, per-poll errors/recoveries, alert-
// rule evaluations. The outbound webhook sender is deferred to v2 (needs an
// in-process pub/sub seam). The daemon's Publish("notifications", ...) is the
// production point; this panel is the consumer.

import { useCallback, useRef, useState } from "react";
import { BellOff } from "lucide-react";
import { useDaemonSSE } from "../api/sse";
import type { DaemonEndpoint } from "../api/client";
import { cn } from "../lib/utils";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ScrollArea } from "../components/ui/scroll-area";

/** FEAT-041: severity triage level carried by notification payloads. */
export type NotificationSeverity = "info" | "warning" | "critical";

const SEVERITIES: readonly NotificationSeverity[] = ["info", "warning", "critical"];

export interface NotificationItem {
  raw: unknown;
  message?: string;
  /**
   * Visual severity (FEAT-041): drives the severity dot/label color. Optional
   * because older daemon payloads omit it; anything unset renders as "info".
   * Ingest raw SSE frames through {@link parseSeverity}.
   */
  severity?: NotificationSeverity;
  /** Stable id for list keys (assigned by the hook). */
  id?: number;
  /**
   * Arrival wall-clock time in ms since epoch (assigned by the hook). Rendered
   * as a relative, tabular-nums timestamp; optional so hand-built items in
   * tests simply omit it.
   */
  timestamp?: number;
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

/**
 * Compact relative age for a row's arrival timestamp ("just now", "5m ago").
 * Pure in `now` so it stays trivially testable; rendered with tabular-nums so
 * the column of ages doesn't jitter as values tick over.
 */
export function formatRelativeTime(ts: number, now: number = Date.now()): string {
  const seconds = Math.max(0, Math.floor((now - ts) / 1000));
  if (seconds < 45) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

interface NotificationsProps {
  items: NotificationItem[];
  unread: number;
  onSeen?: () => void;
  /**
   * Wave C skeleton state: while true the list area renders pulsing
   * placeholders instead of rows (used while a first page/history loads).
   * Additive and optional — the existing hook contract never sets it.
   */
  loading?: boolean;
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

/** Number of skeleton rows shown while `loading`. */
const SKELETON_ROWS = 3;

export function Notifications({ items, unread, onSeen, loading = false }: NotificationsProps) {
  // FEAT-045: ack/snooze parity with the pager PWA, purely local — the
  // desktop panel has no ack/snooze API to call, so these maps live only in
  // component state (keyed by the same id ?? index the list keys use).
  const [acked, setAcked] = useState<Record<string, boolean>>({});
  const [snoozedUntil, setSnoozedUntil] = useState<Record<string, number>>({});
  // Per-row seen marks (wave C): the global unread counter stays the source of
  // truth — a row counts as unseen while it sits within the newest `unread`
  // arrivals and hasn't been individually dismissed. Like ack/snooze this is
  // local-only until the daemon exposes a per-item seen API.
  const [rowSeen, setRowSeen] = useState<Record<string, boolean>>({});

  // An item is snoozed while its expiry lies in the future. Evaluated at
  // render time: once the wall clock passes the timestamp, the next render
  // (any state/prop change) reveals the item again — and refreshes the
  // relative timestamps for free.
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
        <Button
          variant="ghost"
          size="sm"
          className="ml-auto"
          data-testid="mark-seen"
          onClick={onSeen}
          disabled={unread === 0}
        >
          Mark all read
        </Button>
      </div>
      {/* role=log + aria-live=polite (WCAG 4.1.3): incoming notifications are
          announced without stealing focus. The wrapper div keeps the list a
          real list for assistive tech and stays mounted in every state
          (loading / empty / rows) so the live region never unmounts. */}
      <div className="cf-notification-log" role="log" aria-live="polite">
        <ScrollArea className="max-h-[calc(100vh-14rem)]">
          {loading ? (
            // Skeleton rows (wave C): neutral raised surfaces, pulse only —
            // no bounce/elastic motion (FEAT-055 hard rule).
            <div className="flex flex-col gap-2 p-1" aria-hidden data-testid="notifications-skeleton">
              {Array.from({ length: SKELETON_ROWS }, (_, i) => (
                <div key={i} className="h-16 animate-pulse rounded-md bg-raised" />
              ))}
            </div>
          ) : items.length === 0 ? (
            <Card
              className="items-center justify-center gap-2 py-10 text-center"
              data-testid="notifications-empty"
            >
              <BellOff aria-hidden className="size-6 text-muted-foreground" />
              <p className="cf-empty">No notifications</p>
              <p className="m-0 text-xs text-muted-foreground">
                Alerts from the daemon will appear here.
              </p>
            </Card>
          ) : (
            <ul className="cf-notification-list">
              {items.map((it, i) => {
                const key = itemKey(it, i);
                // Snoozed items are hidden until their expiry passes; the
                // testid keeps the original items index so identities stay
                // stable across snooze/un-snooze transitions.
                if (snoozedKeys.has(key)) return null;
                const severity = severityOf(it);
                const isAcked = acked[key] === true;
                // Unseen rows carry the accent dot + semibold message until
                // dismissed (per-row) or the unread count is cleared (all).
                const isUnseen = i < unread && !rowSeen[key];
                return (
                  <li key={key}>
                    <Card
                      data-testid={`notification-${i}`}
                      data-severity={severity}
                      data-acked={isAcked ? "true" : undefined}
                      data-unseen={isUnseen ? "true" : undefined}
                      className={cn(
                        "cf-notification-item",
                        `is-${severity}`,
                        isAcked && "is-acked"
                      )}
                    >
                      <div className="flex items-center gap-2">
                        <span className="cf-notification-severity">
                          <span className="cf-severity-mark" aria-hidden />
                          <span className="cf-severity-label">{severity}</span>
                        </span>
                        {isUnseen && (
                          <span
                            aria-hidden
                            className="size-2 shrink-0 rounded-full bg-accent"
                          />
                        )}
                        {it.timestamp !== undefined && (
                          <span className="ml-auto text-xs tabular-nums text-muted-foreground">
                            {formatRelativeTime(it.timestamp, now)}
                          </span>
                        )}
                      </div>
                      <span
                        className={cn(
                          "cf-notification-message",
                          isUnseen && "font-semibold"
                        )}
                      >
                        {it.message ?? JSON.stringify(it.raw)}
                      </span>
                      {/* FEAT-045: local triage actions, mirroring the pager
                          PWA. Acknowledge dims + strikes the item; Snooze
                          hides it for SNOOZE_MINUTES. Mark seen dismisses the
                          row's unread dot. No network, no persistence. */}
                      <div className="cf-notification-actions">
                        {isUnseen && (
                          <Button
                            variant="ghost"
                            size="sm"
                            data-testid={`seen-${i}`}
                            onClick={() =>
                              setRowSeen((prev) => ({ ...prev, [key]: true }))
                            }
                          >
                            Mark seen
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          data-testid={`ack-${i}`}
                          onClick={() =>
                            setAcked((prev) => ({ ...prev, [key]: true }))
                          }
                        >
                          Acknowledge
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          data-testid={`snooze-${i}`}
                          onClick={() =>
                            setSnoozedUntil((prev) => ({
                              ...prev,
                              [key]: Date.now() + SNOOZE_MINUTES * 60_000,
                            }))
                          }
                        >
                          Snooze 10m
                        </Button>
                      </div>
                    </Card>
                  </li>
                );
              })}
            </ul>
          )}
        </ScrollArea>
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
      appendNotification(prev, {
        raw,
        message,
        severity,
        id: counter.current,
        timestamp: Date.now(),
      })
    );
    setUnread((u) => u + 1);
  });

  const markSeen = useCallback(() => setUnread(0), []);
  return { items, unread, markSeen };
}
