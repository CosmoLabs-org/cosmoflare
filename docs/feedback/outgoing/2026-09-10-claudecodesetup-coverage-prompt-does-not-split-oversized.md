---
ulid: 01M25WE1Y5C67B3MKJEVNHP6MX
title: coverage prompt does not split oversized packages — cmd (433 functions) became one agent
type: idea
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-10T18:44:31.81381+04:00"
updated: "2026-09-10T18:44:31.81381+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 49
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

# FB-pNHP6MX: coverage prompt does not split oversized packages — cmd (433 functions) became one agent

PROBLEM: ccs coverage prompt --save generated ONE test-generation task for the cmd package listing 340 untested functions, despite the /cover skill's own splitting rule ('if a package has 20+ untested functions, split into 2-3 agents').

CURRENT VS EXPECTED: The generated batch-001 manifest's cmd task body contains ~340 function bullet lines. Expected: the prompt generator splits at a threshold into multiple file-grouped tasks (e.g. cmd-part-1: cache/config/plugin; cmd-part-2: dns/zone/bucket...). The dispatched agent predictably timed out having covered only a fraction (cache_run_test.go + config_run_test.go of the listed surface).

WHY IT MATTERS: One agent cannot cover 340 functions within max-turns; the remainder is silently lost — the manifest reports the package as 'attempted' while coverage barely moves.

PRIORITY: Medium — happens on every repo with a large command surface.

REPRO: In cosmoflare: ccs coverage prompt --json --save; inspect docs/unit-testing/prompts/<batch>/batch-manifest.yaml — the 'cmd' task lists the full function inventory in one entry.

AFFECTED FILES: Unverified — the prompt builder in the coverage command path (cmd/coverage_prompt.go or internal/coverage/; needs find-func 'prompt').

SUGGESTED IMPLEMENTATION: In the prompt generator, when a package's gap list exceeds ~30 functions, partition by source file into N tasks with disjoint files: lists (the conflict-detection boundary already exists) and note the split in the task title (cmd 1/3, cmd 2/3...).

SESSION CONTEXT: 2026-09-10 cosmoflare /cover run — the review step of the skill caught it manually ('340 functions exceeds the 20-function split rule — dispatching as-is'); the generator should have done this natively.

