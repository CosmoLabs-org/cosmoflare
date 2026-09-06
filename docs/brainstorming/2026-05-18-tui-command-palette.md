---
created: 2026-05-18T15:00:00-03:00
deliverables:
    - id: BR-01
      title: CommandPalette Bubble Tea component (internal/tui/components/palette/)
    - id: BR-02
      title: Ctrl+P integration into DashboardModel
    - id: BR-03
      title: Command registry populated from Cobra command tree
    - id: BR-04
      title: Fuzzy search with sahilm/fuzzy (already in go.mod as indirect)
    - id: BR-05
      title: Tests for palette component and command matching
issue: FEAT-001
last_review_content_hash: 5801caa8808fc5095eb85db6ddd6e194d56725f8f1b870176b766923eb2a7ccb
last_review_findings: 0
last_review_ref: docs/brainstorming/2026-05-18-tui-command-palette.md
last_reviewed: "2026-09-06T20:14:03.608268+04:00"
status: implemented
title: TUI Command Palette with Fuzzy Search
---

# TUI Command Palette with Fuzzy Search (FEAT-001)

**Goal**: Add a VS Code/lazygit-style Ctrl+P command palette to the TUI dashboard, enabling fuzzy search across all available commands with context-aware filtering.

**Date**: 2026-05-18
**Issue**: FEAT-001

---

## Current Architecture

### TUI Stack
- **Framework**: Bubble Tea v1.3.10 + Bubbles v0.21.0 + Lipgloss v1.1.0
- **Main model**: `DashboardModel` in `internal/tui/model.go`
- **Update loop**: `internal/tui/update.go` — handles key input, section navigation
- **Components**: `internal/tui/components/navigation/menu_selection.go` (MenuModel), `internal/tui/components/installer/header.go`
- **Sections**: Overview, BucketList, ObjectList, Upload, Monitoring, Settings, Help

### Key Bindings (current)
- Number keys 1-6: section switching
- Arrow/vim keys: navigation
- `/`: search (inline, within current section)
- `?`: help toggle
- Letter shortcuts: c=create, u=upload, d=delete, m=monitor, s/p=settings

### Dependencies Already Available
- `github.com/sahilm/fuzzy v0.1.1` — already in go.mod as indirect dep (transitive from bubbles)
- `github.com/charmbracelet/bubbles/list` — already imported in menu_selection.go
- `github.com/charmbracelet/bubbles/textinput` — available in bubbles, not yet used

### CLI Command Count
~132 commands across: analytics, auth, backup, bucket, cache, compare, completion, config, dns, doctor, domains, kv, object, ssl, worker, zone + subcommands.

## Design Decisions

### Q1: Trigger key?

**Decision: Ctrl+P** (primary) + `:` (secondary, vim-style).
- Ctrl+P is universal (VS Code, Sublime, lazygit)
- `:` feels natural for terminal users
- Both are currently unbound in the TUI

### Q2: What's searchable?

**Decision: Three-tier command registry.**

| Tier | Source | Example entries |
|------|--------|-----------------|
| **CLI commands** | Cobra command tree walk | "bucket create", "dns list", "kv put" |
| **Dashboard actions** | Hardcoded action list | "Go to Overview", "Go to Buckets", "Refresh data", "Toggle help" |
| **Context actions** | Current section state | "Delete bucket X" (when in bucket list with selection) |

### Q3: Fuzzy search library?

**Decision: `sahilm/fuzzy`** — already in go.mod, proven, used by bubbles/list internally. Fast enough for ~200 candidates.

### Q4: Component architecture?

**Decision: Standalone Bubble Tea sub-model in `internal/tui/components/palette/`.**

```
internal/tui/components/palette/
├── palette.go       // PaletteModel (tea.Model)
├── command.go       // Command type + registry
└── palette_test.go  // Tests
```

The palette overlays the dashboard view when active. DashboardModel owns a `*PaletteModel` field, delegates key events when palette is visible.

### Q5: How does selection trigger execution?

Each `Command` struct carries an `Action func() tea.Cmd`. When the user selects a command (Enter), the palette returns a `PaletteSelectMsg` containing the action. DashboardModel receives it and executes the cmd.

```go
type Command struct {
    Name        string           // Display name: "bucket create"
    Description string           // Short help text
    Category    string           // "R2", "DNS", "KV", "Navigation", "Settings"
    Icon        string           // Emoji icon
    Action      func() tea.Cmd   // What happens on selection
    ContextFunc func() bool      // Optional: show only when this returns true
}

type PaletteModel struct {
    input      textinput.Model
    commands   []Command
    filtered   []fuzzy.Match
    cursor     int
    visible    bool
    width      int
    height     int
}
```

### Q6: Building the command registry from Cobra?

**Decision: Walk the Cobra command tree at TUI startup.** `cmd.Root().Commands()` gives us the full tree. Each leaf command becomes a palette entry. We skip hidden commands and add a category based on the parent command name.

```go
func BuildRegistryFromCobra(root *cobra.Command) []Command {
    // Walk tree, create Command per leaf
    // Category = parent.Name()
    // Name = "parent subcommand"
    // Description = cmd.Short
}
```

This keeps the palette automatically in sync with CLI commands — no manual maintenance.

### Q7: Visual design?

```
┌─────────────────────────────────────────────┐
│ > bucket cre_                                │
├─────────────────────────────────────────────┤
│   🪣 bucket create     Create a new R2 bucket│
│   📤 bucket import     Create from spec file │
│   ⚙️ config init       Initialize config     │
│                                              │
│  3 results                    Esc to close   │
└─────────────────────────────────────────────┘
```

- Centered overlay, 60% width, max 10 results
- Matched characters highlighted (bold + primary color)
- Category icon on the left
- Description right-aligned or second line
- Result count + Esc hint at bottom

## Scope Boundaries

**In scope (Phase 1):**
- Palette component with fuzzy search
- CLI command registry from Cobra tree
- Dashboard navigation actions
- Ctrl+P / `:` trigger
- Enter to execute, Esc to close
- Arrow keys / j/k to navigate results

**Out of scope (future):**
- Recently used commands (MRU)
- Command history persistence
- Parameter input (e.g., "bucket create" then prompt for name)
- Custom keybinding registration
- Plugin commands
