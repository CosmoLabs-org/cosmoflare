---
brainstorm_ref: docs/brainstorming/2026-06-14-domain-management-center.md
branch: master
covers_brainstorm_deliverables:
    - BR-01
    - BR-02
    - BR-03
    - BR-04
    - BR-05
    - BR-06
    - BR-07
covers_plan_deliverables:
    - P-01
    - P-02
    - P-03
    - P-04
    - P-05
    - P-06
    - P-07
created: "2026-06-14"
id: P-2026-06-14-domain-management-center
plan_ref: docs/planning-mode/2026-06-14-domain-management-center.md
priority: medium
requires_reading:
    - docs/brainstorming/2026-06-14-domain-management-center.md
    - docs/planning-mode/2026-06-14-domain-management-center.md
schema_version: 1
status: PENDING
title: Domain Management Center — Implementation
---
# Domain Management Center — Implementation

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/brainstorming/2026-06-14-domain-management-center.md`** — design rationale and decision history.
2. **`docs/planning-mode/2026-06-14-domain-management-center.md`** — implementation plan with deliverables and file scope.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## Context

Domain Management Center: a complete domain surface for cosmoflare. All of the
operator's domains are already Cloudflare zones; this adds modern Redirect Rules,
a registrar overlay (expiry/auto-renew for CF-registered; `external` badge otherwise),
a richer `domains` command tree, and a split-pane TUI domain browser. Built on the
existing `DomainService` + cloudflare-go v0.116.0 + the Charm stack (lipgloss v1.1.0,
bubbletea v1.3.10, bubbles). Linked issue: **FEAT-006**.

Two waves: **Wave 1** (G-01..G-05, library + CLI) ships first and is independently
useful; **Wave 2** (G-06..G-07, TUI) builds only on Wave 1's library surface.

## Execution Strategy

- **Wave 1** — the library services (G-01 RedirectService, G-02 RegistrarService) and
  CLI commands (G-04 domains tree, G-05 redirects group) are independent and
  GLM-dispatchable in parallel; G-03 (DomainService enrichment) depends on G-01/G-02.
  Review each worktree diff + re-run tests in the worktree before merge (S334 gate).
- **Wave 2** — the TUI is iterative rendering; best done inline or via a single Opus
  subagent after Wave 1 merges.
- All new code is TDD: failing test → run-fail → implement → run-pass → commit.
- CF API specifics (Rulesets phase shapes, SDK param names) are flagged for
  verification against cloudflare-go v0.116.0 at implementation time.

## Goals

### [ ] G-01 RedirectService library — modern CF Redirect Rules CRUD (Rulesets API)
Covers P-01.

### [ ] G-02 RegistrarService library — registration overlay
Covers P-02.

### [ ] G-03 DomainService enrichment — DomainDetail carries Redirects + Registrar + attention helpers
Covers P-03.

### [ ] G-04 domains CLI command tree — get / stats / redirects / ns
Covers P-04.

### [ ] G-05 redirects CLI command group — Redirect Rules CRUD
Covers P-05.

### [ ] G-06 DomainBrowserModel TUI — split-pane browser with per-domain detail
Covers P-06.

### [ ] G-07 Quick-add redirect prompt + domainDataSource dashboard wiring
Covers P-07.

## Related

- Brainstorm: `docs/brainstorming/2026-06-14-domain-management-center.md`
- Plan: `docs/planning-mode/2026-06-14-domain-management-center.md`
