// Notifications center (P-06 continuation): the fired-alert FEED from the
// Ops Worker's /api/alerts/feed first — newest-first rows with a relative
// time, the rule title, the value/threshold line and an unread dot for
// entries newer than the localStorage "pager-alerts-seen" timestamp
// (mark-all-read on render, quota-safe) — then the existing push-history
// section unchanged (IndexedDB records via the store module).
// Clicking a feed row expands an inline detail (motion, DomainProfile
// section patterns): the numbers table, a plain-language explanation for the
// condition, and a muted footer with the rule id and delivery state.
// The section masthead owns the page title.

import { useCallback, useEffect, useRef, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api } from "../../api";
import { acknowledge, listAlerts, snooze, type AlertRecord } from "../../store";
import { relativeTime } from "../../views";
import type { AlertFeed, AlertFeedEntry } from "../../alerts";
import AgentCommandsModal from "../components/AgentCommandsModal";
import type { ViewFetcher } from "./WorkersView";

/** Same mapping as views.ts's alertCard: unknown severities read as info. */
function severityClass(severity: string): string {
  return severity === "warning" || severity === "critical" ? severity : "info";
}

// SEEN_KEY is the localStorage timestamp of the last rendered feed: entries
// newer than it carry the unread dot. Both accesses are quota-safe — a
// throwing localStorage (private mode, full quota) reads as "all read" and
// the mark-all-read write becomes a no-op.
const SEEN_KEY = "pager-alerts-seen";

function readSeen(): number {
  try {
    const raw = window.localStorage.getItem(SEEN_KEY);
    const n = raw === null ? NaN : Number(raw);
    return Number.isFinite(n) ? n : 0;
  } catch {
    return 0;
  }
}

function markSeen(now: number): void {
  try {
    window.localStorage.setItem(SEEN_KEY, String(now));
  } catch {
    // Quota/privacy failure: the dot simply returns next session.
  }
  // Clear the home-screen badge the service worker set on push (feature-
  // detected; a throwing clearBadge must not break the render).
  try {
    void navigator.clearAppBadge?.();
  } catch {
    // Badging unsupported: nothing to clear.
  }
}

/** Feed numbers: % renders one decimal (the wire carries full float precision
 *  — "50.40650406504065" must never reach the screen); counts stay plain. */
function feedNum(v: number, unit: string): string {
  return unit === "%" ? v.toFixed(1) : String(v);
}

/** Local relative-time helper for feed rows: "14m ago", "2h ago", "3d ago". */function timeAgo(ts: string, nowMs: number = Date.now()): string {
  const at = Date.parse(ts);
  if (!Number.isFinite(at)) return "";
  const seconds = Math.max(0, Math.floor((nowMs - at) / 1000));
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

// EXPLANATIONS: one plain-language entry per rules.ts condition id (see
// apps/ops/src/rules.ts CONDITIONS) plus the telemetry-gap pseudo condition:
// what the metric means, why crossing it matters operationally, what to do
// next. Unknown conditions fall back to the generic line.
const EXPLANATIONS: Record<string, string> = {
  "zone-uncached-requests":
    "Uncached requests are the hits Cloudflare had to pass through to your origin because nothing in its cache could answer them. When the volume climbs, your servers are absorbing traffic that should have been absorbed at the edge, which usually means cache rules need tuning or something is hammering a path you thought was cached. Check which URL patterns dominate the traffic and add a Cache Rule for them, or investigate the source of the surge.",
  "zone-cache-miss-pct":
    "Cache miss percentage is the share of requests the edge could not serve from cache, measured only for zones with enough traffic to judge. A high miss ratio means most visitors are waiting on your origin instead of the nearby edge node, which raises latency and origin load at the same time. Review your cache eligibility rules and cache key settings — most fixes are a matter of marking more responses as cacheable.",
  "d1-rows-read":
    "Rows read counts every row your D1 database scanned over the last 24 hours, which is how D1 bills you. A spike usually means a query lost its index and is table-scanning, so the same traffic suddenly costs far more and runs slower. Check your query metrics for the expensive statements and add the missing index before the bill does.",
  "kv-writes":
    "KV writes counts the write operations across a namespace in a day, one of the metered KV classes. Crossing the threshold means something is writing far more often than a cache layer should — a hot key rewritten per request, a session store under load, or a retry loop. Find the writer (per-namespace usage in the Workers view) and batch or raise its write interval before the day's usage bill lands.",
  "telemetry-gap":
    "A telemetry gap means Cloudflare's analytics for one dataset were unavailable, so alert rules that depend on it could not be evaluated this cycle. Nothing is necessarily wrong with your traffic — but for the duration of the gap you are blind to those conditions. It clears on its own when the analytics API recovers; if gaps repeat, check Cloudflare's status page.",
};

function explanationFor(condition: string): string {
  return EXPLANATIONS[condition] ?? "This alert came from a condition this build has no explanation text for — see the rule in the Rules section.";
}

/** One push-history card (unchanged section): severity mark + title,
 *  service/time meta, detail, and the Acknowledge / Snooze actions. */
function AlertCard({ record, onChanged }: { record: AlertRecord; onChanged: () => void }): React.JSX.Element {
  const ack = (): void => {
    void acknowledge(record.id).then(onChanged);
  };
  const snz = (): void => {
    void snooze(record.id, 10).then(onChanged);
  };
  return (
    <article
      className={`cf-alert is-${severityClass(record.severity)}`}
      data-id={record.id}
      data-acked={record.acked ? "true" : undefined}
    >
      <div className="cf-alert-header">
        <span className="cf-severity-mark">{record.severity}</span>
        <strong className="cf-alert-title">{record.title}</strong>
      </div>
      <div className="cf-alert-meta">
        <span className="cf-alert-service">{record.service}</span>
        <time className="cf-alert-time">{relativeTime(record.received_at)}</time>
      </div>
      <p className="cf-alert-detail">{record.detail}</p>
      <div className="cf-alert-actions">
        <button type="button" className="cf-btn" onClick={ack}>
          {record.acked ? "Acknowledged" : "Acknowledge"}
        </button>
        <button type="button" className="cf-btn cf-btn-ghost" onClick={snz}>Snooze 10m</button>
      </div>
    </article>
  );
}

/** The zone a feed entry fired for: the scope segment of the ruleId
 *  ("<ruleName>/<zoneKey>", zoneKey = zone id or name — rules.ts alertId).
 *  Everything after the LAST slash: rule names are operator-chosen and may
 *  contain slashes just as they contain spaces. */
function zoneOf(entry: AlertFeedEntry): string {
  return entry.ruleId.slice(entry.ruleId.lastIndexOf("/") + 1);
}

/** Inline detail under an expanded feed row: the numbers table, the
 *  plain-language explanation, the Commands-to-AI-Agent entry point, and the
 *  muted rule id + delivery footer. */
function FeedDetail({ entry, onAdvisory }: { entry: AlertFeedEntry; onAdvisory: (zone: string) => void }): React.JSX.Element {
  const reduced = useReducedMotion();
  return (
    <motion.div
      className="cf-feed-detail"
      initial={reduced ? false : { opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={reduced ? { duration: 0 } : { duration: 0.2, ease: "easeOut" }}
    >
      <table className="cf-feed-table">
        <tbody>
          <tr><th scope="row">Value</th><td>{feedNum(entry.value, entry.unit)}</td></tr>
          <tr><th scope="row">Threshold</th><td>{feedNum(entry.threshold, entry.unit)}</td></tr>
          <tr><th scope="row">Unit</th><td>{entry.unit || "—"}</td></tr>
          <tr><th scope="row">Fired at</th><td>{entry.ts}</td></tr>
        </tbody>
      </table>
      {entry.context ? <p className="cf-feed-context">{entry.context}</p> : null}
      <p className="cf-feed-explanation">{explanationFor(entry.condition)}</p>
      {/* Advisory dispatch is zone-scoped; a telemetry-gap row's ruleId
          segment is a dataset name, not a zone — the button can never
          succeed there (integration review 2026-10-16). */}
      {entry.condition !== "telemetry-gap" ? (
        <div className="cf-feed-actions">
          <button type="button" className="cf-btn" onClick={() => onAdvisory(zoneOf(entry))}>
            Commands to AI Agent
          </button>
        </div>
      ) : null}
      <p className="cf-feed-footer">
        {entry.ruleId} · {entry.sent ? "push delivered" : "push not delivered"}
      </p>
    </motion.div>
  );
}

/** One feed row: unread dot, relative time, rule title, value line; clicking
 *  toggles the inline detail. */
function FeedRow({ entry, unread, expanded, onToggle, onAdvisory }: {
  entry: AlertFeedEntry;
  unread: boolean;
  expanded: boolean;
  onToggle: () => void;
  onAdvisory: (zone: string) => void;
}): React.JSX.Element {
  return (
    <div className="cf-feed-item" data-rule-id={entry.ruleId}>
      <button type="button" className="cf-feed-row" aria-expanded={expanded} onClick={onToggle}>
        {unread ? <span className="cf-feed-dot" aria-label="unread" /> : null}
        <time className="cf-feed-time">{timeAgo(entry.ts)}</time>
        <strong className="cf-feed-title">{entry.title}</strong>
        <span className="cf-feed-value">{feedNum(entry.value, entry.unit)} / {feedNum(entry.threshold, entry.unit)} {entry.unit}</span>
      </button>
      {expanded ? <FeedDetail entry={entry} onAdvisory={onAdvisory} /> : null}
    </div>
  );
}

const defaultFetchJson: ViewFetcher = (endpoint, opts) => api.fetchJson(endpoint, opts);

export default function AlertsView({ refreshSeq = 0, fetcher }: { refreshSeq?: number; fetcher?: ViewFetcher }): React.JSX.Element {
  const fetchJson: ViewFetcher = fetcher ?? defaultFetchJson;
  const [records, setRecords] = useState<AlertRecord[] | null>(null);
  const [feed, setFeed] = useState<AlertFeed | null>(null);
  const [feedError, setFeedError] = useState<string | null>(null);
  const [retryTick, setRetryTick] = useState(0);
  // seenAt is read once per mount: entries newer than it carry the unread
  // dot for this session; the feed render then marks everything read.
  const [seenAt] = useState<number>(() => readSeen());
  // 60-second tick: a long-lived screen must age the feed rows' relative
  // times honestly — the tick state is never read, it only re-renders so
  // timeAgo recomputes against the new Date.now().
  const [, setTick] = useState(0);
  const seenMarkedRef = useRef(false);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  // The Commands-to-AI-Agent modal opens per firing card: the zone comes
  // from the entry's ruleId scope segment; null = no modal open.
  const [advisoryZone, setAdvisoryZone] = useState<string | null>(null);

  const load = useCallback((): void => {
    void listAlerts().then((visible) => setRecords(visible));
  }, []);

  // refreshSeq > 0 means the shell's global Refresh re-invoked the view.
  useEffect(() => {
    load();
  }, [load, refreshSeq]);

  // Relative times go stale on a long-lived screen: every 60s the tick
  // re-renders the feed rows so "14m ago" ages honestly (cleaned up on
  // unmount).
  useEffect(() => {
    const id = window.setInterval(() => setTick((t) => t + 1), 60_000);
    return () => window.clearInterval(id);
  }, []);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const res = await fetchJson<AlertFeed>("api/alerts/feed");
        if (cancelled) return;
        setFeed(res.data);
        setFeedError(null);
        // Mark-all-read once per session after a successful render: the
        // dots for THIS mount were computed from the pre-write seenAt.
        if (!seenMarkedRef.current) {
          seenMarkedRef.current = true;
          markSeen(Date.now());
        }
      } catch (err) {
        if (cancelled) return;
        setFeedError(err instanceof Error ? err.message : String(err));
      }
    })();
    return () => {
      cancelled = true;
    };
    // refreshSeq re-runs the feed too: the shell's Refresh click and the
    // service worker's alerts-updated message both bump it (operator
    // 2026-10-12: fires appear while the app is open).
  }, [fetchJson, retryTick, refreshSeq]);

  const entries = feed?.feed ?? [];

  return (
    <div>
      <p className="cf-row-detail cf-view-intro">
        Alert feed: every fired rule with what it hit, why it matters and what to do next — plus the push history. Alerts page your paired devices — pair in the Pairing section.
      </p>

      {feedError !== null ? (
        <section className="cf-card cf-level-warning">
          <h3 className="cf-card-title">Could not load the alert feed</h3>
          <p className="cf-row-detail">{feedError}</p>
          <button type="button" className="cf-btn" onClick={() => {
            setFeedError(null);
            setRetryTick((t) => t + 1);
          }}>
            Retry
          </button>
        </section>
      ) : feed === null ? (
        <div className="cf-loading">
          <div className="cf-skeleton" />
          <div className="cf-skeleton" />
          <div className="cf-skeleton" />
        </div>
      ) : entries.length === 0 ? (
        <p className="cf-empty cf-empty-quiet">Nothing has fired yet — alerts land here the moment a rule trips.</p>
      ) : (
        <div className="cf-feed">
          {entries.map((entry) => (
            <FeedRow
              key={entry.ruleId + entry.ts}
              entry={entry}
              unread={Date.parse(entry.ts) > seenAt}
              expanded={expandedId === entry.ruleId + entry.ts}
              onToggle={() => setExpandedId((cur) => (cur === entry.ruleId + entry.ts ? null : entry.ruleId + entry.ts))}
              onAdvisory={setAdvisoryZone}
            />
          ))}
        </div>
      )}

      {advisoryZone !== null ? <AgentCommandsModal zone={advisoryZone} onClose={() => setAdvisoryZone(null)} /> : null}

      {records === null ? null : records.length === 0 ? (
        <p className="cf-empty cf-empty-quiet">No alerts. Your daemon will page you here.</p>
      ) : (
        <div className="cf-alert-list">
          {records.map((record) => (
            <AlertCard key={record.id} record={record} onChanged={load} />
          ))}
        </div>
      )}
    </div>
  );
}
