---
brainstorm_ref: docs/brainstorming/2026-09-09-domain-center-residuals.md
branch: master
covers_brainstorm_deliverables:
    - BR-01
    - BR-02
    - BR-03
    - BR-04
    - BR-05
    - BR-06
    - BR-07
covers_plan_deliverables:
    - P-01
    - P-02
    - P-03
    - P-04
    - P-05
    - P-06
    - P-07
    - P-08
created: "2026-09-09T02:14:59+04:00"
id: P-2026-09-09-domain-center-residuals
plan_ref: docs/planning-mode/2026-09-09-domain-center-residuals.md
priority: medium
requires_reading:
    - docs/brainstorming/2026-09-09-domain-center-residuals.md
    - docs/planning-mode/2026-09-09-domain-center-residuals.md
schema_version: 1
status: PENDING
title: 'Cosmoflare — domain center residuals: pagerules merge + redirect probes'
---
# Cosmoflare — domain center residuals: pagerules merge + redirect probes

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/brainstorming/2026-09-09-domain-center-residuals.md`** — design rationale and decision history.
2. **`docs/planning-mode/2026-09-09-domain-center-residuals.md`** — implementation plan with deliverables and file scope.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## Context

FEAT-010 / ROAD-087 residual scope: complete the two Domain Center requirements that never shipped — the legacy Page-Rule `forwarding_url` merge into redirect visibility (library GetDetail via `WithPageRules`), and the opt-in redirect-target attention check (`RedirectProber` → `DomainStatus.RedirectIssue` → stats/doctor/TUI). The plan contains complete TDD code for every task — implement it spec-exact. Design: `docs/brainstorming/2026-09-09-domain-center-residuals.md`; plan: `docs/planning-mode/2026-09-09-domain-center-residuals.md`.

Both documents passed fresh-context independent review on 2026-09-09 (3 blockers + 8 Tier-1 fixes applied; accepted limitations recorded in the plan). Where the plan and this prompt disagree, the plan wins.

## Goals

### [ ] G-01 RedirectRule.Source + legacyForwardingRules pure helper with table tests
Covers P-01. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/redirect.go`, `pkg/cosmoflare/redirect_test.go` (plan Task 1).

### [ ] G-02 PageRuleLister + DomainService.WithPageRules + GetDetail merge with tests
Covers P-02. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/domains.go`, `pkg/cosmoflare/domains_test.go` (plan Task 2). Depends on G-01; same wave as G-04 is FORBIDDEN (same file) — sequential.

### [ ] G-03 RedirectProber — bounded probes, loop detection, httptest coverage
Covers P-03. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/redirectprobe.go`, `pkg/cosmoflare/redirectprobe_test.go` (plan Task 3, new files). Independent — parallel-safe.

### [ ] G-04 DomainStatus.RedirectIssue + classifyRedirectIssue + attention criterion
Covers P-04. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/domains.go`, `pkg/cosmoflare/domains_test.go` (plan Task 4). Independent — parallel-safe.

### [ ] G-05 domains stats --check-redirects + factory wiring + tests
Covers P-05. **Model:** glm-turbo. **Files:** `cmd/domains.go`, `cmd/domains_stats.go`, `cmd/domains_stats_test.go` (plan Task 5). CRITICAL: the factory call flips to `newDomainService(domainsStatsCheckRedirects)` — without it the flag is a silent no-op. Requires G-01+G-02+G-04.

### [ ] G-06 DiagnosticReport.RedirectTargets + cmd/doctor.go section + tests
Covers P-06. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/doctor.go`, `cmd/doctor.go`, `cmd/doctor_test.go` (plan Task 6). Severity string is `"warning"` (codebase convention), score staleness accepted (plan's Accepted limitations). Requires G-03+G-04.

### [ ] G-07 TUI detail-pane redirect-issue badge + view test
Covers P-07. **Model:** glm-turbo. **Files:** `internal/tui/domain.go`, `internal/tui/domain_test.go` (plan Task 7). Badge reads from `sel` (the list DomainStatus), NOT the `m.detail` block — plan encodes the trap. Independent after G-04 compiles.

### [ ] G-08 docs/USAGE.md — merge completeness + --check-redirects + doctor section
Covers P-08. **Model:** glm-turbo. **Files:** `docs/USAGE.md` (plan Task 8). Sections live under `## Domains Command` → `### Domain subcommands` — not standalone. Last.

## Execution Strategy

GLM dispatch from `docs/prompts/2026-09-09-domain-center-residuals-glm-tasks.yaml`. Four waves:

1. Wave 1 (parallel): library core — legacy helper (Task 1, `redirect.go`) + prober (Task 3, new file) + RedirectIssue (Task 4, `domains.go`)
2. Wave 2 (solo): WithPageRules merge (Task 2 — `domains.go` again, must not run concurrent with Task 4)
3. Wave 3 (parallel): stats flag (Task 5) + doctor section (Task 6) + TUI badge (Task 7) — disjoint files
4. Wave 4 (solo): docs (Task 8)

Every merge passes the quality gate: Opus re-reads the diff, re-runs the full affected package in the worktree, `ccs verify-worktree --approve` + `ccs merge` with ancestry check. Agents run `gofmt -w` on touched files before committing.

## File Scope

```yaml
files_modified:
  - pkg/cosmoflare/redirect.go          # Source field + legacyForwardingRules
  - pkg/cosmoflare/redirect_test.go
  - pkg/cosmoflare/domains.go           # WithPageRules, merge, RedirectIssue, classify
  - pkg/cosmoflare/domains_test.go
  - pkg/cosmoflare/redirectprobe.go     # new — RedirectProber
  - pkg/cosmoflare/redirectprobe_test.go
  - pkg/cosmoflare/doctor.go            # DiagnosticReport.RedirectTargets field
  - cmd/domains.go                      # factory enrich wiring
  - cmd/domains_stats.go                # --check-redirects
  - cmd/domains_stats_test.go
  - cmd/doctor.go                       # redirect-target section
  - cmd/doctor_test.go
  - internal/tui/domain.go              # issue badge
  - internal/tui/domain_test.go
  - docs/USAGE.md
```

## Related

- Brainstorm: `docs/brainstorming/2026-09-09-domain-center-residuals.md`
- Plan: `docs/planning-mode/2026-09-09-domain-center-residuals.md`
- GLM tasks: `docs/prompts/2026-09-09-domain-center-residuals-glm-tasks.yaml`
- Issue: FEAT-010 · Roadmap: ROAD-087
