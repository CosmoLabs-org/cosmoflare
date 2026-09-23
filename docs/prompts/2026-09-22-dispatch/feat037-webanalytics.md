# FEAT-037 wave 3 — Web Analytics

Repo: /Users/gabstudio/PROJECTS/cosmoflare. NO registry edits — a follow-up pass owns internal/cmdmanifest/data.go and cmd/manifest_tree_test.go (do NOT touch them; unregistered this round is expected).

## Files you own

- `pkg/cosmoflare/webanalytics.go` + `pkg/cosmoflare/webanalytics_test.go` — NEW
- `cmd/webanalytics.go` + `cmd/webanalytics_run_test.go` — NEW

## Surface (account-scoped; Cloudflare RUM API)

- `GET /accounts/{account_id}/rum/site_info` — list sites
- `POST /accounts/{account_id}/rum/site_info` — create site (returns the JS snippet token ONCE)
- `DELETE /accounts/{account_id}/rum/site_info/{site_id}` — delete site
- Permission: NOT in the Qwen dataset → registry pass leaves it sparse; note `// Web Analytics permission pending FEAT-011 dataset` in the service doc.
- cloudflare-go may lack RUM resources — if absent, hand-roll with the client's Raw seam (pattern: registrarRaw-style raw helpers in pkg/cosmoflare).

## Commands

`cosmoflare web-analytics site list|create|delete` — create returns the site token + JS snippet with an explicit "paste this into your <head>" instruction; the token is the credential for the beacon (never repeat in list output). Delete: `destructiveDryRun(cliPathOfCmdOr(cmd, "web-analytics site delete"), force)` + `ux.Confirm` (pattern: cmd/dns.go — registry flagging LATER activates dry-by-default). `--force` skips the prompt.

Optional stretch (only if the SDK/stub makes it trivial): `web-analytics site stats <site-id> --days 7` from the RUM analytics endpoint — SKIP if it requires endpoint guessing; sparse is correct.

## Conventions

Rich Long help with examples, --json via outPayload/outResult/outErr (pattern: cmd/tunnel.go), agent-readable errors, missing-creds guards, account ID = ID (no prefix). Tests: httptest stubs (registrar_test.go pattern; stateful-stub lesson from logpush: re-fetch-after-write must return the WRITTEN state), runner guards, confirm/dry-run behavior, token shown-once assertion, JSON envelopes. No network.

## Verify

`go build ./... && go vet ./cmd/ ./pkg/cosmoflare/ && go test ./cmd/ ./pkg/cosmoflare/ -count=1 -timeout 420s` green.

## Commit

`feat(web-analytics): RUM site management (FEAT-037)` — conventional, no AI attribution.
