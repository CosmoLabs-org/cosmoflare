---
brainstorm_ref: docs/brainstorming/2026-05-16-cors-transform-rules.md
branch: master
completed: "2026-05-18"
created: "2026-05-16"
goals_completed: 5
goals_total: 5
priority: medium
related_prompts: []
requires_reading:
    - docs/brainstorming/2026-05-16-cors-transform-rules.md
schema_version: 1
status: COMPLETED
tags: []
title: GLM Parallel Coverage Wave 2 + CORS Brainplan
---

# GLM Parallel Coverage Wave 2 + CORS Brainplan

## File Scope

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. `ccs prompts load-context` enforces this.**

Read in order:

1. **`docs/brainstorming/2026-05-16-cors-transform-rules.md`** — the design spec.

```yaml
files_modified:
  - cmd/bucket.go
  - cmd/object.go
  - internal/cli/batch/manager.go
  - internal/cli/batch/manager_test.go
files_created:
  - pkg/r2go2/cors.go
  - pkg/r2go2/cors_test.go
  - cmd/cors.go
  - cmd/cors_test.go
  - cmd/status.go
  - pkg/r2go2/status.go
  - docs/brainstorming/2026-05-16-cors-transform-rules.md
```

## Context

Cosmoflare is in a coverage-first phase. The previous session dispatched 12 GLM 5.1 agents in parallel — 11 merged, 1 rejected (CORS). Coverage improved across all major packages but cmd/ (21.4%) and pkg/r2go2 (44.8%) still have room. The rejected CORS agent revealed Cloudflare has NO native CORS zone setting — it's handled via Transform Rules API or Workers. ROAD-016 needs a brainplan before dispatch.

The project is at v0.8.0 with full Cloudflare platform coverage (R2, Workers, KV, DNS, Zones, SSL, Cache, Email, Page Rules, Healthchecks, Diagnostics, Domains). The test infrastructure is solid, all commands support --json, and the GLM dispatch pipeline is proven.

## GLM Dispatch Rules

1. **ALWAYS** use `ccs glm-agent exec` for GLM agents (routes through queue with retry)
2. **NEVER** use Agent tool with `model:sonnet`/`model:haiku` for GLM work (bypasses queue)
3. Agent tool with `model:opus` is fine for Opus subagents
4. For parallel work: use `ccs glm-agent exec-batch` or dispatch multiple `ccs glm-agent exec` calls

## What Got Done (Previous Session)

- **12 GLM 5.1 agents dispatched in 3 waves**, 11 merged to master
- **Coverage gains**: interactive 81.7%->92.2%, batch 93.3%->94.7%, cmd/ 12.2%->21.4%, pkg/ 31.6%->44.8%
- **5 bugs fixed**: BUG-012 to BUG-016 (TUI panics + animation overflow)
- **Batch race condition fixed**: goroutine lifecycle restructured with proper channel drain
- **275 new cmd tests** across pagerules, email, bucket, object, copy, dns, zone, ssl, cache, worker, kv, d1, firewall, waf, config, analytics, auth, compare
- **CORS rejected**: Cloudflare doesn't have native CORS settings — needs Transform Rules approach
- **Code review passed**: 2 issues found (gofmt fixed, dead collectResults noted)

## Goals

### [x] 1. Brainplan ROAD-016: CORS via Transform Rules
**Model:** `sonnet` | **Files:** `docs/brainstorming/2026-05-16-cors-transform-rules.md`
Research Cloudflare Transform Rules API for CORS header injection. The cloudflare-go SDK has `CreateRuleset`, `UpdateRuleset`, `ListRulesets` for response header modification. Produce a brainstorm doc with: (a) correct API mapping, (b) CORSService design using Transform Rules, (c) CLI UX for `r2go2 cors` commands. Then plan and dispatch GLM agent with exact implementation.

### [x] 2. Push cmd/ coverage from 21.4% to 50%+
**Model:** `glm-turbo` | **Files:** `cmd/bucket.go`, `cmd/object.go`, `cmd/worker.go`, `cmd/kv.go`
Current tests only cover registration/flags. Add execution-path tests that mock the service layer. Pattern: set `runXxx` functions to use injected services, test with mock service returning canned responses. Focus on the 4 most-used commands first (bucket, object, worker, kv).

### [x] 3. Push pkg/r2go2 coverage from 44.8% to 70%+
**Model:** `glm-turbo` | **Files:** `pkg/r2go2/client.go`, `pkg/r2go2/storage.go`, `pkg/r2go2/upload.go`, `pkg/r2go2/download.go`
Add httptest-based tests for the R2 storage layer (client, storage, upload, download). These are the core functions with lowest coverage. Mock S3 responses via httptest.NewServer.

### [x] 4. ROAD-050: `cosmoflare status` dashboard command
**Model:** `sonnet` | **Files:** `cmd/status.go`, `pkg/r2go2/status.go`
Design and implement the at-a-glance status command (priority 88 on roadmap). Shows: zone count, worker count, KV namespace count, R2 bucket count, recent errors, SSL status summary. Needs brainplan for output format, then GLM dispatch for implementation.

### [x] 5. Remove dead `collectResults()` method
**Model:** `glm-turbo` | **Files:** `internal/cli/batch/manager.go`, `internal/cli/batch/manager_test.go`
Code review flagged `collectResults()` as dead code after the race fix inlined its logic. Remove the method and update `TestCollectResults_*` tests to test via `Execute()` instead. Verify: `go test ./internal/cli/batch/... -race -cover`

## Carry-Over Tasks

- [x] ROAD-016 CORS (was: rejected by quality gate — wrong API mapping)
- [x] ROAD-050 cosmoflare status (was: brainplan needed, priority 88)
- [x] ROAD-040 WAF/Firewall (was: identified as Wave 2 candidate, needs brainplan)

## Where We're Headed

The project is converging on a release-ready state. Coverage is the main gap — once cmd/ hits 50%+ and pkg/ hits 70%+, we can cut v0.9.0. ROAD-050 (`cosmoflare status`) is the highest-priority UX feature. After that, ROAD-040 (WAF) and ROAD-039 (Page Rules — already implemented, just needs more tests) round out Phase 5 services. The Tauri desktop app (ROAD-063) and React Native mobile (ROAD-064) are the next major architectural milestones but live in separate repos.

## Priority Order
1. ROAD-016 CORS brainplan (unblocks feature dispatch)
2. cmd/ + pkg/ coverage (unblocks v0.9.0 release confidence)
3. ROAD-050 status command (highest-priority UX item)
4. Dead code cleanup (quick win)
5. ROAD-040 WAF brainplan (Phase 5 planning)
