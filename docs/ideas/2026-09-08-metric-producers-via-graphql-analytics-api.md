---
ulid: 01M207FRS0T5BZT5HB1RVNQ1RR
id: IDEA-055
id_assigned_at: 2026-09-10T16:45:25.445483+04:00
title: Metric producers via GraphQL Analytics API
created: "2026-09-08T14:02:15.717862+04:00"
status: withered
source: human
origin:
    session: 43
tags:
    - observability
    - metrics
resolution:
    reason: implemented
    date: "2026-09-08T16:06:33.843211+04:00"
    note: 'Landed 2026-09-08: AnalyticsService (GraphQL producers) + daemon partial snapshots + alert evaluator over the same layer'
---

# Metric producers via GraphQL Analytics API

# Metric producers via GraphQL Analytics API

The producer layer that unblocks two features: the local control plane (ROAD-090) and the alert evaluator (audit Phase 2 item 7). Implement per-source collectors over Cloudflare's GraphQL Analytics API — zone http requests, R2 storage/operations/bandwidth per bucket, Workers invocations — with partial snapshots (per-source errors, never all-or-nothing), populated Profile, byte-based deltas. Then 'cosmoflare metrics' becomes real data instead of the current snapshot, and TriggerAlert gets rich inputs. Recommended as the next build wave.
