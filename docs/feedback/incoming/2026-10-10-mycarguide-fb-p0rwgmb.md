---
ulid: 01M4HJJ3TNC4FY16J0WD0RWGMB
id: FB-28
title: ops.cosmolabs.org shows MyCarGuide KV storage as 4GB — the real figure is 46MB (86x off). HIGH.
type: bug
status: implemented
priority: medium
complexity: ""
from_project: MyCarGuide
from_path: /Users/gabstudio/PROJECTS/MyCarGuide
to_project: cosmoflare
to_target: project
created: "2026-10-10T04:14:15.893498+04:00"
updated: "2026-10-10T19:36:33.897958+04:00"
suggested_conversion: bug
converted_to: null
related_issues: []
brainstorm_ref: null
session: 19
suggested_workflow: []
response:
  acknowledged: null
  acknowledged_by: null
  started: null
  implemented: null
  rejected: null
  rejection_reason: null
  notes: 'Verified fixed: BUG-060 (pFAKDN3) — kv.storage is rolling bytes, excluded from all allowance visuals and never priced; pager card hidden with muted note (f093ea3); code+tests+live verified in session 2036.'
---

# FB-p0RWGMB: ops.cosmolabs.org shows MyCarGuide KV storage as 4GB — the real figure is 46MB (86x off). HIGH.

**What happened:** ops.cosmolabs.org reports MyCarGuide KV storage = 4GB. Authoritative measurement of the same namespace (binding CACHE, id c31756364bab468dbc0e08d5df6927ea) via the Cloudflare KV REST API: 878 keys, values sampled at 40-78KB (avg 54KB), total ≈ 46MB = 4.6% of the 1GB included tier. The dashboard number is ~86x the truth.

**Why it matters:** wrong storage numbers drive wrong engineering decisions — we nearly scoped a KV storage-reduction effort (TTL changes, key redesign, purges) off the 4GB figure; the real number needs zero action. A fleet audit tool's numbers must be authoritative, or it trains operators to ignore them.

**Likely root cause (evidence math):** the discrepancy matches a metric-definition bug almost exactly. Measured KV writes for this namespace are 2,541/day × 54KB avg value × 30 days ≈ 4.1GB — within a few percent of the displayed 4GB. Hypotheses in order: (1) cumulative bytes WRITTEN over a window reported as storage; (2) summing kvStorageAdaptiveGroups rows across the window instead of taking the latest/max snapshot; (3) resource-class mislabeling (R2 shown under KV). Check which one ops performs — all three produce ~4GB from this namespace's data.

**Proposed fix:** define and label storage as "current stored bytes" (latest kvStorageAdaptiveGroups max, or REST key-list × sampled sizes as fallback); display key count beside bytes so sanity checks are one glance; label the resource class explicitly (KV vs R2 vs D1 file size); add a regression test comparing the dashboard metric against a REST-derived total on a known namespace.

**Repro/evidence:** keys+values via GET /accounts/287a...6990/storage/kv/namespaces/c31756...7ea/keys and /values/<key> (2026-10-10: 878 keys, avg 54KB). Writes/day measured via kvOperationsAdaptiveGroups (24h: ~2,541 writes). Companion corpus FB: 01M4HHTVMZW101AQCCEJT3ZHNM.

**Priority:** high — wrong cost signal, fleet-wide.


## Recurrence (ingest duplicate)

- twin ulid: 01M4HJJ3TNC4FY16J0WD0RWGMB
- twin path: /Users/gabstudio/PROJECTS/cosmoflare/docs/feedback/incoming/2026-10-10-mycarguide-opscosmolabsorg-shows-mycarguide-kv-st.md
- folded: 2026-10-10T19:35:21+04:00
