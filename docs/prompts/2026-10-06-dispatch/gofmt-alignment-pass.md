---
created: "2026-10-06T22:58:04+04:00"
goals_completed: 0
goals_total: 0
origin: migrated by ccs prompts migrate
priority: medium
related_prompts: []
requires_reading: []
schema_version: 1
status: PENDING
tags: []
title: Repository gofmt alignment pass — 84 files, whitespace-only
---

# Repository gofmt alignment pass — 84 files, whitespace-only

## Task

Run `gofmt -w` on EXACTLY these 84 files (pre-computed drift list — do not add or skip any):

```
pkg/cosmoflare/account_members_test.go
pkg/cosmoflare/ai.go
pkg/cosmoflare/ai_test.go
pkg/cosmoflare/cors.go
pkg/cosmoflare/cost.go
pkg/cosmoflare/d1_sql_splitter.go
pkg/cosmoflare/dns_test.go
pkg/cosmoflare/doctor_test.go
pkg/cosmoflare/export.go
pkg/cosmoflare/fleet_status_test.go
pkg/cosmoflare/guardrails_test.go
pkg/cosmoflare/healthcheck.go
pkg/cosmoflare/hyperdrive.go
pkg/cosmoflare/images.go
pkg/cosmoflare/logpush_test.go
pkg/cosmoflare/multipart_test.go
pkg/cosmoflare/pages_env_test.go
pkg/cosmoflare/plugin_test.go
pkg/cosmoflare/r2_bucket_integration_test.go
pkg/cosmoflare/registrar_test.go
pkg/cosmoflare/retry_test.go
pkg/cosmoflare/stream.go
pkg/cosmoflare/sync.go
pkg/cosmoflare/sync_test.go
pkg/cosmoflare/types.go
pkg/cosmoflare/upload_test.go
pkg/cosmoflare/validate.go
pkg/cosmoflare/vectorize.go
pkg/cosmoflare/waf_test.go
pkg/cosmoflare/worker.go
pkg/cosmoflare/worker_test.go
pkg/cosmoflare/wrangler.go
internal/cli/batch/manager.go
internal/cli/batch/worker.go
internal/cli/batch/worker_test.go
internal/cli/operations/copy.go
internal/cli/progress/progress.go
internal/cli/ux/confirmation.go
internal/cli/ux/confirmation_test.go
internal/cli/ux/retry.go
internal/cli/ux/retry_test.go
internal/cli/visual/animations.go
internal/cli/visual/r2-progress.go
internal/cli/visual/visual_test.go
internal/config/config.go
internal/interactive/accessibility.go
internal/interactive/accessibility_test.go
internal/interactive/advanced_config.go
internal/interactive/advanced_config_test.go
internal/interactive/backup_restore.go
internal/interactive/backup_restore_test.go
internal/interactive/errors.go
internal/interactive/errors_test.go
internal/interactive/first_run.go
internal/interactive/helpers.go
internal/interactive/helpers_test.go
internal/interactive/interactive_test.go
internal/interactive/profile_manager.go
internal/interactive/profile_manager_test.go
internal/interactive/setup.go
internal/interactive/test.go
internal/interactive/themes_deep_test.go
internal/interactive/transitions.go
internal/interactive/tutorials_test.go
internal/interactive/validation.go
internal/keychain/keychain.go
internal/keychain/keychain_test.go
internal/migration/s3.go
internal/migration/worker.go
internal/server/metrics.go
internal/server/metrics_test.go
internal/server/rest_test.go
internal/server/server.go
internal/tui/components/installer/header.go
internal/tui/components/navigation/menu_selection.go
internal/tui/dashboard.go
internal/tui/datasource.go
internal/tui/model.go
internal/tui/model_test.go
internal/tui/render_test.go
internal/tui/update.go
internal/tui/view.go
internal/tui/view_test.go
internal/webhook/manager.go
```

Then verify and commit:

```bash
go build ./...        # must exit 0
go vet ./...          # must exit 0
```

Commit message:

```
style: gofmt alignment across 84 drifted files

Whitespace/alignment-only; no behavior change. Drift accumulated since the
Go version's gofmt alignment rules changed. Verified: go build ./... and
go vet ./... exit 0.
```

## Rules

- gofmt ONLY — no hand-edits, no import reordering beyond gofmt's own, no renames.
- Do not touch any file outside the list.
- Do not reformat the list itself differently (gofmt decides).
- If any file fails to build AFTER gofmt (should be impossible for format-only changes), STOP and report instead of committing.
