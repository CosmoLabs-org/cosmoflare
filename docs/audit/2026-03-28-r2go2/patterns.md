# Codebase Patterns

## Error Handling
- **Wrapping**: Consistent `fmt.Errorf("context: %w", err)` throughout (see agent-1-code-quality.md)
- **Classification**: Two parallel systems — interactive/errors.go (8 types) and cli/ux/retry.go (10 types) — not connected (see agent-7-api-design.md)
- **Critical bug**: `printErrorAndExit()` at cmd/list.go:130 does NOT exit (see agent-1-code-quality.md)
- **PersistentPreRun**: Uses `os.Exit(1)` directly, bypassing Cobra error chain (see agent-7-api-design.md)
- **JSON mode suppression**: `printError` returns silently when JSONOutput=true (see agent-1-code-quality.md)

## State Management
- **Global mutable state**: 30+ package-level vars in cmd/ (AccountID, APIToken, DryRun, JSONOutput, etc.) (see agent-1-code-quality.md, agent-7-api-design.md)
- **TUI state**: Proper Elm architecture in internal/tui/ — immutable model updated via messages (see agent-3-tui-ux.md)
- **Config state**: ConfigManager with load/save lifecycle, Viper-backed (see agent-8-infrastructure.md)

## Naming Conventions
- Go idioms followed: camelCase for unexported, PascalCase for exported
- Copyright headers on every file
- **Inconsistency**: Binary name R2Go2 (Makefile) vs r2go2 (cobra) (see agent-6-seo-content.md)
- **Inconsistency**: Module path CosmoDev-R2Go2 vs repo name r2go2 (see agent-6-seo-content.md)

## File Organization
- Standard Go layout: cmd/ + internal/ + tests/ + docs/
- **Anomaly**: Disabled packages use `_disabled` suffix instead of build tags (see agent-1-code-quality.md)
- **Anomaly**: Duplicate Bucket type in api/client.go:22 and tui/model.go:55 (see agent-1-code-quality.md)
- **Anomaly**: Three separate test directories — internal/*_test.go, tests/unit/, tests/integration/ (see agent-1-code-quality.md)
