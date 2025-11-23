# Security Guidelines & Best Practices

This document outlines security requirements and best practices for R2Go2, focusing on credential management, data protection, and secure development practices.

## 🔒 Core Security Principles

### 1. Never Hardcode Credentials
**❌ FORBIDDEN:**
```go
// NEVER do this
const API_TOKEN = "your_hardcoded_token_here"
const ACCOUNT_ID = "your_account_id_here"
```

**✅ REQUIRED:**
```go
// ALWAYS use environment variables or secure storage
apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
```

### 2. No Sensitive Data in Logs
**❌ FORBIDDEN:**
```go
// NEVER log credentials
log.Printf("Using token: %s", apiToken)  // DANGEROUS
fmt.Printf("Account: %s", accountID)    // DANGEROUS
```

**✅ REQUIRED:**
```go
// ALWAYS mask or omit sensitive data
fmt.Printf("Account: %s", maskAccountID(accountID))  // SAFE
// OR
log.Printf("API client initialized")  // NO sensitive data
```

### 3. Input Validation & Sanitization
**✅ REQUIRED:**
```go
// ALWAYS validate all inputs
func validateBucketName(name string) error {
    if len(name) < 3 || len(name) > 63 {
        return errors.New("bucket name length invalid")
    }
    // Additional validation...
    return nil
}
```

## 🔐 Authentication & Authorization

### Environment Variables

**Current Implementation (Phase 1):**
```bash
export CLOUDFLARE_API_TOKEN="your_api_token_here"
export CLOUDFLARE_ACCOUNT_ID="your_account_id_here"
```

**Security Requirements:**
- API tokens must have minimal required permissions
- Tokens should be rotated regularly
- Never include in configuration files
- Use system environment variable protection

### Token Permissions

**Required Minimum Permissions:**
- **Read Operations**: `Account R2:Read`
- **Write Operations**: `R2:Edit`
- **Full Management**: `Account R2:Read` + `R2:Edit`

**Recommended Token Scopes:**
```json
{
  "permissions": {
    "account.r2_storage.edit": "*",
    "account.r2_storage.read": "*"
  },
  "resources": {
    "com.cloudflare.api.account.*": "*"
  }
}
```

## 🛡️ Data Protection

### Sensitive Data Masking

**Implementation:**
```go
// maskAccountID masks sensitive information in output
func maskAccountID(accountID string) string {
    if len(accountID) <= 8 {
        return strings.Repeat("*", len(accountID))
    }
    return accountID[:4] + strings.Repeat("*", len(accountID)-8) + accountID[len(accountID)-4:]
}

// maskAPIToken masks API tokens for display
func maskAPIToken(token string) string {
    if len(token) <= 8 {
        return strings.Repeat("*", len(token))
    }
    return token[:4] + strings.Repeat("*", len(token)-8)
}
```

### Error Message Sanitization

**✅ SAFE:**
```go
return fmt.Errorf("authentication failed: invalid credentials")
```

**❌ UNSAFE:**
```go
return fmt.Errorf("authentication failed for token %s", token)
```

## 📁 File System Security

### Configuration Files (Future Phase 2)

**Planned Secure Configuration:**
```yaml
# .r2go2.yaml (planned for Phase 2)
profiles:
  default:
    account_id: "encrypted_value"
    api_token: "encrypted_value"
    api_token_source: "keyring"  # env, file, keyring
```

**Security Requirements:**
- Sensitive fields encrypted at rest
- File permissions: 600 (user read/write only)
- Include in `.gitignore`
- Optional integration with system keychain

### Temporary Files

**Security Practices:**
```go
// Use secure temporary file creation
tmpFile, err := os.CreateTemp("", "r2go2-upload-*.tmp")
if err != nil {
    return err
}
defer os.Remove(tmpFile.Name())
defer tmpFile.Close()

// Set secure permissions
if err := os.Chmod(tmpFile.Name(), 0600); err != nil {
    return err
}
```

## 🌐 Network Security

### HTTPS Only

All API communications use HTTPS:
```go
// Cloudflare API endpoints are HTTPS only
const apiBaseURL = "https://api.cloudflare.com/client/v4"
```

### TLS Configuration

**Current:** Uses Go's default TLS configuration
**Future:** Consider custom TLS settings for enterprise environments

### Timeout and Retry Security

```go
// Implement timeouts to prevent hanging
client := &http.Client{
    Timeout: 30 * time.Second,
}

// Implement exponential backoff for retries
func withRetry(fn func() error, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        if err := fn(); err == nil {
            return nil
        }
        time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
    }
    return fmt.Errorf("max retries exceeded")
}
```

## 🔍 Input Validation

### Bucket Name Validation

```go
func validateBucketName(name string) error {
    if name == "" {
        return errors.New("bucket name cannot be empty")
    }
    if len(name) < 3 || len(name) > 63 {
        return errors.New("bucket name must be between 3 and 63 characters")
    }
    // AWS S3 naming rules apply to R2
    if !regexp.MustCompile(`^[a-z0-9.-]+$`).MatchString(name) {
        return errors.New("bucket name can only contain lowercase letters, numbers, dots, and hyphens")
    }
    if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
        return errors.New("bucket name cannot start or end with a hyphen")
    }
    return nil
}
```

### File Path Validation

```go
func validateFilePath(path string) error {
    // Prevent path traversal
    if strings.Contains(path, "..") {
        return errors.New("file path cannot contain '..' for security reasons")
    }

    // Check file exists and is readable
    info, err := os.Stat(path)
    if os.IsNotExist(err) {
        return fmt.Errorf("file does not exist: %s", path)
    }
    if err != nil {
        return fmt.Errorf("cannot access file: %w", err)
    }

    // Check it's a file (not directory)
    if info.IsDir() {
        return errors.New("path must be a file, not a directory")
    }

    return nil
}
```

### Object Key Validation

```go
func validateObjectKey(key string) error {
    if key == "" {
        return errors.New("object key cannot be empty")
    }
    if len(key) > 1024 {
        return errors.New("object key too long (max 1024 characters)")
    }
    // Prevent path traversal in object keys
    if strings.Contains(key, "..") {
        return errors.New("object key cannot contain '..'")
    }
    return nil
}
```

## 🚨 Error Handling Security

### Safe Error Messages

**✅ SECURE:**
```go
func (c *Client) CreateBucket(name string) (*Bucket, error) {
    if err := validateBucketName(name); err != nil {
        return nil, fmt.Errorf("invalid bucket name: %w", err)
    }

    // API call...
    if apiErr != nil {
        // Don't expose API response details
        return nil, fmt.Errorf("failed to create bucket: %w", apiErr)
    }

    return bucket, nil
}
```

### Audit Logging (Future)

**Planned Implementation:**
```go
type AuditLog struct {
    Timestamp time.Time `json:"timestamp"`
    Operation string    `json:"operation"`
    Resource  string    `json:"resource"`
    Success   bool      `json:"success"`
    Error     string    `json:"error,omitempty"`
    UserID    string    `json:"user_id,omitempty"`
    IPAddress string    `json:"ip_address,omitempty"`
}

func (c *Client) logAudit(operation, resource string, success bool, err error) {
    audit := AuditLog{
        Timestamp: time.Now().UTC(),
        Operation: operation,
        Resource:  resource,
        Success:   success,
    }

    if err != nil {
        audit.Error = err.Error()
    }

    // Log to secure audit system
    c.writeAuditLog(audit)
}
```

## 🔒 Credential Storage Roadmap

### Phase 1: Environment Variables ✅
- Current implementation
- Basic but secure
- No persistent storage

### Phase 2: Encrypted Configuration (Planned)
```go
type SecureConfig struct {
    EncryptedFields map[string][]byte
    PlaintextFields map[string]string
}

func (c *SecureConfig) EncryptField(key, value string) error {
    // Encrypt using system key or derived key
    encrypted, err := encrypt(value, c.getEncryptionKey())
    if err != nil {
        return err
    }
    c.EncryptedFields[key] = encrypted
    return nil
}

func (c *SecureConfig) DecryptField(key string) (string, error) {
    encrypted, exists := c.EncryptedFields[key]
    if !exists {
        return "", fmt.Errorf("field not found")
    }

    return decrypt(encrypted, c.getEncryptionKey())
}
```

### Phase 3: System Keychain Integration (Planned)
```go
type KeychainStorage struct {
    ServiceName string
}

func (k *KeychainStorage) GetCredentials() (*Credentials, error) {
    // Integration with system keychain
    // macOS: Keychain
    // Linux: libsecret
    // Windows: Windows Credential Manager
}
```

### Phase 4: Cloud Key Management (Planned)
```go
type CloudKMS struct {
    Provider string // aws, gcp, azure, hashicorp-vault
    Config   map[string]interface{}
}

func (k *CloudKMS) Encrypt(data []byte) ([]byte, error) {
    // Cloud-based encryption
}
```

## 🛡️ Hardening Guidelines

### Build Security

**Makefile Security:**
```makefile
# Ensure no debug symbols in production builds
build-prod:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build -ldflags "-s -w $(LDFLAGS)" -o $(BINARY) .

# Static analysis
security-scan:
	@echo "🔒 Running security scan..."
	@go list -json -m all | nancy sleuth
	@gosec ./...
```

### Dependency Security

**Regular Security Updates:**
```bash
# Check for vulnerabilities
go list -json -m all | nancy sleuth

# Update dependencies
go get -u ./...
go mod tidy

# Security audit
gosec ./...
```

### Runtime Security

**Memory Safety:**
```go
// Use defer for cleanup
file, err := os.Open(path)
if err != nil {
    return err
}
defer file.Close()

// Clear sensitive memory
func clearPassword(password []byte) {
    for i := range password {
        password[i] = 0
    }
}
```

## 📋 Security Checklist

### Development Phase
- [ ] No hardcoded credentials
- [ ] All inputs validated
- [ ] Error messages sanitized
- [ ] Sensitive data masked in logs
- [ ] Dependencies scanned for vulnerabilities
- [ ] Code reviewed for security issues

### Deployment Phase
- [ ] Environment variables properly set
- [ ] File permissions secure (600 or better)
- [ ] TLS/HTTPS enforced
- [ ] Audit logging enabled
- [ ] Monitoring and alerting configured
- [ ] Backup and recovery procedures tested

### Operational Phase
- [ ] Regular security updates
- [ ] Log monitoring for suspicious activity
- [ ] Access reviews and permissions audits
- [ ] Incident response procedures
- [ ] Security training for team members
- [ ] Compliance requirements verified

## 🚨 Incident Response

### Security Incident Response

1. **Immediate Response**
   - Identify and contain the breach
   - Rotate all exposed credentials
   - Enable additional logging

2. **Investigation**
   - Analyze logs and access patterns
   - Identify affected resources
   - Determine root cause

3. **Recovery**
   - Apply security patches
   - Update security procedures
   - Communicate with stakeholders

4. **Prevention**
   - Implement additional security measures
   - Update monitoring and alerting
   - Conduct security training

---

**Last Updated**: 2025-11-24
**Security Status**: Phase 1 (Environment Variables)
**Next Security Milestone**: Phase 2 (Encrypted Configuration)
**Review Date**: Quarterly or after security incidents

---

*This document is part of R2Go2's security framework. All developers must read and follow these guidelines.*