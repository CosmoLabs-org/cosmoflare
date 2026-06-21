---
id: FB-1227
title: GLM pool 'glm-5.2[1m]' unknown — exec-batch dispatch fails
type: idea
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-06-20T21:05:00.936458-03:00"
updated: "2026-06-20T21:05:00.936458-03:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 17
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

# FB-1227: GLM pool 'glm-5.2[1m]' unknown — exec-batch dispatch fails

What happened: ccs glm-agent exec-batch (and a dry-run) failed every task with 'queue: conductor acquire denied: unknown pool: glm-5.2[1m]'. 0/N agents dispatched. The conductor has no pool matching the model id — the '[1m]' context-window suffix appears to not be registered as (or mapped to) a pool name. Why it matters: this blocks ALL GLM dispatch (glm-agent exec, exec-batch, glm-tree). The user's next-session goal (build the Tauri app via glm-tree Wave 1) is blocked until this is fixed. Proposed: in the conductor/glmconfig pool registry, register a pool for the suffixed model id, or strip/normalize the '[1m]' suffix to the base pool before acquire. Repro: ccs glm-agent exec-batch <any-manifest>. Priority: high — blocks the documented next session.

