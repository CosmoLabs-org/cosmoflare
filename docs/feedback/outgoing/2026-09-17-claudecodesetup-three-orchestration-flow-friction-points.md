---
ulid: 01M2R24A2C1N9VMZBQ9JB3VM29
title: 'Three orchestration-flow friction points from cosmoflare session 2026-09-17: (1) ccs prompts set-goal refuses a bare number whenever any declared G-NN id matches the digit (G-05/G-06 collided with ''4''), so ticking an id-less numbered goal took three commands — autotick wait + manual body-checkbox Edit + set-goals resync — where one flip should do; an explicit positional syntax like ''#3 done'' would fix it. (2) ccs commit --direct has no repo target flag; completing a salvaged agent''s work inside its worktree required raw git -C <worktree> commit because the hook blocks raw commits in the main repo — a --repo/-C flag would remove the trap (first attempt committed toward the wrong repo; caught, nothing lost). (3) ccs glm-agent status --json returns a top-level ARRAY of agent objects, not {agents: [...]}; undocumented, and a completion-monitor jq filtering .agents[] false-positived ''both agents terminal'' while both were still running — silently wrong state is a footgun worth one doc line.'
type: improvement
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-17T20:10:23.692049+04:00"
updated: "2026-09-17T20:10:23.692049+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 60
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

# FB-pB3VM29: Three orchestration-flow friction points from cosmoflare session 2026-09-17: (1) ccs prompts set-goal refuses a bare number whenever any declared G-NN id matches the digit (G-05/G-06 collided with '4'), so ticking an id-less numbered goal took three commands — autotick wait + manual body-checkbox Edit + set-goals resync — where one flip should do; an explicit positional syntax like '#3 done' would fix it. (2) ccs commit --direct has no repo target flag; completing a salvaged agent's work inside its worktree required raw git -C <worktree> commit because the hook blocks raw commits in the main repo — a --repo/-C flag would remove the trap (first attempt committed toward the wrong repo; caught, nothing lost). (3) ccs glm-agent status --json returns a top-level ARRAY of agent objects, not {agents: [...]}; undocumented, and a completion-monitor jq filtering .agents[] false-positived 'both agents terminal' while both were still running — silently wrong state is a footgun worth one doc line.

## Summary

[Describe feedback in detail]

## Motivation

[Why is this important?]

## Proposed Solution

[How should this be implemented?]

## Key Considerations

- [Consideration 1]
- [Consideration 2]

