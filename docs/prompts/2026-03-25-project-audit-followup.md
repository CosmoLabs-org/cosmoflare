---
title: "Post-Audit: Critical Fixes for R2Go2"
status: PENDING
created: 2026-03-25
type: continuation
source: project-audit
priority: high
---

# Post-Audit Follow-Up: R2Go2 Critical Fixes

## Context
Comprehensive project audit completed 2026-03-25. Overall score: 5.4/10 (C grade).
Full report: `docs/audit/2026-03-25-r2go2/README.md`

## Critical Bug IDs and Locations

1. **API stubs** — `internal/api/client.go:140-225` — All 9 methods are placeholders
2. **BUG-001** — `internal/api/enhanced_client.go:198,277` — `time.Since(time.Now())` speed calc
3. **BUG-002** — `cmd/root.go:63-89` — PersistentPreRun blocks non-API commands
4. **BUG-003** — `.github/workflows/test-suite.yml:9` — Go 1.26 reference (doesn't exist)
5. **No LICENSE** — Root directory missing LICENSE file

## Top 5 Quick Wins

1. **Add LICENSE file** (~2 min) — Copy MIT template, fill in CosmoLabs copyright
2. **Fix speed calculation** (~5 min) — Store startTime before upload, use in division
3. **Fix CI Go version** (~5 min) — Remove 1.26 from test-suite.yml matrix
4. **Remove tracked binaries** (~5 min) — `git rm simple-setup test-setup`
5. **Fix PersistentPreRun** (~15 min) — Skip validateEnvironment for non-API commands

## Regression Areas
None (baseline audit, no previous data to compare).

## Recommended Session Focus
Start with Phase 0 of the upgrade plan (4 items). These are prerequisites for any real usage.
After Phase 0, tackle Phase 1 cleanup (items 5-9) to stabilize the build.
