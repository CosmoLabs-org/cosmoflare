# Design Quality — Composite 78/100 (methodology 2)

Desktop tier (`desktop/src/`, Tauri + React/TS). Agent 19 (Motion) skipped — 0 animation instances detected.

## Score Roll-Up

| Dimension | Score | Weight | Source |
|-----------|-------|--------|--------|
| Design Taste | 75/100 | 50% | agent-17-design-taste.md |
| Accessibility | 80/100 | 50% | agent-18-accessibility.md |
| Motion | skipped | — | no motion evidence |

Composite = round(0.50×75 + 0.50×80) = **78**. Only the composite enters the overall mean. Prior audit's design row: 40 (single design-taste score, no composite) → **+38** (dimension pair re-scored).

## Accessibility Compliance (STATIC PASS — runtime verification pending)

Agent 18: "A green score is NOT WCAG certification" — runtime pass (axe in shipped webview, tab order, 200% zoom, VoiceOver walkthrough) still owed.

**CRITICAL (escalated to action-plan 🔴 Now)** — Notifications are silent on every non-Notifications tab: per-view `role="log"` live regions unmount on tab switch (App.tsx:152-154, SC 4.1.3 Status Messages). Fix: always-mounted visually-hidden `aria-live=polite` region at App level + unread count in nav button label. (see agent-18-accessibility.md)

Other findings (all with one-attribute or one-token fixes):
- Health transitions never announced; `aria-label` on role-less span fragile (Header.tsx:69-78) → `role=status` on the HealthDot span
- Online/offline conveyed by dot color alone (Header.tsx:67-78, SC 1.4.1) → visible state text
- Interactive control borders 1.19–1.36:1 vs required 3:1 (styles.css:20, SC 1.4.11) → `--border-strong` token ≥ #7a7a7a
- Unread badge announces bare number (Notifications.tsx:47-54) → hidden " unread notifications" text
- Lightning emoji in h1 announced as "high voltage" (Header.tsx:28) → aria-hidden wrapper
- No forced-colors/high-contrast handling (styles.css:1)

## Design Taste Findings

Strengths: verified contrast ratio table in CSS comments, 4px spacing grid, designed async/loading states, `--control-height` token intent, semantic token names (see agent-17-design-taste.md).

Gaps:
- Accent overexposure — every notification gets the same amber border, no severity encoding (styles.css:327) — prerequisite for any alerting UX
- Dark-only theme; no `[data-theme="light"]` token set for a paid desktop tier (styles.css:12)
- Touch-target drift: 44px nav vs 32px switcher/Mark-read (styles.css:121,183,286)
- No typographic identity — system stack only; 12/13px near-duplicate steps (styles.css:29)
- Pure neutral grays, zero warm bias toward the amber accent (styles.css:17)

## Motion

Agent 19 skipped (0 animation instances). Agent 17's motion note: zero motion is itself a gap — hover states snap, notifications pop in with no enter cue. Recommended: 120ms ease-out CSS transitions on interactive states + ~150ms fade/translate on notification enter — plain CSS, no animation library needed. (see agent-17-design-taste.md)

## Impeccable Detector Summary

`impeccable.json`: **unavailable** — detector not installed in this project. Both agents' detector-derived facts are PARTIAL (direct code reading + hand-computed WCAG math instead). Do not treat anti-slop/a11y facts as detector-verified.

## Cross-Agent Design Patterns

- Both design agents independently verified the documented contrast table (agent 17 as strength, agent 18 as the baseline for the 1.4.11 border finding) — the token system is real, maintained, and auditable.
- Agent 5 (distribution) found the desktop tier's release-side gaps (version skew, unsigned, no updater) — the design quality is not matched by shipping mechanics.

## Prioritized Design Fixes

| Fix | Severity | Source Agent | File:Line |
|-----|----------|--------------|-----------|
| App-level always-mounted live region (4.1.3) | CRITICAL (escalated) | agent-18 | desktop/src/App.tsx:152 |
| role=status on HealthDot span | MED | agent-18 | desktop/src/components/Header.tsx:69 |
| --border-strong token (1.4.11) | MED | agent-18 | desktop/src/styles.css:20 |
| Notification severity encoding | MED | agent-17 | desktop/src/styles.css:327 |
| Light theme token set | MED | agent-17 | desktop/src/styles.css:12 |
| Visible state text beside health dots (1.4.1) | MED | agent-18 | desktop/src/components/Header.tsx:67 |
| Unify control heights 44px | LOW | agent-17 | desktop/src/styles.css:121 |
| Emoji aria-hidden in h1 | LOW | agent-18 | desktop/src/components/Header.tsx:28 |
| CSS micro-transitions (motion note) | LOW | agent-17 | desktop/src/styles.css:1 |
| forced-colors media fallbacks | LOW | agent-18 | desktop/src/styles.css:1 |
