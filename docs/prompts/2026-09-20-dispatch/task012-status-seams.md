# TASK-012 — Unify the HTTP-status classification seams

Repo: /Users/gabstudio/PROJECTS/cosmoflare.

## Problem

Three disjoint failure-classification channels coexist in pkg/cosmoflare:
(a) typed constructors `authError`/`accessDenied`/`quotaError` that `internal/server/rest.go` `mapError` keys on — a decodeEnvelope 403 lands as plain R2Error, so mapError reports 502 upstream_error;
(b) the `httpStatusCarrier` interface walk in `pkg/cosmoflare/retry.go` `isHTTPStatusRetryable`;
(c) `R2Error.Status` + `ErrorStatus` (FEAT-014).

## Fix (chosen direction: the first option in the issue)

1. PIN FIRST: write tests capturing current `mapError` behavior in `internal/server/rest_test.go` for each error class (auth → 401/403 mapping, quota → 429, plain R2Error with Status, plain without) BEFORE touching anything. These tests define the contract you must preserve or deliberately improve.
2. Make `R2Error` implement `httpStatusCarrier` (`StatusCode() int` returning Status) so retry.go's interface walk picks the status up.
3. Make `ErrorStatus` (where it duplicates the walk) delegate to the shared seam.
4. Have `decodeEnvelope` attach Status to the typed errors it constructs (401/403/429 paths), so mapError classifies envelope-level auth failures correctly instead of 502 upstream_error. If this changes a pinned mapError expectation from 502→ the correct 4xx, update that pinned test AND note it in the commit body — that is the bug being fixed, not drift.

## Files

`pkg/cosmoflare/errors.go`, `pkg/cosmoflare/retry.go`, `pkg/cosmoflare/client.go` (decodeEnvelope), `internal/server/rest.go` (+ their _test.go files).

## Verify

`go test ./pkg/cosmoflare/ ./internal/server/ -count=1 -timeout 300s` green; `go build ./...`; `go vet` clean.

## Commit

`refactor(errors): R2Error carries HTTP status through one seam — mapError classifies envelope auth failures (TASK-012)` — conventional, no AI attribution.
