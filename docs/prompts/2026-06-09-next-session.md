---
created: "2026-06-09T00:30:00-03:00"
goals_completed: 0
goals_total: 4
priority: medium
related_prompts:
  - docs/prompts/2026-06-07-next-session.md
requires_reading:
  - CLAUDE.md
schema_version: 1
status: PENDING
tags: []
title: Cosmoflare — Next Session
---

# Cosmoflare — Next Session Continuation

## File Scope
```yaml
files_modified:
  - cmd/*.go
  - cmd/*_test.go
  - pkg/cosmoflare/client.go
  - internal/tui/view.go
  - internal/webhook/webhook_test.go
files_created: []
```

## Context

Cosmoflare is firing on all cylinders. The prior session was massive — 3 major ROAD items shipped (ROAD-020 dashboard, ROAD-002 object browser, ROAD-007 S3 migration), plus FEAT-005 (keychain credentials) and a blocker fix. The roadmap is at 75/88 completed items with only 3 ROAD items remaining, all large-scope (Tauri desktop, React Native mobile, CCS integration). The CLI is feature-complete for the open-source tier.

The codebase is in strong shape: 1,733 tests passing, clean build, no open issues or features. The biggest remaining quality gap is **cmd/ test coverage at ~27.5%** — every `RunE` handler calls `NewClient()` which requires live Cloudflare credentials, making commands untestable without DI. This was item #7 on the prior continuation prompt and the only item not tackled.

## GLM Dispatch Rules

When goals involve dispatching subagents:

1. **ALWAYS** use `ccs glm-agent exec` for GLM agents (routes through queue with retry logic)
2. **NEVER** use Agent tool with `model:sonnet` or `model:haiku` for GLM work (bypasses queue, risks 429 rate limits)
3. Agent tool with `model:opus` is fine for Opus subagents
4. For parallel work: use `/glm-sprint` or `ccs glm-agent exec-batch`

## What Got Done (prior session)

**20 commits, 3 ROAD items, 1 FEAT closed:**

- **ROAD-020: Dashboard TUI** — full Bubble Tea dashboard with live monitoring (R2/Workers/KV with delta indicators), bucket CRUD (create/delete with confirmation), basic object list with pagination, tiered refresh (auto-poll monitoring, manual elsewhere), graceful no-credentials launch. `cmd/metrics.go` absorbed into dashboard. DataSource abstraction in `internal/tui/datasource.go`.
- **ROAD-002: Object Browser** — split-pane BrowserModel (`internal/tui/browser.go`, 530 lines) with bucket list left, objects right. Prefix-based folder navigation with stack. Wide mode (side-by-side) / narrow mode (single pane + Tab). Bottom detail panel. HeadObject modal. Object delete with two-keypress confirm. Token-based pagination. Library extended with `CommonPrefixes` and `ContinuationToken` in `ListObjects`.
- **ROAD-007: S3 Migration Resume** — replaced simulated transfer loop with real S3→R2 streaming engine. Worker pool with concurrent transfers (`internal/migration/worker.go`). Checkpoint persistence in `~/.cosmoflare/migrations/` (`internal/migration/checkpoint.go`) with atomic writes. Per-object retry (3x exponential backoff). Graceful SIGINT shutdown. ETag verification mode. Objects >100MB auto-route to multipart upload.
- **FEAT-005: Keychain credentials** — `internal/keychain/` package wrapping `go-keyring`. Config layer stores sentinel in YAML, hydrates from OS keychain. `cosmoflare config migrate-keychain` command. Falls back to file when keychain unavailable.
- **Fixes**: Removed orphaned `internal/api/` test files (1,495 lines). Fixed `TestNewClientValidation` env var dependency. Triaged and closed 2 ideas.

## Goals

### [ ] 1. Release v0.15.0
**Model:** `glm-turbo` | **Files:** `.version-registry.json`, `CHANGELOG.md`
3 ROAD items + 1 FEAT since v0.14.0 = ready for a minor bump. Run `ccs changelog preview` to confirm 4 staged entries (ROAD-020, ROAD-002, ROAD-007, FEAT-005), then `/release` to bump, tag, update registry.

### [ ] 2. cmd/ DI refactoring — brainplan
**Model:** `opus` | **Files:** `cmd/*.go`, `cmd/*_test.go`
`cmd/` is at ~27.5% coverage. The blocker: `RunE` handlers all call `NewClient()` which requires live Cloudflare credentials. To test without credentials, commands need dependency injection (accept a `Client` interface or factory function). This is a design task — run `/brainplan` to explore the DI pattern before implementing. Key questions: inject via global var, per-command factory, or cobra context? How to handle commands that need multiple services (R2Client + WorkerService + KVService)? What's the minimum-touch approach that doesn't restructure every command?

### [ ] 3. Fix internal/webhook flaky test
**Model:** `sonnet` | **Files:** `internal/webhook/webhook_test.go` | **Reason:** requires diagnosis
`TestTriggerAlert_WithStoreMultipleWebhooks` fails intermittently — "expected 2 calls (2 enabled + 1 disabled + 1 missing), got 1". Observed during this session's test runs. Likely a race condition or timing issue in the test's mock HTTP server. Diagnose and fix.

### [ ] 4. Roadmap health cleanup
**Model:** `glm-turbo` | **Files:** `docs/roadmap/items/*.yaml`
`ccs roadmap health` reports 19 findings: 7 drifted items (recent commits touch related files but item not updated), orphaned items (ROAD-002, ROAD-007 linked issues both closed). Run `ccs roadmap health --fix` for the auto-fixable ones, manually review the drifted BASE-* items.

## Carry-Over Tasks
- [ ] cmd/ coverage — DI refactoring (was: pending)

## Where We're Headed

The CLI is feature-complete for the open-source tier. The remaining 3 ROAD items (ROAD-063 Tauri desktop, ROAD-064 React Native mobile, ROAD-029 CCS integration) are all large-scope and represent the paid tier (desktop + mobile apps). Before starting those, the priority is hardening: test coverage (especially cmd/), the flaky webhook test, and a clean v0.15.0 release.

After that, the natural next step is ROAD-063 (Tauri desktop app) — it wraps the same `pkg/cosmoflare/` library in a GUI, which means the library API quality matters. Good cmd/ test coverage validates the library's public surface before building a second consumer on top of it.

## Priority Order
1. Release v0.15.0 (quick win, closes out the sprint)
2. Fix flaky webhook test (small but blocking clean CI)
3. Roadmap health cleanup (housekeeping, 10 min)
4. cmd/ DI brainplan (strategic — sets up the testing foundation)
