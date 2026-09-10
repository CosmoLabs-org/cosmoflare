# Unit Test Generation — . (risk tier: medium)

You are a Go test engineer. Your task is to write unit tests for the package `.`.

## Functions to Test

- `main` in `/Users/gabstudio/PROJECTS/cosmoflare/main.go` (line 19)

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
