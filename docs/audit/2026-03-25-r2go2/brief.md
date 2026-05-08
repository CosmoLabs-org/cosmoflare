# R2Go2 Project Brief

**For Claude session consumption. Load this instead of re-auditing.**

## Identity
- **Name**: R2Go2
- **Purpose**: CLI tool for managing Cloudflare R2 buckets
- **Version**: 0.2.2
- **Language**: Go 1.25.3
- **Module**: `github.com/CosmoLabs-org/CosmoDev-R2Go2`
- **License**: MIT (claimed, but LICENSE file missing)

## Tech Stack
- CLI: Cobra v1.10.1
- TUI: Bubbletea v1.3.10 + Lipgloss v1.1
- API: AWS SDK Go v2 (S3-compatible for R2)
- Config: Viper v1.21, YAML at ~/.r2go2/config.yaml
- Testing: testify v1.11.1

## Architecture
```
main.go -> cmd/root.go -> cmd/*.go (17 commands)
                       -> internal/api/ (S3 client — STUBS)
                       -> internal/config/ (profiles)
                       -> internal/tui/ (Bubbletea dashboard)
                       -> internal/interactive/ (setup wizards)
                       -> internal/cli/ (visual, ux, batch)
```

## Current State (Audit Score: 5.4/10 C)
- **Working**: Build, TUI dashboard, interactive setup, themes, config management, visual progress
- **Not working**: All R2 API operations (100% stubs), speed calculation (BUG-001)
- **Broken packages**: 6 `_disabled` packages fail to build
- **Test suite**: 22/28 packages pass, 37 test files covering unit/integration/security/performance/TUI
- **CI**: 3 workflows (build, test-suite, release) but test-suite references non-existent Go 1.26

## Known Issues
- BUG-001: Speed calc `time.Since(time.Now())` = 0
- BUG-002: PersistentPreRun blocks non-API commands
- BUG-003: CI Go version mismatch
- BUG-004: TUI menu keyboard shortcuts off-by-one
- BUG-005: YAML parsing not implemented
- BUG-006: Object search regex/glob use fake data

## Key Patterns
- Error wrapping with %w, custom error types in ux package
- Dual output: human-readable + --json mode
- Profile-based config with current/default
- Package-level global vars for CLI state (AccountID, APIToken)

## Top Priority (from ROAD-000, priority 95)
Wire up real S3 API client to replace placeholders in `internal/api/client.go`

## Entry Points
- `main.go` -> `cmd.Execute()`
- TUI: `r2go2 dashboard`
- Setup: `r2go2 setup`
- Auth: `r2go2 auth login`
