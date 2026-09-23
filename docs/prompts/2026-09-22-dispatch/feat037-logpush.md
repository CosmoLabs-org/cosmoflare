# FEAT-037 wave 1 — Logpush job management

Repo: /Users/gabstudio/PROJECTS/cosmoflare. Research verdict: REST-only, no CLI coverage anywhere. Logpush first per the issue's wave order (feeds observability). A CONCURRENT lane will add loadbalancer AFTER this merges — you own the registry NOW.

## Files you own

- `pkg/cosmoflare/logpush.go` — NEW service
- `pkg/cosmoflare/logpush_test.go` — NEW
- `cmd/logpush.go` — NEW (own init(); registers `logpush` on rootCmd — follow cmd/tunnel.go's structure)
- `cmd/logpush_run_test.go` — NEW
- `internal/cmdmanifest/data.go` — registry entries
- `cmd/manifest_tree_test.go` — add "logpush" to the group lists + switches

## Surface (account-scoped Logpush; cloudflare-go has typed LogpushJob resources — use them like worker_cron.go does)

- `POST /accounts/{account_id}/logpush/ownership` — destination ownership validation (`POST .../logpush/ownership/validate` for the file-based check) — the issue explicitly calls out destination ownership validation
- `POST /accounts/{account_id}/logpush/jobs` — create
- `GET /accounts/{account_id}/logpush/jobs` — list
- `GET /accounts/{account_id}/logpush/jobs/{job_id}` — get
- `PUT /accounts/{account_id}/logpush/jobs/{job_id}` — update
- `DELETE /accounts/{account_id}/logpush/jobs/{job_id}` — delete (destructive + high, irreversible)

Permission (Qwen dataset, verified): account `Logs: Edit` for ALL Logpush ops — set it for every entry, reads included.

## Commands

`cosmoflare logpush job create --dataset X --destination-conf URI [--name] [--frequency]`, `logpush job list`, `logpush job get ID`, `logpush job update ID [flags]`, `logpush job delete ID [--force]`, `logpush ownership verify --dataset X --destination-conf URI`.

- `delete` uses `destructiveDryRun([]string{"logpush", "job", "delete"}, force)` exactly like cmd/bucket.go.
- Dataset names (edge/request, dns/firewall etc.) validate against a known list ONLY as a help-text hint, not a hard block — Cloudflare owns the real validation.
- Conventions per CLAUDE.md: rich Long help with examples, --json via outPayload/outResult/outErr (pattern: cmd/tunnel.go), agent-readable errors. IDs are IDs — no prefix (TASK-011 rule).

## Registry

One entry per live leaf under `logpush` (6 commands). Add "logpush" to ALL group lists/switches in cmd/manifest_tree_test.go. WranglerEquivalent: none exists ("" for all — that is the differentiation).

## Tests

Service: httptest stub (pattern: registrar_test.go registrarStubServer) — CRUD round trips, ownership validation, malformed envelope errors. Runners: missing-creds guards per subcommand, dry-run on delete, JSON envelope shapes. No network.

## Verify

`go build ./... && go vet ./cmd/ ./pkg/cosmoflare/ && go test ./cmd/ ./pkg/cosmoflare/ ./internal/cmdmanifest/ -count=1 -timeout 420s` green (the tree tests prove both directions).

## Commit

`feat(logpush): job management with ownership validation (FEAT-037)` — conventional, no AI attribution.
