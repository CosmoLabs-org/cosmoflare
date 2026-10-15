---
brainstorm_ref: docs/brainstorming/2026-10-15-commands-to-ai-agent-briefs.md
branch: master
covers_brainstorm_deliverables:
    - BR-01
    - BR-02
    - BR-03
    - BR-04
    - BR-05
    - BR-06
covers_plan_deliverables:
    - P-15
    - P-16
    - P-17
    - P-18
    - P-19
created: "2026-10-15T19:07:33+04:00"
date: "2026-10-15T19:07:33+04:00"
id: P-2026-10-15-commands-to-ai-agent-briefs
plan_ref: docs/planning-mode/2026-10-15-commands-to-ai-agent-briefs.md
priority: medium
requires_reading:
    - docs/brainstorming/2026-10-15-commands-to-ai-agent-briefs.md
    - docs/planning-mode/2026-10-15-commands-to-ai-agent-briefs.md
schema_version: 1
status: PENDING
title: Commands to AI Agent — Full Implementation
type: prompt
---
# Commands to AI Agent — Full Implementation

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/brainstorming/2026-10-15-commands-to-ai-agent-briefs.md`** — design rationale and decision history.
2. **`docs/planning-mode/2026-10-15-commands-to-ai-agent-briefs.md`** — implementation plan with deliverables and file scope.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## Context

_Describe the session context._

## Goals

### [ ] G-01 briefs core — types/registry/collectors/compile in apps/ops/src/briefs/ (pure, tested)
Covers P-15.

### [ ] G-02 v1 templates + agent-brief renderer with golden tests (byte-identical proof)
Covers P-16.

### [ ] G-03 GET /api/advisory/:zone endpoint — parallel collectors, Access auth, no-store
Covers P-17.

### [ ] G-04 Commands-to-AI-Agent modal + alert-card entry point, impeccable detect 0
Covers P-18.

### [ ] G-05 Playwright live pass on the dev URL + README API docs
Covers P-19.

## Related

- Brainstorm: `docs/brainstorming/2026-10-15-commands-to-ai-agent-briefs.md`
- Plan: `docs/planning-mode/2026-10-15-commands-to-ai-agent-briefs.md`
