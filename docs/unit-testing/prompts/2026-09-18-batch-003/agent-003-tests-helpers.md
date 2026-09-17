# Unit Test Generation — tests/helpers (risk tier: medium)

You are a Go test engineer. Your task is to write unit tests for the package `tests/helpers`.

## Functions to Test

- `MockCloudflareServer` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/mocks.go` (line 17) (exported)
- `NewMockS3Client` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/mocks.go` (line 106) (exported)
- `NewMockTerminal` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/mocks.go` (line 228) (exported)
- `SetupTest` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/test_helpers.go` (line 23) (exported)
- `CreateTestFile` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/test_helpers.go` (line 58) (exported)
- `CaptureOutput` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/test_helpers.go` (line 66) (exported)
- `AssertContains` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/test_helpers.go` (line 100) (exported)
- `AssertNotContains` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/test_helpers.go` (line 106) (exported)
- `WaitFor` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/test_helpers.go` (line 112) (exported)
- `MockConfig` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/test_helpers.go` (line 127) (exported)
- `WithTimeout` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/test_helpers.go` (line 156) (exported)
- `FileExists` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/test_helpers.go` (line 174) (exported)
- `DirExists` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/test_helpers.go` (line 180) (exported)
- `CountFiles` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/test_helpers.go` (line 189) (exported)
- `RandomString` in `/Users/gabstudio/PROJECTS/cosmoflare/tests/helpers/test_helpers.go` (line 204) (exported)

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
