# FEAT-037 wave 2 — Load Balancer pools + monitors

Repo: /Users/gabstudio/PROJECTS/cosmoflare. Logpush (wave 1) is merged — the registry is yours now.

## Files you own

- `pkg/cosmoflare/loadbalancer.go` — NEW service
- `pkg/cosmoflare/loadbalancer_test.go` — NEW
- `cmd/loadbalancer.go` — NEW (own init(); registers `loadbalancer` on rootCmd — follow cmd/tunnel.go's structure)
- `cmd/loadbalancer_run_test.go` — NEW
- `internal/cmdmanifest/data.go` — registry entries
- `cmd/manifest_tree_test.go` — add "loadbalancer" to the group lists + switches

## Surface (account-scoped; cloudflare-go has typed LoadBalancerPool/LoadBalancerMonitor resources — use them like worker_cron.go does)

Pools:
- `POST /accounts/{account_id}/load_balancers/pools` — create
- `GET /accounts/{account_id}/load_balancers/pools` — list
- `GET /accounts/{account_id}/load_balancers/pools/{pool_id}` — get (also `.../pools/{pool_id}/health` for health detail if the SDK exposes it)
- `PATCH /accounts/{account_id}/load_balancers/pools/{pool_id}` — update
- `DELETE /accounts/{account_id}/load_balancers/pools/{pool_id}` — delete (destructive + high)

Monitors:
- `POST /accounts/{account_id}/load_balancers/monitors` — create
- `GET /accounts/{account_id}/load_balancers/monitors` — list
- `GET /accounts/{account_id}/load_balancers/monitors/{monitor_id}` — get
- `PATCH /accounts/{account_id}/load_balancers/monitors/{monitor_id}` — update
- `DELETE /accounts/{account_id}/load_balancers/monitors/{monitor_id}` — delete (destructive + high)

Permission (Qwen dataset, verified): account `Load Balancing: Monitors and Pools` — set for every entry, reads included.

## Commands

`cosmoflare loadbalancer pool create|list|get|update|delete`, `loadbalancer monitor create|list|get|update|delete` (10 commands). Flag surface: keep MINIMAL and honest — pool create takes `--name --origins JSON` (array of {name,address,enabled}) and optional `--monitor ID`; monitor create takes `--type http|https --path --port --expected-codes --interval --retries --timeout` with sane defaults documented in help. `delete` commands use `destructiveDryRun([]string{"loadbalancer", "pool", "delete"}, force)` / `("loadbalancer", "monitor", "delete")` exactly like cmd/bucket.go. IDs are IDs — no prefix (TASK-011 rule).

Conventions per CLAUDE.md: rich Long help with examples, --json via outPayload/outResult/outErr (pattern: cmd/tunnel.go), agent-readable errors.

## Registry

One entry per live leaf (10 commands), permissions `Load Balancing: Monitors and Pools`. Add "loadbalancer" to ALL group lists/switches in cmd/manifest_tree_test.go. WranglerEquivalent: none ("" — differentiation surface).

## Tests

Service: httptest stub (pattern: registrar_test.go registrarStubServer, or logpush_test.go from wave 1 — note its stateful-stub lesson: a stub serving re-fetch-after-write must return the WRITTEN state). CRUD round trips + malformed envelope errors. Runners: missing-creds guards, dry-run on both deletes, JSON envelope shapes, origin-JSON parse errors. No network.

## Verify

`go build ./... && go vet ./cmd/ ./pkg/cosmoflare/ && go test ./cmd/ ./pkg/cosmoflare/ ./internal/cmdmanifest/ -count=1 -timeout 420s` green.

## Commit

`feat(loadbalancer): pool and monitor management (FEAT-037)` — conventional, no AI attribution.
