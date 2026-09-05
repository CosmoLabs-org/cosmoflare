---
status: PENDING
type: continuation
priority: high
created: 2026-09-05T00:00:00+02:00
requires_reading:
    - docs/audit/latest/brief.md
    - docs/audit/latest/action-plan.md
schema_version: 1
supersedes: "docs/prompts/2026-08-31-cosmoflare-audit-followup.md"
goals_total: 6
---
# Cosmoflare — Next Session Continuation (2026-09-05)

## Context to Load

Read these first:

- docs/audit/latest/brief.md (project context)
- docs/audit/latest/action-plan.md (prioritized next-steps)
- docs/audit/latest/risk-map.md (where NOT to touch without tests)

## Carry-over Goals (priority order)

### 1. [GOAL] Verify the first-ever GitHub Release

If billing is fixed, run:

```
gh run rerun 33934576121 --repo CosmoLabs-org/cosmoflare
```

and watch it complete (tests, cross-compile, version assertion, checksums, publish).
If still blocked, it remains a billing action item — do **not** retry-loop.

### 2. [GOAL] BUG-035 (last open audit bug)

Auto-generate MCP tools from the cobra tree (8 tools → full surface), mutations gated
behind guardrails. Large effort — **plan first** in docs/planning-mode/.

### 3. [GOAL] Launch readiness (audit growth tier)

- Make repo public + set description/topics/homepage
- Regenerate README service table from CLAUDE.md matrix (13 services marked
  *Planned* are actually shipped)
- GoReleaser + Homebrew tap

### 4. [GOAL] FEAT-008 implementation

Per the 2026-09-05 re-scope: thin bridge `webhook.Manager.TriggerAlert` → existing
`sseHub` notifications channel (`internal/server/sse.go`). The old plan is superseded —
write a fresh short plan first.

### 5. [GOAL] Housekeeping

- ROAD-085 remainder: untrack `GOrchestra/sessions` recovery patches (234MB) + remaining
  committed binaries
- ROAD-086 remainder: audit 9 BASE roadmap items

### 6. [GOAL] /independent-review backlog

24 unreviewed design docs + 6 stale (re-touched by the `plan_ref` repair).
