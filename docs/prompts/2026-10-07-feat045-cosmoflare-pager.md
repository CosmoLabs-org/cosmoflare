---
brainstorm_ref: docs/brainstorming/2026-10-07-feat045-cosmoflare-pager.md
branch: master
completed: "2026-10-07T18:21:58+04:00"
covers_brainstorm_deliverables:
    - BR-01
    - BR-02
    - BR-03
    - BR-04
    - BR-05
    - BR-06
covers_plan_deliverables:
    - P-01
    - P-02
    - P-03
    - P-04
    - P-05
    - P-06
    - P-07
    - P-08
    - P-09
created: "2026-10-07T17:08:05+04:00"
date: "2026-10-07T17:08:05+04:00"
goals_completed: 9
goals_total: 9
id: P-2026-10-07-feat045-cosmoflare-pager
implemented_commits:
    - 68baf3b4f1db
plan_ref: docs/planning-mode/2026-10-07-feat045-cosmoflare-pager.md
priority: medium
related_prompts: []
requires_reading:
    - docs/brainstorming/2026-10-07-feat045-cosmoflare-pager.md
    - docs/planning-mode/2026-10-07-feat045-cosmoflare-pager.md
schema_version: 1
status: COMPLETED
tags: []
title: FEAT-045 Cosmoflare Pager — Full Implementation
type: prompt
---

# FEAT-045 Cosmoflare Pager — Full Implementation

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/brainstorming/2026-10-07-feat045-cosmoflare-pager.md`** — design rationale and decision history.
2. **`docs/planning-mode/2026-10-07-feat045-cosmoflare-pager.md`** — implementation plan with deliverables and file scope.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## Context

_Describe the session context._

## Goals

### [x] G-01 alertspush Store — VAPID keypair + subscriptions, push.json 0600
Covers P-01.

### [x] G-02 alertspush Payload protocol — severity vocabulary, 2KB envelope
Covers P-02.

### [x] G-03 alertspush Sender + Dispatch with 404/410 pruning
Covers P-03.

### [x] G-04 CLI alerts push keygen|add|list|remove with --json + USAGE.md section
Covers P-04.

### [x] G-05 CLI alerts watch interval evaluator + --test-fire
Covers P-05.

### [x] G-06 PWA scaffold — manifest, service worker push handler, payload parser
Covers P-06.

### [x] G-07 PWA views — list/detail/ack/snooze/pairing, IndexedDB store
Covers P-07.

### [x] G-08 Desktop notifications ack/snooze parity
Covers P-08.

### [x] G-09 PRODUCT-VISION principle-6 pivot + ROAD-064 link
Covers P-09.

## Related

- Brainstorm: `docs/brainstorming/2026-10-07-feat045-cosmoflare-pager.md`
- Plan: `docs/planning-mode/2026-10-07-feat045-cosmoflare-pager.md`
