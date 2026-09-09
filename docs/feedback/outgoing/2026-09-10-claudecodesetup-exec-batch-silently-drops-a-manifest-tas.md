---
ulid: 01M23YH3J1PYYS4CM49ZV2RCSJ
title: exec-batch silently drops a manifest task whose only .go output appends to another task's file
type: bug
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-10T00:42:40.065588+04:00"
updated: "2026-09-10T00:42:40.065588+04:00"
suggested_conversion: bug
converted_to: null
related_issues: []
brainstorm_ref: null
session: 47
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

# FB-pV2RCSJ: exec-batch silently drops a manifest task whose only .go output appends to another task's file

What happened: running 'ccs glm-agent exec-batch docs/prompts/2026-09-09-cf-api-knowledge-layer-glm-tasks.yaml --dry-run' on a 7-task manifest validated and listed only 6 tasks — the dropped task (a JSON data pack + a test APPENDED to knowledge_test.go, where task-1 creates that file) never appears in the wave plan. The output mentioned 'context ... deferred — supplied by a dependency', suggesting the dependency detector swallowed it. Why it matters: anyone trusting exec-batch with this manifest silently loses the seed-pack task — the feature ships without its data. We worked around it by mandating per-task 'ccs glm-agent exec' in the manifest header with a WARNING comment, but the default path is lossy. Proposed solution: when the validator drops/merges a task, print its identity explicitly ('task N merged into task M / dropped: reason') and exit non-zero on silent loss; or fix the dependency detection so append-to-existing-file outputs don't count as 'supplied by a dependency'. Repro: the manifest at cosmoflare docs/prompts/2026-09-09-cf-api-knowledge-layer-glm-tasks.yaml, ccs 2026-09-09 build.

