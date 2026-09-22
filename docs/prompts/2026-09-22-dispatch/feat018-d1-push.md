# FEAT-018 P0 — `cosmoflare d1 push-sql`: batched remote SQL push

Repo: /Users/gabstudio/PROJECTS/cosmoflare. Production evidence: MyCarGuide pushed an 84,437-statement seed by hand — the batch-push pattern was re-implemented per-project twice. wrangler cannot handle it.

## Files you own

- `cmd/d1_push.go` — NEW (registers `d1 push-sql` onto `d1Cmd` via its OWN init(); do NOT edit cmd/d1.go)
- `cmd/d1_push_test.go` — NEW

## Command contract (from the issue, verbatim requirements)

`cosmoflare d1 push-sql FILE --database ID [--concurrency N] [--batch-bytes N]`

- Statement-aware SQL splitting: REUSE `splitSQLStatements` in cmd/d1_local.go (same package — quote/comment/semicolon-aware, already tested). Never a naive ';' split.
- Byte-capped batches (default ~40KB): accumulate whole statements until the cap; a single statement larger than the cap becomes its own batch.
- Multi-value `INSERT OR IGNORE` is the CALLER's SQL concern, not ours — we batch whatever statements the file holds.
- Retry: on `internal error code: 7500` (and transient 5xx/network), retry with backoff (3 attempts). On SQLITE_TOOBIG-style failures, split the batch in half and retry each half recursively (depth cap 4).
- Progress lines to stderr (human mode): "batch 12/47 ok (312 statements)"; `--json` prints one final envelope via outPayload/outResult. Agent-friendly terse errors per CLAUDE.md.
- Kill-safe resume: batches are idempotent because callers use INSERT OR IGNORE; add `--from-batch N` to skip ahead after an interrupted run, and print "resume with --from-batch N" on failure.
- Execution path: reuse the remote D1 query path — `getD1Service()` + the query method used by `d1 query` (check cmd/d1.go runD1Query for the exact service call). Concurrency N default 2, cap 8.
- Profile prefix: none — database is an ID, do NOT prefix (TASK-011 rule).

## Tests (cmd/d1_push_test.go, follow d1_local_run_test.go patterns)

- splitter grouping: statements pack to the byte cap; oversized statement becomes a lone batch
- batch splitting on failure: a batch that fails with a TOOBIG-looking error splits and both halves execute
- retry: 7500 error then success → one retry, final success
- resume: --from-batch skips earlier batches (count executed batches)
- --json envelope shape; missing-creds guard; missing-file error

Use a stubbed query seam if the service call allows injection; otherwise table-test the pure grouping/splitting/resume-decision helpers and guard-test the runner. No network in tests.

## Verify

`go build ./... && go vet ./cmd/ && go test ./cmd/ -count=1 -timeout 420s` green.

## Commit

`feat(d1): batched remote SQL push with retry, split-recovery, resume (FEAT-018)` — conventional, no AI attribution.
