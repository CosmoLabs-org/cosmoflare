# R2Go2

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Cli-Cobra-25D366?style=for-the-badge&logo=command-line" alt="Cobra CLI">
  <img src="https://img.shields.io/badge/Cloudflare-R2-F38020?style=for-the-badge&logo=cloudflare" alt="Cloudflare R2">
  <img src="https://img.shields.io/badge/License-MIT-9cf?style=for-the-badge" alt="License">
</p>

<p align="center">
  <strong>A production-ready CLI tool for managing Cloudflare R2 buckets</strong>
</p>

<p align="center">
  Built by <a href="https://cosmolabs.org">CosmoLabs</a> for the <a href="https://cosmolabs.org/cosmodev">CosmoDev</a> ecosystem
</p>

---

## Overview

R2Go2 is a **revolutionary dual-use CLI tool** that serves both as a **beautiful interactive interface** for human users and a **powerful JSON API backend** for GUI applications. Built with Go 1.21+, it provides comprehensive Cloudflare R2 management with an exceptional user experience and seamless programmatic integration.

## 🎯 Vision

**R2Go2** is designed to be the **foundational CLI component** of a comprehensive Cloudflare management ecosystem. While it provides a beautiful TUI for human interaction, its true power lies in being a **programmable, JSON-based tool** that can drive sophisticated GUI applications for managing Pages, Workers, and other Cloudflare services.

### 🎮 Interactive TUI Mode
- **Beautiful Visual Interface**: Keyboard navigation, real-time dashboards
- **Professional UX**: Matches enterprise tools like GitHub CLI
- **Interactive Wizards**: Guided setup with smart defaults
- **Real-time Monitoring**: Live progress indicators and statistics

### 🔌 Programmatic API Mode
- **JSON Output**: Every command supports `--json` flag for structured data
- **Consistent Format**: Standardized response structure across all operations
- **Error Handling**: Machine-readable error codes with recovery actions
- **Streaming Support**: Real-time updates for GUI applications

## Features

### 🔐 Authentication & Configuration
- **Interactive Setup Wizard**: Beautiful guided configuration with password masking
- **Multi-Profile Support**: Manage multiple Cloudflare accounts and environments
- **Secure Credential Storage**: Encrypted config file with masked output
- **Auto-Detection**: Smart account info extraction from API tokens
- **Environment Variable Support**: Seamless CI/CD integration

### 🎨 Beautiful Interactive Experience
- **Rich Visual Progress**: Animated spinners, progress bars, and status indicators
- **Smart Error Handling**: Contextual troubleshooting steps and recovery guidance
- **Cross-Platform**: Works beautifully on macOS, Linux, and Windows
- **Accessibility Support**: Screen reader friendly with high contrast options

### 🔧 Core Bucket Management
- **Full R2 Operations**: Create, list, delete, upload, download objects
- **Lifecycle Policies**: Automated object deletion rules
- **Analytics & Monitoring**: Real-time usage statistics and performance metrics
- **Batch Operations**: Efficient handling of multiple files and operations
- **Interactive Setup**: Guided configuration for new users

### 🪣 Bucket Management
- **Complete CRUD Operations**: Create, read, update, delete buckets
- **Metadata Management**: Tags, descriptions, and custom properties
- **Import/Export**: Bulk operations from YAML/JSON specifications
- **Existence Checks**: Fast boolean checks for automation
- **Detailed Information**: Size, object count, location data

### 📁 Object Management
- **Advanced Operations**: Upload, download, copy, delete, search
- **Batch Processing**: Multiple files with progress bars
- **Metadata Support**: Custom headers, content types, cache control
- **Range Downloads**: Partial content support
- **Search & Filter**: Prefix, regex, glob pattern matching
- **Multipart Uploads**: Efficient large file handling

### 🌐 Custom Domains & CDN
- **Domain Management**: Attach, verify, detach custom domains
- **CDN Configuration**: Cache rules, SSL settings, WAF integration
- **Cache Purging**: Selective or complete cache invalidation
- **Edge Analytics**: CDN performance metrics
- **SSL Automation**: Automatic certificate provisioning

### 📊 Analytics & Monitoring
- **Usage Analytics**: Storage, operations, bandwidth metrics
- **Cost Estimation**: Project monthly costs based on usage
- **Health Monitoring**: Connectivity, performance, availability checks
- **Alert Management**: Threshold-based notifications
- **Export Capabilities**: CSV, JSON, PDF report generation

### 🔄 Bulk Operations & Migration
- **S3 Migration**: Complete AWS S3 to R2 migration tools
- **Sync Operations**: Two-way bucket synchronization
- **Backup & Restore**: Complete and incremental backups
- **Resume Capability**: Interrupted operation recovery
- **Parallel Processing**: Configurable concurrency for speed

### 🚀 CI/CD Integration
- **Template Generation**: GitHub Actions, GitLab CI, Jenkins, Azure DevOps
- **Docker Integration**: Containerized deployment workflows
- **n8n Automation**: Visual workflow integration
- **Terraform Support**: Infrastructure as code templates
- **Shell Scripts**: Ready-to-use deployment scripts

### 🛠️ Developer Experience
- **JSON Output**: Machine-readable for automation
- **Dry Run Mode**: Safe operation preview
- **Progress Indicators**: Visual feedback for long operations
- **Shell Completion**: Tab completion support
- **Verbose Logging**: Detailed debugging information

## 🚀 Installation

### ⚡ Quick Install (Recommended)

**macOS & Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/CosmoLabs-org/CosmoDev-R2Go2/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/CosmoLabs-org/CosmoDev-R2Go2/main/install.ps1" -UseBasicParsing | Invoke-Expression
```

That's it! The installer will:
- ✅ Download the correct binary for your system
- ✅ Install it to your PATH automatically
- ✅ Handle all dependencies
- ✅ Show you exactly what to do next

### 📚 Getting Started Guide

**New to R2Go2?** 🎉 Start with our comprehensive [Getting Started Guide](GETTING_STARTED.md) - it will have you up and running in 5 minutes!

### 🔄 Alternative Installation Methods

#### Pre-built Binaries
Download the appropriate binary for your platform from our [Releases](https://github.com/CosmoLabs-org/CosmoDev-R2Go2/releases) page.

#### Build from Source
```bash
# Clone the repository
git clone https://github.com/CosmoLabs-org/CosmoDev-R2Go2.git
cd CosmoDev-R2Go2

# Install dependencies
go mod tidy

# Build the CLI
make build

# Or build for multiple platforms
make build-all
```

### 📋 Prerequisites

- Cloudflare API token with R2 permissions
- Cloudflare Account ID

**Don't have these yet?** Our [Getting Started Guide](GETTING_STARTED.md) shows you exactly how to get them!

## 🎯 Quick Start

Ready to dive in? Here's the fastest path to success:

### 1. Set Up Your Account (2 minutes)
```bash
r2go2 setup
```
This beautiful wizard will securely guide you through API token setup and account validation.

### 2. Launch the Dashboard (instant)
```bash
r2go2 dashboard
```
Welcome to your professional R2 management interface! 🎉

### 3. Try Basic Operations
```bash
# List your buckets
r2go2 list

# Create a new bucket
r2go2 create my-test-bucket

# Upload a file
r2go2 upload my-test-bucket ./my-file.txt
```

**Want more guidance?** Our [Getting Started Guide](GETTING_STARTED.md) has detailed walkthroughs, troubleshooting, and pro tips!

## Configuration

### Environment Variables

Set these environment variables before using R2Go2:

```bash
# Required: Your Cloudflare API token with R2 permissions
export CLOUDFLARE_API_TOKEN="your_api_token_here"

# Optional: Your Cloudflare Account ID (can be provided with --account-id flag)
export CLOUDFLARE_ACCOUNT_ID="your_account_id_here"
```

### API Token Permissions

Your Cloudflare API token needs the following permissions:
- `R2:Edit` for full bucket management
- `Account R2:Read` for listing operations

## Usage

### 🚀 Quick Start

#### 1. Interactive Setup (Recommended)

```bash
# Run the beautiful interactive setup wizard
r2go2 setup
```

The setup wizard provides:
- 🎨 **Beautiful visual interface** with step-by-step guidance
- 🔐 **Secure password masking** for API token input
- 🏢 **Smart auto-detection** of account information
- ✅ **Real-time validation** with your Cloudflare account
- 🎯 **Professional onboarding** with next steps

#### 2. Programmatic Usage (for GUI Integration)

```bash
# JSON output for applications
r2go2 buckets list --json

# Real-time streaming for dashboards
r2go2 monitor my-bucket --stream --json

# Background operations
r2go2 upload ./file.txt my-bucket --json
```

#### 3. Traditional Configuration

```bash
# Manual environment variable setup
export CLOUDFLARE_API_TOKEN="your_api_token_here"
export CLOUDFLARE_ACCOUNT_ID="your_account_id_here"

# Verify configuration
r2go2 config validate
```

#### 2. Bucket Operations

```bash
# List all buckets
r2go2 bucket list

# Create a new bucket
r2go2 bucket create my-awesome-bucket

# Get bucket details
r2go2 bucket get my-awesome-bucket

# Delete a bucket
r2go2 bucket delete my-awesome-bucket
```

#### 3. Object Management

```bash
# Upload files
r2go2 object put my-bucket ./local-file.txt --key="remote/file.txt"

# List objects
r2go2 object ls my-bucket

# Download objects
r2go2 object get my-bucket remote/file.txt --output=local-file.txt

# Search objects
r2go2 object search my-bucket "*.jpg"
```

### Advanced Workflows

#### Custom Domain Setup

```bash
# Attach custom domain
r2go2 domain attach my-bucket --domain=cdn.example.com

# Verify domain configuration
r2go2 domain verify cdn.example.com

# Purge CDN cache
r2go2 domain purge cdn.example.com --path=/images/*
```

#### S3 to R2 Migration

```bash
# Migrate entire S3 bucket
r2go2 migrate from-s3 my-s3-bucket to-r2 my-r2-bucket

# With filtering and concurrency
r2go2 migrate from-s3 my-s3-bucket to-r2 my-r2-bucket \
  --filter="images/*" --concurrency=20

# Resume interrupted migration
r2go2 migrate from-s3 my-s3-bucket to-r2 my-r2-bucket --resume
```

#### Backup & Restore

```bash
# Create backup
r2go2 migrate backup my-bucket --to-local=/backup/ --compress

# Restore from backup
r2go2 migrate restore /backup/my-bucket.tar.gz to my-bucket

# Incremental backup
r2go2 migrate backup my-bucket --to-local=/backup/ --incremental
```

#### Analytics & Monitoring

```bash
# Query usage analytics
r2go2 analytics query my-bucket --start=2025-01-01 --end=2025-01-31

# Export report
r2go2 analytics export my-bucket --output=report.csv --type=detailed

# Health check
r2go2 analytics health check

# Cost estimation
r2go2 analytics cost estimate my-bucket --objects=1000 --size=10GB
```

#### CI/CD Integration

```bash
# Generate GitHub Actions workflow
r2go2 cicd template github --bucket=my-bucket --output=.github/workflows/deploy.yml

# Initialize CI/CD for current project
r2go2 cicd init --platform=github --bucket=production-assets

# Generate deployment script
r2go2 cicd script deploy --bucket=my-bucket --output=deploy.sh
```

### Command Reference

#### Authentication Commands
```bash
r2go2 auth login                    # Interactive authentication
r2go2 auth status                   # Show current auth status
r2go2 auth rotate                   # Rotate API tokens
r2go2 auth logout                   # Clear credentials
```

#### Configuration Commands
```bash
r2go2 config init                   # Initialize configuration
r2go2 config list                   # List all profiles
r2go2 config show [profile]         # Show profile details
r2go2 config set [profile]          # Create/update profile
r2go2 config switch [profile]       # Switch active profile
r2go2 config export [profile]       # Export as env vars
```

#### Bucket Commands
```bash
r2go2 bucket create [name]          # Create bucket
r2go2 bucket ls                     # List buckets
r2go2 bucket get [name]             # Get bucket details
r2go2 bucket update [name]          # Update metadata
r2go2 bucket delete [name]          # Delete bucket
r2go2 bucket exists [name]          # Check if exists
r2go2 bucket import [spec]          # Bulk create from spec
```

#### Object Commands
```bash
r2go2 object ls [bucket]            # List objects
r2go2 object get [bucket] [key]     # Download object
r2go2 object put [bucket] [path]    # Upload object
r2go2 object delete [bucket] [key]  # Delete object
r2go2 object copy [src] [dst]       # Copy object
r2go2 object head [bucket] [key]    # Get metadata
r2go2 object search [bucket] [q]    # Search objects
r2go2 object batch [bucket] [spec]  # Batch operations
```

### Global Flags

All commands support these global flags:

```bash
--account-id string    Cloudflare Account ID (overrides env var)
--dry-run             Show what would happen without executing
--json                Output in JSON format
--verbose, -v         Enable verbose output
```

### Global Flags

All commands support these global flags:

```bash
--account-id string    Your Cloudflare Account ID (overrides CLOUDFLARE_ACCOUNT_ID)
--dry-run            Show what would happen without executing
--json               Output in JSON format
```

### Advanced Usage

#### Scripting with JSON Output

```bash
#!/bin/bash
# List all buckets and process with jq
r2go2 list --json | jq -r '.buckets[].name'

# Upload file and check result
result=$(r2go2 upload my-bucket ./file.txt --key="uploads/file.txt" --json)
if echo "$result" | jq -e '.success' > /dev/null; then
    echo "Upload successful!"
else
    echo "Upload failed: $(echo "$result" | jq -r '.error')"
fi
```

#### Batch Operations

```bash
#!/bin/bash
# Create multiple buckets
buckets=("app-backups" "user-uploads" "log-storage")
for bucket in "${buckets[@]}"; do
    r2go2 create "$bucket"
    echo "Created bucket: $bucket"
done

# Upload multiple files
for file in ./uploads/*; do
    filename=$(basename "$file")
    r2go2 upload user-uploads "$file" --key="$filename"
done
```

## Output Formats

### Default (Human-Readable)

```
🪣 Buckets:
  • my-bucket-1      (Created: 2024-03-15 10:30:00)
  • my-bucket-2      (Created: 2024-03-16 14:22:00)
  • log-storage      (Created: 2024-03-17 09:15:00)

Total: 3 buckets
```

### JSON Output

```json
{
  "success": true,
  "buckets": [
    {
      "name": "my-bucket-1",
      "creation_date": "2024-03-15T10:30:00Z"
    },
    {
      "name": "my-bucket-2",
      "creation_date": "2024-03-16T14:22:00Z"
    }
  ],
  "total": 2
}
```

## Development

### Project Structure

```
R2Go2/
├── cmd/
│   ├── root.go          # Cobra root command and global flags
│   ├── create.go        # Bucket creation command
│   ├── list.go          # Bucket listing command
│   ├── delete.go        # Bucket deletion command
│   ├── upload.go        # File upload command
│   └── policy.go        # Lifecycle policy command
├── internal/
│   └── api/             # R2 API wrapper functions
├── docs/                # Documentation
├── tests/               # Unit tests
├── main.go              # Application entry point
├── go.mod               # Go module definition
├── Makefile             # Build automation
└── README.md            # This file
```

### Building

```bash
# Development build
make build

# Production builds for multiple platforms
make build-all

# Run tests
make test

# Run tests with coverage
make test-coverage
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v ./internal/api -run TestListBuckets
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes using conventional commits
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

### Commit Message Format

This project follows [Conventional Commits](https://conventionalcommits.org/) specification:

```
feat(scope): add new bucket listing feature
fix(scope): resolve memory leak in upload function
docs: update installation guide
chore: update dependencies
```

## Integration with AI IDEs

R2Go2 is designed to work seamlessly with AI coding assistants like Claude Code:

```bash
# Claude Code can execute these commands directly in terminals
r2go2 create my-test-bucket --dry-run
r2go2 list --json
r2go2 upload my-bucket ./test.txt --key="demo/file.txt" --json
```

The JSON output format makes it easy for AI assistants to parse and process results.

## Security Considerations

- **API Tokens**: Never commit API tokens to version control
- **Permissions**: Use minimal necessary API token permissions
- **Input Validation**: All inputs are validated before API calls
- **Error Handling**: Sensitive information is never logged

## Troubleshooting

### Common Issues

1. **API Token Issues**:
   ```bash
   # Verify token permissions in Cloudflare dashboard
   # Ensure token has R2 permissions for your account
   ```

2. **Account ID Not Found**:
   ```bash
   # Set account ID in environment or use --account-id flag
   export CLOUDFLARE_ACCOUNT_ID="your-account-id"
   ```

3. **Connection Issues**:
   ```bash
   # Check network connectivity and Cloudflare API status
   curl -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
        https://api.cloudflare.com/client/v4/user/tokens/verify
   ```

### Debug Mode

Enable verbose logging for debugging:

```bash
r2go2 list --debug
```

## 🌌 GUI Integration & Ecosystem

### 🔌 Programmatic API for GUI Applications

R2Go2 is designed to be the **backend powerhouse** for sophisticated Cloudflare management GUI applications:

```json
// Every command supports JSON output
{
  "success": true,
  "data": {
    "buckets": [
      {
        "name": "production-bucket",
        "size": 2567348257,
        "object_count": 1247,
        "created_date": "2025-01-15T10:30:00Z"
      }
    ]
  },
  "metadata": {
    "timestamp": "2025-01-24T17:30:00Z",
    "execution_time_ms": 245
  }
}
```

### 🎯 GUI Integration Examples

**React/Vue.js Frontend:**
```typescript
import { R2Go2Client } from '@cosmolabs/r2go2-client';

const client = new R2Go2Client();
const buckets = await client.listBuckets(); // Returns typed JSON
```

**Python Desktop Application:**
```python
import asyncio
from r2go2_client import R2Go2Client

async def list_buckets():
    client = R2Go2Client()
    response = await client.execute_command(['buckets', 'list', '--json'])
    return response.data['buckets']
```

**Electron Desktop Application:**
```typescript
// Real-time monitoring for live dashboards
const updates = client.monitorBucket('production');
for await (const update of updates) {
    updateDashboard(update);
}
```

### 🏗️ Cloudflare Management Ecosystem

R2Go2 is the **foundation** for a unified Cloudflare management experience:

```bash
# Current: R2Go2 for storage
r2go2 buckets list --json              # R2 storage management

# Future ecosystem tools:
cf-pages deploy --project=my-app --json    # Pages management
cf-workers deploy --script=api.js --json   # Workers management
cf-dns list --domain=example.com --json    # DNS management
cf-manager status --service=all --json     # Unified dashboard
```

**Learn more:**
- 📖 [GUI Integration Strategy](docs/roadmap/GUI-INTEGRATION-STRATEGY.md)
- 🗺️ [Vision & Architecture Roadmap](docs/roadmap/VISION-AND-ARCHITECTURE.md)
- 🔧 [Parallel Session Coordination](docs/prompts/PARALLEL-SESSION-COORDINATION.md)

## Version History

See [CHANGELOG.md](CHANGELOG.md) for detailed version history.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/CosmoLabs-org/CosmoDev-R2Go2/issues)
- **Discussions**: [GitHub Discussions](https://github.com/CosmoLabs-org/CosmoDev-R2Go2/discussions)

## About CosmoLabs

R2Go2 is part of the [CosmoDev](https://cosmolabs.org/cosmodev) ecosystem by [CosmoLabs](https://cosmolabs.org), a collection of developer tools designed to simplify modern software development workflows.

**Our Vision**: Create a unified, professional command-line experience that makes Cloudflare management accessible to everyone, from individual developers to enterprise teams, while providing the programmable foundation for next-generation GUI applications.

---

<p align="center">
  <strong>Built with ❤️ by the CosmoLabs team</strong>
  <br>
  <em>"One CLI to rule them all, one JSON to bind them"</em>
</p>