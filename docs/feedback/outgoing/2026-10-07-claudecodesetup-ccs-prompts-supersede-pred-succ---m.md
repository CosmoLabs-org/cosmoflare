---
ulid: 01M4BYH1NVCSPCDSRW2C5KJ8BW
title: '`ccs prompts supersede <pred> <succ> --migrate-goals` appends migrated goal checkboxes without deduplicating against goals the successor prompt already contains.'
type: bug
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-10-07T23:47:57.243764+04:00"
updated: "2026-10-07T23:47:57.243764+04:00"
suggested_conversion: bug
converted_to: null
related_issues: []
brainstorm_ref: null
session: 65
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

# FB-p5KJ8BW: `ccs prompts supersede <pred> <succ> --migrate-goals` appends migrated goal checkboxes without deduplicating against goals the successor prompt already contains.

What happened: `ccs prompts supersede <pred> <succ> --migrate-goals` appends migrated goal checkboxes without deduplicating against goals the successor prompt already contains. Observed three times in cosmoflare in two days: Session-2028 continuation (fixed in e204903), Session-2029 continuation (d1a2aac), Session-2030 continuation (fixed today minutes after creation) — each time the operator-authored goals and the migrated ones described the same items, doubling the checkbox count and desyncing goals_total.

Why it matters: every session-end that uses the standard supersede flow produces a corrupted ledger unless the operator manually re-checks and dedupes; set-goals' mismatch guard fires only if run AFTER the migration, so the corruption ships silently otherwise.

Proposed fix: in supersede --migrate-goals, skip a migrated goal whose normalized title (or referenced predecessor goal id already covered by a "Covers migrated" note) matches an existing successor goal; or emit a warning listing duplicates so the operator dedupes before the counter locks.

Evidence: cosmoflare docs/prompts/2026-10-07-session-2030-continuation.md pre-fix (8 checkboxes, 4 unique); commits e204903, d1a2aac, and today's dedupe commit; pattern identical across all three.

