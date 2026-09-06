---
ulid: 01M1WEXBVH1NJT32Y5AVJ081RA
title: changelog add silently ignores unknown flags and still writes the entry
type: bug
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-07T02:55:03.537234+04:00"
updated: "2026-09-07T02:55:03.537234+04:00"
suggested_conversion: bug
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

# FB-pJ081RA: changelog add silently ignores unknown flags and still writes the entry

Problem: 'ccs changelog add "x" --type added --source d8f8f3c' prints 'Unknown flag --source ignored' as a warning and then ADDS the entry anyway — with source: manual in docs/changelog/unreleased.yaml. The caller believes the source was recorded; it was silently dropped.

Current vs expected: expected a hard error on an unknown flag (cobra strict parsing), entry not written. Got: warning + successful write with wrong source field. Repro (any project): ccs changelog remove <staged>; ccs changelog add 'test entry' --type added --source abc123; grep source docs/changelog/unreleased.yaml → shows 'manual'.

Why it matters: the Source field is the primary dedupe key (source-based rule in staging.go; the planned stage-from-log fix passes SHA as Source per FB-pGVA5Z3). A caller pre-supplying source to avoid near-dupes gets silently defeated — exactly the failure mode that landed a duplicate error-contract entry in cosmoflare today (removed manually via substring).

Priority: medium — data integrity of changelog staging, not a crash.

Affected files: changelog command flag parsing — Unverified (needs investigation): likely tools/ccsession/cmd/changelog.go or the shared flag-parsing wrapper; investigate where the 'Unknown flag' warning string is printed.

Suggested implementation: remove any FParseErrorsWhitelist / lenient unknown-flag handling for changelog subcommands (or globally for ccs); unknown flag = exit non-zero before any write.

Session context: session-end Phase 2.75 in cosmoflare (2026-09-07), auto-staging the feat commit for the v0.21.0 release. The flag was invented to attach a commit SHA as source; the warning-plus-write split the operation's intent from its result.

