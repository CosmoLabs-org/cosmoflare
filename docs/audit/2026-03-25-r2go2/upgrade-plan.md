# R2Go2 Upgrade Plan

## Phase 0: Critical Fixes (Must-Do Before Any Release)

**Effort**: ~2 days | **Impact**: Product goes from non-functional to usable

1. **Wire up real S3 API client** (ROAD-000)
   - Replace all 9 placeholder methods in `internal/api/client.go`
   - Initialize `s3.Client` properly in `NewClient()` using AWS SDK v2
   - Use `accountID` to construct R2 endpoint: `https://{accountID}.r2.cloudflarestorage.com`
   - Test against real R2 credentials

2. **Fix speed calculation** (BUG-001)
   - `enhanced_client.go:198` — capture `startTime` before upload, use in speed calc
   - `enhanced_client.go:277` — same fix for multipart upload

3. **Add LICENSE file**
   - Create `LICENSE` with MIT license text
   - Verify copyright holder: "CosmoLabs"

4. **Fix PersistentPreRun** (BUG-002)
   - Move `validateEnvironment()` to individual command `PreRun` functions
   - Or check if current command needs API access before validating
   - Commands that should work without credentials: `help`, `version`, `completion`, `setup`, `config`, `demo`

## Phase 1: Foundation (Clean Build, Stable CI)

**Effort**: ~2 days | **Impact**: Clean builds, reliable CI, no contributor confusion

5. **Remove disabled packages**
   - Delete `cmd_disabled/`, `internal/analytics_disabled/`, `internal/api_disabled/`, `internal/domain_disabled/`, `internal/migration_disabled/`
   - Fix or delete `internal/migration/s3.go` (references undefined functions from old cmd package)
   - Use git tags to preserve history reference if needed

6. **Remove tracked binaries from git**
   - `git rm simple-setup test-setup`
   - Add `simple-setup` and `test-setup` to `.gitignore`
   - Verify `CosmoDev-R2Go2` and `r2go2-enhanced` are already covered by gitignore patterns

7. **Fix CI Go version matrix**
   - Remove Go 1.26 from `test-suite.yml` (doesn't exist)
   - Standardize on Go 1.25.x across all workflows
   - Update `build.yml` test command to exclude `_disabled` and `migration` packages

8. **Add golangci-lint to CI**
   - Add `.golangci.yml` config
   - Add lint step to `build.yml`

9. **Update deprecated GitHub Actions**
   - Replace `actions/create-release@v1` with `softprops/action-gh-release@v2`
   - Verify `actions/checkout@v5` is correct (latest is v4 as of 2025)

## Phase 2: Quality (Testing, Security, Patterns)

**Effort**: ~1 week | **Impact**: Production-readiness, security hardening

10. **Config encryption**
    - Integrate macOS Keychain via `go-keyring` for API tokens
    - Fallback to encrypted file with user passphrase on Linux
    - Migration path from plaintext config

11. **Proper token validation**
    - Validate Cloudflare API token format (40-char alphanumeric)
    - Test token permissions via `/user/tokens/verify` API call
    - Cache validation result with TTL

12. **Retry logic** (ROAD-014)
    - Wire up existing `ux.RetryStrategy` to API client operations
    - Add exponential backoff with jitter for transient failures
    - Classify retryable vs non-retryable errors

13. **Shell completions** (ROAD-012)
    - `cmd/completion.go` already exists — verify it works
    - Add install instructions to README

14. **Cmd-level integration tests** (ROAD-017)
    - Test each command with mock API client
    - Verify JSON output format
    - Test flag parsing and validation

## Phase 3: Growth (Features, Competitive Parity)

**Effort**: ~2-4 weeks | **Impact**: Feature completeness, market competitiveness

15. **Complete object operations** (ROAD-010)
    - copy, head, delete-version, range downloads

16. **Multipart uploads** (ROAD-013)
    - Upload infrastructure exists in `enhanced_client.go`
    - Need to wire to real S3 API and add resume capability

17. **rsync-like sync** (ROAD-003)
    - Directory synchronization with R2 buckets
    - Differential upload based on size/mtime/checksum

18. **Competitive parity with wrangler r2**
    - `wrangler r2 object list/put/get/delete` equivalents working
    - Advantages: TUI dashboard, multi-profile, visual progress
