# TASK-013 — AlertService cannot produce a disabled rule through its API

Repo: /Users/gabstudio/PROJECTS/cosmoflare (Go, module github.com/CosmoLabs-org/cosmoflare)

## Problem (verified by coverage batch 005, 2026-09-18)

`pkg/cosmoflare/alerts.go:255` — `Create` force-sets `rule.Enabled = true` on every created rule. `AlertRuleUpdate` has no `Enabled` field. Consequence: no API path creates or updates a rule into a disabled state — only hand-editing the YAML store can. `runAlertsCheck` correctly skips work when only disabled rules exist, but that path is unreachable via the CLI. `TestRunAlertsCheck_OnlyDisabledRules` (cmd/alerts_check_test.go) pins the behavior by writing the YAML directly.

## Fix (the issue's fix direction)

Files you own: `pkg/cosmoflare/alerts.go`, `pkg/cosmoflare/alerts_test.go`, and if the wire requires it `cmd/alerts.go`.

1. `AlertRuleUpdate`: add `Enabled *bool` (pointer — nil means "leave unchanged").
2. `Create`: honor an explicit enabled/disabled value from the create input instead of force-enabling. When the caller specifies nothing, keep the current default (rule created enabled).
3. `Update`: when `Enabled != nil`, set the stored rule's enabled state to the dereferenced value.
4. Tests in `pkg/cosmoflare/alerts_test.go` following the file's existing patterns:
   - create with explicit disabled state → stored rule stays disabled
   - update enable→disable flips the state
   - update with nil `Enabled` leaves the state unchanged
   - the existing default-enabled create behavior still holds

## Verify

`go test ./pkg/cosmoflare/ -run Alert -count=1 -v` all pass. `go build ./...` clean. Do NOT weaken `TestRunAlertsCheck_OnlyDisabledRules`.

## Rules

- Conventional commit: `fix(alerts): honor explicit enabled state in create/update (TASK-013)`, no AI attribution.
