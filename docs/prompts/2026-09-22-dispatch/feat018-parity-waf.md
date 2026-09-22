# FEAT-018 P1 — `cosmoflare d1 parity` + `cosmoflare waf ratelimit`

Repo: /Users/gabstudio/PROJECTS/cosmoflare. Two P1 items from the MyCarGuide friction report, both fully specified by the issue. Work them SEQUENTIALLY in this one dispatch.

## Files you own

- `cmd/d1_parity.go` — NEW (own init() onto d1Cmd; do NOT edit cmd/d1.go)
- `cmd/d1_parity_test.go` — NEW
- `cmd/waf_ratelimit.go` — NEW (own init() onto the waf group — READ cmd/waf.go first for the group var name; do NOT edit waf.go)
- `cmd/waf_ratelimit_test.go` — NEW

## 1. `cosmoflare d1 parity DATABASE_ID --local PATH --tables a,b,c`

- Local side: open the sqlite file at --local with the same driver d1_local.go uses (`modernc.org/sqlite`), run `SELECT COUNT(*)` per table.
- Remote side: `getD1Service()` query path (see runD1Query in cmd/d1.go), `SELECT COUNT(*)` per table.
- Output: table (human) / JSON envelope of {table, local, remote, match} rows (match = exact equality). NON-ZERO exit when any table mismatches (agent-verifiable), unless --no-fail.
- Both sides unreachable-safe: a per-table error is a row with an error field, not a crash.

## 2. `cosmoflare waf ratelimit ZONE_ID PATH --requests N --period SECONDS [--action block|challenge] [--dry-run]`

- Creates a WAF rate-limiting rule for PATH (URL path match). NOTE (FEAT-011 finding): rate limiting is governed by the Zone **WAF Edit** permission, not a legacy Rate Limiting permission — use the WAF/ratelimit service per pkg/cosmoflare (check what exists; cloudflare-go has RateLimit / Ruleset resources — prefer the modern phase-based entry-point ruleset rule: phase `http_ratelimit`, expression `starts_with(http.request.uri.path, "PATH")`, action per flag, ratelimit characteristics: requests N / period SECONDS).
- Guidance output: after creation print the threshold summary and a one-line "tune with --requests/--period; rule ID X".
- DryRun: print the rule that WOULD be created (full JSON) and exit 0.
- Default action: challenge when flag omitted. Zone ID is an ID — no prefix.

## Tests

Pure logic first: parity row building + match/exit decision as functions (table-test); ratelimit expression + characteristics builder as a pure function returning the rule struct (assert expression string, action, N/period). Runner guards: missing creds, missing --local file, DryRun short-circuit, JSON envelope shapes. No network.

## Verify

`go build ./... && go vet ./cmd/ && go test ./cmd/ -count=1 -timeout 420s` green.

## Commit

Two commits, one per feature: `feat(d1): local-vs-remote parity check (FEAT-018)` then `feat(waf): rate-limit rule creation from the terminal (FEAT-018)` — conventional, no AI attribution.
