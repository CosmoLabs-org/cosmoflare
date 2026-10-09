---
branch: master
covers_plan_deliverables:
    - P-01
    - P-02
    - P-03
    - P-04
    - P-05
    - P-06
    - P-07
created: "2026-10-09T20:36:13+04:00"
date: "2026-10-09T20:36:13+04:00"
goals_completed: 5
goals_total: 10
id: P-2026-10-09-cf-cli-review
plan_ref: docs/planning-mode/2026-10-09-ops-billing-ui-caching.md
priority: medium
related_prompts: []
requires_reading:
    - docs/planning-mode/2026-10-09-ops-billing-ui-caching.md
schema_version: 1
status: PENDING
tags: []
title: CF agent CLI review + Ops push completion
type: prompt
---

# CF agent CLI review + Ops push completion

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/planning-mode/2026-10-09-ops-billing-ui-caching.md`** — implementation plan with deliverables and file scope.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## File Scope
```yaml
files_modified:
  - apps/ops/src/index.ts
  - apps/ops/src/scheduled.ts
  - apps/ops/README.md
  - docs/USAGE.md
  - pager/src/main.ts
  - pager/src/routes.ts
files_created:
  - docs/research/2026-10-10-cf-agent-cli-gap-analysis.md
  - pager/src/rules.ts
  - apps/ops/src/rules-api.ts
```

## Context

Cosmoflare Ops is live and private: https://ops.cosmolabs.org (custom domain only; workers.dev and preview URLs off; the Access app "[cosmoflare-ops] Owner only" covers ops.cosmolabs.org only since 2026-10-09 — the stale workers.dev destination was removed; one policy, operator email). Deployed version 9f012605 (2026-10-09): billing-period cost view with per-product overage projection and per-project attribution, a responsive redesign (hamburger drawer < 768px, sidebar ≥ 1024px, Overview/Billing/Workers/D1/Zones/Alerts/Pairing), a KV-backed upstream cache (OPS_KV = cosmoflare-ops-kv; the Cache API is unavailable behind Access per CF docs), a new icon set, and a 5-minute cron that evaluates the FEAT-049 zone/D1 rules and sends Web Push. Push cannot deliver yet: the VAPID_PRIVATE_KEY and VAPID_SUBJECT Worker secrets are not set (the permission classifier blocks Claude from secret writes — the operator runs them), and no phone is paired.

Live billing findings (2026-10-09, day 9 of 31, calendar fallback): total projected overage ≈ $1.56 — mycarguide KV storage 3.9 GB vs 1 GB included (≈ $1.44) and D1 rows read projected ≈ 24.99B vs 25B (mycarguide 97%). Billing uses 5 upstream calls per refresh (aliased GraphQL).

Next session's main job (operator 2026-10-09): review Cloudflare's newest agent-friendly CLI end to end and find what cosmoflare is missing — features and, above all, better strategies to pull the information we need.

## GLM Dispatch Rules

1. Use `ccs glm-agent exec` for every subagent; the Agent tool refuses sonnet/haiku (hook).
2. `--model sonnet` resolves to glm-5.3-flash via config/glm-models.json (observed in agent logs 2026-10-09). Use `--model claude-sonnet-4-6` only if the operator asks for real Sonnet.
3. Flash agents hit turn limits on large tasks (2 of 5 died at 120-150 turns): split big work into bounded briefs; `--skip-validate` for design work; `--worktree` into a finished agent's worktree fails ("dispatch lost") — use `--base-branch <agent-branch>` instead.
4. Opus reviews every diff and re-runs tests in the worktree (S334). Agents stated false API facts this session ("cpuTimeUs / payloadSize / byteCount not exposed"): re-check every "not available" claim by live introspection.
5. Deploys and secret writes need an explicit operator request in-session (classifier).

## What Got Done (2026-10-09)

- Plan with shared JSON contracts: docs/planning-mode/2026-10-09-ops-billing-ui-caching.md (ff53903).
- Cache + summary v2 (a43bb75, 7bd6492), icons (9e5280e), cron engine + fixes (d3a9ab9, f2fc377), billing collector + live-schema fixes + KV ID normalisation (34b98dd, 825f083, 7479c84), UI + polish + legend (9004e0a, c5eba99 and the C/C2 commits), integration routes + cron wiring (c7b8590), OPS_KV binding (445514f). Tests: apps/ops 101, pager 54, all green; deployed.
- Roadmap: Ops tier ROAD-pJ25XG5 with phases ROAD-pMA5QCQ/pG66HHM/pR73WHE/pHVTJDK (done), ROAD-pMBBH62 (cron push, in progress), ROAD-p0WZ2JP (billing period), ROAD-pKHNBPX (rules editor); CLI review ROAD-pCC8KKR under ROAD-102. Changelog FEAT-052 entry updated. Nothing pushed (pre-push guard; local master ahead of origin).

## Goals

### [x] G-01 Upstream caching layer for the Ops Worker (per-dataset TTL, stale-while-revalidate, single-flight)
Covers P-01. Done: a43bb75, 7bd6492.

### [x] G-02 Summary v2 — per-Worker 24h rows, zone status breakdown, cache metadata
Covers P-02. Done: a43bb75.

### [x] G-03 /api/billing collector — billing period, per-product allowance/usage/projection/overage, top consumers, project attribution, verified prices
Covers P-03. Done: 34b98dd, 825f083, 7479c84.

### [x] G-04 Pager UI — responsive layout, hamburger drawer, overview/billing/workers/D1/zones views
Covers P-04. Done: 9004e0a, c5eba99; deployed 9f012605.

### [x] G-05 App icon set (home-screen, maskable, favicon)
Covers P-05. Done: 9e5280e.

### [ ] G-06 Cron push from the Worker — finish delivery to the iPhone
Covers P-06. **Model:** operator + opus. Code is live (d3a9ab9, f2fc377, c7b8590). Remaining, in order: (1) operator runs, in apps/ops: `python3 -c "import json,os;print(json.load(open(os.path.expanduser('~/.cosmoflare/push.json')))['vapid_private_key'],end='')" | npx wrangler secret put VAPID_PRIVATE_KEY` and `printf 'mailto:alerts@cosmolabs.org' | npx wrangler secret put VAPID_SUBJECT`; (2) operator opens https://ops.cosmolabs.org on the iPhone, Add to Home Screen, opens it from the Home Screen, Pairing → Enable notifications (POSTs to /api/subscribe), then "Send test alert"; (3) acceptance: the test push lands; `wrangler tail` (or Workers observability) shows a `{"cron":"alerts",...}` line with sent ≥ 1 on the next breach. If Apple rejects, read the issue text in the /api/test-fire response (BadJwtToken / BadVapidPublicKey / VapidPkHashMismatch).

### [ ] G-07 Integration docs — Ops section in docs/USAGE.md
Covers P-07. **Model:** `glm-turbo` | **Files:** `docs/USAGE.md`. Integration and deploy are done (c7b8590, 445514f, 9f012605); apps/ops/README.md is current. Add a "Cosmoflare Ops (web/phone)" section to docs/USAGE.md that summarises apps/ops/README.md: URL, Access lockdown, endpoints (/api/summary, /api/billing, /api/subscribe, /api/test-fire), cron cadence, cache TTL table, secrets list, deploy command. Acceptance: `grep -c "/api/billing" docs/USAGE.md` ≥ 1.

### [ ] G-08 Full-repo review of Cloudflare's newest agent-friendly CLI vs cosmoflare (ROAD-pCC8KKR)
**Model:** `opus` orchestrates; parallel read-only flash scouts per area. **Files:** `docs/research/2026-10-10-cf-agent-cli-gap-analysis.md` (new).
Steps: (1) identify the tool and its repo URL live (Cloudflare blog/changelog/GitHub via web subagents — do not guess the name); (2) `ccs analyze-clone <github-url>` (out-of-tree clone); (3) map its command surface, output formats (JSON/agent modes), auth model, MCP/agent integration, config, error/exit-code conventions; (4) map HOW it pulls account data — which REST/GraphQL endpoints and datasets, batching, pagination, caching, rate-limit handling — and compare against pkg/cosmoflare/ and apps/ops (summary.ts, billing.ts, cache.ts); (5) write a gap matrix: feature | them | cosmoflare CLI | Ops | adopt? | effort, plus a "data-pull strategies to adopt" list with expected upstream-call savings; (6) file roadmap/issue items for the top gaps (`/feature`, `ccs roadmap add --parent ROAD-102`). Acceptance: the research doc exists with the matrix and ≥ 5 concrete adopt/skip decisions, each citing a file in their repo.

### [ ] G-09 Ops rules editor (ROAD-pKHNBPX)
**Model:** `sonnet` (split into two bounded briefs) | **Files:** `apps/ops/src/rules-api.ts` (new: GET/PUT /api/rules over KV key "rules", validated against rules.ts CONDITIONS), `apps/ops/src/index.ts` (route, behind the Access check), `pager/src/rules.ts` (new Rules view: list, enable toggle, threshold, exclude list), `pager/src/routes.ts`/`main.ts` (#/rules). Also: `runScheduled` returns early with zero upstream calls when KV "subs" is empty. Acceptance: tsc + tests green in both packages; PUT with an unknown condition → 400.

### [ ] G-10 Billing period from the real subscription (ROAD-p0WZ2JP)
**Model:** operator decision. GET /accounts/{id}/subscriptions returns 403 for the ops token (billing falls back to the UTC calendar month). Either the operator adds Billing Read to the token (then `wrangler secret put CF_API_TOKEN`), or sets a `BILLING_ANCHOR_DAY` var. Acceptance: /api/billing `period.source` is "subscription" or "anchor".

## Carry-Overs

1. **Session 2032 prompt** (2/8) → `docs/prompts/2026-10-09-session-2032-continuation.md`: goal 2 v0.33.0 release (operator go only), goal 3 FEAT-050 design (non-colliding command name; ROAD-102 rule catalog), goals 4-6 operator-driven (cosmolabs.org TASK-010 deploy, launch posts, research pastes). Goal 1 = G-06 here.
2. **CF limits corpus research ingestion** (5/6) → `docs/prompts/2026-09-10-cf-limits-corpus-research-ingestion.md`.

## Where We're Headed

Cosmoflare's governance pillar (ROAD-102) now has a working phone tier. The CLI review decides the next big moves: which agent-UX conventions to adopt, and how to pull richer data with fewer calls — useful for both the CLI and Ops. After that: FEAT-050 (pre-launch verdict command), the rules editor, and the v0.33.0 release once the operator gives the go.

## Priority Order
1. G-06 (operator: two secret commands + phone pairing) — unblocks paging, minutes of work.
2. G-08 CLI review — the session's main goal.
3. G-07 docs, G-09 rules editor (parallel flash agents while G-08 runs).
4. G-10 billing period (operator decision), carry-overs.

## Related

- Plan: `docs/planning-mode/2026-10-09-ops-billing-ui-caching.md`
