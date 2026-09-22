# FEAT-018 residue — `cosmoflare waf ratelimit` only

Repo: /Users/gabstudio/PROJECTS/cosmoflare. The d1 parity half of this lane already landed on master — THIS dispatch is the WAF rate-limit half ONLY.

## Files you own

- `cmd/waf_ratelimit.go` — NEW (own init() registering onto the waf group — READ cmd/waf.go first for the group var name; do NOT edit waf.go)
- `cmd/waf_ratelimit_test.go` — NEW

## Command contract

`cosmoflare waf ratelimit ZONE_ID PATH --requests N --period SECONDS [--action block|challenge] [--dry-run]`

- Creates a WAF rate-limiting rule for PATH (URL path match). NOTE (FEAT-011 finding): rate limiting is governed by the Zone **WAF Edit** permission, not a legacy Rate Limiting permission — use the modern phase-based entry-point ruleset rule: phase `http_ratelimit`, expression `starts_with(http.request.uri.path, "PATH")`, action per flag (default `challenge` when omitted), ratelimit characteristics: requests N / period SECONDS.
- Service: check what pkg/cosmoflare offers for rulesets/waf (grep for Ruleset/WAF services); use the cloudflare-go typed resources like worker_cron.go does. If no service exists, add `pkg/cosmoflare/waf_ratelimit.go` + `pkg/cosmoflare/waf_ratelimit_test.go` to your owned files and build a minimal `WAFRatelimitService` (NewWAFRatelimitServiceFromCreds pattern).
- After creation print the threshold summary and a one-line "tune with --requests/--period; rule ID X".
- DryRun: print the rule that WOULD be created (full JSON) and exit 0. Zone ID is an ID — no prefix.
- Conventions: rich Long help with examples, --json via outPayload/outResult/outErr, agent-readable errors, missing-creds guard.

## Tests

- Pure logic first: expression + characteristics builder as a function returning the rule struct (assert expression string, action, N/period, default action)
- Runner guards: missing creds, DryRun short-circuit, JSON envelope shape
- Service tests if you add one: httptest stub (pattern: registrar_test.go registrarStubServer)
- No network in tests.

## Verify

`go build ./... && go vet ./cmd/ && go test ./cmd/ ./pkg/cosmoflare/ -count=1 -timeout 420s` green. NOTE: the wave-4 registry mirror test covers the `waf` group — if `waf ratelimit` is a NEW live leaf, add its registry entry to `internal/cmdmanifest/data.go` (zone-scoped, `Zone WAF Edit` permission per the Qwen dataset — grep docs/research/2026-09-17-cf-perms-next-waves-tiers/qwen-results.md for the exact name) and add "waf" to the group lists in cmd/manifest_tree_test.go if not present.

## Commit

`feat(waf): rate-limit rule creation from the terminal (FEAT-018)` — conventional, no AI attribution.
