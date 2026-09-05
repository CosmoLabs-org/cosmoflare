# Design Quality — Composite 40/100 (methodology 2)

> Frontend detected: `desktop/src/` (Tauri + React 18 shell). Impeccable detector UNAVAILABLE — Agent 17's anti-slop facts are partial (manual derivation).

## Score Roll-Up

| Dimension | Score | Weight | Source |
|-----------|-------|--------|--------|
| Design Taste | 30/100 | 50% | agent-17-design-taste.md |
| Accessibility | 50/100 | 50% | agent-18-accessibility.md |
| Motion | skipped | — | no motion evidence (`animation_instances: 0`) |

**Composite**: round(0.50×30 + 0.50×50) = **40/100** (Developing). Only the composite enters the overall score; child scores recorded in scorecard.json `design.child_scores`.

## Accessibility Compliance (STATIC PASS — runtime verification pending)

Three HIGH findings, all escalated to action-plan.md 🔴 Now (per methodology: a11y CRITICALs escalate regardless of composite):

1. **WCAG 4.1.3 Status Messages (AA)** — no `aria-live`/`role="status"` anywhere; every SSE notification, unread-badge change, and connection-state swap is silent to screen readers. For a real-time monitoring dashboard this is a core-function failure. — `desktop/src/views/Notifications.tsx:44`, `App.tsx:146` (see agent-18-accessibility.md)
2. **WCAG 1.4.1 Use of Color (A) + 4.1.2 Name/Role/Value (A)** — health online/offline state lives in an `aria-hidden` dot plus a `title` that never computes into the accessible name; `data-online` doesn't enter the accessibility tree. — `Header.tsx:66-74`
3. **WCAG 2.5.8 Target Size Minimum (AA)** — the missing stylesheet leaves every interactive at UA-default height (<24px); blocks verification of 1.4.11 non-text contrast and 1.4.3 on state colors. — `styles.css:1-15`

Medium: no `<h1>` (headings start at h2), `is-active` not `aria-current="page"`, sidebar is `<aside>` not `<nav>`, unread badge announces a bare number, account select label-less with UA-default colors on a dark theme. Low: card error state is the literal word "Error"; notification fallback dumps raw JSON into the accessible tree.

**What already passes** (preserve): native controls everywhere, no positive tabindex, no `outline:none` (SC 2.1.1, 2.4.7), `lang="en"`+title (SC 3.1.1), base contrast 13.95:1 (AAA headroom), landmarks and real lists. Keyboard nav scored 8/10 — the highest design sub-score of any kind.

## Design Taste Findings

The app is a semantic DOM skeleton with no visual design layer (agent-17):

- **Dead design system**: 30+ `cf-*` classes referenced by components, zero defined in the 15-line `styles.css` (committed once in scaffold `e7401cd`, never expanded; built bundle = 167 bytes). The shipped app renders as an unstyled vertical stack.
- **Health states visually identical**: `is-online`/`is-offline` have no styles — a monitoring dashboard whose primary signal is invisible (compounds the a11y finding: invisible to AT *and* to sighted users).
- **No layout**: `<aside>` stacks above `<main>`; `cf-card-grid` never grids.
- **Two hardcoded hex values total** (`#e6e6e6` on `#1a1a1a` — excellent 13.9:1, but no tokens, no accent, no state colors).
- **Typography**: browser defaults end to end; no scale, weights, or tabular-nums for the dashboard's numeric payload.
- **Stack divergence**: React 18 (profile mandates 19), no Tailwind 4 / shadcn / Zustand 5 / Framer Motion / icon set; brand mark is the `⚡` emoji.
- **anti_slop 9/10 is vacuous** — nothing is styled, so nothing is slop. Not a real signal.

## Motion

Agent 19 skipped (no motion evidence). Agent 17's motion note: zero transitions/animations exist — acceptable for a v0.16 read-only dashboard, no `prefers-reduced-motion` exposure to get wrong. The gap is *informational*: health-dot flips, badge increments, and load transitions all happen with no feedback — a user watching the screen cannot tell anything changed. Once the stylesheet exists, a ~150-200ms ease-out pulse on dot transitions and badge increments is the minimum purposeful motion.

## Impeccable Detector Summary

`impeccable.json`: **unavailable** (detector not installed in this environment; install was deliberately not attempted mid-audit). Agent 17 labeled its anti-slop facts partial. Anti-slop score derived manually: `10 - ceil(1 violation / 3)` = 9 (single violation: default system font, no deliberate choice).

## Cross-Agent Design Patterns

| Finding | Agents | Evidence |
|---------|--------|----------|
| Stylesheet never written (blocked by + blocks everything else) | 17, 18 | desktop/src/styles.css:1-15 |
| Health state invisible (visually AND to AT) | 17, 18 | Header.tsx:65-75 |
| Error states carry no information | 17, 18, 2 (CLI --json parallel) | Dashboard.tsx:30-41 |
| Desktop version drift + one-platform sidecar | 5 (distribution) | desktop/src-tauri/tauri.conf.json:4 |

## Prioritized Design Fixes

| Fix | Severity | Source Agent | File:Line |
|-----|----------|--------------|-----------|
| Write the `cf-*` stylesheet: two-pane flex, card grid, ≥24px (target 44px) targets, `:focus-visible`, semantic tokens, ≥3:1 indicators | HIGH | 17, 18 | desktop/src/styles.css |
| `role="log"` + `aria-live="polite"` notifications; `role="status"` badge + connecting pane | HIGH | 18 | Notifications.tsx:44 |
| Accessible health state via `aria-label="Systems: online\|offline"` + non-color cue | HIGH | 18 | Header.tsx:66 |
| Brand `<div>` → `<h1>`; `aria-current="page"`; `<aside>` → `<nav>` | MED | 18 | Header.tsx:26, App.tsx:134 |
| Error cards: real message + retry + `role="alert"` | MED | 17, 18 | Dashboard.tsx:34 |
| Stack alignment to v2026.02 desktop profile (React 19, Tailwind 4, shadcn, Zustand, Lucide) | MED | 17 | desktop/package.json |
| Micro-motion pulse (~150-200ms ease-out) once styling lands | LOW | 17 (motion note) | — |
