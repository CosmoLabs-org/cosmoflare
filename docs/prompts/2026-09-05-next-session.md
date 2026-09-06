---
created: "2026-09-05T00:00:00+02:00"
goals_completed: 5
goals_total: 6
priority: high
related_prompts: []
requires_reading:
    - docs/audit/latest/brief.md
    - docs/audit/latest/action-plan.md
schema_version: 1
status: SUPERSEDED
tags: []
title: Cosmoflare — Next Session Continuation (2026-09-05)
type: continuation
superseded_by: "docs/prompts/2026-09-06-launch-and-hardening.md"
completed: "2026-09-06T23:12:02+04:00"
---

# Cosmoflare — Next Session Continuation (2026-09-05)

## Context to Load

Read these first:

- docs/audit/latest/brief.md (project context)
- docs/audit/latest/action-plan.md (prioritized next-steps)
- docs/audit/latest/risk-map.md (where NOT to touch without tests)

## Carry-over Goals (priority order)

### 1. [DONE 2026-09-05] First GitHub Release — shipped locally

v0.19.0 published with 6 assets (5 binaries + checksums) via `gh release create`
from the maintainer machine. **CI is permanently disabled on this repo**
(all 3 workflows `disabled_manually`) — releases are ALWAYS local:
cross-compile 5 platforms with version ldflags → `shasum -a 256` → assert
`--version` → `gh release create <tag> dist/* --notes`. Never wait on Actions.

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
