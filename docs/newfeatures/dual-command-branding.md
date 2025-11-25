# Case-Insensitive Command: R2Go2

**Feature**: Fully case-insensitive CLI command
**Added**: v0.3.1
**Status**: Implemented

## Overview

R2Go2 is **fully case-insensitive**. Users can type the command in any case variation and it will work:

```bash
R2Go2 bucket list    # Branded (recommended)
r2go2 bucket list    # Lowercase
R2GO2 bucket list    # All caps
r2Go2 bucket list    # Mixed case
R2go2 bucket list    # Mixed case
```

**All of these work identically.**

## How It Works

### macOS & Linux
The installer creates **symbolic links** for all common case variations:

```
/usr/local/bin/r2go2    # Main binary
/usr/local/bin/R2Go2    # Symlink → r2go2 (branded)
/usr/local/bin/R2GO2    # Symlink → r2go2 (all caps)
/usr/local/bin/r2Go2    # Symlink → r2go2 (mixed)
/usr/local/bin/R2go2    # Symlink → r2go2 (mixed)
/usr/local/bin/r2GO2    # Symlink → r2go2 (mixed)
```

### Windows
Since Windows symlinks require administrator privileges, the installer creates **copies** of the binary for each variation. However, Windows filesystems are typically case-insensitive by default, so this is mainly for consistency.

## Why This Matters

### No More "Command Not Found" Errors
On case-sensitive systems (macOS, Linux), typing the wrong case normally causes errors. With R2Go2, users never have to worry about capitalization.

### Branding Consistency
- Documentation uses `R2Go2` (the branded form)
- Users typing from memory can use any case
- Professional appearance in examples and tutorials

### User-Friendly Experience
- New users don't need to remember exact capitalization
- Copy-paste from different sources always works
- Reduces support issues and frustration

## Supported Case Variations

| Command | Status |
|---------|--------|
| `R2Go2` | ✅ Works (Recommended - branded) |
| `r2go2` | ✅ Works (lowercase) |
| `R2GO2` | ✅ Works (all caps) |
| `r2Go2` | ✅ Works |
| `R2go2` | ✅ Works |
| `r2GO2` | ✅ Works |

## Technical Implementation

Located in `cmd/installer_tui/main.go`:

```go
// createBrandedSymlinks creates case-insensitive command aliases
// Users can type R2Go2, r2go2, R2GO2, r2GO2, etc. - all will work
func createBrandedSymlinks(installDir string) error {
    aliases := []string{
        "R2Go2",  // Branded (recommended)
        "R2GO2",  // All caps
        "r2Go2",  // Mixed
        "R2go2",  // Mixed
        "r2GO2",  // Mixed
        // "r2go2" is the main binary, not a symlink
    }

    for _, alias := range aliases {
        symlinkPath := installDir + "/" + alias

        if runtime.GOOS == "windows" {
            // Windows: copy the binary
            // ...
        } else {
            // macOS and Linux: create symbolic link
            os.Symlink("r2go2", symlinkPath)
        }
    }
    return nil
}
```

## Installation Verification

After installation, verify the commands work:

```bash
# All of these should show the same version
R2Go2 --version
r2go2 --version
R2GO2 --version

# Check symlinks exist (macOS/Linux)
ls -la /usr/local/bin/ | grep -i r2go2
```

## Cross-Platform Compatibility

| Platform | Filesystem | Solution |
|----------|-----------|----------|
| macOS | Case-sensitive | Symlinks for all variations |
| Linux | Case-sensitive | Symlinks for all variations |
| Windows | Case-insensitive | Binary copies (filesystem handles case) |

## Related: SuperMac Integration Idea

For a more comprehensive solution across all CLI tools, consider implementing case-insensitive command aliasing as a SuperMac feature. This would automatically create aliases for common case variations of any installed command, eliminating "command not found" errors caused by capitalization differences across the entire system.

---

*Part of the R2Go2 CLI Tool by CosmoLabs*
