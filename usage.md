# R2Go2 Usage Guide

Complete guide to using R2Go2 - the production-ready CLI tool for managing Cloudflare R2 buckets.

## 🚀 Quick Start

### 1. Installation

#### Build from Source
```bash
git clone https://github.com/CosmoLabs-org/CosmoDev-R2Go2.git
cd CosmoDev-R2Go2
make build
sudo make install-system  # or: cp build/r2go2 /usr/local/bin/
```

#### Download Binary
Download the appropriate binary for your platform from the [Releases](https://github.com/CosmoLabs-org/CosmoDev-R2Go2/releases) page.

### 2. Authentication

Set up your Cloudflare credentials:

```bash
export CLOUDFLARE_API_TOKEN="your_api_token_here"
export CLOUDFLARE_ACCOUNT_ID="your_account_id_here"
```

**Required Token Permissions:**
- `R2:Edit` for full bucket management
- `Account R2:Read` for listing operations

### 3. Verify Installation

```bash
r2go2 --version
r2go2 --help
```

## 📋 Commands Reference

### Global Options

All commands support these global flags:

```bash
--account-id string    Override CLOUDFLARE_ACCOUNT_ID
--dry-run             Show what would happen without executing
--json                Output in JSON format
--verbose, -v         Enable verbose output
```

### 🪣 Bucket Management

#### Create Bucket
```bash
# Basic bucket creation
r2go2 create my-awesome-bucket

# Preview without creating
r2go2 create my-awesome-bucket --dry-run

# JSON output
r2go2 create my-awesome-bucket --json
```

**Naming Rules:**
- 3-63 characters long
- Lowercase letters, numbers, hyphens, periods
- Must start and end with letter or number
- Globally unique across Cloudflare

#### List Buckets
```bash
# Human-readable output
r2go2 list

# JSON output for scripting
r2go2 list --json

# Verbose with account info
r2go2 list --verbose
```

**Example Output:**
```
🪣 Buckets:

  • my-backups         (Created 7 days ago)
  • user-uploads       (Created 14 days ago)
  • log-storage        (Created 30 days ago)

Total: 3 buckets
```

#### Delete Bucket
```bash
# Interactive deletion (recommended)
r2go2 delete my-bucket

# Skip confirmation (use with caution)
r2go2 delete my-bucket --confirm

# Preview deletion
r2go2 delete my-bucket --dry-run
```

**⚠️ Warning:** Deleting a bucket permanently removes all objects within it. This action cannot be undone.

### 📤 File Uploads

#### Upload Single File
```bash
# Basic upload
r2go2 upload my-bucket ./local-file.txt --key="remote-file.txt"

# Upload with custom path
r2go2 upload my-bucket ./image.jpg --key="photos/2024/profile.jpg"

# JSON output
r2go2 upload my-bucket ./data.json --key="api/data.json" --json
```

#### Upload Examples
```bash
# Upload backup with date
r2go2 upload backups ./backup.tar.gz --key="backups/$(date +%Y-%m-%d)/backup.tar.gz"

# Upload multiple files (scripting)
for file in ./logs/*.log; do
    r2go2 upload log-storage "$file" --key="logs/$(basename $file)"
done

# Upload with dry-run
r2go2 upload my-bucket ./important.doc --key="docs/important.doc" --dry-run
```

**Upload Features:**
- Automatic content-type detection
- File size validation
- Object key validation
- Progress information (planned)
- Resume capability (planned)

### ⚙️ Lifecycle Policies

#### Set Basic Policy
```bash
# Delete objects after 30 days
r2go2 policy my-bucket set --days=30

# Delete objects after 1 year
r2go2 policy my-bucket set --days=365

# Preview policy change
r2go2 policy my-bucket set --days=90 --dry-run
```

**Policy Behavior:**
- Applied to all existing and future objects
- Automatic deletion after specified days
- No notification before deletion
- Can be modified or removed later

### 🔧 Shell Completion

Enable auto-completion for your shell:

```bash
# Bash
eval "$(r2go2 completion bash)"
# Or add to ~/.bashrc
echo 'eval "$(r2go2 completion bash)"' >> ~/.bashrc

# Zsh
r2go2 completion zsh > "${fpath[1]}/_r2go2"
# Or add to ~/.zshrc
echo 'autoload -U compinit; compinit' >> ~/.zshrc

# Fish
r2go2 completion fish | source

# PowerShell
r2go2 completion powershell | Out-String | Invoke-Expression
```

## 📊 JSON Output Format

All commands support JSON output with consistent structure:

### Success Response
```json
{
  "success": true,
  "message": "Operation completed successfully",
  "data": {
    // Command-specific data
  },
  "dry_run": false
}
```

### Error Response
```json
{
  "success": false,
  "error": "Error message description",
  "dry_run": false
}
```

### Bucket List Response
```json
{
  "success": true,
  "buckets": [
    {
      "name": "my-bucket",
      "creation_date": "2024-03-15T10:30:00Z"
    }
  ],
  "total": 1,
  "dry_run": false
}
```

## 🔄 Scripting Examples

### Bash Scripting

#### Backup Script
```bash
#!/bin/bash

# Daily backup script
BACKUP_DIR="/backups"
BUCKET_NAME="daily-backups"
DATE=$(date +%Y-%m-%d)

# Create backup
tar -czf "$BACKUP_DIR/backup-$DATE.tar.gz" /data

# Upload to R2
r2go2 upload "$BUCKET_NAME" "$BACKUP_DIR/backup-$DATE.tar.gz" \
  --key="backups/$DATE/backup.tar.gz" --json

# Cleanup old backups (keep 7 days)
find "$BACKUP_DIR" -name "backup-*.tar.gz" -mtime +7 -delete

echo "Backup completed: $DATE"
```

#### Batch Upload
```bash
#!/bin/bash

# Upload all images from directory
BUCKET="photo-storage"
SOURCE_DIR="./photos"

for file in "$SOURCE_DIR"/*.{jpg,jpeg,png,gif}; do
    if [ -f "$file" ]; then
        filename=$(basename "$file")
        r2go2 upload "$BUCKET" "$file" --key="images/$filename"
        echo "Uploaded: $filename"
    fi
done
```

#### Bucket Management
```bash
#!/bin/bash

# Bucket maintenance script
BUCKETS=$(r2go2 list --json | jq -r '.buckets[].name')

for bucket in $BUCKETS; do
    echo "Processing bucket: $bucket"

    # Set 30-day retention
    r2go2 policy "$bucket" set --days=30

    # Get bucket info
    r2go2 list --json | jq --arg bucket "$bucket" \
      '.buckets[] | select(.name == $bucket)'
done
```

### Python Scripting

#### Python Integration
```python
import subprocess
import json
import os

def list_buckets():
    """List all R2 buckets"""
    cmd = ['r2go2', 'list', '--json']
    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode == 0:
        data = json.loads(result.stdout)
        return data['buckets']
    else:
        raise Exception(f"Command failed: {result.stderr}")

def upload_file(bucket, local_path, object_key):
    """Upload file to R2 bucket"""
    cmd = ['r2go2', 'upload', bucket, local_path, '--key', object_key, '--json']
    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode == 0:
        return json.loads(result.stdout)
    else:
        raise Exception(f"Upload failed: {result.stderr}")

# Usage example
try:
    buckets = list_buckets()
    print(f"Found {len(buckets)} buckets")

    upload_result = upload_file('my-bucket', './test.txt', 'uploads/test.txt')
    print(f"Upload successful: {upload_result['message']}")

except Exception as e:
    print(f"Error: {e}")
```

## 🐛 Troubleshooting

### Common Issues

#### Authentication Errors
```bash
❌ Error: Configuration error: CLOUDFLARE_API_TOKEN environment variable is required
```
**Solution:** Set your Cloudflare API token:
```bash
export CLOUDFLARE_API_TOKEN="your_token_here"
```

#### Account ID Issues
```bash
❌ Error: Cloudflare Account ID is required
```
**Solution:** Set your account ID or use flag:
```bash
export CLOUDFLARE_ACCOUNT_ID="your_account_id"
# OR
r2go2 list --account-id="your_account_id"
```

#### Permission Issues
```bash
❌ Error: Failed to create bucket: permission denied
```
**Solution:** Verify your API token permissions:
- Token must have `R2:Edit` permission
- Token must be for the correct account

#### Network Issues
```bash
❌ Error: Failed to list buckets: connection timeout
```
**Solution:** Check network connectivity:
```bash
# Test API connectivity
curl -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
     https://api.cloudflare.com/client/v4/user/tokens/verify
```

### Debug Mode

Enable verbose output for debugging:

```bash
r2go2 list --verbose
r2go2 upload my-bucket file.txt --key="test.txt" --verbose --dry-run
```

### Environment Check

Verify your setup:

```bash
# Check version
r2go2 --version

# Check environment
echo "API Token: ${CLOUDFLARE_API_TOKEN:0:10}..."
echo "Account ID: $CLOUDFLARE_ACCOUNT_ID"

# Test API token
curl -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
     https://api.cloudflare.com/client/v4/user/tokens/verify
```

## 📚 Advanced Usage

### Dry Run Mode

Always use `--dry-run` to preview operations:

```bash
# Preview all operations
r2go2 create production-backups --dry-run
r2go2 upload prod-db ./backup.sql --key="backups/db.sql" --dry-run
r2go2 delete old-bucket --dry-run --confirm
```

### JSON Processing with jq

Combine R2Go2 with jq for powerful data processing:

```bash
# Get bucket names only
r2go2 list --json | jq -r '.buckets[].name'

# Find buckets created in last 30 days
r2go2 list --json | jq '.buckets[] | select(.creation_date > (now - 30*24*3600 | strftime("%Y-%m-%dT%H:%M:%SZ")))

# Count total buckets
r2go2 list --json | jq '.total'

# Format bucket list as CSV
r2go2 list --json | jq -r '.buckets[] | "\(.name),\(.creation_date)"'
```

### Automation with Cron

Set up automated tasks:

```bash
# Edit crontab
crontab -e

# Daily backup at 2 AM
0 2 * * * /usr/local/bin/r2go2 upload backups /data/daily-backup.tar.gz --key="backups/$(date +\%Y\%m\%d)/daily.tar.gz"

# Weekly cleanup
0 3 * * 0 /usr/local/bin/r2go2 policy temp-storage set --days=7
```

## 🔒 Security Best Practices

### API Token Management
- Use minimal required permissions
- Rotate tokens regularly
- Never commit tokens to version control
- Use environment variables or secure storage

### Object Key Security
- Avoid sensitive information in object keys
- Use random names for sensitive files
- Implement access controls at bucket level

### Network Security
- Use HTTPS connections (automatic)
- Consider VPN for sensitive operations
- Monitor Cloudflare API usage

---

## 📞 Support

- **Documentation**: [README.md](README.md)
- **Issues**: [GitHub Issues](https://github.com/CosmoLabs-org/CosmoDev-R2Go2/issues)
- **Discussions**: [GitHub Discussions](https://github.com/CosmoLabs-org/CosmoDev-R2Go2/discussions)
- **Session History**: [docs/sessions/](docs/sessions/)

---

**Version**: 0.1.0 | **Last Updated**: 2025-11-24 | **License**: MIT