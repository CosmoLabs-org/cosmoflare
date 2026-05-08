# R2Go2 Codebase Patterns

## Error Handling

- **Wrapped errors**: Consistent use of `fmt.Errorf("context: %w", err)` throughout
- **Custom error types**: `ux.ErrorInfo` with error classification (network, auth, timeout, etc.)
- **Retry strategy**: `ux.RetryStrategy` with exponential backoff, jitter, configurable delays
- **Output helpers**: `printError()`, `printWarning()`, etc. in cmd/root.go suppress output in JSON mode
- **Gap**: No structured error codes for API responses; error messages are free-form strings

## State Management

- **Config**: Viper-based YAML config at `~/.r2go2/config.yaml`, profiles pattern with current/default
- **TUI State**: Bubbletea elm-architecture (Model -> Update -> View) in `internal/tui/`
- **Global State**: Package-level vars in `cmd/root.go` (AccountID, APIToken, DryRun, JSONOutput, Verbose)
- **Theme State**: Package-level color vars initialized by `InitializeStyles()`
- **Gap**: Global vars in root.go create tight coupling; should use context or dependency injection

## Naming Conventions

- **Packages**: Lowercase, single-word (`api`, `config`, `tui`, `utils`)
- **Files**: Snake_case for tests, camelCase or hyphenated for source (`enhanced_client.go`, `r2-progress.go`)
- **Types**: PascalCase with descriptive names (`DashboardModel`, `UploadProgressCallback`)
- **Disabled code**: `*_disabled/` directory suffix or `*.go.disabled` file suffix
- **Commands**: Verb-based Cobra commands (`create`, `list`, `delete`, `auth login`)

## File Organization

```
cmd/          - One file per command (flat)
internal/     - Feature-based packages
  api/        - Client types (basic + enhanced)
  config/     - Config management (single file)
  tui/        - MVC split (model.go, update.go, view.go)
  interactive/ - Many small files by concern (validation, themes, etc.)
  cli/        - Sub-packages by UX concern (visual, ux, batch, progress)
tests/        - Mirror structure with additional categorization (unit, integration, security, etc.)
```

## Testing Patterns

- **Test framework**: `testify/assert` and `testify/require`
- **Table-driven tests**: Used in validation and API tests
- **Mock helpers**: `tests/helpers/mocks.go` with test fixtures in `tests/fixtures/`
- **Test categories**: Unit, integration (mock + real), security, performance, TUI, accessibility
- **Test suppression**: `visual.DisableAnimations()` for cleaner test output
- **Input abstraction**: `InputReader` interface for testable interactive prompts
- **Gap**: No cmd-level tests (commands not tested end-to-end), config package has no tests

## Output Patterns

- **Dual mode**: All commands support `--json` flag via `JSONOutput` global
- **Visual feedback**: Spinners, progress bars, animations via `internal/cli/visual/`
- **Emoji-prefixed**: `printSuccess()` = checkmark, `printError()` = cross, etc.
- **Standard response**: `OutputResponse{Success, Message, Data, Error, DryRun}` JSON envelope

## Security Patterns

- **Credential masking**: `utils.MaskAccountID()`, `config.MaskKey()` for display
- **File permissions**: Config saved with 0600
- **Sanitized output**: `SanitizeForOutput()` strips tokens before display
- **Gap**: Plaintext storage, no keychain/vault integration, token validation is length-only
