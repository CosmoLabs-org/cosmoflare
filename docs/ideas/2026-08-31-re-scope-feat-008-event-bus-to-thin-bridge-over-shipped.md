---
ulid: 01M1AF1EWD03RXB14DPCP3X5R6
id: IDEA-048
title: Re-scope FEAT-008 event bus to thin bridge over shipped sseHub
created: "2026-08-31T03:10:57.93397+04:00"
status: seed
source: agent
origin:
    session: 36
    trigger: audit-agent-15-work-completion
    file: docs/audit/2026-08-31-cosmoflare/agent-15-work-completion.md
tags:
    - audit
    - work-completion
    - desktop
---

# Re-scope FEAT-008 event bus to thin bridge over shipped sseHub

FEAT-008 (71 days open, 0/19 plan goals) scopes a standalone internal/events.Bus — but the sseHub transport it assumed now EXISTS (internal/server/sse.go). TriggerAlert (internal/webhook/manager.go:182) only does HTTP delivery; the notifications channel has one source (online/offline transitions). Re-scope to: bridge TriggerAlert onto sseHub notifications. Re-validate the 644-line plan against current code before executing its 4 TDD tasks.
