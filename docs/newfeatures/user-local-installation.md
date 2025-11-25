# User-Local Installation (No Sudo Required)

**Feature**: Install R2Go2 to user directory with automatic PATH configuration
**Added**: v0.3.1
**Status**: Implemented

## Overview

R2Go2 now installs to `~/.local/bin` by default, eliminating the need for `sudo` or administrator privileges. The installer automatically:

1. **Copies the binary** to `~/.local/bin/r2go2`
2. **Creates case-insensitive symlinks** (R2Go2, R2GO2, etc.)
3. **Adds to PATH** automatically by updating shell configs
4. **Saves install location** to `~/.r2go2/install-info.txt`

## Installation Locations

| Platform | Install Directory | Config Location |
|----------|------------------|-----------------|
| macOS | `~/.local/bin` | `~/.r2go2/` |
| Linux | `~/.local/bin` | `~/.r2go2/` |
| Windows | `%LOCALAPPDATA%\R2Go2\bin` | `%USERPROFILE%\.r2go2\` |

## Automatic PATH Configuration

The installer automatically updates your shell configuration:

### macOS (zsh default)
Adds to `~/.zshrc`:
```bash
# R2Go2 CLI
export PATH="~/.local/bin:$PATH"
```

Also updates `~/.bash_profile` if it exists.

### Linux
Updates whichever exists:
- `~/.zshrc` (if using zsh)
- `~/.bashrc` (if using bash)
- `~/.profile` (fallback)

### Windows
Windows filesystem is case-insensitive by default. The installer creates executables in the user's local app data. Users may need to manually add to PATH via System Settings.

## After Installation

After installation, the TUI shows:

```
✅ R2Go2 installed successfully!

📁 Install location:
  /Users/yourname/.local/bin

✅ PATH updated automatically
   Restart your terminal or run: source ~/.zshrc

Command is case-insensitive: R2Go2, r2go2, R2GO2 all work!
```

**To activate immediately** (without restarting terminal):
```bash
source ~/.zshrc    # macOS
source ~/.bashrc   # Linux
```

## Install Info File

The installer saves metadata to `~/.r2go2/install-info.txt`:

```
install_dir=/Users/yourname/.local/bin
install_date=2025-11-25 12:30:45
version=0.3.1
```

This file can be used by:
- `R2Go2 --help` to show install location
- Update scripts to find the installed binary
- Uninstallers to know what to remove

## Case-Insensitive Commands

The installer creates symlinks so all these work:

```bash
R2Go2 bucket list    # Branded (recommended)
r2go2 bucket list    # Lowercase
R2GO2 bucket list    # All caps
r2Go2 bucket list    # Any mix
```

## Benefits

### No Sudo Required
- Users don't need admin/root access
- Safer - only modifies user's home directory
- Works in restricted environments

### Automatic PATH
- No manual configuration needed
- Works immediately after terminal restart
- Idempotent - won't add duplicates

### Clean Uninstall
- Everything in `~/.local/bin` and `~/.r2go2`
- Easy to remove completely
- Doesn't affect system directories

## Technical Implementation

### Binary Installation
```go
installDir = homeDir + "/.local/bin"
os.MkdirAll(installDir, 0755)
copyFile(sourceBinary, installDir + "/r2go2")
os.Chmod(destBinary, 0755)
```

### PATH Update
```go
pathLine := fmt.Sprintf("\n# R2Go2 CLI\nexport PATH=\"%s:$PATH\"\n", installDir)
// Appends to ~/.zshrc, ~/.bashrc, etc.
```

### Symlink Creation
```go
aliases := []string{"R2Go2", "R2GO2", "r2Go2", "R2go2", "r2GO2"}
for _, alias := range aliases {
    os.Symlink("r2go2", installDir + "/" + alias)
}
```

## Comparison: User vs System Install

| Aspect | User Install (~/.local/bin) | System Install (/usr/local/bin) |
|--------|----------------------------|--------------------------------|
| Sudo required | No | Yes |
| Affects other users | No | Yes |
| Survives user deletion | No | Yes |
| PATH auto-config | Yes | Usually already in PATH |
| Recommended for | Personal use | Shared systems |

## Future Enhancements

- **Custom install location**: Allow users to specify directory
- **System-wide install option**: For admins who want /usr/local/bin
- **Uninstaller**: `R2Go2 uninstall` command
- **Update checker**: `R2Go2 update` to fetch latest version

---

*Part of the R2Go2 CLI Tool by CosmoLabs*
