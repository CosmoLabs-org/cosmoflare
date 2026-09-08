---
ulid: 01M20MQCF8BY2PTX548W1JN6SZ
title: Alert evaluation loop in cosmoflare is inline cmd-layer functions while the analogous MetricsProducer is a constructed component
type: improvement
status: pending
priority: low
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-08T17:53:36.744481+04:00"
updated: "2026-09-08T17:53:36.744481+04:00"
suggested_conversion: feature
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

# FB-p1JN6SZ: Alert evaluation loop in cosmoflare is inline cmd-layer functions while the analogous MetricsProducer is a constructed component

What happened: the 2026-09-08 simplify review (altitude angle) flagged that cosmoflare's daemon alert evaluator (cmd/serve.go runServeAlertEvalLoop/runServeAlertEvalCycle free functions + a mutable interval global) mirrors the MetricsProducer component one file away but stopped short of the component shape. Policy duplication followed: the daemon hardcodes a 15m cooldown that silently shadows the library default, the 24h window literal appears in both serve and the CLI one-shot, and the CLI re-implements the eval cycle with different error policy. Why it matters: this is the recurring pattern GLM agents produce when briefs specify function-level additions instead of component seams — the fix is a brief-writing guideline, not just this instance. Proposed fix: (1) in cosmoflare, extract internal AlertProducer with Start(ctx)/RunOnce(ctx) like MetricsProducer (deferred this session, recorded in the continuation prompt context); (2) in CCS, add to the GLM brief-writing guidance: when a brief adds a background loop next to an existing component with the same shape, specify the component pattern explicitly or the agent will inline free functions. Evidence: cmd/serve.go:180-243 vs internal/server/metrics.go:31-60, session 2026-09-08, simplify altitude finding 3 (skipped as too wide for the session-end sweep).

