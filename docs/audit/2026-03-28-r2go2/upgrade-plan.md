# Upgrade Plan: CosmoDev-R2Go2

## Phase 0: Critical Bugs (must-fix, 1-2 days)

| # | Action | File:Line | Effort |
|---|--------|-----------|--------|
| 1 | Fix `printErrorAndExit()` — add `os.Exit(1)` | cmd/list.go:130 | 5 min |
| 2 | Fix speed calc — `time.Since(startTime)` | enhanced_client.go:198,277 | 10 min |
| 3 | Fix PersistentPreRun — skip for non-API commands | cmd/root.go:63 | 30 min |
| 4 | Fix keyboard off-by-one — `int(num-'1')` | menu_selection.go:170 | 5 min |
| 5 | Add MIT LICENSE file | repo root | 5 min |
| 6 | Fix install.sh binary name (R2Go2 → r2go2) | install.sh:131 | 10 min |

## Phase 1: Foundation (1-2 weeks)

| # | Action | Impact | Effort |
|---|--------|--------|--------|
| 1 | **Wire up real S3 API client** (ROAD-000) | Enables core product | Large |
| 2 | Extract Client interface (TASK-002) | Enables testing + mocking | Medium |
| 3 | Remove tracked binaries from git | Repo hygiene | Small |
| 4 | Fix CI Go version (1.26 → 1.25) | Unblocks CI pipeline | Small |
| 5 | Update README to match reality | Honest positioning | Medium |
| 6 | Replace deprecated GitHub Actions | CI reliability | Small |
| 7 | Add HTTP client timeout (30s) | Security hardening | Small |

## Phase 2: Quality (2-4 weeks)

| # | Action | Impact | Effort |
|---|--------|--------|--------|
| 1 | Add structured logging (slog/zerolog) | Observability | Medium |
| 2 | Consolidate theme systems (3 → 1 lipgloss) | Maintainability | Medium |
| 3 | Consolidate mock systems (3 → 1) | Test quality | Medium |
| 4 | Unify error classification systems | Consistency | Medium |
| 5 | Implement config encryption | Security | Medium |
| 6 | Fix search (real regex/glob) | Correctness | Small |
| 7 | Add bucket name validation | Correctness | Small |
| 8 | Fix YAML output (use yaml.v3) | Correctness | Small |

## Phase 3: Growth (ongoing)

| # | Action | Impact | Effort |
|---|--------|--------|--------|
| 1 | Complete object operations (ROAD-010) | Feature parity | Large |
| 2 | Add sync/mirror command (ROAD-003) | Key differentiator gap | Large |
| 3 | Wire TUI dashboard to real API data | UX value | Medium |
| 4 | Implement accessibility modes (high contrast, large text) | Inclusivity | Medium |
| 5 | Add `go install` support | Discoverability | Small |
| 6 | Define milestones (v0.3/v0.4/v0.5/v1.0) | Strategic clarity | Small |
| 7 | Run roadmap grooming cycle | Planning health | Small |
