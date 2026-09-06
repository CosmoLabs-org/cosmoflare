---
brainstorm: docs/brainstorming/2026-05-18-tui-command-palette.md
completed: "2026-05-24T00:00:00-03:00"
created: "2026-05-18T15:00:00-03:00"
deliverables:
    - id: P-01
      title: internal/tui/components/palette/command.go — Command type + registry builder
    - id: P-02
      title: internal/tui/components/palette/palette.go — PaletteModel tea.Model
    - id: P-03
      title: internal/tui/components/palette/palette_test.go — Component tests
    - id: P-04
      title: Wire Ctrl+P into DashboardModel (update.go + model.go)
    - id: P-05
      title: All tests pass, palette renders correctly
goals_completed: 7
goals_total: 7
issue: FEAT-001
last_review_content_hash: 7e3952c7de97423eccc73de8c013aca9cbac059bba80bf3c81f083d47914dedd
last_review_findings: 0
last_review_ref: docs/planning-mode/2026-05-18-tui-command-palette.md
last_reviewed: "2026-09-06T20:14:03.99216+04:00"
related_prompts: []
requires_reading: []
schema_version: 1
status: COMPLETED
tags: []
title: TUI Command Palette Implementation
---

# Plan: TUI Command Palette (FEAT-001)

**Date**: 2026-05-18
**Issue**: FEAT-001
**Brainstorm**: docs/brainstorming/2026-05-18-tui-command-palette.md

---

## Execution Model

**Single worktree**: `ccs spawn palette-feat` or `ccs glm-agent exec`.
**Estimated effort**: ~400 lines of new code across 4 files.

## Steps

### Step 1: Create command type and registry (P-01)

**File**: `internal/tui/components/palette/command.go`

```go
package palette

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/spf13/cobra"
)

type Command struct {
    Name        string
    Description string
    Category    string
    Icon        string
    Action      func() tea.Cmd
    ContextFunc func() bool
}

func (c Command) FilterValue() string {
    return c.Name + " " + c.Description
}

func BuildRegistryFromCobra(root *cobra.Command) []Command {
    // Walk root.Commands() recursively
    // Skip hidden commands
    // Category = parent.Name() or "root"
    // Name = full path: "bucket create", "dns list"
    // Description = cmd.Short
    // Action = nil for now (CLI commands run via os/exec or tea.ExecProcess)
}

func DefaultDashboardActions() []Command {
    // Hardcoded navigation + dashboard actions:
    // "Go to Overview" → SectionOverview
    // "Go to Buckets" → SectionBucketList
    // "Refresh data" → refreshDataCmd()
    // "Toggle help" → showHelp
    // "Quit" → tea.Quit
}
```

### Step 2: Create palette model (P-02)

**File**: `internal/tui/components/palette/palette.go`

```go
package palette

import (
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/lipgloss"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/sahilm/fuzzy"
)

type PaletteSelectMsg struct {
    Command Command
}

type PaletteDismissMsg struct{}

type PaletteModel struct {
    input    textinput.Model
    commands []Command
    matches  []fuzzy.Match
    cursor   int
    visible  bool
    width    int
    height   int
    maxShow  int // max visible results (10)
}

func New(commands []Command, width, height int) *PaletteModel
func (m *PaletteModel) Show()
func (m *PaletteModel) Hide()
func (m *PaletteModel) IsVisible() bool
func (m PaletteModel) Init() tea.Cmd
func (m PaletteModel) Update(msg tea.Msg) (PaletteModel, tea.Cmd)
func (m PaletteModel) View() string
```

**Key behaviors:**
- On Show(): focus text input, reset filter
- On typing: run fuzzy.Find against commands
- Up/Down or j/k: move cursor through results
- Enter: return PaletteSelectMsg with selected command
- Esc: return PaletteDismissMsg

**Rendering:**
- Centered overlay box using lipgloss.Place()
- Text input at top
- Filtered results list below
- Matched characters highlighted with lipgloss bold+primary
- Footer: result count + "Esc to close"

### Step 3: Write tests (P-03)

**File**: `internal/tui/components/palette/palette_test.go`

Test cases:
1. `TestNewPalette` — creates with commands, correct initial state
2. `TestFuzzyFiltering` — typing "buc cre" matches "bucket create"
3. `TestCursorNavigation` — up/down moves cursor, wraps at boundaries
4. `TestEnterSelectsCommand` — Enter returns PaletteSelectMsg
5. `TestEscDismisses` — Esc returns PaletteDismissMsg
6. `TestEmptyQuery` — shows all commands (or top N)
7. `TestContextFiltering` — commands with ContextFunc=false are hidden
8. `TestBuildRegistryFromCobra` — walks a mock Cobra tree correctly

### Step 4: Wire into DashboardModel (P-04)

**Files**: `internal/tui/model.go`, `internal/tui/update.go`

Changes to `model.go`:
- Add `palette *palette.PaletteModel` field to DashboardModel
- Initialize palette in `initialModel()` with `palette.BuildRegistryFromCobra(root) + palette.DefaultDashboardActions()`

Changes to `update.go`:
- In `handleKeyMsg`: if `m.palette.IsVisible()`, delegate all keys to `m.palette.Update(msg)`
- Add Ctrl+P handler: `case "ctrl+p":` → `m.palette.Show()`
- Add `:` handler (when not in search mode): → `m.palette.Show()`
- Handle `PaletteSelectMsg`: execute `msg.Command.Action()`
- Handle `PaletteDismissMsg`: palette already hidden, no-op

Changes to `view.go`:
- If `m.palette.IsVisible()`, overlay `m.palette.View()` on top of current view

### Step 5: Verify (P-05)

- `go build ./...` passes
- `go test ./internal/tui/...` passes
- `go vet ./...` passes
- Manual test: run `r2go2 dashboard`, press Ctrl+P, type partial command, Enter

## Acceptance Criteria

- [x] Ctrl+P opens palette overlay
- [x] Typing filters commands with fuzzy matching
- [x] Enter executes selected command (navigation actions work)
- [x] Esc closes palette
- [x] All existing TUI tests still pass
- [x] Palette component has its own test suite
- [x] CLI commands appear in palette (from Cobra tree walk)
