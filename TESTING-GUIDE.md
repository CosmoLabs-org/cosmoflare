# R2Go2 Testing & Setup Guide

## 🚀 Quick Start

### 1. Build the Tool
```bash
go build -o R2Go2 .
```

### 2. Setup Configuration
```bash
# Interactive setup wizard
./R2Go2 setup

# Or set environment variables manually
export CLOUDFLARE_API_TOKEN="your-api-token-here"
export CLOUDFLARE_ACCOUNT_ID="your-account-id-here"
```

## 📋 Prerequisites

### Cloudflare Requirements
1. **Cloudflare Account** with R2 access
2. **API Token** with R2 permissions:
   - Account ID read access
   - R2 bucket read/write permissions
   - Zone read permissions (if needed for zone-level access)

### Creating an API Token
1. Go to Cloudflare Dashboard → My Profile → API Tokens
2. Create token with "Custom token" template
3. Required permissions:
   - `Account:Cloudflare R2:Edit`
   - `Account:Account Settings:Read`
4. Account resources: `Include All accounts` or specify your account ID

## 🔧 Configuration Methods

### Method 1: Interactive Setup (Recommended)
```bash
./r2go2 setup
```
Follow the prompts to:
- Enter your Cloudflare Account ID
- Provide your API Token
- Configure default profile
- Set preferred bucket

### Method 2: Environment Variables
```bash
export CLOUDFLARE_API_TOKEN="your_api_token"
export CLOUDFLARE_ACCOUNT_ID="your_account_id"
```

### Method 3: Profile Configuration
```bash
# Create a profile
./r2go2 config create my-profile --account-id="your_account_id" --token="your_api_token"

# Switch to profile
./r2go2 switch my-profile

# List profiles
./r2go2 config list
```

## 🧪 Testing the Tool

### Basic Functionality Tests

#### 1. Test Connection
```bash
# Test basic connectivity
./r2go2 list

# With verbose output
./r2go2 list --verbose

# JSON output
./r2go2 list --json
```

#### 2. Bucket Operations
```bash
# Create a test bucket
./r2go2 create test-bucket-12345

# List buckets
./r2go2 list

# Test dry-run (won't actually create)
./r2go2 create test-bucket-dryrun --dry-run
```

#### 3. File Operations
```bash
# Create a test file
echo "Hello R2Go2!" > test.txt

# Upload file
./r2go2 upload test-bucket-12345 test.txt

# List objects in bucket
./r2go2 list-objects test-bucket-12345

# Download file
./r2go2 download test-bucket-12345 test.txt --output="downloaded.txt"

# Delete file
./r2go2 delete-object test-bucket-12345 test.txt
```

#### 4. Interactive TUI Dashboard
```bash
# Launch TUI dashboard
./r2go2 dashboard
```

### Advanced Testing Scenarios

#### 1. Multiple Profiles
```bash
# Create production profile
./r2go2 config create prod --account-id="prod_account_id" --token="prod_token"

# Create development profile
./r2go2 config create dev --account-id="dev_account_id" --token="dev_token"

# Switch between profiles
./r2go2 switch prod
./r2go2 switch dev

# Show current profile
./r2go2 config current
```

#### 2. Theme Customization
```bash
# List available themes
./r2go2 theme list

# Apply a theme
./r2go2 theme apply dark

# Create custom theme
./r2go2 theme create my-theme --colors="primary:#ff0000,secondary:#00ff00"
```

#### 3. Backup and Restore
```bash
# Backup configurations
./r2go2 backup create --output="my-backup.json"

# Restore configurations
./r2go2 backup restore --from="my-backup.json"

# List backups
./r2go2 backup list
```

## 🐛 Common Issues & Troubleshooting

### Authentication Issues
```bash
# Verify credentials
./r2go2 auth verify

# Test token permissions
./r2go2 auth test

# Refresh token
./r2go2 auth refresh
```

### Connection Issues
```bash
# Test connectivity
./r2go2 test-connection

# Check verbose output for errors
./r2go2 list --verbose

# Test with different endpoint (if custom)
./r2go2 list --endpoint="https://custom.r2.endpoint.com"
```

### Permission Issues
```bash
# Check file permissions for uploads
ls -la test.txt

# Verify bucket permissions
./r2go2 bucket info test-bucket-12345
```

## 📊 Performance Testing

### Upload Speed Test
```bash
# Create a larger test file
dd if=/dev/zero of=test-large.bin bs=1M count=10

# Upload with progress
./r2go2 upload test-bucket-12345 test-large.bin --progress

# Time the upload
time ./r2go2 upload test-bucket-12345 test-large.bin
```

### Batch Operations Test
```bash
# Upload multiple files
for i in {1..10}; do
  echo "Test file $i" > "test-$i.txt"
  ./r2go2 upload test-bucket-12345 "test-$i.txt"
done

# Clean up
for i in {1..10}; do
  ./r2go2 delete-object test-bucket-12345 "test-$i.txt"
  rm "test-$i.txt"
done
```

## 🎯 Feature Testing Checklist

### Core Features
- [ ] Basic bucket operations (create, list, delete)
- [ ] File operations (upload, download, delete)
- [ ] Profile management
- [ ] Configuration management
- [ ] Authentication and authorization

### Advanced Features
- [ ] Interactive TUI dashboard
- [ ] Theme customization
- [ ] Backup and restore
- [ ] Batch operations
- [ ] Progress reporting
- [ ] JSON output for scripting

### Cross-Platform Features
- [ ] Works on macOS, Linux, Windows
- [ ] Shell completion generation
- [ ] Environment variable support
- [ ] Config file management

## 💡 Improvement Areas to Test

### 1. User Experience
- **Setup Process**: How intuitive is the interactive setup?
- **Error Messages**: Are error messages clear and actionable?
- **Progress Indicators**: Do upload/downloads show progress clearly?
- **Help System**: Is help information comprehensive and accessible?

### 2. Performance
- **Large File Handling**: How does it perform with files >100MB?
- **Batch Operations**: How efficient are multiple file operations?
- **Memory Usage**: What's the memory footprint during operations?
- **Startup Time**: How quickly does the tool start?

### 3. Integration
- **Shell Integration**: Does tab completion work well?
- **Scripting**: Is JSON output parseable and useful for scripts?
- **CI/CD**: Can it be effectively used in automated workflows?

### 4. TUI/Interactive Features
- **Dashboard Usability**: Is the TUI dashboard intuitive?
- **Keyboard Navigation**: Are keyboard shortcuts logical?
- **Visual Feedback**: Is status information clearly presented?
- **Error Handling**: How does the TUI handle errors and interruptions?

## 🔍 Testing Command Reference

### Debug Mode
```bash
# Enable verbose output
./r2go2 --verbose list

# Dry run mode
./r2go2 --dry-run create test-bucket

# JSON output for parsing
./r2go2 --json list
```

### Profile Management
```bash
# Profile operations
./r2go2 config create
./r2go2 config list
./r2go2 config current
./r2go2 switch
./r2go2 config delete
```

### Advanced Configuration
```bash
# Theme management
./r2go2 theme list
./r2go2 theme apply
./r2go2 theme create
./r2go2 theme delete

# Backup operations
./r2go2 backup create
./r2go2 backup restore
./r2go2 backup list
```

---

## 📝 Testing Feedback

Please test the tool and provide feedback on:
1. **Setup Experience**: Was initial setup smooth?
2. **Core Functionality**: Do basic operations work as expected?
3. **Performance**: How does it perform with your use cases?
4. **User Interface**: Is the CLI/TUI intuitive?
5. **Missing Features**: What functionality would you like to see?
6. **Bugs/Issues**: Any problems or unexpected behavior?

Your feedback will help prioritize the next development phases and improve the tool for the open-source launch!