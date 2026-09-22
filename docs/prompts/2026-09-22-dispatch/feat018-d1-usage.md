# FEAT-018 P0 — `cosmoflare d1 usage` + `d1 status`: usage-vs-limits visibility

Repo: /Users/gabstudio/PROJECTS/cosmoflare. OUTAGE-PREVENTION command: since 2026-09-01 Cloudflare ENFORCES free-tier D1 daily limits by failing queries; there is no terminal way to see today's rows read/written vs budget.

## Files you own

- `pkg/cosmoflare/d1_usage.go` — NEW service
- `pkg/cosmoflare/d1_usage_test.go` — NEW
- `cmd/d1_usage.go` — NEW (registers `d1 usage` + `d1 status` onto `d1Cmd` via its OWN init(); do NOT edit cmd/d1.go)
- `cmd/d1_usage_run_test.go` — NEW

## Service (pkg/cosmoflare/d1_usage.go)

D1 daily usage comes from Cloudflare's GraphQL Analytics API endpoint (`POST https://api.cloudflare.com/client/v4/graphql`), node `viewer.accounts(filter:{accountTag:$account}).d1AnalyticsAdaptiveGroups` — dimensions: date, databaseId; measures: rowsRead, rowsWritten (sum). Query the last N days, one node call. If the exact field names differ, adjust to what compiles against cloudflare-go v0.116.0 — the SDK may expose a GraphQL helper; if not, use the client's raw-REST seam the way pkg/cosmoflare/d1.go's Raw calls work (check how registrarRaw uses s.cf.Raw and mirror that auth/transport). DB size: `GET /accounts/{account_id}/d1/database/{database_id}` returns file_size (D1Service.Get already exists — reuse it).

Types: `D1DailyUsage{Date string; RowsRead, RowsWritten int64}`, `D1UsageReport{Days []D1DailyUsage; TotalRowsRead, TotalRowsWritten int64; FileSizeBytes int64; NumTables int}`.

## Commands

- `cosmoflare d1 usage DATABASE_ID --days 7 [--json]` — per-day read/written table + totals (human table via tabwriter like kv.go; JSON envelope via outResult).
- `cosmoflare d1 status DATABASE_ID` — one-screen health: today's usage, DB size, table count. FREE-TIER BUDGETS are plan-dependent — do NOT hardcode thresholds; display usage and, when `--budget-read/--budget-written` flags are given (or D1_USAGE_BUDGET_READ/WRITTEN env vars), print a percent-used bar and exit NON-ZERO when ≥90% (warning threshold, agent-visible failure). Document both flags in --help.

Conventions: rich Long help with examples, --json everywhere, outPayload/outResult/outErr, agent-readable errors. Database ID is an ID — no prefix (TASK-011 rule). MISSING CREDS guard like other runners.

## Tests

Service: httptest stub server returning a fixed GraphQL envelope (pattern: pkg/cosmoflare tests use `cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))` — see registrar_test.go's registrarStubServer); assert date filtering, sum parsing, malformed-response error. Runners: missing-creds guards, JSON envelope shape, budget-threshold exit decision as a pure function (table-test it). No network in tests.

## Verify

`go build ./... && go vet ./cmd/ ./pkg/cosmoflare/ && go test ./cmd/ ./pkg/cosmoflare/ -count=1 -timeout 420s` green.

## Commit

`feat(d1): usage-vs-budget visibility — daily analytics, status, warning threshold (FEAT-018)` — conventional, no AI attribution.
