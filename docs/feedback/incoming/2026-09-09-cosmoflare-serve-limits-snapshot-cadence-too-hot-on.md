---
ulid: 01M21S2GXEYA5NQCYE00DS7Z68
title: Serve limits-snapshot cadence too hot once wired
type: idea
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: cosmoflare
to_target: self
created: "2026-09-09T04:28:50.478829+04:00"
updated: "2026-09-09T04:28:50.478829+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 45
suggested_workflow: []
response:
  acknowledged: null
  acknowledged_by: null
  started: null
  implemented: null
  rejected: null
  rejection_reason: null
  notes: ""
---

# FB-pDS7Z68: Serve limits-snapshot cadence too hot once wired

What happened: the session-end efficiency review found the serve alert cycle now collects a full limits snapshot (workers + buckets + zones + subscriptions + N DNS usage calls) every evaluation cycle — metrics that move on an hours timescale polled at the alert interval (~45 calls/cycle, ~13k/day at 5min for a 41-zone account).
Why it matters: wasted API quota and avoidable latency inline before eval.Evaluate.
Proposed solution: cache the limits snapshot with a 30-60min TTL keyed by account, or move limits collection to a slower sub-ticker that publishes into the shared metrics. Also add fail-fast for dnsUsage on the first 4xx (a token without DNS Read currently fails all 41 calls one by one) — needs HTTP status plumbed through restClient errors.
Priority: medium — correctness is fine (v0.23.0 shipped inert-free), this is efficiency headroom.

