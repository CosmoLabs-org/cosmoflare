# FEAT-037 wave 3b — CLI wiring for three landed services

Repo: /Users/gabstudio/PROJECTS/cosmoflare. The SERVICES for Page Shield, Turnstile, and Web Analytics are merged in pkg/cosmoflare (pageshield.go, turnstile.go, webanalytics.go — READ all three first for their exact method signatures and options). Their CLI command files were never written. This lane wires them.

## Files you own

- `cmd/pageshield.go` + `cmd/pageshield_run_test.go` — NEW
- `cmd/turnstile.go` + `cmd/turnstile_run_test.go` — NEW
- `cmd/webanalytics.go` + `cmd/webanalytics_run_test.go` — NEW

Template: cmd/spectrum.go (same wave, already merged) — copy its structure exactly: group command, subcommands with Use lines, init() registration, flag vars, runners calling the service, outPayload/outResult/outErr, missing-creds guard.

## Command surfaces

`cosmoflare page-shield` (zone-scoped): `connections ZONE_ID`, `scripts ZONE_ID`, `policy create|list|get|update|delete` (policy create/update need the fields the service exposes — derive flags from the pkg option types).
`cosmoflare turnstile widget` (account-scoped): `create|list|get|update|delete` — create takes --name + repeatable --hostname; the create response carries the secret key: print it ONCE with a "store it now" warning; get/list/update must never repeat it (the service already handles this — assert in tests).
`cosmoflare web-analytics site` (account-scoped): `list`, `create` (token + JS snippet printed once with a paste-into-head instruction), `delete SITE_ID`.

## Delete contract (all three deletes)

`destructiveDryRun(cliPathOfCmdOr(cmd, "page-shield policy delete"), force)` (+ `ux.Confirm`, pattern: cmd/dns.go runDNSDelete), `--force` skips the prompt. Same for turnstile widget delete and web-analytics site delete.

## Tests

Follow cmd/spectrum_run_test.go patterns: missing-creds guards per runner, confirm/dry-run delete behavior, JSON envelope shapes, secret/token shown-once assertions (capture output, assert it appears exactly in create and never in list/get). No network. NOTE stateful-stub lesson if you stub re-fetches: serve the WRITTEN state.

## Verify

`go build ./... && go vet ./cmd/ && go test ./cmd/ -count=1 -timeout 420s` green.

## Commits

Three (one per service): `feat(page-shield): CLI — connections, scripts, policies (FEAT-037)`, `feat(turnstile): widget CLI (FEAT-037)`, `feat(web-analytics): site CLI (FEAT-037)` — conventional, no AI attribution.
