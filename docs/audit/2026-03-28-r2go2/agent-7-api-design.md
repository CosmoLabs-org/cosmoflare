# API Design Audit: CosmoDev-R2Go2

## 1. Contract Quality (Score: 30/100)

**Interfaces found:**
- `S3API` in `internal/api/enhanced_client.go:63` -- 5 methods, focused on S3 multipart upload operations. This is the only interface in the API layer.
- No interface exists for the base `Client` struct itself.

**Critical problems:**

The base `Client` struct (`internal/api/client.go:46`) is a concrete type with no interface. Every method on it (`CreateBucket`, `ListBuckets`, `GetBucket`, `ListObjects`, `DeleteBucket`, `BucketExists`, `GetObject`, `DeleteObject`, `HeadObject`) is a placeholder returning mock data. Every single one has a `// Placeholder implementation` comment. This is BUG-007 confirmed at the source level.

The `ClientOptions` struct is the only contract for client creation, and it mixes concerns: `Profile`, `AccountID`, `APIToken`, `HTTPClient`, and `S3Client` are all optional with no clear documentation of what combinations are valid. `NewClient` accepts nil for everything except the options struct itself.

The response types (`Bucket`, `Object`) are concrete structs with JSON tags, which is acceptable, but they mix concerns -- `Bucket` has both API response fields (`Name`, `Created`) and derived/display fields (`Size`, `ObjectCount`, `Tags`) with no separation of API response from domain model.

There is no API response envelope type. The `OutputResponse` in `cmd/root.go:179` is a CLI-level output wrapper, not an API contract.

**Speed calculation bug:** Lines 198 and 277 in `enhanced_client.go` calculate speed as `float64(fileSize) / time.Since(time.Now()).Seconds()` which divides by essentially zero (time.Now minus time.Now), producing `+Inf` or `NaN`. This is a correctness defect in the only non-placeholder API code.

## 2. Interface Design (Score: 35/100)

**Three interfaces exist in the entire codebase:**

1. **`S3API`** (`enhanced_client.go:63`) -- 5 methods. Well-scoped to the S3 upload operations the `EnhancedClient` actually uses. This is the best interface in the project. It has a proper mock (`mockS3Client` in `enhanced_client_test.go:18`) with function-field injection for per-test customization. This is textbook Go test design.

2. **`InputReader`** (`interactive/input.go:12`) -- 1 method (`ReadLine`). Excellent: minimal, focused, with a production `stdinReader` and test-injectable design. All interactive functions (`PromptWithReader`, `SelectFromListWithReader`, `ConfirmWithReader`) accept this interface.

3. **`ProgressBar`** (`cli/progress/progress.go:56`) -- 7 methods. Slightly large for an interface (Go convention favors 1-3 methods), but the methods are cohesive lifecycle operations.

**The critical gap (TASK-002):** There is **no interface for the base `Client`**. The `cmd/` layer constructs `*api.Client` directly via `getAPIClient()` in `cmd/object.go:775`. Every command function (`runBucketCreate`, `runBucketList`, etc.) calls `getAPIClient()` which returns a concrete `*api.Client`. This makes it impossible to:
- Mock the API client in command-level tests
- Swap the placeholder implementation for a real one without modifying call sites
- Write integration tests that don't depend on environment variables

The `EnhancedClient` embeds `*Client` directly (composition of concrete type, not interface), further coupling the layers.

**Dependency injection is absent** at the command layer. All commands use package-level variables (`AccountID`, `APIToken`, `DryRun`, `JSONOutput`, `Verbose` in `cmd/root.go:21-29`) as implicit global state.

**Mock fragmentation:** Three separate mock systems exist:
- `tests/helpers/mocks.go` -- `MockS3Client` struct implementing raw S3 SDK methods (but does not implement `S3API` interface)
- `tests/integration/api/mock/mock.go` -- `MockR2Client` with its own `MockBucket`/`MockObject` types that duplicate `api.Bucket`/`api.Object`
- `internal/api/enhanced_client_test.go` -- `mockS3Client` implementing `S3API` (the only properly interface-aligned mock)

## 3. Error Contracts (Score: 40/100)

**No custom error types exist.** All errors are created via `fmt.Errorf` with `%w` wrapping.

**Two parallel error classification systems:**

1. `internal/interactive/errors.go` -- `ErrorType` enum (8 types) + `ErrorContext` struct with troubleshooting guidance. Factory functions (`NetworkError`, `AuthError`, etc.) produce rich error contexts for user display.

2. `internal/cli/ux/retry.go` -- A completely separate `ErrorType` enum (10 types) + `ErrorInfo` struct with retry logic. `ClassifyError()` does string-matching on `err.Error()` to categorize errors. This is brittle.

These two systems have **overlapping `ErrorType` names** in different packages with different values. They are never connected.

**R2 API errors are not mapped to Go errors at all.** Since the `Client` methods are all placeholders, there is no error mapping from HTTP status codes, S3 error codes, or Cloudflare-specific errors.

**The `PersistentPreRun` in `cmd/root.go:63` calls `os.Exit(1)` directly** on validation failure, bypassing Cobra's error handling.

## 4. Extensibility (Score: 35/100)

**Adding new commands** is straightforward via Cobra's `init()` registration pattern.

**Adding new API operations** is blocked by the concrete `Client` type. Without an interface, adding a new provider requires modifying `client.go` directly.

**Configuration extensibility** is reasonable. Multi-profile support, env var fallback, YAML persistence with Viper.

**No plugin/hook points exist.** The `RetryStrategy` in `ux/retry.go` is configurable but not wired into the API client.

**YAML parsing is not implemented** (`bucket.go:679` returns error for YAML despite claiming support).

## Summary of Files Read (14 total)

1. `internal/api/client.go` -- Base API client, all placeholder methods
2. `internal/api/enhanced_client.go` -- S3API interface, multipart upload
3. `internal/api/client_test.go` -- Tests against placeholder methods
4. `internal/api/enhanced_client_test.go` -- Good mock-based S3 tests
5. `internal/config/config.go` -- Profile-based config management
6. `internal/interactive/input.go` -- InputReader interface
7. `internal/interactive/errors.go` -- Error context with UX guidance
8. `internal/cli/ux/retry.go` -- Retry strategy, error classification
9. `internal/cli/progress/progress.go` -- ProgressBar interface
10. `cmd/root.go` -- Root command, global state, output helpers
11. `cmd/bucket.go` -- Bucket commands
12. `cmd/object.go` -- Object commands, getAPIClient()
13. `tests/helpers/mocks.go` -- MockS3Client, MockTerminal
14. `tests/integration/api/mock/mock.go` -- MockR2Client
