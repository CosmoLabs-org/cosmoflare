# Unit Test Generation — internal/migration (risk tier: high)

You are a Go test engineer. Your task is to write unit tests for the package `internal/migration`.

## Functions to Test

- `checkpointDir` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/migration/checkpoint.go` (line 40)
- `saveCheckpoint` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/migration/checkpoint.go` (line 69)
- `deleteCheckpoint` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/migration/checkpoint.go` (line 89)
- `printMigrationSummary` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/migration/s3.go` (line 432)
- `printInfo` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/migration/s3.go` (line 454)
- `printSuccess` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/migration/s3.go` (line 458)
- `printWarning` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/migration/s3.go` (line 462)
- `FormatBytes` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/migration/s3.go` (line 467) (exported)
- `transferWorker` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/migration/worker.go` (line 46)
- `transferObject` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/migration/worker.go` (line 60)
- `doTransfer` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/migration/worker.go` (line 80)

## Test Style Requirements

- Use t.Run subtests for all functions with multiple cases.
- Each test case must have a descriptive name.
- Add GoDoc comments above each test function explaining what it validates.
- Test files follow the `*_test.go` convention.
- Prohibit: testing unexported internals via reflection, mocking without understanding dependencies.

## Risk-Tier Instructions (high)

This is a HIGH-risk package. Correctness and round-trip fidelity are paramount. You MUST include:

- **Round-trip** tests: encode then decode (or write then read) and assert identity.
- **Idempotency** tests: calling the function twice with the same input must produce the same output.
- Error propagation tests: verify errors bubble up correctly without data corruption.
- Concurrent safety: where applicable, test with parallel execution and a race detector.

## Verification

After writing tests, verify they compile and pass:

```
go test ./... -v -count=1
```

Ensure `go test` exits 0 before submitting.
