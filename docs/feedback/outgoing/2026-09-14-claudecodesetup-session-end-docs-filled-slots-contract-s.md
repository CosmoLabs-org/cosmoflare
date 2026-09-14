---
ulid: 01M2G4DT7XW016HSH1ZRNJFN8N
title: session-end-docs filled-slots contract should publish its schema
type: idea
status: pending
priority: medium
complexity: ""
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-14T18:16:36.861268+04:00"
updated: "2026-09-14T18:16:36.861268+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 53
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

# FB-pNJFN8N: session-end-docs filled-slots contract should publish its schema

The --apply path silently falls back to deterministic output on slot-shape mismatch. Observed live in cosmoflare session 2027 (2026-09-14): slots as array failed (want object), object-of-strings failed (KEY_DECISIONS wants []string), object-of-arrays failed (NARRATIVE wants string) — three blind attempts before hand-writing the artifacts. Fix: scaffold.json should include a per-slot type map or a literal example filled.json; --apply should validate + report the exact expected shape on mismatch instead of warning-then-fallback. Why: the fallback makes failures silent-ish and costs the LLM 3 extra rounds at session-end (worst time to burn context). Complexity: small.

