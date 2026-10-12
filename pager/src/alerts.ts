// Types for the Ops Worker's fired-alert feed (FEAT-052 Notifications
// center): the wire shape of GET /api/alerts/feed. The Worker stores the
// same objects in KV "alert-feed" (apps/ops/src/scheduled.ts's
// AlertFeedEntry) — newest first, capped at 100 entries — and this file
// mirrors that shape verbatim so the pager can render entries as-is.

// AlertFeedEntry is one fired alert: when it fired, which rule and condition
// tripped, the observed numbers against the threshold, the push detail line,
// and whether the push actually reached a device.
export interface AlertFeedEntry {
  ts: string; // fire time, ISO
  ruleId: string; // stable alert ID ("rule/scope" or "telemetry-gap/dataset")
  condition: string; // rules.ts condition id, or "telemetry-gap"
  title: string; // rule name
  value: number; // observed value at fire time
  threshold: number; // rule threshold
  unit: string; // "%" | "requests" | "rows" | "writes" ("" for gaps)
  message: string; // the page detail line
  sent: boolean; // whether the push reached at least one device
  context: string; // absolute base for ratio fires ("46 misses of 139 eligible requests"); "" for counts
}

// AlertFeed is the /api/alerts/feed response envelope.
export type AlertFeed = { feed: AlertFeedEntry[] };
