---
brainstorm_ref: docs/brainstorming/2026-10-15-commands-to-ai-agent-briefs.md
branch: master
completed: "2026-10-16T02:16:41+04:00"
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
goals_completed: 5
goals_total: 5
id: P-2026-10-15-commands-to-ai-agent-briefs
implemented_commits:
    - covers:
        - P-15
      sha: f3686b54951d
    - covers:
        - P-16
      sha: 47c22907087a
    - covers:
        - P-17
      sha: 206b0196bb40
    - covers:
        - P-18
      sha: 8e616176b95a
    - covers:
        - P-19
      sha: e9a2a979b070
    - covers:
        - BR-03
      sha: 47c22907087a
    - covers:
        - BR-04
      sha: 206b0196bb40
    - covers:
        - BR-05
      sha: 8e616176b95a
    - covers:
        - BR-06
      sha: e9a2a979b070
    - covers:
        - BR-01
      sha: f3686b54951d
    - covers:
        - BR-02
      sha: f3686b54951d
plan_ref: docs/planning-mode/2026-10-15-commands-to-ai-agent-briefs.md
priority: medium
related_prompts: []
requires_reading:
    - docs/brainstorming/2026-10-15-commands-to-ai-agent-briefs.md
    - docs/planning-mode/2026-10-15-commands-to-ai-agent-briefs.md
schema_version: 1
status: COMPLETED
tags: []
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

### [x] G-01 briefs core — types/registry/collectors/compile in apps/ops/src/briefs/ (pure, tested)
Covers P-15.

### [x] G-02 v1 templates + agent-brief renderer with golden tests (byte-identical proof)
Covers P-16.

### [x] G-03 GET /api/advisory/:zone endpoint — parallel collectors, Access auth, no-store
Covers P-17.

### [x] G-04 Commands-to-AI-Agent modal + alert-card entry point, impeccable detect 0
Covers P-18.

### [x] G-05 Playwright live pass on the dev URL + README API docs
Covers P-19.

## Related

- Brainstorm: `docs/brainstorming/2026-10-15-commands-to-ai-agent-briefs.md`
- Plan: `docs/planning-mode/2026-10-15-commands-to-ai-agent-briefs.md`
