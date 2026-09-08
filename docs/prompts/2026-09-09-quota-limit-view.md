---
brainstorm_ref: docs/brainstorming/2026-09-09-quota-limit-view.md
branch: master
covers_brainstorm_deliverables:
    - BR-01
    - BR-02
    - BR-03
    - BR-04
    - BR-05
covers_plan_deliverables:
    - P-01
    - P-02
    - P-03
    - P-04
    - P-05
    - P-06
    - P-07
    - P-08
    - P-09
created: "2026-09-09T00:56:34+04:00"
id: P-2026-09-09-quota-limit-view
plan_ref: docs/planning-mode/2026-09-09-quota-limit-view.md
priority: medium
requires_reading:
    - docs/brainstorming/2026-09-09-quota-limit-view.md
    - docs/planning-mode/2026-09-09-quota-limit-view.md
schema_version: 1
status: PENDING
title: 'Cosmoflare — limits: quota proximity view + alert feed'
implemented_commits:
    - {sha: '773de39d45cd', covers: [P-01]}
    - {sha: '7159245a1d28', covers: [P-03]}
    - {sha: '1bf625352bb7', covers: [P-06]}
    - {sha: '5e967efb6026', covers: [P-02]}
---
# Cosmoflare — limits: quota proximity view + alert feed

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/brainstorming/2026-09-09-quota-limit-view.md`** — design rationale and decision history.
2. **`docs/planning-mode/2026-09-09-quota-limit-view.md`** — implementation plan with deliverables and file scope.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## Context

FEAT-009 / ROAD-090: `cosmoflare limits` shows how close the account is to its Cloudflare plan limits. A new `LimitsService` joins live usage counts (workers scripts, R2 buckets, zones, per-zone DNS record quotas) against documented plan tables, with partial-failure snapshot semantics and an alert-evaluator feed. The plan contains complete TDD code for every task — implement it spec-exact. Design: `docs/brainstorming/2026-09-09-quota-limit-view.md`; plan: `docs/planning-mode/2026-09-09-quota-limit-view.md`.

Both documents passed fresh-context independent review on 2026-09-09. The plan's Go code is signature-verified against the codebase. Where the plan and this prompt disagree, the plan wins.

## Goals

### [ ] G-01 Static limit tables + limitFor pure function with table tests
Covers P-01. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/limits.go`, `pkg/cosmoflare/limits_test.go` (plan Task 1).

### [ ] G-02 LimitsService types, constructor, consumer interfaces
Covers P-02. **Model:** glm-turbo. **Files:** same two files (plan Task 2). Depends on G-01 — same file, sequential.

### [ ] G-03 Workers plan resolution — subscriptions API + config/flag fallback
Covers P-03. **Model:** glm-turbo. **Files:** same two files (plan Task 3). Resolution order: subscriptions → flag → config → unknown.

### [ ] G-04 DNS usage endpoint + ZonePlan.LegacyID extension
Covers P-04. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/limits.go`, `pkg/cosmoflare/limits_test.go`, `pkg/cosmoflare/zone.go` (plan Task 4).

### [ ] G-05 Snapshot assembly with partial-failure semantics
Covers P-05. **Model:** glm-turbo. **Files:** same library files (plan Task 5). Zero rows + all sources errored = hard failure; anything less = partial snapshot.

### [ ] G-06 ProjectConfig workers_plan field
Covers P-06. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/config.go`, `pkg/cosmoflare/config_test.go` (plan Task 6). Independent — can run parallel to G-01..G-05.

### [ ] G-07 cmd/limits.go CLI — table, JSON, flags, exit codes
Covers P-07. **Model:** glm-turbo. **Files:** `cmd/limits.go`, `cmd/limits_test.go` (plan Task 7). Requires G-01..G-06 landed.

### [ ] G-08 Alert-evaluator feed — EvalMetrics fields, conditions, serve wiring
Covers P-08. **Model:** glm-turbo. **Files:** `internal/webhook/evaluator.go`, `internal/webhook/evaluator_test.go`, `cmd/serve.go` (plan Task 8). Requires G-05. Limits failure logs and continues — never skips analytics rules.

### [ ] G-09 Documentation — docs/USAGE.md limits section
Covers P-09. **Model:** glm-turbo. **Files:** `docs/USAGE.md` (plan Task 9). Last — documents final flags and conditions.

## Execution Strategy

GLM batch dispatch from `docs/prompts/2026-09-09-quota-limit-view-glm-tasks.yaml`. Four waves:

1. Wave 1 (parallel): limits library part 1 (Tasks 1-3, `limits.go`) + config field (Task 6, `config.go`)
2. Wave 2 (solo): limits library part 2 (Tasks 4-5 — same file as part 1, must not run concurrent)
3. Wave 3 (parallel): CLI (Task 7, `cmd/`) + alert feed (Task 8, `internal/webhook/` + `cmd/serve.go`)
4. Wave 4 (solo): docs (Task 9)

Every merge goes through the quality gate: Opus re-reads the diff, re-runs the full affected package in the worktree, then `ccs verify-worktree --approve` + `ccs merge`. The quality gate also runs the Open Risk pins: one live curl of the DNS usage endpoint to confirm response field names, and a spot-check of the four static joined values against the live docs.

## File Scope

```yaml
files_modified:
  - pkg/cosmoflare/limits.go        # new — LimitsService
  - pkg/cosmoflare/limits_test.go   # new
  - pkg/cosmoflare/zone.go          # ZonePlan.LegacyID
  - pkg/cosmoflare/config.go        # workers_plan field
  - pkg/cosmoflare/config_test.go
  - cmd/limits.go                   # new — CLI
  - cmd/limits_test.go              # new
  - cmd/serve.go                    # alert-cycle limits feed
  - internal/webhook/evaluator.go   # EvalMetrics + conditions + collector
  - internal/webhook/evaluator_test.go
  - docs/USAGE.md                   # limits section
```

## Related

- Brainstorm: `docs/brainstorming/2026-09-09-quota-limit-view.md`
- Plan: `docs/planning-mode/2026-09-09-quota-limit-view.md`
- GLM tasks: `docs/prompts/2026-09-09-quota-limit-view-glm-tasks.yaml`
- Issue: FEAT-009 · Roadmap: ROAD-090
