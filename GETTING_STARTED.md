# 🚀 Getting Started with R2Go2

Welcome to R2Go2! This guide will help you get up and running in minutes with our beautiful Cloudflare R2 management tool.

## 📋 Quick Overview

**R2Go2** is a professional command-line tool for managing Cloudflare R2 storage with two modes:
- **🎮 Interactive TUI Mode**: Beautiful terminal dashboard with keyboard navigation
- **🔌 Programmatic Mode**: JSON output for automation and GUI applications

## ⚡ One-Command Installation

### macOS & Linux
```bash
curl -fsSL https://raw.githubusercontent.com/CosmoLabs-org/CosmoDev-R2Go2/main/install.sh | bash
```

### Windows (PowerShell)
```powershell
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/CosmoLabs-org/CosmoDev-R2Go2/main/install.ps1" -UseBasicParsing | Invoke-Expression
```

That's it! The script will:
- ✅ Download the correct binary for your system
- ✅ Install it to `~/.local/bin` (or add to Windows PATH)
- ✅ Handle all dependencies automatically
- ✅ Show you exactly what to do next

## 🎯 Your First 5 Minutes

### Step 1: Set Up Your Cloudflare Credentials

R2Go2 needs two things from your Cloudflare account:

1. **API Token** with R2 permissions
2. **Account ID**

#### Option A: Interactive Setup (Recommended)
```bash
r2go2 setup
```

This will launch a beautiful wizard that:
- 🔒 Securely collects your API token (password-style input)
- 🔍 Auto-detects your account information
- ✅ Validates everything works
- 🎯 Creates a secure profile for you

#### Option B: Manual Setup
```bash
# Set environment variables
export CLOUDFLARE_API_TOKEN="your_api_token_here"
export CLOUDFLARE_ACCOUNT_ID="your_account_id_here"

# Verify it works
r2go2 list
```

### Where to Find Your Cloudflare Credentials

1. **Account ID**:
   - Go to [Cloudflare Dashboard](https://dash.cloudflare.com/)
   - Right side of the sidebar → "Get your API token"
   - Copy your Account ID

2. **API Token**:
   - Click "Get your API token" → "Create Token"
   - Use "Custom token" template
   - Permissions needed:
     - **Zone** → **Zone** → **Read**
     - **Account** → **Cloudflare R2:Edit** (or **R2:Read** for read-only)

### Step 2: Launch the Beautiful Dashboard

```bash
r2go2 dashboard
```

Welcome to your professional R2 management interface! 🎉

**Dashboard Features:**
- 📊 **Overview**: See all your buckets and usage statistics
- 🎮 **Keyboard Navigation**: Use arrow keys, Enter to select
- 🔍 **Search & Filter**: Find buckets and objects instantly
- 📈 **Real-Time Monitoring**: Live upload/download statistics
- ⚙️ **Settings**: Switch profiles and customize themes

**Quick Dashboard Keys:**
- `↑↓` or `jk`: Navigate up/down
- `Enter`: Select item
- `C`: Create bucket
- `U`: Upload files
- `F1`: Show help
- `Q`: Quit

### Step 3: Try Basic Operations

#### In the Dashboard:
1. Press `C` to create a new bucket
2. Press `U` to upload files
3. Use arrow keys to explore your buckets

#### Or Use Commands:
```bash
# List all buckets
r2go2 list

# Create a new bucket
r2go2 create my-test-bucket

# Upload a file
r2go2 upload my-test-bucket ./my-file.txt --key="uploads/my-file.txt"

# List files in a bucket
r2go2 objects list my-test-bucket
```

## 🔧 Power User Features

### JSON Output for Automation
Every command supports `--json` for programmatic use:

```bash
# Get JSON output for scripts
r2go2 list --json

# Upload and get JSON response
r2go2 upload my-bucket ./file.txt --key="uploads/file.txt" --json
```

### Multiple Profiles
Manage multiple Cloudflare accounts:

```bash
# Create a new profile
r2go2 setup --profile=work

# Switch profiles
r2go2 config switch work

# List all profiles
r2go2 config list
```

### Environment Variable Mode
Perfect for CI/CD and automation:

```bash
# Set once, use everywhere
export CLOUDFLARE_API_TOKEN="your_token"
export CLOUDFLARE_ACCOUNT_ID="your_account"

# All commands now work without setup
r2go2 list --json
```

## 🎨 Pro Tips

### 1. Use the Dashboard for Daily Work
The dashboard (`r2go2 dashboard`) is your primary interface for:
- ✅ Managing buckets and files
- ✅ Monitoring usage and activity
- ✅ Quick uploads and downloads
- ✅ Visual file management

### 2. Use Commands for Scripting
Perfect for automation and CI/CD:
```bash
# Backup script
r2go2 upload backup-$(date +%Y%m%d).tar.gz backups/

# Deploy script
r2go2 upload build/ production-assets --recursive
```

### 3. Enable Shell Completion
```bash
# Add to your ~/.bashrc or ~/.zshrc
eval "$(r2go2 completion bash)"  # or zsh, fish, powershell
```

### 4. Use Dry Run Mode
Test operations without making changes:
```bash
r2go2 create test-bucket --dry-run
```

## 🛠️ Common Workflows

### Web Development Deployment
```bash
# Deploy static site
r2go2 upload ./dist my-website-bucket --recursive

# Update specific files
r2go2 upload ./css/main.css my-website-bucket --key="css/main.css"
```

### Backup and Restore
```bash
# Backup important files
r2go2 upload ./documents my-backups --recursive

# Restore files
r2go2 download my-backups ./restored-documents --recursive
```

### Media Management
```bash
# Upload photos
r2go2 upload ./photos my-photo-gallery --recursive

# List by size
r2go2 list --sort=size --json | jq '.buckets | sort_by(.size)'
```

## 🔍 Troubleshooting

### "Command not found" Error
If `r2go2` command isn't found:

**macOS/Linux:**
```bash
# Add to PATH (temporary)
export PATH="$PATH:$HOME/.local/bin"

# Add permanently (add to ~/.bashrc or ~/.zshrc)
echo 'export PATH="$PATH:$HOME/.local/bin"' >> ~/.bashrc
source ~/.bashrc
```

**Windows:**
```powershell
# Restart PowerShell after installation
# Or manually add to PATH:
$env:PATH += ";$env:USERPROFILE\.local\bin"
```

### API Token Issues
1. **Token Permissions**: Ensure your token has **R2:Edit** permissions
2. **Account ID**: Copy from Cloudflare dashboard sidebar
3. **Token Format**: Should start with `v1_...` or similar

### Connection Issues
```bash
# Test connectivity
curl -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
     https://api.cloudflare.com/client/v4/user/tokens/verify

# Enable verbose logging
r2go2 list --verbose
```

## 📚 Next Steps

Now you're ready to explore more advanced features:

- 📖 **Full Documentation**: [README.md](README.md)
- 🎯 **TUI Dashboard**: `r2go2 dashboard`
- 🔧 **Advanced Commands**: `r2go2 --help`
- 💬 **Get Help**: [GitHub Issues](https://github.com/CosmoLabs-org/CosmoDev-R2Go2/issues)

## 🌟 What Makes R2Go2 Special?

- **🎮 Professional TUI**: Beautiful dashboard that works over SSH anywhere
- **🔌 Dual Use**: Perfect for both humans and automation/GUI applications
- **⚡ Lightning Fast**: Written in Go for maximum performance
- **🌍 Cross-Platform**: Works on macOS, Linux, and Windows
- **🛡️ Enterprise Ready**: Secure credential handling and error recovery

---

**Welcome to the R2Go2 family!** 🎉

You now have a professional, beautiful Cloudflare R2 management tool that works everywhere. Whether you're managing files through the dashboard or automating with JSON output, R2Go2 makes R2 storage delightful to use.

Need help? Found a bug? Have a feature request?
- 📚 [Documentation](https://github.com/CosmoLabs-org/CosmoDev-R2Go2)
- 💬 [Discussions](https://github.com/CosmoLabs-org/CosmoDev-R2Go2/discussions)
- 🐛 [Issues](https://github.com/CosmoLabs-org/CosmoDev-R2Go2/issues)

**Happy R2 managing!** 🚀