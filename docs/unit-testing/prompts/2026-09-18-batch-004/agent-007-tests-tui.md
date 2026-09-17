# Unit Test Generation — tests/tui (risk tier: low)

You are a Go test engineer. Your task is to write unit tests for the package `tests/tui`.

## Functions to Test

- `NewTestModel` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/tui/test_harness.go` (line 26) (exported)
- `BenchmarkNavigation` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/tui/test_harness.go` (line 74) (exported)
- `BenchmarkUpdates` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/tui/test_harness.go` (line 91) (exported)
- `AssertAccessibleNavigation` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/tui/test_harness.go` (line 115) (exported)
- `AssertKeyboardOnly` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/tui/test_harness.go` (line 134) (exported)
- `MeasureMemoryUsage` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/tui/test_harness.go` (line 182) (exported)
- `CloneModel` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/tui/test_harness.go` (line 192) (exported)
- `AssertModelState` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/tui/test_harness.go` (line 199) (exported)
- `RunFullWorkflow` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/tui/test_harness.go` (line 218) (exported)
- `RunConcurrentUpdates` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/tui/test_harness.go` (line 277) (exported)

## Test Style Requirements

- Use t.Run subtests for all functions with multiple cases.
- Each test case must have a descriptive name.
- Add GoDoc comments above each test function explaining what it validates.
- Test files follow the `*_test.go` convention.
- Prohibit: testing unexported internals via reflection, mocking without understanding dependencies.

## Risk-Tier Instructions (low)

Focus on happy-path coverage and basic error handling.

## Verification

After writing tests, verify they compile and pass:

```
go test ./... -v -count=1
```

Ensure `go test` exits 0 before submitting.
