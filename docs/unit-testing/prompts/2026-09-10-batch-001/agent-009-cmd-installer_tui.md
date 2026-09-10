# Unit Test Generation — cmd/installer_tui (risk tier: low)

You are a Go test engineer. Your task is to write unit tests for the package `cmd/installer_tui`.

## Functions to Test

- `NewInstallerModel` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/installer_tui/main.go` (line 95) (exported)
- `createInstallerStyles` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/installer_tui/main.go` (line 110)
- `addToPath` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/installer_tui/main.go` (line 466)
- `saveInstallLocation` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/installer_tui/main.go` (line 533)
- `copyFile` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/installer_tui/main.go` (line 561)
- `createBrandedSymlinks` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/installer_tui/main.go` (line 571)
- `createMenuModel` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/installer_tui/main.go` (line 731)
- `createPostInstallMenu` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/installer_tui/main.go` (line 774)
- `maskToken` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/installer_tui/main.go` (line 1040)
- `main` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/installer_tui/main.go` (line 1285)

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
