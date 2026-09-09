---
brainstorm_ref: docs/brainstorming/2026-09-09-cf-api-knowledge-layer.md
branch: master
covers_brainstorm_deliverables:
    - BR-01
    - BR-02
    - BR-03
    - BR-04
    - BR-05
    - BR-06
covers_plan_deliverables:
    - P-01
    - P-02
    - P-03
    - P-04
    - P-05
    - P-06
    - P-07
created: "2026-09-09T18:52:56+04:00"
id: P-2026-09-09-cf-api-knowledge-layer
plan_ref: docs/planning-mode/2026-09-09-cf-api-knowledge-layer.md
priority: medium
requires_reading:
    - docs/brainstorming/2026-09-09-cf-api-knowledge-layer.md
    - docs/planning-mode/2026-09-09-cf-api-knowledge-layer.md
schema_version: 1
status: PENDING
title: 'Cosmoflare — CF API knowledge layer: endpoint registry, plan caps, error decoding'
---
# Cosmoflare — CF API knowledge layer: endpoint registry, plan caps, error decoding

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/brainstorming/2026-09-09-cf-api-knowledge-layer.md`** — design rationale and decision history.
2. **`docs/planning-mode/2026-09-09-cf-api-knowledge-layer.md`** — implementation plan with deliverables and file scope.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## Context

FEAT-012 (origin: MyCarGuide feedback FB-6): one Cloudflare rate-limit rule took six failed API calls because CF explains none of its failure classes at the point of failure (10405 on nonexistent routes reads as auth trouble; missing phase entrypoints look like zero rules; 20155 reveals the cf.colo.id requirement late; Free-plan caps are enforced nowhere client-side). This session implements the knowledge layer that encodes that tribal knowledge: embedded JSON packs, a route-blocking transport, pure payload validators, central error decoding — proven by a minimal RateLimitService. The plan contains complete TDD code for every task — implement it spec-exact. Design: `docs/brainstorming/2026-09-09-cf-api-knowledge-layer.md`; plan: `docs/planning-mode/2026-09-09-cf-api-knowledge-layer.md`.

Both documents passed fresh-context independent review on 2026-09-09. Where the plan and this prompt disagree, the plan wins.

## Goals

### [ ] G-01 Knowledge package — types, embedded loader, matchPath, CheckRoute with tests
Covers P-01. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/knowledge/knowledge.go`, `knowledge_test.go`, `packs/ratelimit.json` (placeholder `{}` — embed glob must match a file; plan Task 1).

### [ ] G-02 ratelimit.json seed pack + pack-validation test
Covers P-02. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/knowledge/packs/ratelimit.json`, `knowledge_test.go` (plan Task 2 — replaces the placeholder). Sequential after G-01 (same package).

### [ ] G-03 knowledge.Transport (route block) + DecodeCFError + newError hook with tests
Covers P-03. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/knowledge/transport.go`, `transport_test.go`, `decode.go`, `pkg/cosmoflare/errors.go` (plan Task 3). After G-02 (pack must exist for transport tests).

### [ ] G-04 ValidatePayload — cap interpreter + invariants (pure) with tests
Covers P-04. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/knowledge/validate.go`, `validate_test.go` (plan Task 4). After G-02; parallel-safe with G-03 (disjoint files, same package).

### [ ] G-05 RateLimitService — list + create via correct rulesets flow with tests
Covers P-05. **Model:** glm-turbo. **Files:** `pkg/cosmoflare/ratelimit.go`, `ratelimit_test.go` (plan Task 5). Requires G-02+G-03+G-04.

### [ ] G-06 CLI — decode, knowledge list, ratelimit list/create with tests
Covers P-06. **Model:** glm-turbo. **Files:** `cmd/decode.go`, `cmd/knowledge_cmd.go`, `cmd/ratelimit.go`, `cmd/decode_test.go` (plan Task 6). Requires G-05. REUSES existing `resolveZoneID` (cmd/bucket_domain.go:145) — do not add a resolver.

### [ ] G-07 docs/USAGE.md — knowledge layer, decode, ratelimit commands
Covers P-07. **Model:** glm-turbo. **Files:** `docs/USAGE.md` (plan Task 7). Last.

## Execution Strategy

GLM dispatch from `docs/prompts/2026-09-09-cf-api-knowledge-layer-glm-tasks.yaml`. Three waves:

1. Wave 1 (sequential): knowledge package core — Task 1 then Task 2 (same package, Task 2's tests need Task 1's types + the pack)
2. Wave 2 (parallel): Task 3 (transport+decode+errors.go) + Task 4 (validate) — disjoint files
3. Wave 3 (sequential): Task 5 (ratelimit service) → Task 6 (CLI) → Task 7 (docs)

Every merge passes the quality gate: Opus re-reads the diff, re-runs the full affected package in the worktree, `ccs verify-worktree --approve` + `ccs merge` with ancestry check. Agents run `gofmt -w` on touched files before committing.

## File Scope

```yaml
files_modified:
  - pkg/cosmoflare/knowledge/knowledge.go      # types, loader, matchPath, CheckRoute
  - pkg/cosmoflare/knowledge/knowledge_test.go
  - pkg/cosmoflare/knowledge/packs/ratelimit.json
  - pkg/cosmoflare/knowledge/transport.go      # route-blocking RoundTripper
  - pkg/cosmoflare/knowledge/transport_test.go
  - pkg/cosmoflare/knowledge/decode.go         # DecodeCFError + KnowledgeError
  - pkg/cosmoflare/knowledge/validate.go       # caps + invariants interpreter
  - pkg/cosmoflare/knowledge/validate_test.go
  - pkg/cosmoflare/errors.go                   # newError knowledge hook
  - pkg/cosmoflare/ratelimit.go                # RateLimitService
  - pkg/cosmoflare/ratelimit_test.go
  - cmd/decode.go
  - cmd/knowledge_cmd.go
  - cmd/ratelimit.go
  - cmd/decode_test.go
  - docs/USAGE.md
```

## Related

- Brainstorm: `docs/brainstorming/2026-09-09-cf-api-knowledge-layer.md`
- Plan: `docs/planning-mode/2026-09-09-cf-api-knowledge-layer.md`
- GLM tasks: `docs/prompts/2026-09-09-cf-api-knowledge-layer-glm-tasks.yaml`
- Issue: FEAT-012 · Companions: FEAT-011 (permission catalog), FEAT-013 (traffic classes)
