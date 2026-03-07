---
project: CosmoDev-R2Go2
version: 0.3.0
date: 2025-01-25
slug: release-notes
title: "release-notes Release"
---

# R2Go2 v0.3.0 Release Notes

**Release Date**: 2025-01-25  
**Codename**: TUI Installer Redesign

---

## Overview

This release focuses on the professional TUI installer experience, fixing critical usability bugs and adding a post-installation setup wizard. Additionally, enhanced CLI features and comprehensive testing infrastructure have been added.

---

## TUI Installer Overhaul

### Bug Fixes

| Issue | Before | After |
|-------|--------|-------|
| **Gigantic menu items** | Each item wrapped in bordered card with `Padding(2,3)` | Compact single-line items |
| **Selection not tracking** | Hardcoded `selectedIndex := 0` | Uses `m.list.Index()` for actual state |
| **Nested borders** | Header + Menu + Container all had borders | Single outer container border |
| **Bloated help section** | 60+ line table with sections | Inline: `↑↓ navigate · enter select · 1-5 quick · esc` |

### New Features

- **Post-Installation Menu**: After installation completes, users get an interactive menu:
  - Configure API credentials
  - Test connection
  - View quick start guide
  - Open documentation
  - Exit to terminal

- **State Machine Flow**: `StateMenu` → `StateInstalling` → `StatePostInstall`

- **Design Improvements**:
  - Unified color palette (`#00D4AA` accent, `#2D3748` borders)
  - Selection indicator: `▸`
  - Number shortcuts: `1-5` for quick selection

---

## Enhanced CLI Features

### New Components (`internal/cli/`)

- **Batch Manager** (`batch/manager.go`): Worker pool for parallel operations
- **Progress Tracking** (`progress/progress.go`): Visual progress indicators
- **UX Improvements** (`ux/`): Confirmation dialogs, retry logic
- **Visual Animations** (`visual/`): R2-themed progress displays
- **Copy Command** (`cmd/copy.go`): Recursive file copying

### Enhanced API Client

- `internal/api/enhanced_client.go`: Improved HTTP client with better error handling

---

## Testing Infrastructure

### New Test Suites

| Category | File | Coverage |
|----------|------|----------|
| Multi-account | `tests/integration/multiaccount/` | Account switching |
| Performance | `tests/performance/` | Benchmarks |
| Security | `tests/security/` | Auth, input validation |
| Platform | `tests/platform/` | Cross-platform |
| HTTP | `tests/integration/http/` | Client validation, rate limits |

---

## Files Changed

- **45 files changed**
- **13,624 insertions**
- **606 deletions**

### Key Files

```
cmd/installer_tui/main.go                    # TUI installer
internal/tui/components/navigation/menu_selection.go  # Menu component
internal/tui/components/installer/header.go  # Header component
internal/cli/batch/manager.go                # Batch operations
internal/cli/progress/progress.go            # Progress tracking
```

---

## Upgrade Notes

No breaking changes. The TUI installer is backwards compatible.

---

## Contributors

- CosmoLabs Development Team

---

*Built with Bubble Tea & Lip Gloss*
