# Code Patterns Analysis

**Source**: Synthesized from agent-1-code-quality.md, agent-2-core-logic.md, agent-3-api-design.md

## Strong Patterns

### 1. Functional Options (client.go)

The `R2Client` uses the standard Go functional options pattern for configuration:

```go
client, err := r2go2.NewClient(
    r2go2.WithAccountID("abc123"),
    r2go2.WithAPIToken("token"),
    r2go2.WithEndpoint("https://custom.endpoint"),
)
```

**Why it works**: Extensible without breaking callers. New options can be added without changing the constructor signature. Clean default behavior. Agent-friendly -- options are self-documenting.

**Source**: agent-1-code-quality.md, Strength #1.

### 2. Typed Error Hierarchy (errors.go)

Errors are categorized by kind, enabling callers to match without string parsing:

```go
var notFound *r2go2.R2NotFoundError
if errors.As(err, &notFound) {
    // handle 404
}
```

Error types: `R2NotFoundError`, `R2ValidationError`, `R2AuthError`, `R2RateLimitError`, `R2Error`. Each carries contextual fields (bucket name, object key, underlying cause).

**Why it works**: Callers can make decisions based on error type. No brittle string matching. Good for agent consumers that need programmatic error handling.

**Source**: agent-1-code-quality.md, Strength #2; agent-2-core-logic.md.

### 3. Service Struct Pattern

All 12 services follow the same structural template:

```go
type XxxService struct {
    accountID string
    apiToken  string
    httpClient *http.Client
    baseURL   string
}

func NewXxxService(accountID, apiToken string) *XxxService { ... }
func NewXxxServiceFromCreds(creds Credentials) *XxxService { ... }
```

**Why it works**: Predictable. Once you understand one service, you understand all 12. Consistent constructor pattern. Easy to generate code for new services.

**Source**: agent-1-code-quality.md, Strength #4.

### 4. Input Validation at Boundary

Every public method validates inputs before making API calls:

```go
func (s *DNSService) Create(zoneID string, record DNSRecord) error {
    if zoneID == "" {
        return &R2ValidationError{Message: "zone ID is required"}
    }
    if record.Type == "" {
        return &R2ValidationError{Message: "record type is required"}
    }
    // ... proceed with API call
}
```

**Why it works**: Prevents cryptic API errors. Error messages are actionable. Validates early, fails fast. Good for both human and agent callers.

**Source**: agent-1-code-quality.md, Strength #3.

### 5. Consistent `--json` Output

All commands support `--json` for machine-readable output, making the CLI agent-friendly:

```bash
r2go2 bucket list --json
r2go2 dns list --zone-id abc --json
```

**Source**: agent-3-api-design.md.

## Weak Patterns / Anti-Patterns

### 1. Error Misclassification

Some errors are wrapped with the wrong type. Network errors appear as validation errors; transient failures are reported as permanent. This causes callers to make incorrect retry/bail decisions.

**Impact**: An agent receiving `R2ValidationError` will not retry, even if the underlying cause is a transient network issue. A human will see "invalid input" when the real problem is connectivity.

**Source**: agent-2-core-logic.md.

### 2. Global Mutable CLI State

CLI commands use package-level global variables for flags:

```go
var jsonOutput bool
var profileName string
var accountID string
```

**Why it hurts**: Commands are non-reentrant. Cannot safely test commands in parallel. State leaks between test cases. Makes the CLI harder to embed in other tools.

**Better pattern**: Pass a `CommandContext` struct through the Cobra command tree:
```go
type CommandContext struct {
    JSONOutput  bool
    ProfileName string
    AccountID   string
}
```

**Source**: agent-1-code-quality.md, Additional Findings.

### 3. O(n) List-and-Scan Lookups

`GetBucket` and `BucketExists` list all buckets, then iterate to find the target:

```go
func (c *R2Client) GetBucket(name string) (*BucketInfo, error) {
    buckets, err := c.ListBuckets()
    // ... iterate to find by name
}
```

**Why it hurts**: O(n) on bucket count. Unnecessary API call overhead. The Cloudflare API supports direct lookup by name.

**Source**: agent-1-code-quality.md, Weakness #3.

### 4. Silent Error Suppression in JSON Mode

`printError` in `root.go` discards errors when `--json` is active:

```go
func printError(err error) {
    if jsonOutput {
        return // error silently dropped
    }
    fmt.Fprintf(os.Stderr, "Error: %s\n", err)
}
```

**Why it hurts**: Debugging is impossible in JSON mode. Agents that parse `--json` output receive empty results with no error indication. Should output `{"error": "message"}` instead.

**Source**: agent-3-api-design.md.

### 5. No-Op Command (bucket update)

The `bucket update` command accepts flags but does not apply any changes. It parses input, validates it, and returns success without modifying anything.

**Why it hurts**: Users believe they have updated their bucket configuration. The command is a silent lie.

**Source**: agent-3-api-design.md.

### 6. Legacy Command Duplication

Some operations are accessible through both old and new command paths (e.g., `r2go2 create` and `r2go2 bucket create`). The legacy paths are not deprecated or documented as aliases.

**Source**: agent-3-api-design.md.

## Pattern Health Summary

| Pattern | Health | Notes |
|---------|--------|-------|
| Functional options | Healthy | Idiomatic, extensible |
| Typed errors | Healthy (classification issues) | Good hierarchy, some misclassification |
| Service struct | Healthy (isolation issue) | Consistent but no unified accessor |
| Input validation | Healthy | Thorough, early, actionable |
| JSON output | Healthy (error gap) | Consistent but errors dropped |
| Global CLI state | Unhealthy | Non-reentrant, state leakage |
| Lookup efficiency | Unhealthy | O(n) where O(1) is possible |
| Error output in JSON | Unhealthy | Silent suppression |
