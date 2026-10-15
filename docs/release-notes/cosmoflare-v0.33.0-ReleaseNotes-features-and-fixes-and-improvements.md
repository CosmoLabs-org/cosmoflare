---
project: cosmoflare
type: release
version: 0.33.0
source: .version-registry.json
date: 2026-10-15T23:10:11.663279+04:00
previous: 0.32.0
slug: features-and-fixes-and-improvements
title: "features-and-fixes-and-improvements Release"
---

# cosmoflare v0.33.0 Release Notes

**Release Date**: October 15, 2026

**Previous**: v0.32.0

## Overview

This release brings 93 new features, 52 bug fixes, and 2 improvements.

## Highlights

Advisory era: GET /api/advisory plus the pager's Commands-to-AI-Agent modal turn firing alerts into deterministic, copy-paste agent directives — measured per-zone facts (7d coverage, origin probe, rules quota, live alerts) compiled and rendered byte-identically, verified live against the real account and in a real browser. The cost command now estimates from measured account analytics instead of zero-value placeholders (R2/Workers/KV with 30d projection and window honesty), dns update gains its documented --content flag, and the limits catalog test counts catch up to the kv.requests_monthly entry.

## What's New

- BR-03: LimitsService v2 — embedded 66-entry limits catalog (schema v1, provenance + 90-day stale warning) as single source of truth; hardcoded limit maps deleted; per-service plan resolution chain; limits catalog export command
- Usage-waste alerting wave 1: stuck-work thresholds for Workers and Durable Objects (FEAT-047) (FEAT-047)
- Usage pacing wave 2: monthly usage view + projected-overage alerts (FEAT-048) (FEAT-048)
- zone cache + D1 rows-read alerts (FEAT-049 wave 3a): zone-cache-miss-pct, zone-uncached-requests, d1-rows-read — name the zone or database, page at most hourly per offender unless it doubles, page a telemetry gap when data cannot be collected; per-rule --exclude; D1 monthly rows-read in cosmoflare usage; opt-in live API smoke test
- Cosmoflare Ops (FEAT-052): private dashboard and pager at ops.cosmolabs.org, behind Cloudflare Access (owner email only, custom domain only) — month-to-date usage pacing, billing-period cost view with per-product overage projection and per-project attribution, per-Worker/D1/zone detail with a responsive hamburger-drawer UI and new icon set, a KV-backed upstream cache, and a 5-minute Worker cron that sends Web Push alerts
- Desktop app redesign on Tailwind v4 + shadcn/ui: Ops design tokens, logo mark, icon sidebar with unread badge, Dashboard and Notifications rebuilt on shadcn primitives, light/dark theme with dark default (FEAT-055)
- CLI: cosmoflare search — fuzzy command discovery with --json envelope (FEAT-pEKR6K6)
- CLI: stable error_code field in --json error output, wave 1 mechanism + envelope (FEAT-p8KYM5K)
- Ops projects view: per-project consumption and overage across Cloudflare, zero new upstream calls (FEAT-p0QKZT0)
- CLI: --help gains an agent contract preamble when run inside agent harnesses (FEAT-pN68ZRM)
- Ops domains view: full domain list with status, paused badges, and renewal attention coloring (FEAT-p9GD588)
- Ops tables: top-right sort controls with direction toggle and saved preference, largest-first by default (FEAT-pRC6EDA)
- Ops D1/Workers lists: graded per-row bars showing each row's share of the product allowance
- Ops domains: tap-through profile page per domain — status, renewal, plan, nameservers, dates
- Ops sidebar/drawer: per-section counters (domains, workers, D1, zones, projects, rules)
- Ops domain profiles: enriched sections (renewal, registration, DNS with original nameservers, DNSSEC, SSL, 24h traffic) + sidebar section counters
- Ops Durable Objects section: namespaces with per-script 24h usage; first shared view components; Nunito brand/link face (FEAT-p3KJAC2)
- kv-writes alert condition on both engines — per-namespace KV write volume over 24h; the Ops Worker pages from kvOperationsAdaptiveGroups (write rows only) and the Go watch/check evaluates the same condition with cached titles, ID fallback and telemetry-gap paging (FEAT-066) (FEAT-066)
- bounded retry on the control plane and worker loads — RetryTransport wraps the knowledge registry at both chokepoints (2 retries, 250ms exponential backoff with jitter, 429 always with body rewind, 5xx idempotent-only); the worker's nine upstream fetch sites adopt the mirrored fetchRetry (FEAT-061)
- Ops mobile app rebuilt on React + Tailwind v4 + shadcn/ui + Framer Motion — React shell with quick nav, drawer, section mastheads and a type system (Spline Sans Mono instrument labels, Schibsted Grotesk figures); every view ported behind a bridge that kept the app green; exact 132px ring endpoints, grade+overage pairs, ranked rows, worker and domain profiles, gradient hierarchy with reduced-motion respected
- wire measured usage into all five cost commands (commit:a3afb07d)
- efficiency ttl — verdicts from real D1 write cadence (P-14, FEAT-068/070) (commit:80371fe3)
- efficiency caching — P1-P5 pattern findings from Worker source (P-13, FEAT-067) (commit:2ba53e43)
- Overview rule-quota strip — rotating fleet summary (ROAD-p9X66J9) (commit:55559365)
- article wave 2 — 25 more Cloudflare articles (ROAD-p9X66J9) (commit:fedf555d)
- source links, mobile-safe tables — article upgrades (ROAD-p9X66J9) (commit:e0a0b9bb)
- education hub first slice — 30 Cloudflare articles + rule-quota display (ROAD-p9X66J9) (commit:fbc81d20)
- table block type + renderer — reference tables for the education hub (ROAD-p9X66J9) (commit:9ecd0e09)
- KV operations dimension — spend row with honest limit-unknown (commit:56ffe135)
- HTML caching-trap detector — s-maxage on HTML the edge is not honoring (commit:a010281e)
- zone cache coverage — low-hit findings from httpRequests1dGroups (commit:edd9f2c8)
- analytics kv — per-namespace KV operation totals (FEAT-069) (commit:8e1f72e9)
- worker profile Tail section — bounded NDJSON live tail UI (commit:12ad41c4)
- idle-waste line from efficiency collectors — counts + honest $0 basis (commit:46260cb6)
- cosmoflare efficiency — idle-resource findings from real analytics (commit:d8afc1f0)
- EfficiencyService — idle workers, dead KV namespaces, empty R2 buckets from real analytics (commit:3a97a4d3)
- profiles everywhere — D1, Durable Objects and zone drill-ins (commit:9e432859)
- pill tuning round two — wider lens edge, taller, calmer tabs (commit:9338992d)
- tighter pill with Plus overflow + update button as progress (commit:f393c2a7)
- rules editor overhaul — switches, data numerals, iconed actions (commit:5c1a02ed)
- the pill owns the bottom — status strip retired (commit:d2006fba)
- pill nav tuning — droplet lens + graceful activation (commit:ea8dbfaa)
- worker profile expansion — bindings, crons, routes, versions (commit:1f8d7ee2)
- liquid glass tab pill — phase 2, WebGL refraction (commit:a23b67d7)
- liquid glass tab pill — phase 1, the CSS material (commit:7e4c132f)
- the month on one line — billing + overview share the cost lockup (commit:8e428bcf)
- month as an equation + period dial + update check on open (commit:71a7deaf)
- monthly cost lockup + ring craft pass + JetBrains Mono data face (commit:fdca0ddf)
- stable error codes — workers wraps + USAGE codes table (FEAT-058 wave 2c) (commit:c88066c1)
- audit dead-functions — shadowed functions/ beside _worker.js (FEAT-071) (commit:f019b988)
- email routing metrics collector + overview tile (FEAT-059) (commit:7a87cadc)
- d1 schema-diff — local migrations vs remote parity (FEAT-051 slice) (commit:bff7a423)
- auth preflight — real token scopes + per-command scope map (FEAT-072) (commit:954db30b)
- tab bar polish + LeverButton — one reusable action lever app-wide (FEAT-pCT8CT2) (commit:e32ce8a5)
- WAF custom rules list + enable/disable action buttons (wave 3, FEAT-074 thread) (commit:c4a8184a)
- update modal — centered, smooth, appears the moment a deploy lands (commit:c7fecfb5)
- push polish — 6h repeat cooldown, one digest push per run, human headlines, live in-app alerts (FEAT-p2H8GZ6) (commit:0d5a4495)
- notifications center — fired-alert feed with plain-language explanations (commit:1dacec93)
- mobile bottom tab bar (5 primary tabs + More) (commit:fb47c1b7)
- logo mark v2 — pulse-hero C, full-bleed, icon master (commit:47fe6a93)
- install guidance banner (FEAT-pXNQ5GW) (commit:48d8cacf)
- pairing polish — paired state + unpair, step checklist, friendly errors, install hint (commit:4e5e0dcf)
- per-section action hints — every page says what to do (commit:dbb65b3f)
- proper skeleton, empty and error states across all views (commit:8e211b9c)
- section totals move to masthead heading (commit:365b0a44)
- bottom status bar rebuilt — day progress bar right, updated-ago left (commit:4993c968)
- section nav as a glowing button list (operator pass 2026-10-11) (commit:1086e85a)
- AI Gateway collector + /api/ai — per-gateway model usage (BR-02) (commit:5de0f2c2)
- React views tier 2b — Projects + Domains (P-05b) (commit:3ca9f774)
- React views tier 2a — tables family (P-05a) (commit:c062d89c)
- gradient hierarchy + microanimations (P-07 layer) (commit:f2f1c02e)
- section masthead, nav icons everywhere, bigger clear rings (commit:82759ce7)
- stat indicators, whole-row tap, icon sort buttons, ranked separation (commit:17effb0e)
- profile + brand polish — instrument-sheet profiles, live section lockup (commit:4462bc6a)
- React shell swap — P-02 tokens + P-03 shell (rebuild phases 2-3) (commit:4a108a48)
- React rebuild phase 0 — ADR flipped, toolchain staged (commit:d55b41f8)
- type system pass + exact ring endpoints + statusbar styling (commit:bbed724b)
- quick section nav + ranked rows in Projects (IMP-003) (commit:1d6d3596)
- kv-writes condition — per-namespace KV writes/day (FB-29) (commit:27932474)
- overview rings carry their projected overage dollars (commit:e1619b10)
- worker profile pages — tap a worker for its 24h profile (commit:113c8df1)
- tappable attention rows; centered KPI type (commit:98895808)
- billing cards always show grade + overage pair (commit:47dcfcf4)
- top consumers as rankings — rank, who, share bar, % right (commit:68584a29)
- Durable Objects section + shared view components + Nunito brand face (commit:2733246c)
- enriched domain profile + nav counters (commit:3df8f782)
- right-aligned section counters in sidebar and drawer (commit:3bb81ecc)
- domain profile page — tap a domain, get the full picture (commit:3f9722df)
- graded per-row usage bars on D1 and Workers lists (commit:bdad3c98)
- tappable domain detail sheet + animated pairing glyph (commit:3ba5d220)
- centered pairing ritual with stateful buttons; overview plan line (commit:f3cddc54)
- refresh icon button with loading spin, success check and reduced-motion fallback (IMP-pPFDXJA) (commit:cba22ec6)
- limits corpus synthesis — unified catalog-draft.json (66 entries), conflicts register, brainstorm corpus findings (goals G-02..G-06 ticked) (commit:b47b5ed4)

## Bug Fixes

- alert fan-out conditions now page every offending script/DO, not only the first one per cycle (BUG-054)
- cosmoflare metrics zone traffic works again — the zone analytics query no longer fails with unknown field count (BUG-055)
- alert page text prints percent values with one decimal
- push alerts never reached iPhones: VAPID subject was mailto:mailto:, and the pager subscribed without the VAPID application server key
- Ops pager: 7 integration fixes (BUG-057) — rules cache after save, honest data age with revalidation repaint, refresh keeps old DOM and button, legend CSS scoped, request-level rule errors, warning-text style, test-fire redirect handling
- Ops Worker: cron KV keys renamed cron:d1-list/cron:zones with 24h TTL and scheduled-handler rejection audit line (BUG-058)
- Ops pager gauge nits: one status per billing card, solid legend swatches, storage rings show the size once (IMP-002)
- Ops pager: background revalidation no longer resets table sort/filter or wipes unsaved rule edits; storage rings show the allowance once (combined-review findings)
- Ops overview rings: fixed hairline arcs (stroke-width was unset), softer projected halo, allowance moved below the ring
- CLI: errors carry HTTP status + API cause; --json stdout is pure envelope; auth status shows real token scopes (BUG-pPF2BCF)
- Ops pager: fixed an infinite fetch loop (BUG-pZKJ5VB) that re-requested API endpoints ~50x/second on every page since the React rebuild and stuck iOS on the skeleton loader; fetchers now use stable module-level identity, with regression tests on the production path
- review type fixes — ZoneFacts import, synthetic ctx cast, probe null type, narrowed gap probe (commit:90527bfa)
- rules-quota — shared paginated zone list + summary age honesty (BUG-pP1YH9W, BUG-p9GM11M) (commit:2edf2de1)
- end kvCacheOptions at the attachment close (commit:9008e203)
- independent review of brainplan pair — real routing/alert/quota shapes, pager paths, verified CLI surface (commit:a238ccfc)
- review fixes — scanner window/auth/test-file/keyPrefix, TTL default+cadence, JSON null findings (commit:56b036ff)
- KV operations split — the actionType dimension exists after all (commit:f9d82eac)
- httpRequests1dGroups is date-granular — date_geq/date_lt with Date vars (commit:458f284e)
- match live KV analytics schema — dimensions.namespaceId + sum.requests (commit:006aa488)
- critique minors sweep — recognition, order, honest chrome (commit:3d94a930)
- frosted AND bending — the full Apple material (commit:bce60118)
- overscroll burn-in caught + the distortion becomes visible (commit:b2ba4cfe)
- the lens owns the full pill height — distortion in both directions (commit:9bf639d0)
- the pill is fixed to the viewport again — settled refresh (commit:080f39bb)
- glass recalibrated, burn-in fixed, build marker in the drawer (commit:6e2c7743)
- selected tab calmer — smaller zoom, end padding, gliding lift (commit:18a43127)
- clearer glass — the pill stops reading as opaque (commit:adb98c4d)
- pill fully overlays + the update modal can actually fire (commit:1b6c0bce)
- critique wave — glyph-only nav, true copy, distilled overview (commit:6b0b9361)
- drawer keyboard hardening — Escape, focus in, focus restore (commit:e676b325)
- craft wave — intro font, honest aging, two-step delete (commit:fb920d72)
- malformed font link broke the HTML build; overshoot moves from the easing curve into the keyframes (detector-clean) (commit:44ab2d9a)
- mobile cards render rich cells — purge lever + worker drill-in live on phones (critique P0/P1) (commit:0eafee32)
- one active tab mark + a selected state you can feel (commit:72ae7d0d)
- forced update path + billing progress redesign + layout tidying (commit:62b276f7)
- cache-miss alert floor + absolute base (BUG-pEPB03A) (commit:7d9042f6)
- pair Enable errors name the install-first path when Notification is missing (reviewer amendment) (commit:5eec1ab2)
- stable default fetcher ends infinite fetch loop (stuck skeleton on iOS) (commit:a43e03b2)
- quicknav as a real navigation bar — rectangular segments, no counters (commit:23e0bc24)
- complete P-06 salvage — starter-flag test seam, nullable hint, unused import (commit:e50258a1)
- complete P-04 salvage — dedupe import, finish overage rename (commit:53384f35)
- AIGatewayLog matches the live REST schema (BR-01) (commit:757881e9)
- domain profile labels original nameservers as historical (commit:213574ab)
- token-only S3 data plane fails fast with guidance (BUG-p3Y31ZQ) (commit:3fbc2bca)
- DO allowance bars — the lookup used the wrong product id (commit:95e78a9f)
- rolling KV never drives allowance visuals (BUG-pFAKDN3 follow-up) (commit:f093ea3d)
- overview rings drew as hairlines — real stroke width + decluttered center (commit:94f781a8)
- storage rings showed the allowance twice (commit:839be473)
- preserve view state across background revalidation (commit:c756dc1c)
- gauge polish — projected-first ring centers, proper labels, legend, top-8 rings, today dot (FEAT-pDQ8JET) (commit:4accd32d)
- billing cycle anchored on the 23rd (BUG-pNXAHPD); file usage-gauges feature and refresh-control improvement (commit:67350ca1)
- add doc.go to 12 test-only packages — go build ./... passes again (commit:0641bb71)

## Improvements

- Ops pager styles pass: token-driven sections (spacing/type/motion), Space Grotesk display face, amber-tinted neutrals, deduplicated stylesheet (G-11)
- simplify pass over wave 3a — shared HumanCount, pacing-exempt flag, scope constants, one telemetry entry point (commit:351e7bfd)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 618 |
| Files changed | 703 |
| New features | 93 |
| Bug fixes | 52 |

---
_Full changelog: CHANGELOG.md_
