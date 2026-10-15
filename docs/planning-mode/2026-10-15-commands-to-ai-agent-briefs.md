---
title: 'Commands to AI Agent — implementation (FEAT-pC1EV32 / ROAD-pX0RYK9)'
status: PLANNED
branch: master
created: "2026-10-15T15:20:00+04:00"
brainstorm_ref: docs/brainstorming/2026-10-15-commands-to-ai-agent-briefs.md
related_issues:
    - FEAT-pC1EV32
roadmap_ref: ROAD-pX0RYK9
tags: [ops, advisory, agent-briefs, wave-1]
deliverables:
    - id: P-15
      title: "briefs core — types/registry/collectors/compile in apps/ops/src/briefs/ (pure, tested)"
    - id: P-16
      title: "v1 templates + agent-brief renderer with golden tests (byte-identical proof)"
    - id: P-17
      title: "GET /api/advisory/:zone endpoint — parallel collectors, Access auth, no-store"
    - id: P-18
      title: "Commands-to-AI-Agent modal + alert-card entry point, impeccable detect 0"
    - id: P-19
      title: "Playwright live pass on the dev URL + README API docs"
---

# Commands to AI Agent — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development or executing-plans, task-by-task.

**Goal:** Ship the v1 of FEAT-pC1EV32: per-zone deterministic advisory briefs in Ops, rendered into a copy-paste "Commands to AI Agent" modal.

**Architecture:** New `apps/ops/src/briefs/` module family (types → registry → collectors → pure compiler → renderer) behind one new API route; one modal component in the `pager/` PWA with an entry button on firing alert cards. Design: `docs/brainstorming/2026-10-15-commands-to-ai-agent-briefs.md` (decisions D1-D8 binding).

**Tech Stack:** TypeScript, Cloudflare Worker (Hono-style routing as exists in `src/index.ts`), `cached()` from `src/cache.ts`, vitest (`apps/ops/package.json` runner — verify with `bun test` per repo convention), pager PWA build.

**Spec:** the brainstorm doc above; FEAT-pC1EV32 body; Ops taste rules (no tile outlines/left borders, impeccable detect = 0).

## Global Constraints

Determinism is the contract: same facts → byte-identical `agentBrief` string (goldens prove it). No LLM calls anywhere. Real numbers only — every figure in a brief traces to a collector fact. `--json`-equivalent envelope discipline: API responses carry `Cache-Control: private, no-store`. No mutations: advisory is read-only. Conventional commits; TDD red→green per named test; never break existing `/api/*` routes.

## Reference files (read before implementing)

`apps/ops/src/index.ts` (route registration + Access auth gate) · `apps/ops/src/cache.ts` (cached() — THE cache layer) · `apps/ops/src/summary.ts` (GraphQL analytics query pattern; already computes zone cache health) · `apps/ops/src/rules_quota.ts` (collector + KV pattern) · `apps/ops/src/rules.ts` + alerts feed (firing-alert store shape) · `pkg/cosmoflare/caching.go` (the HTML-trap verdict logic to port: content-type + cache-control parse + cf-cache-status classes) · `pkg/cosmoflare/analytics.go` ZoneCacheCoverage query shape · `pager/` tree (existing component vocabulary) · `apps/ops/README.md` (API docs to extend in P-19).

---

## Task B1: briefs core — types, collectors, compiler (P-15)

**Model:** sonnet · **Files:** create `apps/ops/src/briefs/types.ts`, `briefs/collect.ts`, `briefs/compile.ts`, `briefs/compile.test.ts`, `briefs/collect.test.ts`

**Interfaces (exact — later tasks compile against these):**

```ts
// types.ts
export type BriefKind = 'low-cache-hit' | 'html-cache-trap' | 'cache-rule-gap' | 'alert-firing' | 'quota-pressure';
export type BriefScope = 'zone';                    // v2: 'worker' | 'project' | 'account'
export type BriefFormat = 'agent-brief';            // v2: 'slack' | 'ticket'
export interface BriefFinding { kind: BriefKind; severity: 'critical' | 'warning' | 'info'; evidence: string }
export interface BriefAction { directive: string; command?: string }
export interface Brief {
  scope: BriefScope; zone: string; zoneId: string; window: string;
  findings: BriefFinding[];           // severity-ranked, stable order for identical facts
  actions: BriefAction[];             // numbered at render time
  verify: string[];                   // acceptance commands
}
export interface ZoneFacts {
  zone: string; zoneId: string; window: string;
  coverage?: { requests: number; cached: number; hitRatePct: number };   // httpRequests1dGroups 7d
  probe?: { contentType: string; cacheControl: string; cfCacheStatus: string };
  cacheRuleCount?: number;
  firingAlerts: { ruleId: string; title: string; value: number; threshold: number; unit: string }[];
  quota?: { usedPct: number; family: string };
}
```

- `collect.ts`: `collectZoneFacts(env, zone): Promise<ZoneFacts>` — five collectors in parallel (`Promise.all`): coverage (GraphQL, port the CLI's `httpRequests1dGroups` shape — reference `summary.ts`'s zone-cache code first; if summary.ts already exposes a reusable per-zone coverage function, CALL it, do not duplicate), origin probe (plain fetch of `https://<zone>/`, 10s timeout, headers only), cache-rule count (REST, best-effort — 403/404 → undefined, never fails the advisory), firing alerts (existing store filtered by zone name), quota (existing rules-quota summary lookup). Every remote read goes through `cached()` (1h TTL) EXCEPT the alert store (live KV read).
- `compile.ts`: `compileBrief(facts: ZoneFacts): Brief` — PURE, no env, no fetch. Thresholds mirror the CLI exactly: low-cache-hit fires when `requests >= 1000 && hitRatePct < 5` (severity warning; `hitRatePct < 1` → critical); html-cache-trap ports `assessHTMLCacheTrap`'s verdict (HTML + public + s-maxage/max-age>=60 + cf-cache-status not HIT/STALE/REVALIDATED/UPDATING; warning; critical when cacheRuleCount === 0 and HTML); cache-rule-gap (info: HTML probe present, cache-control cacheable, ruleCount === 0 — dedupe: if html-cache-trap already fired, skip); alert-firing (one finding per alert, warning, evidence with value/threshold/unit); quota-pressure (usedPct >= 80 warning, >= 95 critical). Empty findings → brief with empty arrays (still renders a clean "no concerns" agent-brief? NO — the endpoint returns `brief: null` when zero findings; the modal is only reachable from a concern anyway).
- Findings order: critical → warning → info, then by kind alphabetical. Actions/verify: v1 actions come from the template registry (B2) — compile accepts an optional registry param defaulting to the imported registry; keep B2's import cycle safe by having compile.ts take `templates: TemplateMap` as an argument the endpoint passes (registry import lives in the endpoint, not compile).

**TDD:** `TestCompileBriefLowCacheHit` (thresholds incl. critical tier + sub-1000-request suppression), `TestCompileBriefHTMLTrap` (trap/clean/edge classes from caching.go's test matrix), `TestCompileBriefDedupe` (rule-gap suppressed when trap fired), `TestCompileBriefAlertAndQuota`, `TestCompileBriefOrdering` (identical facts → identical ordering), `TestCollectZoneFacts` (mocked fetch + cached() short-circuit; probe error → probe undefined, no throw). Red → implement → green.

**Commit:** `feat(ops): briefs core — zone fact collectors + pure advisory compiler (P-15, FEAT-pC1EV32)`

## Task B2: templates + agent-brief renderer + goldens (P-16) — after B1

**Model:** sonnet · **Files:** create `apps/ops/src/briefs/templates.ts`, `briefs/render.ts`, `briefs/render.test.ts`, `briefs/templates.test.ts`

- `templates.ts`: `TemplateMap = Record<BriefKind, (f: ZoneFacts) => { findings: BriefFinding[]; actions: BriefAction[]; verify: string[] }>` — the registry (one entry per v1 kind). Action directives derive from the FEAT-067/068/070 rulebook (the CLI's Fix texts are the phrasing source — quote their directives, adapted to imperative agent direction). Verify commands use real CLI invocations (`cosmoflare efficiency --json | jq '.data.zone_coverage[] | select(.zone=="<zone>")'`, `cosmoflare zone cache-rules list <zone>` — check the actual command names in `cmd/` before writing them into templates; never invent flags).
- `render.ts`: `renderAgentBrief(b: Brief): string` — the exact format from the brainstorm §3: title line, Problem (one bullet per finding, evidence inline), Actions (numbered), Verify (bulleted), Context footer. Pure string building — no Date.now inside (generatedAt lives in the API envelope, not the rendered text — that is what makes byte-identical goldens possible).
- **Golden tests:** one per kind + one mixed-zone brief: assert the FULL exact output string (not snapshots of structure — literal expected strings in the test file). Also `TestRenderDeterministic`: render twice from identical Brief → `expect(a).toBe(b)` and `expect(a).toEqual(expectedGolden)`.

**Commit:** `feat(ops): agent-brief templates + renderer with byte-identical goldens (P-16, FEAT-pC1EV32)`

## Task B3: `/api/advisory/:zone` endpoint (P-17) — after B1+B2

**Model:** glm-turbo · **Files:** modify `apps/ops/src/index.ts` (route registration only), create `apps/ops/src/advisory.test.ts`

- Route: `GET /api/advisory/:zone` behind the existing Access gate; resolve zone name→id via the existing zone list (cached, the `domains.ts`/`summary.ts` pattern — reuse, don't re-fetch); `collectZoneFacts` → `compileBrief(facts, templates)` → `renderAgentBrief`. Response: `{ zone, zoneId, generatedAt: <ISO now>, facts, brief, rendered: { agentBrief } }`, `Cache-Control: private, no-store`. Zero findings → `{ ..., brief: null, rendered: { agentBrief: "" } }`. Unknown zone → 404 `{ error }`. Collector hard-failure → 502 `{ error, partial: true }` (probe/best-effort failures already degrade inside facts).
- Tests follow the existing route-test pattern in `advisory.test.ts` (mock env/fetch, assert envelope + auth-refusal-when-unconfigured).

**Commit:** `feat(ops): GET /api/advisory/:zone — deterministic zone advisory endpoint (P-17, FEAT-pC1EV32)`

## Task B4: the modal + alert-card entry (P-18) — after B3 (parallel-safe with nothing; last code task)

**Model:** sonnet · **Files:** create/modify under `pager/` following its component layout (read the tree first): one `AgentCommandsModal` component + wiring on the Alerts page firing-card component.

- Button label: "Commands to AI Agent" (secondary style on alert cards).
- Modal: fetch `/api/advisory/<zone>` on open; shimmer while loading; Retry on error; mono brief block (selectable); header `Commands to AI Agent · <zone>` + subtitle `measured <window> · CosmoLabs Ops`; primary Copy → `navigator.clipboard.writeText(rendered.agentBrief)`, label→checkmark with 150ms ease-out opacity fade (NO bounce/elastic), auto-revert 1.5s; Esc/overlay-click closes; focus trap basic.
- Taste gate: no tile outlines, no left borders, reuse pager's existing color/spacing vocabulary. `npx impeccable detect` on changed/new files = 0 findings — paste the output.
- Tests: stubbed-fetch modal flow (open → loading → text → copy called with exact string → checkmark state) in the pager's test setup.

**Commit:** `feat(ops): Commands-to-AI-Agent modal — copy-paste advisory dispatch (P-18, FEAT-pC1EV32)`

## Task B5: live pass + docs (P-19) — MAIN SESSION executes (live-data rule)

- `bun run build` + dev URL (portless `make url`), Playwright: open Alerts (or force an advisory for a known-concern zone), click the button, modal renders the brief for a REAL zone, copy lands on the clipboard, text starts with the expected title and ends with the Context footer. Screenshot for the record.
- README: add `GET /api/advisory/:zone` to the API list.
- Full gates: `go test ./pkg/cosmoflare/ ./cmd/` still green (CLI untouched — regression guard), ops test suite green, `go vet ./...`.
- Record P-15..P-19 coverage via `ccs prompts add-commit`.

**Commit:** `docs(ops): advisory API + live verification record (P-19, FEAT-pC1EV32)`

## Sequencing

1. B1 → B2 (registry/templates consume types).
2. B3 after B1+B2 (endpoint composes them).
3. B4 after B3 (modal calls the endpoint; may stub in tests).
4. B5 last, main session.

## Acceptance

- `curl` the endpoint for a real zone with a known concern → deterministic agentBrief, byte-identical across two calls with unchanged facts.
- Modal on a firing alert card copies the brief in one click.
- Goldens green; impeccable 0; no existing route broken; README updated.
