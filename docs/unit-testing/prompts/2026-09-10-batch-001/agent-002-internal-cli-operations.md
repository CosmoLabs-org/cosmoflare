# Unit Test Generation — internal/cli/operations (risk tier: medium)

You are a Go test engineer. Your task is to write unit tests for the package `internal/cli/operations`.

## Functions to Test

- `DefaultCopyOptions` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/cli/operations/copy.go` (line 37) (exported)
- `NewCopyOperation` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/cli/operations/copy.go` (line 77) (exported)
- `NewBatchCopy` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/cli/operations/copy.go` (line 337) (exported)
- `isRetryableError` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/cli/operations/copy.go` (line 404)
- `contains` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/cli/operations/copy.go` (line 428)
- `findSubstring` in `/Users/gabstudio/PROJECTS/cosmoflare/internal/cli/operations/copy.go` (line 436)

## Test Style Requirements

- Use t.Run subtests for all functions with multiple cases.
- Each test case must have a descriptive name.
- Add GoDoc comments above each test function explaining what it validates.
- Test files follow the `*_test.go` convention.
- Prohibit: testing unexported internals via reflection, mocking without understanding dependencies.

## Risk-Tier Instructions (medium)

This is a MEDIUM-risk package. Focus on algorithm correctness and edge cases.

- Algorithm correctness: verify outputs match expected results for representative inputs.
- Empty/nil/zero-value inputs must not panic or raise unhandled exceptions.
- Test at least one error path per exported function.

## Verification

After writing tests, verify they compile and pass:

```
go test ./... -v -count=1
```

Ensure `go test` exits 0 before submitting.
