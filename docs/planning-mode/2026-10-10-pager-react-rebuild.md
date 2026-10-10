---
title: Ops mobile app — React + Tailwind + shadcn + Framer Motion rebuild
created: 2026-10-10T21:25:00+04:00
status: in_progress
issue: IMP-003
deliverables:
    - P-01: "Toolchain — React 18, Tailwind v4 (@tailwindcss/vite), framer-motion, shadcn/ui base (components.json, cn util), deps installed, build green with the vanilla app still mounted"
    - P-02: "Design tokens as Tailwind theme — the :root custom properties (bg/surface/border/text/muted/accent/sev-*, spacing 4-32, radius, the four font roles) map 1:1 into the Tailwind v4 theme config; impeccable detector stays 0"
    - P-03: "React shell — App root mounts, hash router preserved (#/overview..#/worker/<name>), drawer+sidebar as shadcn Sheet at <1024px / static sidebar at >=1024, quicknav, top bar with global Refresh, bottom status strip; service worker + push untouched"
    - P-04: "Data views tier 1 — Overview (ring gauges as React SVG components, pure arc math reused verbatim from gauges.ts; KPI tiles; attention disclosures; plan line), Billing (grade+overage pair cards, bar gauges, ranked consumers)"
    - P-05: "Data views tier 2 — Workers + worker profile route, D1, Zones, Durable Objects, Projects (ranked rows), Domains (list + profile) on the shared table/row components"
    - P-06: "Interactive views — Rules editor (draft state, chips, validation), Alerts (IndexedDB list), Pairing (push ritual)"
    - P-07: "Motion + polish — Framer Motion: ease-out page/view enters, ring draw-in, drawer sheet spring, reduced-motion respected; shadcn primitives adopted where they fit (Sheet, Toggle, Dialog for confirmations)"
    - P-08: "Test port — pure-module tests survive untouched; view tests ported to @testing-library/react; suite green throughout; production build + impeccable detector 0"
requires_reading:
    - docs/SPEC.md
    - docs/brainstorming/2026-10-10-ai-gateway-monitoring.md
schema_version: 1
---

# Ops mobile app — React + Tailwind + shadcn + Framer Motion rebuild

## Goal

Rebuild the Ops mobile app (still "pager/" in the tree during the port) on
React 18 + Tailwind v4 + shadcn/ui + Framer Motion. Operator decisions
2026-10-10: bundle weight accepted (PWA install makes it one-time); one
component language across desktop and mobile tiers; "the best CSS possible
and the best libraries".

## What survives untouched (the port's safety net)

- `pager/src/api.ts` — ApiClient, all payload types, fixtures fallback
- `pager/src/store.ts` — IndexedDB alert history
- Pure modules and their tests (the majority of the 179): `format`,
  `gauges` (arc math, level bands, label splitting), `attention`, `sort`,
  `routes` (hash parsing), `payload`, `vapid`, `statusstrip` (pure half)
- `public/sw.js`, manifest, icons — push delivery unchanged
- `apps/ops` worker — no API changes anywhere in this plan

## Execution Model

Sequential phases P-01..P-08, app green at each step. Phases P-04..P-06
dispatch as bounded GLM briefs (one view tier per agent, exact file lists);
P-01..P-03 and P-07..P-08 land in-session or as single briefs. Merge order
is phase order; no parallel view-tier agents (shared shell files).

## Steps

1. **P-01 Toolchain**: `bun add react react-dom framer-motion`; Tailwind v4
   via `@tailwindcss/vite` in `pager/vite.config.ts`; shadcn base
   (`components.json`, `src/lib/utils.ts` with cn — prefer
   `@cosmolabs-org/cosmokit/utils` per the CosmoKit adoption rule, fallback
   clsx+tailwind-merge). Verify: `bun run build` green, vanilla app intact.
2. **P-02 Tokens**: Tailwind theme maps the :root tokens verbatim (no new
   colors); font roles load from the existing Google Fonts links. Verify:
   build + detector 0.
3. **P-03 Shell**: `src/main-react.tsx` mounts `App`; hash router hook
   (keep custom — routes and `#/worker/<name>` semantics preserved);
   Sheet-drawer; quicknav; global Refresh in the top bar; status strip.
   Mount swap: index.html script tag switches to the React entry. Verify:
   build, manual route smoke via tests.
4. **P-04..P-06 Views**: port per tier (briefs). Ring/bar gauges become
   React SVG components rendering EXACTLY the same geometry (butt caps,
   dash math from gauges.ts reused as-is). Rules editor keeps its draft +
   PUT semantics (BUG-057 defects stay fixed). Verify per tier: tests +
   build + detector.
5. **P-07 Motion**: page/view enters ease-out 150-200ms, ring arcs draw in,
   Sheet spring; `prefers-reduced-motion` gates all of it.
6. **P-08 Close**: full suite green, `bun run build` green, detector 0,
   deploy on operator go, SPEC note updated, IMP-003 progressed.

## Non-goals

- No worker/API changes; no route changes; no copy changes; no new data.
- No desktop-tier changes (it already runs this stack).
