---
branch: master
created: "2026-10-14T13:04:53+04:00"
date: "2026-10-14T13:04:53+04:00"
goals_completed: 2
goals_total: 7
priority: medium
related_prompts: []
requires_reading:
    - cmd/doctor.go
    - cmd/cost.go
    - cmd/cost_test.go
    - docs/prompts/2026-10-13-session-2040-continuation.md
schema_version: 1
status: PENDING
tags: []
title: 'CLI pivot — indispensable Cloudflare integration: audit + efficiency program'
type: prompt
supersedes: "docs/prompts/2026-10-13-session-2040-continuation.md"
implemented_commits:
    - 'afff8bcda9b6'
    - {sha: 'a4459ea77cac', covers: [P-01, P-02, P-03, P-05]}
---

# CLI pivot — indispensable Cloudflare integration: audit + efficiency program

## Context

Operator pivot (2026-10-14, closing the Ops marathon): "shift gears from the Ops app to the infrastructure of cosmoflare — the CLI tools and workflows. Make cosmoflare an indispensable tool for each project to integrate with Cloudflare, audit its usage, and ensure we are being resource efficient."

The Ops pager reached its milestone and is DEPLOYED (build DQHsMwHV, version 7ce99d7d): liquid-glass pill nav through 8 on-device tuning rounds, entity profiles everywhere (worker/D1/DO/zone), Guides section, critique findings all fixed. The update-notification root cause (unbumped SW cache name) is fixed permanently with a build stamp. Ops work continues only on operator request — the default focus is now the CLI core (`cmd/` + `pkg/cosmoflare/`).

The existing backlog already matches the vision one-to-one: FEAT-047 (usage-waste alerting), FEAT-048 (usage pacing), FEAT-067 (patterns + auto-remediation), FEAT-068 (TTL strategy engine), FEAT-069 (usage review gaps), FEAT-070 (encode CF best practices). This program is about sequencing and shipping those as one coherent "audit + efficiency" story.

House rules that bind: real API numbers only (never modeled), `--json` on every command, agent-readable errors, TDD for new cmd code, conventional commits, no git push without explicit operator ask.

## GLM Dispatch Rules

1. **ALWAYS** `ccs glm-agent exec` for GLM agents (queue + 429 retry)
2. **NEVER** Agent tool with sonnet/haiku models for GLM work
3. Agent tool with `model:opus` is fine
4. Parallel: `/glm-sprint` or `ccs glm-agent exec-batch`
5. Stale-base discipline: agents dispatched before a landing wave go stale — port genuine files by `git checkout <branch> -- <paths>`, never merge (0434/0435/0437/0439/0440/0441 precedent)

## What Got Done (the arc this continues from)

- **Ops app milestone** (deployed): ~25 commits across 2026-10-13/14 — cost lockup redesign, glyph-only nav then off-ramp section hues, liquid glass pill (CSS + WebGL refraction, overscroll-safe refresh, build-stamped SW update chain), Rules editor overhaul (switches, data-face numerals, iconed actions), critique sweeps (all P0-P2 + minors), Guides section, profiles for D1/DO/Zone, drawer/update-modal focus traps.
- **Fleet discipline proven**: 6 stale-base agent salvages ported by hand; the morning 09:00 deploy routine ran on schedule.
- **Ledger**: session-2040 prompt closed 8/10 (G-07 on-device acceptance + G-10 cron push remain — operator's confirmations only).

## Goals

### [x] G-01 Sequence the audit + efficiency program (brainplan)
**Model:** main session | **Files:** `docs/brainstorming/2026-10-14-cli-audit-efficiency.md`, `docs/planning-mode/2026-10-14-cli-audit-efficiency.md`
Run `/brainplan` over FEAT-047, FEAT-048, FEAT-067, FEAT-068, FEAT-069, FEAT-070 with the operator's vision as the brief. Output: one program doc that sequences the six FEATs into waves, names the first shippable user-facing slice, and writes tight GLM-executable goals for wave 1. Acceptance: `ccs prompts validate` passes on the plan; every wave-1 goal names exact files + test cases.

### [x] G-02 `cosmoflare efficiency` — the audit command, first slice
**Model:** `sonnet` | **Files:** `cmd/efficiency.go`, `cmd/efficiency_test.go`, `pkg/cosmoflare/efficiency.go`
The CLI's flagship audit command (FEAT-047/069 first slice): `cosmoflare efficiency [--json]` reports resource waste from REAL API numbers only — idle workers (0 requests over 7d), KV namespaces with 0 reads/writes in 7d, R2 buckets with 0 objects. Follow `cmd/doctor.go`'s diagnostic-command pattern (cobra command, rich --help with examples, --json output, agent-readable findings "what/why/how to fix"). Every finding carries its evidence (counts, dates). Tests: `TestEfficiencyIdleWorkers`, `TestEfficiencyEmptyKV`, `TestEfficiencyJSONOutput`, `TestEfficiencyEmptyAccount` — happy, empty, error paths; `go test ./cmd/ -run TestEfficiency` green. Gate: `go vet ./...` clean.

### [ ] G-03 Wire efficiency findings into `cosmoflare cost`
**Model:** `glm-turbo` | **Files:** `cmd/cost.go`, `cmd/cost_test.go`
The existing `cosmoflare cost` estimation gains a waste line: projected monthly dollars attributed to idle resources found by G-02's collectors (reused from `pkg/cosmoflare/efficiency.go` — no second collector). Test: `TestCostIncludesIdleWaste`. Acceptance: `go test ./cmd/ -run TestCost` green.

### [ ] G-04 Worker-profile live tail (the G-06 stretch, unchanged)
**Model:** `sonnet` | **Files:** `apps/ops/src/index.ts`, `pager/src/app/views/WorkerProfileView.tsx`
Stretch carried from the profiles program: a "Tail" affordance on the worker profile streaming `wrangler tail`-class events through the ops worker. Verify the CF tail API's auth shape first (it needs its own scope); if the token can't support it, ship the profile's "Open in Cloudflare →" deep link to the live-tail view instead and note the gap in the issue.

### [ ] G-05 Operator decisions round
The non-acceptance closes, all operator-gated: kill decision on worktrees 0434/0435 (work already ported), v0.33.0 release (operator go only), FEAT-029 auth design decision (the no-keychain constraint). The on-device acceptance itself is G-06 below.

### [ ] G-06 On-device acceptance round (carried from docs/prompts/2026-10-13-session-2040-continuation.md G-07)
Guides in the drawer, D1/DO/zone profile drill-ins, glass pill on the phone — the operator confirms on device.

### [ ] G-07 Cron push from the Worker — finish delivery to the iPhone (carried from P-2026-10-09-cf-cli-review G-06)
## Carry-Over Tasks

- [ ] Operator confirmations: G-07 acceptance round + G-10 cron push (was: pending, session-2040)
- [ ] Worktree kills 0434/0435 awaiting explicit say-so (was: pending; work already ported)
- [ ] v0.33.0 release — operator go only (was: pending)
- [ ] FEAT-029 auth — blocked on the no-keychain design decision (was: pending)
- [ ] CosmoKit `glass-surface` extraction — after the pill survives a week on devices (was: queued)

## File Scope

```yaml
files_modified:
  - cmd/cost.go
  - cmd/cost_test.go
  - apps/ops/src/index.ts
  - pager/src/app/views/WorkerProfileView.tsx
files_created:
  - cmd/efficiency.go
  - cmd/efficiency_test.go
  - pkg/cosmoflare/efficiency.go
  - docs/brainstorming/2026-10-14-cli-audit-efficiency.md
  - docs/planning-mode/2026-10-14-cli-audit-efficiency.md
```

## Where We're Headed

The strategic frame: every CosmoLabs project touches Cloudflare; cosmoflare should be the tool each one installs first — integrate (init/templates/apply), then continuously audit (efficiency, waste alerts) and remediate (patterns/auto-remediation, TTL strategy). Wave 1 = the audit command + cost wiring (G-02/G-03) because "see the waste" unlocks everything else. FEAT-067/068 (remediation) follow once findings exist. The Ops app becomes the read-only window on the same collectors — no separate implementations per tier (the library-first principle).

## Priority Order

1. G-01 brainplan the program (everything sequences from it)
2. G-02 `cosmoflare efficiency` first slice (the flagship)
3. G-03 cost wiring (quick, rides G-02's collectors)
4. G-05 operator closes (any time)
5. G-04 live tail (stretch)
