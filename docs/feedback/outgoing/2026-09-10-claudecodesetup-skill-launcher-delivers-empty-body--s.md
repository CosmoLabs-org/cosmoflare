---
ulid: 01M25WAWDT4EDTCYQJ5XBTDZ5T
title: Skill launcher delivers empty body — session recovered 3 skills from disk
type: idea
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-10T18:42:47.86653+04:00"
updated: "2026-09-10T18:42:47.86653+04:00"
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

# FB-pBTDZ5T: Skill launcher delivers empty body — session recovered 3 skills from disk

PROBLEM: The Skill tool returned only 'Launching skill: X' with no body, then 'Skill X is already loaded above; instructions unchanged' — while no skill content was ever delivered. Hit 3x this session (test-coverage, triage-issues, session-reflect via launcher).

CURRENT VS EXPECTED: Skill(test-coverage) x2 -> 'Launching skill: test-coverage' then 'already loaded above; instructions unchanged' with zero content in context. Expected: the skill body loads. WORKAROUND USED: read the skill file directly from ~/.claude/skills/<name>.md — worked every time, which proves the file is intact and the delivery/cache layer is what fails. The harness appears to cache 'delivered' keyed to the session after the first empty load, so retries cannot recover (BUG-789 retry also fails).

WHY IT MATTERS: Every skill invocation that silently loads nothing risks BUG-109-style manual freelancing. This session caught it only because the failure mode was known.

PRIORITY: High-frequency (3/15 skills this session) and silent for sessions that do not check.

REPRO: In a fresh Claude Code session in cosmoflare: invoke Skill(test-coverage), then use it. Observe no body arrives; invoke again; observe the dedupe message. Then cat ~/.claude/skills/test-coverage.md — content exists.

AFFECTED FILES: the skill delivery/registry cache layer (harness-side); session cache of loaded skills. Unverified which component owns it — needs investigation on the harness side.

SUGGESTED IMPLEMENTATION: On skill launch, verify the response actually contains the SKILL_LOADED marker + body before marking the skill as delivered in the session cache; on empty delivery, invalidate the cache entry so the BUG-789 retry can succeed.

SESSION CONTEXT: 2026-09-10 cosmoflare mega-session running /cover, /triage, /session-end in sequence — three separate skills hit the empty-delivery failure; all three were recovered by disk reads, so work continued, but each cost a debug detour.

