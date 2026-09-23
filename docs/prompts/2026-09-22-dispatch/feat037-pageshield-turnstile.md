# FEAT-037 wave 3 — Page Shield + Turnstile

Repo: /Users/gabstudio/PROJECTS/cosmoflare. NO registry edits — a follow-up pass owns internal/cmdmanifest/data.go and cmd/manifest_tree_test.go (do NOT touch them; unregistered this round is expected).

## Files you own

- `pkg/cosmoflare/pageshield.go` + `pkg/cosmoflare/pageshield_test.go` — NEW
- `pkg/cosmoflare/turnstile.go` + `pkg/cosmoflare/turnstile_test.go` — NEW
- `cmd/pageshield.go` + `cmd/pageshield_run_test.go` — NEW
- `cmd/turnstile.go` + `cmd/turnstile_run_test.go` — NEW

## Page Shield (zone-scoped; command group `page-shield`; API path stays page_shield)

- `GET /zones/{zone_id}/page_shield/connections` — list detected connections
- `GET /zones/{zone_id}/page_shield/scripts` — list detected scripts
- `GET/POST /zones/{zone_id}/page_shield/policies` — list/create policies; `GET/PATCH/DELETE /zones/{zone_id}/page_shield/policies/{policy_id}` — get/update/delete
- Permission — CRITICAL: the token-UI group was RENAMED; it is zone `Client-side security` (verified Qwen dataset, formerly "Page Shield" — the dataset caught this rename). Record it in the service doc comment; the command NAME stays `page-shield` (the dataset's own example: `cosmoflare page-shield list`).
- cloudflare-go may lack Page Shield resources — if absent, hand-roll with the client's Raw seam (pattern: pkg/cosmoflare/d1.go registrarRaw-style raw helpers).

Commands: `cosmoflare page-shield connections`, `page-shield scripts`, `page-shield policy create|list|get|update|delete`.

## Turnstile (ACCOUNT-scoped)

- `GET/POST /accounts/{account_id}/challenges/widgets` — list/create
- `GET/PATCH/DELETE /accounts/{account_id}/challenges/widgets/{widget_id}` — get/update/delete
- Permission (Qwen dataset, verified): account `Turnstile`.

Commands: `cosmoflare turnstile widget create|list|get|update|delete` — create needs --name and at least one --hostname (repeatable); create's response carries the secret key — print it ONCE with an explicit "store it now, this is not shown again" warning (never repeat it in get/list output; rotate via the dashboard note in help).

## Delete contract (both)

`destructiveDryRun(cliPathOfCmdOr(cmd, "page-shield policy delete"), force)` / `("turnstile widget delete")` + `ux.Confirm` (pattern: cmd/dns.go — registry flagging LATER activates dry-by-default automatically). `--force` skips the prompt.

## Conventions

Rich Long help with examples, --json via outPayload/outResult/outErr (pattern: cmd/tunnel.go), agent-readable errors, missing-creds guards, zone/account IDs are IDs (no prefix). Tests: httptest stubs (registrar_test.go pattern; stateful-stub lesson: re-fetch-after-write must return the WRITTEN state), runner guards, confirm/dry-run behavior, JSON envelopes, secret-key shown-once assertion. No network.

## Verify

`go build ./... && go vet ./cmd/ ./pkg/cosmoflare/ && go test ./cmd/ ./pkg/cosmoflare/ -count=1 -timeout 420s` green.

## Commits

Two: `feat(page-shield): connections, scripts, policy management (FEAT-037)` then `feat(turnstile): widget management (FEAT-037)` — conventional, no AI attribution.
