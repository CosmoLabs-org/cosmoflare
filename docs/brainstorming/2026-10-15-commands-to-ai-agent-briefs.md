---
title: 'Commands to AI Agent — deterministic per-domain advisory briefs in Ops'
status: COMPLETED
branch: master
created: "2026-10-15T14:55:00+04:00"
related_issues:
    - FEAT-pC1EV32
roadmap_ref: ROAD-pX0RYK9
tags: [ops, advisory, agent-briefs, determinism, caching]
deliverables:
    - id: BR-01
      title: "Message architecture — kinds x scopes x formats registry in apps/ops/src/briefs/ with the Brief IR"
    - id: BR-02
      title: "Fact collectors — zone cache coverage, origin probe, cache-rule count, firing alerts, quota state (Ops-measured only)"
    - id: BR-03
      title: "agent-brief v1 format — deterministic paste-text template with golden tests"
    - id: BR-04
      title: "GET /api/advisory?zone=<zone> endpoint — parallel collectors, pure compile, Access auth"
    - id: BR-05
      title: "Commands-to-AI-Agent modal — entry points on alerts + domain rows, instant copy UX"
    - id: BR-06
      title: "Verification — vitest goldens, mocked collectors, Playwright live pass, impeccable detect 0"
---

# Commands to AI Agent — deterministic per-domain advisory briefs in Ops

## Why

Operator directive (2026-10-15): "For every domain that has alerts and concerns about caching, intelligently lay out the problem and offer suggestions and solutions we can take to our AI agents in Claude Code / Codex / agentic sessions." Today a finding in Ops dies in the dashboard: a human reads it, remembers the project, opens an agent session, and re-types context. The knowledge of WHAT to do already exists — encoded deterministically in cosmoflare's finding Fix texts (FEAT-067/068/070 rulebook) — but it has no path from the dashboard to the agent that can act.

The bridge: deterministic, copy-paste agent briefs. Future evolution (filed as idea 2026-10-15-direct-agent-integration): connected agents take action autonomously — the manual paste is the v1 stepping stone that ships the message system first.

## Decisions (from the operator Q&A, 2026-10-15)

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | Per-domain consolidation — one brief per zone covering ALL its concerns | "For every domain" — the domain is the unit of dispatch; scattered per-finding cards would force the operator to paste N messages |
| D2 | Ops-measured facts only (no project source) | Works for every zone on the account, including projects without local repos; source-level findings stay in the CLI, which the brief can direct the agent to run |
| D3 | Action depth = CF config directives + cosmoflare verify commands | "The AI agents already know our project — they just need proper direction"; harness-neutral plain text (Claude Code, Codex, anything) |
| D4 | "Commands to AI Agent" button opens a modal with the copy-paste message | Operator-specified UX; entry from alerts and other pages |
| D5 | Extensible message architecture: kinds x formats x scopes | "Cater to any error, any optimization need, anything" — one template registration extends the system |
| D6 | Texts generated deterministically; no LLM in the app | Same state, byte-identical brief; the app is a Worker with no model calls — determinism is the product promise |
| D7 | Library-first respected: compiler is pure and portable (TS v1; Go CLI reuse noted, out of v1 scope) | CosmoLabs 3-tier principle; no premature dual implementation |
| D8 | Approach A (domain advisory) over standalone fleet dashboard (B) or alert-embedded (C) | Rides the FEAT-060 domains surface; B is a v2 layer over the same endpoint; C misses non-alert-class concerns (the 0.4%-hit zone with no alert) |

## Architecture

### 1. Message system (the core extensible artifact) — `apps/ops/src/briefs/`

```
briefs/
  types.ts        — BriefKind, BriefScope, BriefFormat, Brief (the IR)
  registry.ts     — kind -> template registration (THE extension point)
  collect.ts      — fact collectors, one per source, KV-cached via cached()
  compile.ts      — pure: (scope, facts) -> Brief
  render.ts       — format renderers; agent-brief v1
  templates/      — one file per finding class
```

- **Kinds (v1)**: `low-cache-hit`, `html-cache-trap`, `cache-rule-gap`, `alert-firing`, `quota-pressure`. A new kind = one template file + one registry line.
- **Scopes**: `zone` v1; `worker`/`project`/`account` reserved in the type union.
- **Formats**: `agent-brief` v1; `slack`/`ticket` later — a format is one render function over the Brief IR; adding one never touches collectors or templates.
- **Brief IR** (plain data): `{ scope, zone, window, findings: [{kind, severity, evidence}], actions: [{order, directive, command?}], verify: [commands] }`. Templates emit findings+actions; renderers shape prose. Same facts -> same IR -> byte-identical text.

### 2. Fact collectors (Ops-measured only)

| Fact | Source | Cache | Feeds kind |
|---|---|---|---|
| Zone cache coverage | GraphQL `httpRequests1dGroups` per zone (7d window; the CLI `ZoneCacheCoverage` query ported to TS) | 1h via `cached()` | `low-cache-hit` |
| Origin probe | `GET https://<zone>/` — content-type, cache-control, cf-cache-status | 1h | `html-cache-trap` |
| Cache Rules count | derived from the rules-quota collector's `cache-rules` family (`rules_quota.ts` reads the `rulesets/phases/http_request_cache_settings/entrypoint`) | 5 min (self-cached) | `cache-rule-gap` |
| Firing alerts | KV `alert-feed` (live `AlertFeedEntry[]`), filtered by the scope segment of `ruleId` (`<ruleName>/<zoneKey>`) | live | `alert-firing` |
| Rule-quota state | existing rules-quota collector | existing | `quota-pressure` |

Thresholds mirror the CLI (low-cache-hit: requests >= 1000 && hitRate < 5%) so Ops and `cosmoflare efficiency` never disagree. Ops then grades severity for dispatch priority (the CLI's own finding severities are `info`/`warning`): low-cache-hit < 5% → warning, < 1% → critical; html-cache-trap → warning, critical when the zone has zero Cache Rules; quota-pressure ≥ 80% of an allowance → warning, ≥ 95% → critical.

### 3. agent-brief format v1

```
# <zone> — Cloudflare optimization directive

## Problem
- <severity-ranked findings, each: measured numbers + window>

## Actions
1. <CF config directive — exact dash path or API call>
2. <cosmoflare verify command with expected outcome>

## Verify
- <acceptance command(s) the agent runs>

## Context
Zone <name> (<id>) — measured <window> by CosmoLabs Ops.
```

Prose derives from the FEAT-067/068/070 rulebook already encoded in the CLI's Fix texts — directives, not education. No filler.

### 4. API

`GET /api/advisory?zone=<zone>` -> `{ zone, generatedAt, facts, brief, rendered: { agentBrief } }`. Collectors read through their KV caches; compile is pure per-request. Access-JWT auth like every `/api/*` route; `Cache-Control: private, no-store` envelope. Registered as a `switch (url.pathname)` case with a query param, following the existing per-entity pattern (see `/api/zone-profile?name=` and `/api/dns/records?id=` — the worker has no path-param routing). Fleet index (`GET /api/advisory` with no params, "N domains need attention") is v2.

### 5. UI — the modal

Entry points v1: each firing alert card (Alerts page — exists today, the primary path) + the domain row on the domains view (lands WITH FEAT-060, which is still open — the modal component and endpoint ship first; the domain-row button is a one-line add when FEAT-060's view merges). Modal: fetch-on-open, mono brief block, header "Commands to AI Agent · <zone>", one primary Copy button -> clipboard write, checkmark + 150ms ease-out fade, auto-dismiss 1.5s. Shimmer while loading; Retry on error. Kind-filter chips deferred to v2 (v1 always full severity-ordered brief). Taste rules apply: no tile outlines, no left borders, existing Ops component vocabulary; impeccable detect = 0 on new files.

### 6. Testing / determinism proof

- Golden tests: fixture fact-sets -> exact output string assertions, per v1 kind + one mixed brief — the byte-identical proof.
- Collector tests: mocked fetch (existing `*.test.ts` patterns), cache short-circuit coverage.
- UI: stubbed-API flow tests + a Playwright pass against real handlers on the dev URL (live-data rule — fixtures hid a real bug once).
- `npx impeccable detect` = 0 on new UI files.

## Future expansion (roadmap)

- **Fleet advisory index** (v2): `GET /api/advisory` + "N domains need attention" count on the domains view — zero template changes.
- **New kinds**: worker errors/CPU (`alert-firing` generalization), DNS misconfig, SSL expiry, WAF coverage, R2/D1 spend — each is one template + one collector.
- **New formats**: Slack card, GitHub issue, Linear ticket — one render function each.
- **New scopes**: worker-scoped briefs (worker profile pages), project-scoped (multi-zone).
- **Direct agent integration** (idea 2026-10-15): authenticated dispatch channel, agents act with human-gated approvals — the manual paste retires.
- **Go CLI parity**: `cosmoflare advisory <zone> --format agent-brief` reusing the same templates — library-first bridge, after v1 proves the template set.

## Out of scope (v1)

- Any LLM call inside Ops (determinism is the contract).
- Project-source facts (CLI's `efficiency caching`/`ttl` own that layer).
- Mutation actions — briefs direct, agents execute; Ops stays read-only + purge/toggle as today.
