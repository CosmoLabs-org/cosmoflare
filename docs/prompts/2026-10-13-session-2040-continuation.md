---
title: "Session 2040 Continuation — full design review + finish what we started"
created: 2026-10-13T00:30:00+04:00
status: PENDING
branch: master
goals_total: 9
goals_completed: 0
supersedes: "docs/prompts/2026-10-12-session-2037-continuation.md"
schema_version: 1
---

## Context

Operator directive (2026-10-13): "look over the ENTIRE design of the app, find better opportunities, strengthen the experience, placement and layouts, address the shortcomings you didn't really do for me, and finish what we started."

State at handoff:
- Live worker version: 565da19e (2026-10-12 ~21:41 UTC). Critique baseline: 32/40, snapshot at `.impeccable/critique/2026-10-12T19-45-33Z__pager-src.md`.
- **UNCOMMITTED working-tree changes (tested green: 299 pager tests + tsc, but NOT built/committed/deployed — the operator stopped that chain):** `pager/src/app/views/BillingView.tsx` (24/30 counter lockup + is-hot period track), `pager/src/app/App.tsx` (forced `reg.update()` on every open so the update modal fires on iOS), `pager/src/styles.css` (.cf-progress instrument track + .cf-cycle-lg). Review them, run gates, commit, deploy.
- Master is ~30 commits ahead of origin (evening design wave unsynced).

## Goals

### [ ] G-01 Land the stopped period-dial work
Review the three uncommitted files above; gates (pager vitest + tsc + build + impeccable detect 0); commit; deploy; have the operator close-reopen and confirm.

### [ ] G-02 FULL design review pass (the operator's headline ask)
Every view, not just Billing: placement, layout, hierarchy, spacing, states, copy. Load the impeccable skill, read the critique snapshot, fix its open items AND find new ones on a fresh pass. Explicitly address the shortcomings the operator named but never got fully: view-intro description lines styling (only margins were fixed — they also asked for a better font there), billing card composition, Overview KPI row balance.

### [ ] G-03 Drawer keyboard hardening (critique P1)
Escape closes the drawer; focus moves in on open and restores on close (UpdateModal already does this — mirror it).

### [ ] G-04 Distill + typeset (critique P2)
Drop the billing-day KPI tile on Overview (status bar already carries it; KPI row drops to 4, fixing the chunking failure). Lever labels off 10px (≥11px); armed-state text must not reflow narrow rows (min-width or fixed label slot).

### [ ] G-05 Theme wave 2 (FEAT-080 continues + FEAT-046)
Micro-animations and shader-adjacent polish within house rules (severity-gated glows, reduced-motion safe, detector 0). One authored moment per screen, not scattered effects.

### [ ] G-06 Workers profile expansion
The profile exists but is shallow: add bindings, triggers, routes, deploy versions; live tail is the stretch. Follow DomainProfile's section patterns; real numbers only.

### [ ] G-07 On-device acceptance round
Have the operator confirm: sticky tab bar under scroll (iOS backdrop-filter fix), mobile purge lever + worker drill-in, update modal on open, billing 24/30 dial + $5.00 cost lockup.

### [ ] G-08 Critique minor items
timeAgo goes stale on long-lived screens (needs a tick); card h2 hierarchy (12px uppercase muted vs masthead); RulesView draft Delete needs the arm/confirm convention.

### [ ] G-09 Gate + merge the two in-flight quick-win agents
Dispatched at session close (2026-10-13, not yet returned): one CSS craft agent (description-line font, lever label legibility, card heading weight) and one TSX agent (AlertsView 60s time-ago tick + RulesView two-step draft delete). Find them via `ccs glm-agent status` (the two newest, running/done), review each diff, re-run pager gates in the worktrees, S334-gate, and `ccs merge` both.

## Carry-Over Tasks

1. Operator decisions pending: IDEA-019/020 pushed-duplicate repair (needs explicit go — it pushes); 10 stale-issue candidates (offer the verify-then-close scout sweep); `/independent-review --scan` for 142 unreviewed docs; v0.33.0 release (operator go only); FEAT-029 auth (conflicts with the no-keychain rule — needs a design decision).
2. Sync ~30 commits (operator asks for it explicitly each time — do not push unprompted).
3. G-01 of the previous prompt (cron push acceptance on the phone) was never explicitly confirmed by the operator — close it during the G-07 round.

## File Scope

- Uncommitted: pager/src/app/views/BillingView.tsx, pager/src/app/App.tsx, pager/src/styles.css
- Likely: pager/src/app/views/* (all), pager/src/app/components/*, pager/src/styles.css, apps/ops/src/* for G-06 data
