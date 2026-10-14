---
branch: master
covers_plan_deliverables:
    - P-01
    - P-02
    - P-03
    - P-04
    - P-05
created: "2026-10-14T20:40:42+04:00"
date: "2026-10-14T20:40:42+04:00"
id: P-2026-10-14-www-redirect-automation
plan_ref: docs/planning-mode/2026-10-14-www-redirect-automation.md
priority: medium
requires_reading:
    - docs/planning-mode/2026-10-14-www-redirect-automation.md
schema_version: 1
status: PENDING
title: www-redirect automation — detect, alert, fix
type: prompt
---
# www-redirect automation — detect, alert, fix

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/planning-mode/2026-10-14-www-redirect-automation.md`** — implementation plan with deliverables and file scope.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## Context

The www→non-www fleet standard is verified manually (44/45 zones, 2026-10-14)
but nothing enforces it — new zones and deletions drift silently, and
Cloudflare offers no account-level mechanism (live-verified: the API rejects
account-scope redirect phases). This prompt makes the standard self-enforcing
in three layers plus setup integration. All design decisions, verified facts,
in-tree recon pointers, and the sequencing live in the plan — read it first;
it is the spec.

House rules bind: real API numbers only, `--json` everywhere, TDD for new
cmd code, conventional commits, agent worktrees through the S334 gate.

## Goals

### [ ] G-01 Library: WWWRedirectService — per-zone audit and idempotent fix
**Model:** sonnet (bounded-with-discovery: mirrors existing services) | **Files:** create `pkg/cosmoflare/wwwredirect.go`, `pkg/cosmoflare/wwwredirect_test.go`
Covers P-01. Status/Assess/Fix shapes are specified in the plan; REST-mock tests per the kv_test.go pattern; variants: compliant / rule-only / record-only / neither / fix-idempotence.

### [ ] G-02 Detect: www-redirect-missing finding in efficiency
**Model:** sonnet (same agent as G-01) | **Files:** modify `pkg/cosmoflare/efficiency.go`, `pkg/cosmoflare/efficiency_test.go`, `cmd/efficiency.go`, `cmd/efficiency_test.go`
Covers P-02. Wire via the WithCaching-style optional-field pattern; runs in the zone loop; 2 REST calls per zone.

### [ ] G-03 Fix: `cosmoflare domains www-redirect [--all] [--fix] [--json]`
**Model:** glm-turbo (exact code derivable from P-01) | **Files:** modify `cmd/domains.go` (or create `cmd/domains_wwwredirect.go`), its test, `docs/USAGE.md`
Covers P-03. Seam pattern per usage.go/efficiency.go; USAGE.md section beside the efficiency one.

### [ ] G-04 Alert: www-redirect-missing ops condition, zero extra fetches
**Model:** sonnet | **Files:** modify `apps/ops/src/rules_quota.ts`, `rules_quota.test.ts`, `scheduled.ts` (+ its test)
Covers P-04. Derive wwwRedirect-present per zone inside collectPageRules (already extracts targets + forwarding actions each rotation); evaluate from the summary store; registry follows zone-scoped precedents. Live-verify: delete one test rule → alert fires → --fix restores.

### [ ] G-05 init/apply integration
**Model:** glm-turbo | **Files:** the apply/init code paths (locate via the plan's recon table)
Covers P-05. `www_redirect: true` config default; rides P-01's Fix. Run LAST (needs P-01 stable).

## Related

- Plan: `docs/planning-mode/2026-10-14-www-redirect-automation.md`
