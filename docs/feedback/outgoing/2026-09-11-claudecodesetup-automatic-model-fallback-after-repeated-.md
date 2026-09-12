---
ulid: 01M26NRCZYRA451QRPN59YG6N0
title: Automatic model-fallback after repeated GLM agent idle-timeouts
type: feature
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-11T02:07:05.214176+04:00"
updated: "2026-09-11T02:07:05.214176+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 51
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

# FB-p9YG6N0: Automatic model-fallback after repeated GLM agent idle-timeouts

WHAT HAPPENED: Dispatched the same medium implementation task (cosmoflare FEAT-024, bounded brief with complete code specs) to GLM 5.3 twice via ccs glm-agent exec. Both agents died with 'idle timeout: no file activity detected' (4m window, then 8m window with the same result) while producing real work (442 and then ~1080 lines salvaged). A third dispatch with --model sonnet completed the identical task in one 19-minute pass. The compile-silence hypothesis (fresh-worktree go build produces no file activity) does not explain it — the 8-minute window still tripped.

WHY IT MATTERS: Each GLM idle-timeout wastes a dispatch + leaves salvage worktrees that must be extracted and re-ported manually. The failure repeated deterministically on the same task, then vanished on a different model — this is a per-task/per-model failure signature, not transient API load.

PROPOSED SOLUTION: In ccs glm-agent exec, track idle-timeout history per task brief hash. When a brief has already idle-timed-out once on model M, print an advisory (or add a flag like --fallback-model sonnet) recommending a different model for the re-dispatch. Even advisory-only output would have saved one wasted dispatch here. Consider also logging WHERE the agent was when it stalled (last tool call in the tmux transcript) to the status output — 'no file activity' gives no diagnostic.

PRIORITY JUSTIFICATION: Medium — wasted-dispatch cost, not data loss. Observed 2026-09-11 in cosmoflare session (agents 0118, 0119, 0120).

