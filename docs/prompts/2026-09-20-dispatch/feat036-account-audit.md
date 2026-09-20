# FEAT-036 — Account management + audit logs

Repo: /Users/gabstudio/PROJECTS/cosmoflare (Go). Research verdict: no CLI covers this; it is the security-layer cornerstone (every cosmoflare mutation becomes auditable from the CLI itself).

## Files you own

- `pkg/cosmoflare/account_members.go` — NEW: members + roles service
- `pkg/cosmoflare/account_audit.go` — NEW: Cloudflare account audit-log query service
- `pkg/cosmoflare/account_members_test.go`, `pkg/cosmoflare/account_audit_test.go` — NEW
- `cmd/account_members.go` — NEW: `cosmoflare account member` subcommands
- `cmd/account_audit.go` — NEW: `cosmoflare account audit-logs` subcommands
- `cmd/account_members_run_test.go`, `cmd/account_audit_run_test.go` — NEW

Do NOT edit `cmd/audit.go` or `cmd/audit_wire.go` — those are the LOCAL mutation audit log, a different feature. The Cloudflare account audit log lives under the `account` group to keep the two unambiguous.

## Verified endpoints (coverage-catalog-draft.json, "Account/Org Management & Audit Logs")

Members:
- `GET /accounts/{account_id}/members` — list
- `POST /accounts/{account_id}/members` — invite (email + role)
- `PUT /accounts/{account_id}/members/{member_id}` — change role
- `DELETE /accounts/{account_id}/members/{member_id}` — remove

Roles:
- `GET /accounts/{account_id}/roles` — list available roles

Audit logs:
- `GET /accounts/{account_id}/logs/audit` — query; supports filters actor.id, actor.type, action, zone, since, before, per_page
- `GET /accounts/{account_id}/logs/audit/{id}/history` — one log entry's change history

Permissions (token-UI names, verified): account `Audit Logs: Read` for audit queries; `Account: Audit Logs: Read` variant if the SDK names it that way — use the SDK's typed resources (`cloudflare.AuditLog`, `cloudflare.AccountMember`) via cloudflare-go like worker_cron.go does; members management needs `Members: Read/Edit` equivalents from the SDK.

## Commands

`cosmoflare account member list|invite|update|remove`, `cosmoflare account role list`, `cosmoflare account audit-logs list [--actor --action --zone --since --before --json]` and `audit-logs history <id>`. The `account` parent group exists in cmd/account.go — add subcommands via a NEW file registering onto `accountCmd` (see how bucket subresources register). `member remove` uses `destructiveDryRun([]string{"account", "member", "remove"}, force)` like cmd/bucket.go.

Conventions per CLAUDE.md: rich `Long` help with examples, `--json` everywhere via outPayload/outResult/outErr (pattern: cmd/hyperdrive.go), agent-readable errors. Export: `audit-logs list --format ndjson|csv` writes records to stdout (NDJSON: one JSON object per line; CSV: header row + rows) — this is the export surface.

Registry entries: NONE (wave 4 owns data.go — do not touch it).

## Tests

Follow the sibling patterns: construction error paths (missing account/token), missing-creds runner guards, JSON envelope shape, NDJSON/CSV export formatting on a fixed record set, filter flag plumbing. Use creds-guard + capturePrint from cmd/doctor_run_test.go.

## Verify

`go build ./...`, `go vet ./cmd/ ./pkg/cosmoflare/`, `go test ./cmd/ ./pkg/cosmoflare/ -count=1` green.

## Commit

`feat(account): members, roles, and audit-log query/export (FEAT-036)` — conventional, no AI attribution.
