# Entry Points and Execution Paths

## Primary Entry Points
- **main.go:19** — `cmd.Execute()` → Cobra root command
- **cmd/root.go:94** — `Execute()` → `rootCmd.Execute()`
- **cmd/dashboard.go** — `tui.Run()` → Bubbletea program

## Request Lifecycle: CLI Command
1. User runs `r2go2 <command> [flags]`
2. Cobra parses flags, binds to package-level vars
3. `PersistentPreRun` fires: validates env vars, sets AccountID (root.go:63)
4. Command `RunE` function executes
5. `getAPIClient()` creates `*api.Client` from env/config (object.go:775)
6. Client method called → returns placeholder data
7. Output formatted (JSON/table/csv) and printed to stdout

## Request Lifecycle: TUI Dashboard
1. `r2go2 dashboard` → `runDashboard()`
2. `tea.NewProgram()` with alt-screen + mouse support
3. `Init()` → `loadDataCmd()` → creates Client → `ListBuckets()` → empty
4. `Update()` loop processes key messages, timer ticks
5. `View()` renders current section via lipgloss

## Background Processes
- **Real-time stats ticker**: 5-second interval in TUI (update.go:50) — simulated data
- **Batch worker goroutines**: Pool managed by BatchManager (batch/manager.go)
- **Upload progress goroutine**: visual.ShowSimpleProgress() in background (enhanced_client.go:396)

## Key Construction Points
- `api.NewClient(opts)` — client.go:64
- `api.NewEnhancedClient(base)` — enhanced_client.go:78
- `config.NewConfigManager()` — config.go:47
- `tui.initialModel()` — model.go:269
