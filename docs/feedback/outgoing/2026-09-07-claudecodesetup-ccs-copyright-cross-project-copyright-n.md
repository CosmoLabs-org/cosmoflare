---
ulid: 01M1WECFDMRKVM72VZZH97SDF1
title: 'ccs copyright: cross-project copyright notice scan + fix'
type: feature
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-07T02:45:50.132033+04:00"
updated: "2026-09-07T02:45:50.132033+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 40
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

# FB-p97SDF1: ccs copyright: cross-project copyright notice scan + fix

What happened: mid-session in cosmoflare (2026-09-07) the user noticed the Dockerfile still said 'Copyright (c) 2025 CosmoLabs'. A scan found 72 files carrying notices; 67 were stale or non-standard against the canonical form 'Copyright (c) <first>-<current> CosmoLabs (https://cosmolabs.org)'. The year rolls over every January in every CosmoLabs repo and nothing catches it.

Why it matters: copyright notices are legal metadata present in every distributed file (MIT headers, Dockerfiles, READMEs, Tauri/React sources). Manual sweeps are per-repo, forgettable, and drift again the next New Year. This is the same class of cross-project hygiene as 'ccs timestamps backfill'.

Proposed solution: 'ccs copyright scan' (detect stale years + non-standard forms vs a configurable canonical template, single SSOT for the template) and 'ccs copyright fix --apply' (mechanical rewrite, excluding session logs/audit snapshots). Surface the scan as a one-line warning in 'ccs inject'/session-start, like roadmap health.

Evidence: cosmoflare sweep = 67 fixes across 72 files on 2026-09-07, forms found: 65x '(c) 2025', 2x '(c) 2025', 7x already canonical. Low urgency, zero-decision recurring work; one implementation removes a permanent drift class.

