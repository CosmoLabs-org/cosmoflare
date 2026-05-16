---
branch: master
created: "2026-03-07"
goals_completed: 0
goals_total: 5
origin: /continuation-prompt
priority: medium
related_prompts:
    - docs/prompts/2026-03-02-testing-strategy-progress.md
started: "2026-03-07"
status: STARTED
tags:
    - continuation
    - testing
    - quality
    - refactoring
title: R2Go2 Testing & Quality - Next Steps
schema_version: 1
---

# R2Go2 Testing & Quality - Next Steps

**Date**: 2026-03-07
**Branch**: master
**Context**: Session 016 completed all 7 goals from the testing strategy. API at 88.2%, TUI at 35.9%, interactive at 6.6%. All 38 vet issues fixed. CI/CD updated. Now focusing on deeper coverage and testability improvements.

## Current State

### Coverage Summary (v0.2.2)

| Package | Coverage | Notes |
|---------|----------|-------|
| `internal/api` | 88.2% | Only `NewClientFromProfile` at 0% (config dependency) |
| `internal/config` | 86.4% | Meets 80% target |
| `internal/interactive` | 6.6% | Limited by stdin/visual code; encryption 74-86% |
| `internal/tui` | 35.9% | DashboardModel cycle tested |
| `internal/cli/ux` | new | Error classification, retry strategies tested |
| `internal/cli/batch` | new | Manager operations tested |

### Key Architecture
- `S3API` interface in `enhanced_client.go` for mock-based upload testing
- `mockS3Client` functional mock pattern (reusable for other packages)
- AES-256 encryption tested with round-trip, wrong password, tampering
- Performance benchmark baseline saved in `tests/performance/benchmark_baseline.txt`

### What's Clean
- `go vet ./internal/cli/... ./internal/interactive/...` — zero issues
- All test packages passing (api, batch, ux, interactive, tui, utils)
- CI/CD pipeline uses Go 1.25/1.26 with latest action versions

## Goals

### [ ] 1. Refactor interactive package for testability
Extract stdin-dependent code behind interfaces to enable mocking. Key files: `setup.go` (4 steps with fmt.Scanln), `profile_manager.go` (ShowProfileSwitcher), `advanced_config.go` (wizard flow). See IDEA-MMGDH6L6.

### [ ] 2. Add httptest-based validation tests
Create mock HTTP servers for `ValidateAPIToken`, `TestConnection`, `getAccountName`, and `autoDetectAccountInfo` in `internal/interactive/validation.go`. Currently only basic format validation is tested. See IDEA-MMGDH7M8.

### [ ] 3. Add visual output suppression for faster tests
`UploadWithRealTimeProgress` tests take 47s due to animation delays. Add a `--quiet` or `DisableAnimations` flag to visual package functions to skip sleeps in test mode. See IDEA-MMGDH93M.

### [ ] 4. Push TUI coverage toward 60%
Currently 35.9%. Key gaps: `handleKeyMsg` (keyboard navigation), view rendering sections (`renderOverview`, `renderBuckets`, etc.), and `dashboard.go` (DashboardCmd, Run). Test more Update message types and View output content.

### [ ] 5. Add integration test for complete upload flow
End-to-end test: create mock S3 server → NewClient → EnhancedClient → UploadFile (single + multipart) → verify results. This would exercise the full stack without network calls.

## Reference Files
- `internal/api/enhanced_client.go` — S3API interface + upload implementation
- `internal/api/enhanced_client_test.go` — Mock pattern to replicate
- `internal/api/client_test.go` — Client function tests
- `internal/interactive/interactive_test.go` — Encryption + validation tests
- `internal/tui/tui_test.go` — DashboardModel tests
- `tests/performance/benchmark_baseline.txt` — Performance baseline
- `docs/sessions/Session-016-testing-coverage-and-vet-fixes.md` — Session summary
