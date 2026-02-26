---
branch: tui-installer-wizard-implementation
created: "2025-01-25"
origin: migrated by ccs prompts migrate
priority: medium
status: PENDING
title: 'Continuation Prompt: TUI Installer Post-Installation Wizard Implementation'
---

# Continuation Prompt: TUI Installer Post-Installation Wizard Implementation

## Project Context

**Project**: R2Go2 - Professional Cloudflare R2 CLI Tool
**Company**: CosmoLabs
**Repository**: `/Users/gabstudio/PROJECTS/CosmoDev-R2Go2`
**Technology Stack**: Go, Bubble Tea (bubbletea), Lip Gloss (lipgloss)

## Previous Session Summary

We completed a major TUI installer redesign:

### 1. Menu System Overhaul
- **Fixed gigantic menu items** - Changed from card-based design (each item wrapped in bordered card with `Padding(2, 3)`) to compact single-line items
- **Fixed selection tracking** - Was hardcoded `selectedIndex := 0`, now uses `m.list.Index()` for actual selection
- **Compact help line** - Replaced 60+ line help section with inline: `↑↓ navigate · enter select · 1-5 quick · esc exit`

### 2. Post-Installation Menu Added
After installation completes (100%), a new menu appears with options:
```
▸ 1. 🔑 Configure API credentials · Set up Cloudflare R2 access
  2. 🔌 Test connection · Verify R2 bucket access
  3. 📖 View quick start guide · Learn basic commands
  4. 📚 Open documentation · Full usage reference
  5. ✅ Exit to terminal · Start using R2Go2
```

### 3. Clean Container Design
- Single outer border around entire installer
- Removed nested borders from header and menu components
- Unified visual design

## Current State - Stubbed Handlers Need Implementation

The following handlers in `cmd/installer_tui/main.go` are stubbed with TODOs:

```go
// startAPISetup launches the API credential configuration
func (m *InstallerModel) startAPISetup() tea.Cmd {
	// TODO: Implement actual API setup wizard
	return tea.Sequence(
		tea.Printf("\n🔑 Launching API setup wizard...\n"),
		tea.Quit,
	)
}

// testConnection tests the R2 connection
func (m *InstallerModel) testConnection() tea.Cmd {
	// TODO: Implement actual connection test
	return nil
}

// showQuickStart displays the quick start guide
func (m *InstallerModel) showQuickStart() tea.Cmd {
	// TODO: Implement quick start view
	return nil
}

// openDocumentation opens the documentation
func (m *InstallerModel) openDocumentation() tea.Cmd {
	// TODO: Open docs in browser
	return nil
}
```

## Files Modified This Session

1. **`cmd/installer_tui/main.go`** - Main installer with new `StatePostInstall` state and post-install menu
2. **`internal/tui/components/navigation/menu_selection.go`** - Compact menu rendering
3. **`internal/tui/components/installer/header.go`** - Simplified header styles

## Key Architecture Decisions

1. **State Machine Flow**: `StateMenu` → `StateInstalling` → `StatePostInstall` (via `installCompleteMsg`)
2. **Menu Reuse**: Same `MenuModel` component used for both main menu and post-install menu
3. **Handler Pattern**: `handleMenuSelection()` for main menu, `handlePostInstallSelection()` for post-install

## Next Steps - Implement These Features

### Priority 1: API Credential Setup Wizard (`startAPISetup`)
Create an interactive wizard flow:
1. Ask for Cloudflare Account ID
2. Ask for R2 Access Key ID
3. Ask for R2 Secret Access Key
4. Validate credentials format
5. Save to config file (check existing config system in codebase)
6. Return to post-install menu or show success

### Priority 2: Test Connection (`testConnection`)
1. Load saved credentials
2. Attempt to list buckets or perform health check
3. Show success/failure with details
4. Return to post-install menu

### Priority 3: Quick Start Guide (`showQuickStart`)
1. Create a scrollable text view with basic commands
2. Show common operations: list buckets, upload, download, sync
3. Allow return to menu

### Priority 4: Open Documentation (`openDocumentation`)
1. Use `exec.Command` to open browser with docs URL
2. Handle cross-platform (macOS: `open`, Linux: `xdg-open`, Windows: `start`)

## Relevant Existing Code to Check

- `cmd/root.go` - May have existing config setup
- `internal/` - Check for existing credential/config management
- Look for `.r2go2` or similar config files

## Build Command
```bash
go build -o r2go2-tui-installer ./cmd/installer_tui/
```

## Testing
```bash
./r2go2-tui-installer
```

## Design Guidelines Established

- **Compact menus**: Single-line items with `▸` indicator
- **Color palette**:
  - Accent: `#00D4AA` (teal)
  - Primary text: `#FFFFFF`
  - Secondary: `#94A3B8`
  - Muted: `#64748B`
  - Border: `#2D3748`
- **No nested borders** - Single container border only
- **Lip Gloss patterns** - Use `lipgloss.NewStyle()` with method chaining

---

*Generated: 2025-01-25*
*Session: TUI Menu Redesign & Post-Installation Menu*
