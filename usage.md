# R2Go2 Complete Usage Guide

**The definitive guide to mastering R2Go2 - your production-ready command-line interface for Cloudflare R2**

## 📚 Table of Contents

1. [Quick Start](#-quick-start)
2. [Authentication & Configuration](#-authentication--configuration)
3. [Command Categories](#-command-categories)
4. [Advanced Workflows](#-advanced-workflows)
5. [Integration Examples](#-integration-examples)
6. [Troubleshooting](#-troubleshooting)
7. [Best Practices](#-best-practices)

---

## 🚀 Quick Start

### Installation

```bash
# Option 1: Download binary
curl -fsSL https://github.com/CosmoLabs-org/CosmoDev-R2Go2/releases/latest/download/r2go2-linux-amd64 -o r2go2
chmod +x r2go2
sudo mv r2go2 /usr/local/bin/

# Option 2: Build from source
git clone https://github.com/CosmoLabs-org/CosmoDev-R2Go2.git
cd CosmoDev-R2Go2
make build
sudo cp build/r2go2 /usr/local/bin/

# Option 3: Install via package manager (when available)
brew install r2go2  # macOS
sudo apt install r2go2  # Ubuntu/Debian
```

### Initial Setup

```bash
# Interactive configuration (recommended)
r2go2 config init

# Follow the prompts to set up your Cloudflare credentials

# Verify setup
r2go2 auth status
```

### First Operations

```bash
# Create your first bucket
r2go2 bucket create my-app-bucket

# Upload files
r2go2 object put my-app-bucket ./dist/ --recursive

# List objects
r2go2 object ls my-app-bucket

# Set up custom domain
r2go2 domain attach my-app-bucket --domain=cdn.example.com
```

---

## 🔐 Authentication & Configuration

### Interactive Setup

```bash
# Complete guided setup
r2go2 config init
```

**Setup Process:**
1. Enter Cloudflare API token
2. Provide Account ID (auto-detected if possible)
3. Add profile description
4. Test connection
5. Save to `~/.r2go2/config.yaml`

### Manual Configuration

```bash
# Set environment variables
export CLOUDFLARE_API_TOKEN="your_api_token_here"
export CLOUDFLARE_ACCOUNT_ID="your_account_id_here"

# Or create profile manually
r2go2 config set production \
  --account-id="your_account_id" \
  --api-token="your_api_token" \
  --description="Production environment"
```

### Profile Management

```bash
# List all profiles
r2go2 config list

# Switch between profiles
r2go2 config switch staging

# Show profile details (masked)
r2go2 config show production

# Export profile as environment variables
eval $(r2go2 config export production)

# Delete profile
r2go2 config delete old-profile
```

### Authentication Commands

```bash
# Login with interactive flow
r2go2 auth login --interactive

# Validate current credentials
r2go2 auth status

# Rotate API token
r2go2 auth rotate --profile=production

# Logout (clears session)
r2go2 auth logout
```

### Token Permissions

**Required Permissions:**
- `R2:Edit` - Full bucket and object management
- `Account R2:Read` - Read account information
- `Zone:Cache:Edit` - Cache management (for CDN features)
- `Zone:Zone:Read` - Zone information (for domains)

---

## 📋 Command Categories

### 🪣 Bucket Management

#### Create Buckets

```bash
# Basic creation
r2go2 bucket create my-new-bucket

# With location and tags
r2go2 bucket create my-app-bucket \
  --location=eu \
  --tags=env=production,tier=standard

# From specification file
r2go2 bucket import buckets.yaml
```

**Specification File (`buckets.yaml`):**
```yaml
buckets:
  - name: production-assets
    location: us
    tags:
      env: prod
      tier: standard
    metadata:
      description: "Production assets bucket"

  - name: staging-assets
    location: eu
    tags:
      env: staging
      tier: standard
```

#### List and Inspect Buckets

```bash
# List all buckets
r2go2 bucket list

# JSON output for scripts
r2go2 bucket list --json

# Filter by prefix
r2go2 bucket list --prefix=prod-

# Get detailed bucket information
r2go2 bucket get my-bucket

# Include object count and size
r2go2 bucket get my-bucket --include-objects

# Check if bucket exists (good for CI/CD)
r2go2 bucket exists my-bucket && echo "Bucket exists"
```

#### Update Bucket Metadata

```bash
# Add tags
r2go2 bucket update my-bucket --add-tags=environment=staging

# Replace all tags
r2go2 bucket update my-bucket --tags=env=prod,tier=premium

# Remove specific tags
r2go2 bucket update my-bucket --remove-tags=temp,experimental

# Update metadata
r2go2 bucket update my-bucket --metadata=owner=team-a,project=website
```

#### Delete Buckets

```bash
# Interactive deletion (safest)
r2go2 bucket delete my-bucket

# Force deletion (use with caution)
r2go2 bucket delete my-bucket --force

# Preview deletion
r2go2 bucket delete my-bucket --dry-run
```

### 📁 Object Management

#### Upload Objects

```bash
# Upload single file
r2go2 object put my-bucket ./file.txt --key="remote/file.txt"

# Upload with custom metadata
r2go2 object put my-bucket ./image.jpg \
  --key="assets/images/profile.jpg" \
  --content-type="image/jpeg" \
  --cache-control="max-age=31536000,immutable" \
  --metadata=author=admin,department=marketing

# Upload entire directory
r2go2 object put my-bucket ./dist/ --recursive --progress

# Upload with custom naming
r2go2 object put my-bucket ./backup.tar.gz \
  --key="backups/$(date +%Y-%m-%d)/backup.tar.gz"

# Dry run to preview
r2go2 object put my-bucket ./important-data/ --recursive --dry-run
```

#### List and Search Objects

```bash
# List all objects
r2go2 object ls my-bucket

# List with prefix filtering
r2go2 object ls my-bucket --prefix="images/"

# Recursive listing
r2go2 object ls my-bucket --recursive --max-keys=1000

# JSON output
r2go2 object ls my-bucket --json

# Search objects
r2go2 object search my-bucket "*.jpg" --type=glob

# Search with regex
r2go2 object search my-bucket ".*\.png$" --type=regex

# Exact match
r2go2 object search my-bucket "config.json" --type=exact
```

#### Download Objects

```bash
# Download single object
r2go2 object get my-bucket remote/file.txt --output=local-file.txt

# Download with progress
r2go2 object get my-bucket large-file.zip --progress

# Download specific byte range
r2go2 object get my-bucket video.mp4 \
  --range-start=0 --range-end=1048575 \
  --output=video-header.bin

# Download to stdout (for piping)
r2go2 object get my-bucket data.json | jq '.users'
```

#### Object Metadata

```bash
# Get object metadata without downloading
r2go2 object head my-bucket remote/file.txt

# JSON output
r2go2 object head my-bucket remote/file.txt --json

# Example output:
# Object: remote/file.txt
# Size: 1.2 KB
# Last Modified: 2025-01-15T10:30:00Z
# ETag: "abc123def456"
# Content-Type: text/plain
# Cache-Control: max-age=3600
```

#### Copy and Move Objects

```bash
# Copy object within same bucket
r2go2 object copy my-bucket/old-path/file.txt my-bucket/new-path/file.txt

# Copy between buckets
r2go2 object copy source-bucket/data.csv dest-bucket/backup/data.csv

# Copy with new metadata
r2go2 object copy my-bucket/temp.txt my-bucket/production.txt \
  --metadata=environment=prod,status=final

# Delete objects
r2go2 object delete my-bucket temporary-file.txt

# Delete with dry run
r2go2 object delete my-bucket old-backup/ --dry-run
```

#### Batch Operations

```bash
# Create batch specification
cat > operations.json << EOF
{
  "operations": [
    {
      "action": "upload",
      "local_path": "./dist/app.js",
      "object_key": "js/app.js"
    },
    {
      "action": "delete",
      "object_key": "old-version/app.js"
    },
    {
      "action": "copy",
      "source": "assets/logo.png",
      "dest": "assets/legacy-logo.png"
    }
  ]
}
EOF

# Execute batch operations
r2go2 object batch my-bucket operations.json

# Continue on errors
r2go2 object batch my-bucket operations.json --continue

# Dry run preview
r2go2 object batch my-bucket operations.json --dry-run
```

### 🌐 Custom Domains & CDN

#### Domain Management

```bash
# Attach custom domain
r2go2 domain attach my-bucket --domain=cdn.example.com

# With SSL mode specification
r2go2 domain attach my-bucket \
  --domain=assets.example.com \
  --ssl-mode=full

# List all domains
r2go2 domain list

# Filter by bucket
r2go2 domain list --bucket=my-bucket

# JSON output
r2go2 domain list --json
```

#### Domain Verification

```bash
# Verify domain configuration
r2go2 domain verify cdn.example.com

# Detailed verification
r2go2 domain verify cdn.example.com --detailed

# Example output:
# ✅ Domain Verification Results:
# CHECK              STATUS   DETAILS
# DNS Configuration ✅       CNAME record correctly configured
# SSL Certificate   ✅       Valid certificate installed
# Origin Connectivity ✅       R2 bucket accessible
# CDN Status        ✅       Cloudflare edge active
```

#### Cache Management

```bash
# Purge entire cache
r2go2 domain purge cdn.example.com --everything

# Purge specific paths
r2go2 domain purge cdn.example.com --path="/images/*"

# Purge multiple paths
r2go2 domain purge cdn.example.com \
  --path="/css/*" \
  --path="/js/*" \
  --path="/assets/*"

# Purge by cache tags
r2go2 domain purge cdn.example.com --tags=static,versioned
```

#### Cache Rules

```bash
# Create cache rule (example)
r2go2 domain cache create \
  --domain=cdn.example.com \
  --name="static-assets" \
  --ttl=30d \
  --expression="http.host eq \"cdn.example.com\""

# List cache rules
r2go2 domain cache list --domain=cdn.example.com
```

### 📊 Analytics & Monitoring

#### Usage Analytics

```bash
# Query storage usage
r2go2 analytics query my-bucket --metric=storage \
  --start=2025-01-01 --end=2025-01-31

# Query all metrics
r2go2 analytics query my-bucket --metric=all \
  --start=2025-01-01 --end=2025-01-31 --format=csv

# JSON output for processing
r2go2 analytics query my-bucket --metric=operations --json
```

**Example JSON Output:**
```json
{
  "bucket": "my-bucket",
  "period": {
    "start": "2025-01-01T00:00:00Z",
    "end": "2025-01-31T23:59:59Z"
  },
  "storage": {
    "used_bytes": 1073741824,
    "object_count": 1000,
    "average_size": 1073742
  },
  "operations": {
    "class_a_count": 5000,
    "class_b_count": 50000
  },
  "bandwidth": {
    "inbound_bytes": 1073741824,
    "outbound_bytes": 10737418240
  }
}
```

#### Cost Estimation

```bash
# Estimate costs based on current usage
r2go2 analytics cost analyze my-bucket --period=30d

# Estimate costs for planned usage
r2go2 analytics cost estimate my-bucket \
  --objects=10000 \
  --size=100

# Example output:
# Cost Breakdown (Monthly Estimates):
# Storage (100 GB)         $1.50
# Class A Operations (10,000)  $0.05
# Class B Operations (100,000) $0.40
# -------------------------------
# Total Monthly Cost        $1.95
```

#### Health Monitoring

```bash
# Comprehensive health check
r2go2 analytics health check

# Specific bucket health
r2go2 analytics health check --bucket=my-bucket

# Performance testing
r2go2 analytics health performance --bucket=my-bucket --count=10
```

#### Export Capabilities

```bash
# Export detailed report
r2go2 analytics export my-bucket \
  --output=report.csv \
  --type=detailed \
  --start=2025-01-01 \
  --end=2025-01-31

# Export JSON for processing
r2go2 analytics export my-bucket \
  --output=data.json \
  --format=json \
  --type=billing
```

### 🔄 Migration & Bulk Operations

#### S3 to R2 Migration

```bash
# Migrate entire S3 bucket
r2go2 migrate from-s3 my-s3-bucket to-r2 my-r2-bucket

# With filtering and concurrency
r2go2 migrate from-s3 my-s3-bucket to-r2 my-r2-bucket \
  --filter="images/*" \
  --concurrency=20 \
  --verify

# Resume interrupted migration
r2go2 migrate from-s3 my-s3-bucket to-r2 my-r2-bucket \
  --resume

# Dry run to preview
r2go2 migrate from-s3 my-s3-bucket to-r2 my-r2-bucket \
  --dry-run

# With AWS profile
r2go2 migrate from-s3 my-s3-bucket to-r2 my-r2-bucket \
  --aws-profile=production \
  --aws-region=us-west-2
```

#### Sync Operations

```bash
# Sync local directory to R2
r2go2 migrate sync ./dist r2://my-bucket

# Two-way sync
r2go2 migrate sync r2://source-bucket r2://dest-bucket

# Delete files not in source
r2go2 migrate sync ./dist r2://my-bucket --delete-extras

# Verify after sync
r2go2 migrate sync ./dist r2://my-bucket --verify

# Dry run preview
r2go2 migrate sync ./dist r2://my-bucket --dry-run
```

#### Backup Operations

```bash
# Create complete backup
r2go2 migrate backup my-bucket --to-local=/backups/

# Compressed backup with versions
r2go2 migrate backup my-bucket \
  --to-local=/backups/ \
  --compress \
  --include-versions

# Incremental backup
r2go2 migrate backup my-bucket \
  --to-local=/backups/ \
  --incremental

# JSON output for automation
r2go2 migrate backup my-bucket --to-local=/backups/ --json
```

#### Restore Operations

```bash
# Restore from backup
r2go2 migrate restore /backups/my-bucket.tar.gz to my-bucket

# Restore specific paths
r2go2 migrate restore /backups/my-bucket.tar.gz to my-bucket \
  --path="images/*"

# Verify after restore
r2go2 migrate restore /backups/my-bucket.tar.gz to my-bucket \
  --verify

# Resume interrupted restore
r2go2 migrate restore /backups/my-bucket.tar.gz to my-bucket \
  --resume
```

#### Batch Processing

```bash
# Batch delete from manifest
echo "old-file.txt\ntemp-file.json" > files_to_delete.txt
r2go2 migrate batch my-bucket files_to_delete.txt

# Batch operations with JSON manifest
r2go2 migrate batch my-bucket operations.json --concurrency=10

# Continue on errors
r2go2 migrate batch my-bucket operations.json --continue
```

### 🚀 CI/CD Integration

#### Template Generation

```bash
# GitHub Actions workflow
r2go2 cicd template github \
  --bucket=my-app \
  --env=production \
  --output=.github/workflows/deploy.yml

# GitLab CI pipeline
r2go2 cicd template gitlab \
  --bucket=my-app \
  --output=.gitlab-ci.yml

# Jenkins pipeline
r2go2 cicd template jenkins \
  --bucket=my-app \
  --output=Jenkinsfile

# Azure DevOps pipeline
r2go2 cicd template azure \
  --bucket=my-app \
  --output=azure-pipelines.yml

# Dockerfile
r2go2 cicd template docker \
  --bucket=my-app \
  --output=Dockerfile

# n8n workflow
r2go2 cicd template n8n \
  --bucket=my-app \
  --webhook=https://hooks.slack.com/... \
  --output=n8n-workflow.json

# Terraform configuration
r2go2 cicd template terraform \
  --bucket=my-app \
  --output=main.tf
```

#### Project Initialization

```bash
# Auto-detect project and setup CI/CD
r2go2 cicd init --platform=github --bucket=production-assets

# For specific project type
r2go2 cicd init \
  --platform=gitlab \
  --bucket=staging-assets \
  --env=staging
```

#### Script Generation

```bash
# Deployment script
r2go2 cicd script deploy \
  --bucket=my-app \
  --output=deploy.sh

# Backup script
r2go2 cicd script backup \
  --bucket=my-app \
  --output=backup.sh

# Sync script
r2go2 cicd script sync \
  --bucket=my-app \
  --output=sync.sh

# Cleanup script
r2go2 cicd script cleanup \
  --bucket=my-app \
  --output=cleanup.sh
```

---

## 🔧 Advanced Workflows

### Multi-Environment Deployment

```bash
# Setup profiles for different environments
r2go2 config set development --account-id="dev_account" --api-token="dev_token"
r2go2 config set staging --account-id="staging_account" --api-token="staging_token"
r2go2 config set production --account-id="prod_account" --api-token="prod_token"

# Deploy to different environments
r2go2 config switch development
r2go2 object put dev-app ./dist-dev/ --recursive

r2go2 config switch staging
r2go2 object put staging-app ./dist-staging/ --recursive

r2go2 config switch production
r2go2 object put prod-app ./dist-prod/ --recursive
```

### Automated Backup Strategy

```bash
#!/bin/bash
# Comprehensive backup script

BACKUP_DATE=$(date +%Y-%m-%d)
BACKUP_DIR="/backups"
BUCKETS=("app-assets" "user-uploads" "database-backups")

# Create backup directory
mkdir -p "$BACKUP_DIR/$BACKUP_DATE"

# Backup each bucket
for bucket in "${BUCKETS[@]}"; do
    echo "Backing up $bucket..."
    r2go2 migrate backup "$bucket" \
        --to-local="$BACKUP_DIR/$BACKUP_DATE/" \
        --compress \
        --include-versions

    # Generate analytics report
    r2go2 analytics export "$bucket" \
        --output="$BACKUP_DIR/$BACKUP_DATE/${bucket}-analytics.csv" \
        --type=detailed
done

# Compress all backups
tar -czf "$BACKUP_DIR/backup-$BACKUP_DATE.tar.gz" \
    -C "$BACKUP_DIR" "$BACKUP_DATE"

# Cleanup
rm -rf "$BACKUP_DIR/$BACKUP_DATE"

echo "Backup completed: backup-$BACKUP_DATE.tar.gz"
```

### Content Delivery Workflow

```bash
#!/bin/bash
# Production deployment with CDN management

APP_NAME="my-app"
DOMAIN="cdn.example.com"

# 1. Build application
npm run build

# 2. Upload to R2
echo "Uploading assets..."
r2go2 object put "$APP_NAME" ./dist/ --recursive --progress

# 3. Setup custom domain (if not already)
echo "Configuring domain..."
r2go2 domain attach "$APP_NAME" --domain="$DOMAIN" || \
echo "Domain already configured"

# 4. Verify domain configuration
echo "Verifying domain..."
r2go2 domain verify "$DOMAIN"

# 5. Clear CDN cache
echo "Clearing CDN cache..."
r2go2 domain purge "$DOMAIN" --everything

# 6. Generate deployment report
echo "Generating report..."
r2go2 analytics export "$APP_NAME" \
    --output="deployment-$(date +%Y%m%d).csv" \
    --type=summary

echo "Deployment completed successfully!"
echo "Your site is now available at: https://$DOMAIN"
```

### Migration Automation

```bash
#!/bin/bash
# AWS S3 to R2 migration automation

S3_BUCKET="my-aws-bucket"
R2_BUCKET="my-r2-bucket"
AWS_PROFILE="production"
CONCURRENCY=50

# 1. Create R2 bucket if it doesn't exist
if ! r2go2 bucket exists "$R2_BUCKET"; then
    echo "Creating R2 bucket: $R2_BUCKET"
    r2go2 bucket create "$R2_BUCKET" --location=auto
fi

# 2. Start migration
echo "Starting migration from $S3_BUCKET to $R2_BUCKET"
r2go2 migrate from-s3 "$S3_BUCKET" to-r2 "$R2_BUCKET" \
    --aws-profile="$AWS_PROFILE" \
    --concurrency="$CONCURRENCY" \
    --verify \
    --resume

# 3. Post-migration verification
echo "Verifying migration..."
S3_COUNT=$(aws s3 ls "s3://$S3_BUCKET" --recursive | wc -l)
R2_COUNT=$(r2go2 object ls "$R2_BUCKET" --json | jq length)

echo "S3 objects: $S3_COUNT"
echo "R2 objects: $R2_COUNT"

if [ "$S3_COUNT" -eq "$R2_COUNT" ]; then
    echo "✅ Migration successful!"
else
    echo "⚠️ Object count mismatch - investigate"
fi
```

---

## 🔍 Integration Examples

### GitHub Actions Workflow

```yaml
# .github/workflows/deploy.yml
name: Deploy to R2

on:
  push:
    branches: [ main ]
  workflow_dispatch:

jobs:
  deploy:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v4

    - name: Setup Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '20'
        cache: 'npm'

    - name: Install dependencies
      run: npm ci

    - name: Build application
      run: npm run build

    - name: Install R2Go2
      run: |
        curl -fsSL https://github.com/CosmoLabs-org/CosmoDev-R2Go2/releases/latest/download/r2go2-linux-amd64 -o r2go2
        chmod +x r2go2

    - name: Deploy to R2
      env:
        CLOUDFLARE_API_TOKEN: ${{ secrets.CLOUDFLARE_API_TOKEN }}
        CLOUDFLARE_ACCOUNT_ID: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
      run: |
        ./r2go2 object put ${{ secrets.R2_BUCKET }} ./dist/ --recursive --progress
        ./r2go2 domain purge ${{ secrets.R2_DOMAIN }} --everything

    - name: Update deployment info
      run: |
        ./r2go2 analytics query ${{ secrets.R2_BUCKET }} --metric=storage --json > deployment-info.json
        cat deployment-info.json
```

### GitLab CI Pipeline

```yaml
# .gitlab-ci.yml
stages:
  - build
  - deploy

variables:
  R2_BUCKET: $R2_BUCKET
  NODE_ENV: production

build:
  stage: build
  image: node:20-alpine
  cache:
    paths:
      - node_modules/
  script:
    - npm ci
    - npm run build
  artifacts:
    paths:
      - dist/
    expire_in: 1 hour

deploy:
  stage: deploy
  image: alpine:latest
  dependencies:
    - build
  before_script:
    - apk add --no-cache curl
    - curl -fsSL https://github.com/CosmoLabs-org/CosmoDev-R2Go2/releases/latest/download/r2go2-linux-amd64 -o r2go2
    - chmod +x r2go2
  script:
    - r2go2 object put $R2_BUCKET ./dist/ --recursive --progress
    - echo "Deployment completed successfully"
  only:
    - main
```

### Docker Integration

```dockerfile
# Dockerfile
FROM node:20-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM alpine:latest

# Install R2Go2
RUN apk add --no-cache curl && \
    curl -fsSL https://github.com/CosmoLabs-org/CosmoDev-R2Go2/releases/latest/download/r2go2-linux-amd64 -o /usr/local/bin/r2go2 && \
    chmod +x /usr/local/bin/r2go2

WORKDIR /app
COPY --from=builder /app/dist ./dist

# Environment variables
ENV R2_BUCKET=my-app
ENV CLOUDFLARE_API_TOKEN=""
ENV CLOUDFLARE_ACCOUNT_ID=""

# Deploy script
RUN echo '#!/bin/sh' > /deploy.sh && \
    echo 'r2go2 object put $R2_BUCKET ./dist/ --recursive --progress' >> /deploy.sh && \
    echo 'r2go2 domain purge $R2_DOMAIN --everything' >> /deploy.sh && \
    chmod +x /deploy.sh

CMD ["/deploy.sh"]
```

---

## 🐛 Troubleshooting

### Common Issues

#### Authentication Problems

```bash
# Check if token is valid
curl -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
     https://api.cloudflare.com/client/v4/user/tokens/verify

# Verify account access
r2go2 auth status

# Test with different profile
r2go2 auth status --profile=production
```

#### Connection Issues

```bash
# Check connectivity
r2go2 analytics health check

# Verbose output for debugging
r2go2 bucket list --verbose

# Test API directly
curl -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
     -H "X-Account-Id: $CLOUDFLARE_ACCOUNT_ID" \
     https://api.cloudflare.com/client/v4/accounts
```

#### Permission Issues

```bash
# Verify token permissions
r2go2 auth status --verbose

# Test basic operation
r2go2 bucket list --dry-run

# Check account ID format
echo $CLOUDFLARE_ACCOUNT_ID | grep -E '^[a-f0-9]{32}$'
```

### Debug Mode

```bash
# Enable verbose logging
export R2GO2_DEBUG=1

# Or use verbose flag
r2go2 bucket list --verbose

# Dry run for safe testing
r2go2 object put test-bucket ./test.txt --key=test.txt --dry-run --verbose
```

### Environment Validation Script

```bash
#!/bin/bash
# validate-environment.sh

echo "🔍 Validating R2Go2 Environment..."

# Check R2Go2 installation
if command -v r2go2 &> /dev/null; then
    echo "✅ R2Go2 is installed: $(r2go2 --version)"
else
    echo "❌ R2Go2 is not installed or not in PATH"
    exit 1
fi

# Check environment variables
if [ -z "$CLOUDFLARE_API_TOKEN" ]; then
    echo "❌ CLOUDFLARE_API_TOKEN is not set"
    exit 1
fi

if [ -z "$CLOUDFLARE_ACCOUNT_ID" ]; then
    echo "❌ CLOUDFLARE_ACCOUNT_ID is not set"
    exit 1
fi

# Validate token
echo "🔑 Validating API token..."
response=$(curl -s -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
                   https://api.cloudflare.com/client/v4/user/tokens/verify)

if echo "$response" | grep -q '"result":true'; then
    echo "✅ API token is valid"
else
    echo "❌ API token is invalid"
    echo "Response: $response"
    exit 1
fi

# Validate account ID
if [[ "$CLOUDFLARE_ACCOUNT_ID" =~ ^[a-f0-9]{32}$ ]]; then
    echo "✅ Account ID format is valid"
else
    echo "❌ Account ID format is invalid (should be 32-character hex)"
    exit 1
fi

# Test R2Go2 connectivity
echo "🌐 Testing R2Go2 connectivity..."
if r2go2 bucket list --dry-run > /dev/null 2>&1; then
    echo "✅ R2Go2 can connect to Cloudflare API"
else
    echo "❌ R2Go2 cannot connect to Cloudflare API"
    exit 1
fi

echo "🎉 Environment validation completed successfully!"
```

---

## 📈 Best Practices

### Security

```bash
# Use minimal permissions
r2go2 config set production \
  --account-id="your_account_id" \
  --api-token="token_with_r2_edit_only"

# Rotate tokens regularly
r2go2 auth rotate --profile=production

# Use environment variables in CI/CD
export CLOUDFLARE_API_TOKEN="$SECRET_TOKEN"

# Never commit credentials
echo ".r2go2/" >> .gitignore
echo "*.env" >> .gitignore
```

### Performance

```bash
# Use parallel operations for large datasets
r2go2 migrate from-s3 large-bucket to-r2 r2-large-bucket --concurrency=50

# Use appropriate batch sizes
r2go2 object put my-bucket ./large-dataset/ --recursive --batch-size=100

# Monitor performance
r2go2 analytics health performance --bucket=my-bucket --count=10
```

### Cost Optimization

```bash
# Monitor costs regularly
r2go2 analytics cost analyze my-bucket --period=30d

# Set lifecycle policies for old data
r2go2 bucket update temp-bucket --add-tags=auto-expire=30d

# Use cache headers appropriately
r2go2 object put my-bucket ./static.css \
  --key="css/static.css" \
  --cache-control="max-age=31536000,immutable"

# Clean up unused buckets
r2go2 bucket list --json | jq '.buckets[] | select(.creation_date < (now - 90*24*3600 | strftime("%Y-%m-%dT%H:%M:%SZ")))'
```

### Automation

```bash
# Use dry run for critical operations
r2go2 bucket delete production-backup --dry-run

# Implement proper error handling
#!/bin/bash
set -euo pipefail

r2go2 object put "$BUCKET" "$FILE" --key="$KEY" || {
    echo "Upload failed: $FILE"
    exit 1
}

# Use JSON output for scripting
result=$(r2go2 bucket create "my-bucket" --json)
success=$(echo "$result" | jq -r '.success')

if [ "$success" = "true" ]; then
    echo "Bucket created successfully"
else
    echo "Bucket creation failed: $(echo "$result" | jq -r '.error')"
    exit 1
fi
```

---

## 📞 Support & Resources

- **Documentation**: [README.md](README.md)
- **Project Roadmap**: [docs/roadmap/roadmap.md](docs/roadmap/roadmap.md)
- **Issues**: [GitHub Issues](https://github.com/CosmoLabs-org/CosmoDev-R2Go2/issues)
- **Discussions**: [GitHub Discussions](https://github.com/CosmoLabs-org/CosmoDev-R2Go2/discussions)
- **Change Log**: [CHANGELOG.md](CHANGELOG.md)

### Getting Help

```bash
# Built-in help
r2go2 --help
r2go2 bucket --help
r2go2 object --help

# Command completion
r2go2 completion bash > ~/.local/share/bash-completion/completions/r2go2

# Version information
r2go2 --version
r2go2 auth status
```

---

**Version**: 1.0.0 | **Last Updated**: 2025-01-24 | **License**: MIT

*Built with ❤️ by CosmoLabs for the modern development ecosystem*