---
brainstorm_ref: docs/brainstorming/2026-09-10-ratelimit-traffic-classes.md
branch: master
covers_brainstorm_deliverables:
    - BR-01
    - BR-02
    - BR-03
    - BR-04
covers_plan_deliverables:
    - P-01
    - P-02
    - P-03
    - P-04
created: "2026-09-10T16:31:25+04:00"
id: P-2026-09-10-ratelimit-traffic-classes
plan_ref: docs/planning-mode/2026-09-10-ratelimit-traffic-classes.md
priority: medium
requires_reading:
    - docs/brainstorming/2026-09-10-ratelimit-traffic-classes.md
    - docs/planning-mode/2026-09-10-ratelimit-traffic-classes.md
schema_version: 1
status: PENDING
title: 'Cosmoflare — FEAT-013: rate-limit traffic classes + trip probe'
---
# Cosmoflare — FEAT-013: rate-limit traffic classes + trip probe

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/brainstorming/2026-09-10-ratelimit-traffic-classes.md`** — design rationale and decision history.
2. **`docs/planning-mode/2026-09-10-ratelimit-traffic-classes.md`** — implementation plan with deliverables and file scope.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## Context

FEAT-013 (origin: MyCarGuide feedback FB-7): a rate-limit rule verified `enabled=true` produced zero 429s across 45+-request bursts against a Pages static asset on a custom domain — WAF rate limiting silently skips certain traffic classes. This session extends the FEAT-012 knowledge layer: a docs-derived, evidence-linked traffic-class matrix in the pack, plus an opt-in, capped (60), half-cache-busted live trip probe reporting `tripped | not-counted | inconclusive` instead of trusting `enabled=true`. The plan contains complete TDD code for every task — implement it spec-exact. Where the plan and this prompt disagree, the plan wins.

## Goals

### [ ] G-01 TrafficClass schema + SkippedTrafficClasses + evidence-linked seed matrix
Covers P-01. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/knowledge/knowledge.go`, `knowledge_test.go`, `packs/ratelimit.json` (plan Task 1).

### [ ] G-02 RateLimitProber — ExpressionPath, classifyProbe, bounded burst
Covers P-02. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/ratelimitprobe.go`, `ratelimitprobe_test.go` (plan Task 2). After G-01 (classifyProbe cites the matrix).

### [ ] G-03 CLI — ratelimit probe command, --probe on create, pack-driven advisory
Covers P-03. **Model:** glm-turbo. **Files:** `cmd/ratelimit.go`, `cmd/ratelimit_probe_test.go` (plan Task 3). After G-02. REUSES `ratelimitServiceAndZone` in `cmd/ratelimit.go` — do not add a resolver.

### [ ] G-04 docs/USAGE.md — traffic-class matrix and probe documentation
Covers P-04. **Model:** glm-turbo. **Files:** `docs/USAGE.md` (plan Task 4). Last.

## Execution Strategy

GLM dispatch from `docs/prompts/2026-09-10-ratelimit-traffic-classes-glm-tasks.yaml`. All four tasks share the dependency chain 1→2→3→4 (same packages, later tasks consume earlier symbols) — dispatch **sequentially**, one agent at a time. Every merge passes the quality gate: Opus re-reads the diff, re-runs the full affected package in the worktree, `ccs verify-worktree --approve` + `ccs merge` with ancestry check. Agents run `gofmt -w` on touched files before committing.

## File Scope

```yaml
files_modified:
  - pkg/cosmoflare/knowledge/knowledge.go      # TrafficClass type + SkippedTrafficClasses
  - pkg/cosmoflare/knowledge/knowledge_test.go
  - pkg/cosmoflare/knowledge/packs/ratelimit.json  # traffic_classes matrix
  - pkg/cosmoflare/ratelimitprobe.go           # ExpressionPath, classifyProbe, RateLimitProber
  - pkg/cosmoflare/ratelimitprobe_test.go
  - cmd/ratelimit.go                           # probe command + --probe + advisory
  - cmd/ratelimit_probe_test.go
  - docs/USAGE.md
```

## Related

- Brainstorm: `docs/brainstorming/2026-09-10-ratelimit-traffic-classes.md`
- Plan: `docs/planning-mode/2026-09-10-ratelimit-traffic-classes.md`
- GLM tasks: `docs/prompts/2026-09-10-ratelimit-traffic-classes-glm-tasks.yaml`
- Issue: FEAT-013 · Predecessor: FEAT-012 (knowledge layer, shipped 2026-09-10)
