# FEAT-037 wave 3 — Waiting Room + Spectrum

Repo: /Users/gabstudio/PROJECTS/cosmoflare. Zone long-tail services, per-service services + commands. NO registry edits — a follow-up pass owns internal/cmdmanifest/data.go and cmd/manifest_tree_test.go (do NOT touch them; your new groups stay unregistered this round, that is expected).

## Files you own

- `pkg/cosmoflare/waitingroom.go` + `pkg/cosmoflare/waitingroom_test.go` — NEW
- `pkg/cosmoflare/spectrum.go` + `pkg/cosmoflare/spectrum_test.go` — NEW
- `cmd/waitingroom.go` + `cmd/waitingroom_run_test.go` — NEW
- `cmd/spectrum.go` + `cmd/spectrum_run_test.go` — NEW

## Waiting Room (zone-scoped; use cloudflare-go's typed WaitingRoom resources like worker_cron.go does)

- `GET/POST /zones/{zone_id}/waiting_rooms` — list/create
- `GET/PATCH/DELETE /zones/{zone_id}/waiting_rooms/{room_id}` — get/update/delete
- Permission (Qwen dataset, verified exact name): zone `Waiting Room`. If the SDK splits reads/writes use it; record `Permissions{Zone: []string{"Waiting Room"}}` in your service doc comment — the registry pass consumes it.

Commands: `cosmoflare waiting-room create|list|get|update|delete` — create needs --name, --host, --new-users-per-minute, --total-active-users; document sensible defaults in help.

## Spectrum (zone-scoped; cloudflare-go SpectrumApp resources)

- `GET/POST /zones/{zone_id}/spectrum/apps` — list/create
- `GET/PATCH/DELETE /zones/{zone_id}/spectrum/apps/{app_id}` — get/update/delete
- Permission: NOT in the Qwen dataset → the registry pass leaves it sparse; note `// Spectrum permission pending FEAT-011 dataset` in the service doc. Spectrum TCP monitoring is Enterprise-gated (dataset note) — say so in the command Long help.

Commands: `cosmoflare spectrum app create|list|get|update|delete` — create needs --name, --protocol (e.g. tcp/80), --origin-port; help documents Enterprise gating.

## Delete contract (both)

`destructiveDryRun(cliPathOfCmdOr(cmd, "waiting-room delete"), force)` / `("spectrum app delete")` + `ux.Confirm` (pattern: cmd/dns.go runDNSDelete — flag destructive in the registry LATER activates dry-by-default automatically). `--force` skips the prompt. Use `cliPathOfCmdOr` from cmd/destructive.go.

## Conventions

Rich Long help with examples, --json via outPayload/outResult/outErr (pattern: cmd/tunnel.go), agent-readable errors, missing-creds guards, zone ID = ID (no prefix). Tests: httptest stubs (registrar_test.go pattern; NOTE the stateful-stub lesson from logpush — a stub serving re-fetch-after-write must return the WRITTEN state), runner guards, dry-run/confirm behavior, JSON envelopes. No network.

## Verify

`go build ./... && go vet ./cmd/ ./pkg/cosmoflare/ && go test ./cmd/ ./pkg/cosmoflare/ -count=1 -timeout 420s` green.

## Commits

Two: `feat(waiting-room): zone waiting room CRUD (FEAT-037)` then `feat(spectrum): app management (FEAT-037)` — conventional, no AI attribution.
