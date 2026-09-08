---
ulid: 01M20MS4KBPNT1GRY93AFM110F
title: 'FB-2189 persists: GLM agent timeout kills mid-TDD dispatches — 3 of 7 in one cosmoflare session, all salvageable'
type: bug
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-08T17:54:34.219421+04:00"
updated: "2026-09-08T17:54:34.219421+04:00"
suggested_conversion: bug
converted_to: null
related_issues: []
brainstorm_ref: null
session: 43
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

# FB-pFM110F: FB-2189 persists: GLM agent timeout kills mid-TDD dispatches — 3 of 7 in one cosmoflare session, all salvageable

What happened: 2026-09-08 cosmoflare session dispatched 7 GLM agents (ccs glm-agent exec --max-turns 40-45 --idle-timeout 5m --max-timeout 45-50m). Three exited with status timeout / exit_code -2 mid-run: 0060 (sync correctness — full work present, just uncommitted), 0066 (daemon metrics — tests written, implementation missing entirely), 0067 (alert evaluator — complete work, uncommitted). Current vs expected: all three had active work flowing; a 5m idle budget should not fire during continuous file writes. Expected: agents finish or die with a reason. Why it matters: FB-2189 documented this class on 2026-09-08 from WikipediaDB (3 of 8, post-green pre-commit); this adds a second project same-day, 3 of 7, including a tests-only salvage (the most expensive shape — red tests with no implementation). Priority: the salvage path works (auto-commit recovered everything; I finished 0066 manually against its own red tests), but each timeout costs a gate cycle. Repro: dispatch GLM agents with TDD briefs at --max-turns 40+ repeatedly; observe timeout exits with uncommitted complete work. Affected files: GLM queue/agent runner (tools/ccsession glm-agent exec path) — Unverified which module owns the idle timer; needs investigation. Suggested implementation: log the agent's last-activity timestamp and the firing timer (idle vs max) into the status JSON on timeout exit; if idle fires during active writes, the heartbeat source is wrong (tool-result gap vs true idle). Session context: cosmoflare five-wave session, waves 1/3/4 each lost an agent to timeout; salvage+manual-completion recovered 100% of work.

