---
ulid: 01M1W1XVYHJHQYA77AF4V3SGZ8
title: doc-review status reports stale COUNT but never lists WHICH docs — operators must reverse-engineer the set
type: improvement
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-06T23:08:08.529877+04:00"
updated: "2026-09-06T23:08:08.529877+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 38
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

# FB-pV3SGZ8: doc-review status reports stale COUNT but never lists WHICH docs — operators must reverse-engineer the set

PROBLEM: 'ccs doc-review status' prints 'Reviewed: 28 | Stale: 6 | Unreviewed: 110' with no way to enumerate the 6. CURRENT VS EXPECTED: Current — I spent ~10 tool calls reverse-engineering the stale set on 2026-09-06 (guessed the content-hash basis via python3, re-stamped one doc as an experiment, finally found them via grep for old last_reviewed stamps + git log forensics on the plan_ref repair commit). Expected — 'ccs doc-review status --stale' (or plain status) lists the stale paths; 'ccs doc-review status --json' exposes a stale_docs array. WHY IT MATTERS: stale docs are the actionable half of the output — a count with no names forces the exact workaround above every time, and the wrong guess risks re-stamping or missing docs. SUGGESTED IMPLEMENTATION: status already computes staleness (hash compare vs last_review_content_hash) — emit the matched paths. Add a --stale flag printing one path per line for scripting. AFFECTED FILES: tools/ccsession/cmd/doc_review.go (status subcommand; verified path pattern from 'ccs doc-review --help' output listing stamp/status/chain/verify-commands subcommands — exact status func name unverified). SESSION CONTEXT: cosmoflare session-end 2026-09-06, G6 backlog clearance — 6 stale stamps needed identifying; the enumeration gap turned a 1-command job into forensics.

