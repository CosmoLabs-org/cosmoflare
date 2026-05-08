# Project Brief: CosmoDev-R2Go2

**Purpose**: Claude-consumable project context (load instead of re-auditing)

## Tech Stack
- **Language**: Go 1.25.3
- **CLI**: Cobra 1.10 + Viper 1.21
- **TUI**: Bubbletea 1.3 + Lipgloss 1.1
- **Cloud SDK**: aws-sdk-go-v2 (S3 compatible)
- **Testing**: testify (assert + require + suite)
- **Config**: YAML via Viper, stored at ~/.r2go2/config.yaml

## Architecture
- `cmd/` — 16 Cobra commands (bucket, object, config, auth, setup, dashboard, theme, etc.)
- `internal/api/` — Client (placeholder stubs) + EnhancedClient (real S3 SDK uploads)
- `internal/config/` — Multi-profile ConfigManager with Viper
- `internal/tui/` — Bubbletea dashboard (7 sections, 4 are stubs)
- `internal/interactive/` — 18 files: setup wizard, themes, animations, accessibility
- `internal/cli/` — batch manager, progress bars, visual effects, retry logic
- `cmd_disabled/`, `internal/*_disabled/` — 5 disabled packages (analytics, cicd, domain, migrate, api_disabled)

## Key Patterns
- Commands use `RunE` (newer) or `Run` with `printErrorAndExit` (legacy)
- API client is concrete struct with no interface (TASK-002 pending)
- S3API interface exists only for EnhancedClient uploads
- InputReader interface for testable interactive input
- Three parallel error classification systems (interactive, cli/ux, cmd)
- Three parallel theme systems (tui/lipgloss, interactive/fatih-color, ANSI)
- Global mutable state in cmd/ package (30+ package-level vars)

## Known Critical Issues
1. **All 9 API methods are stubs** — no real R2 operations (BUG-007, ROAD-000)
2. **Speed calc produces Inf/NaN** — time.Since(time.Now()) (BUG-001)
3. **PersistentPreRun blocks setup** — requires API token for all commands (BUG-002)
4. **printErrorAndExit doesn't exit** — nil pointer panics follow
5. **Keyboard off-by-one** — number shortcuts select wrong item (BUG-004)
6. **README documents non-existent features** — 30+ disabled/aspirational

## Strengths
- TUI architecture and visual polish (score: 72/100)
- Interactive onboarding (setup wizard, tutorials, first-run detection)
- Distribution infrastructure (Dockerfile, 7-platform Makefile, 3 CI workflows)
- Config management (multi-profile, validation, 0600 permissions)
- 100/100 commit quality

## Entry Points
- `main.go` → `cmd.Execute()` → `rootCmd`
- `cmd/dashboard.go` → `tui.Run()` (TUI entry)
- `internal/api/client.go:64` — `NewClient()` (API entry)
- `internal/config/config.go:47` — `NewConfigManager()` (config entry)

## Audit Score: 46.5/100 (D+)
Three CRITICAL areas: Core Logic (22), API Design (35), Competitive (34).
Three areas above passing: TUI/UX (63), Distribution (64), Infrastructure (65).
