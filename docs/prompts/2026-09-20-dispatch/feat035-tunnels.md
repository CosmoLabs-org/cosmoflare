# FEAT-035 — Cloudflare Tunnels management CLI

Repo: /Users/gabstudio/PROJECTS/cosmoflare (Go). Research verdict: wrangler has NO tunnel management commands — this is differentiation surface.

## Files you own (all NEW — do not touch existing files except where named)

- `pkg/cosmoflare/tunnel.go` — service
- `pkg/cosmoflare/tunnel_test.go` — service tests
- `cmd/tunnel.go` — CLI commands
- `cmd/tunnel_run_test.go` — runner tests

## Verified endpoints (from docs/research/2026-09-10-cf-limits-corpus/coverage-catalog-draft.json, "Cloudflare Tunnels")

- `POST /accounts/{account_id}/cfd_tunnel` — create
- `GET /accounts/{account_id}/cfd_tunnel` — list
- `GET /accounts/{account_id}/cfd_tunnel/{tunnel_id}` — get
- `DELETE /accounts/{account_id}/cfd_tunnel/{tunnel_id}` — delete (`?cascade=true` deletes connected resources)
- `GET /accounts/{account_id}/cfd_tunnel/{tunnel_id}/token` — connector token
- `DELETE /accounts/{account_id}/cfd_tunnel/{tunnel_id}/connections` — force-disconnect active connectors
- `PUT /accounts/{account_id}/cfd_tunnel/{tunnel_id}/configurations` — ingress configuration

Permissions (token-UI names, verified): account `Cloudflare Tunnel: Read` for reads, `Cloudflare Tunnel: Edit` for writes.

## Service shape (`pkg/cosmoflare/tunnel.go`)

Follow `pkg/cosmoflare/queue.go` or `hyperdrive.go` as the structural template: a `TunnelService` with `NewTunnelServiceFromCreds(accountID, apiToken)`, typed `Tunnel`/`TunnelConnection` structs, context-first methods. The library wraps the `cloudflare-go` SDK like worker_cron.go does (`cloudflare.Tunnel` etc.) — use the SDK's tunnel resources, do NOT hand-roll HTTP.

## Commands (`cmd/tunnel.go`)

Group `cosmoflare tunnel` with subcommands: `create <name>`, `list`, `get <id-or-name>`, `delete <id> [--force] [--cascade]`, `token <id>`, `connections <id>`, plus `cleanup <id>` (delete + connections teardown). Conventions per CLAUDE.md: rich `Long` help with examples, `--json` on every command via the outPayload/outResult/outErr helpers (see cmd/hyperdrive.go for the pattern), agent-readable errors (what failed, why, how to fix). `delete` uses `destructiveDryRun([]string{"tunnel", "delete"}, force)` exactly like cmd/bucket.go does, and registers `--force`.

Registry entries: NONE — wave 4 adds them to internal/cmdmanifest/data.go (a separate agent owns that file right now; do not touch it).

## Tests

- Service tests: follow tunnel_test.go patterns of queue_test.go — construction requires account ID + token (error paths), happy paths via the SDK seams the sibling tests use.
- Runner tests (cmd/tunnel_run_test.go): missing-creds guards per subcommand, dry-run default on delete (registry lookup returns false today — assert the flag plumbing, not the registry), JSON envelope shape. Use the creds-guard + capturePrint patterns from cmd/doctor_run_test.go.

## Verify

`go build ./...`, `go vet ./cmd/ ./pkg/cosmoflare/`, `go test ./cmd/ ./pkg/cosmoflare/ -count=1` — all green.

## Commit

`feat(tunnel): Cloudflare Tunnels management — CRUD, token, connections, cleanup (FEAT-035)` — conventional, no AI attribution.
