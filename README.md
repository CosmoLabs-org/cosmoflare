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

R2Go2 is a powerful, user-friendly command-line interface (CLI) tool designed to simplify Cloudflare R2 bucket management. Built with Go 1.21+ and Cobra, R2Go2 provides intuitive commands for creating, listing, deleting, uploading files to, and managing lifecycle policies for R2 buckets.

## Features

- **🪣 Bucket Management**: Create, list, and delete R2 buckets
- **📤 File Uploads**: Upload files to buckets with custom object keys
- **🔄 Lifecycle Policies**: Set automatic object deletion policies
- **🔍 JSON Output**: Script-friendly output format for automation
- **🔒 Secure Authentication**: Uses Cloudflare API tokens
- **✨ Dry Run Mode**: Preview actions without execution
- **📊 Universal Versioning**: Built with CosmoLabs' universal versioning system

## Installation

### Prerequisites

- Go 1.21 or higher
- Cloudflare API token with R2 permissions
- Cloudflare Account ID

### Build from Source

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

### Binary Installation

Download the appropriate binary for your platform from the [Releases](https://github.com/CosmoLabs-org/CosmoDev-R2Go2/releases) page.

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

### Basic Commands

#### List All Buckets

```bash
# Simple list output
r2go2 list

# JSON output for scripting
r2go2 list --json

# With specific account
r2go2 list --account-id="your-account-id"
```

#### Create a New Bucket

```bash
# Create a bucket
r2go2 create my-awesome-bucket

# Dry run (shows what would happen)
r2go2 create my-awesome-bucket --dry-run
```

#### Delete a Bucket

```bash
# Delete a bucket (requires confirmation)
r2go2 delete my-awesome-bucket

# Force delete without confirmation
r2go2 delete my-awesome-bucket --confirm

# Dry run
r2go2 delete my-awesome-bucket --dry-run
```

#### Upload Files

```bash
# Upload a file with specific object key
r2go2 upload my-bucket ./local-file.txt --key="remote-file.txt"

# Upload with dry run
r2go2 upload my-bucket ./local-file.txt --key="remote-file.txt" --dry-run

# JSON output
r2go2 upload my-bucket ./local-file.txt --key="remote-file.txt" --json
```

#### Manage Lifecycle Policies

```bash
# Set policy to delete objects after 30 days
r2go2 policy my-bucket set --days=30

# Set policy to delete objects after 1 year
r2go2 policy my-bucket set --days=365

# Dry run
r2go2 policy my-bucket set --days=30 --dry-run
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

---

<p align="center">
  <strong>Built with ❤️ by the CosmoLabs team</strong>
</p>