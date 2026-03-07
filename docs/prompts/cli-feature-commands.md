---
completed: "2026-03-07"
created: ""
goals_completed: 5
goals_total: 5
origin: migrated by ccs prompts migrate
priority: medium
status: COMPLETED
title: R2Go2 CLI Feature Commands
---

# R2Go2 CLI Feature Commands

## Claude Code Integration Prompts

These prompts are designed for AI coding assistants (like Claude Code) to help with R2Go2 development and usage.

## Development Prompts

### Adding New Commands

```markdown
Add a new R2Go2 command called `sync` that synchronizes local directories with R2 buckets.

Requirements:
- Support bidirectional sync (local ↔ R2)
- Include --dry-run flag for preview
- Add --delete flag to remove remote files not present locally
- Show progress with file-by-file status
- Handle large file uploads with resume capability
- Implement conflict resolution strategies

The command should follow the existing pattern:
```bash
r2go2 sync <bucket> <local-path> [flags]
```

Integration points:
- Add to cmd/sync.go following the existing command structure
- Use internal/api package for R2 operations
- Include comprehensive error handling
- Add JSON output support for scripting
```

### Testing Commands

```markdown
Write comprehensive unit tests for the listBuckets function in internal/api/buckets.go.

Test cases to include:
- Successful API response with multiple buckets
- Empty bucket list response
- API authentication errors
- Network connectivity issues
- Invalid JSON response handling
- Rate limiting scenarios

Use testify framework and mock the Cloudflare SDK responses.
Achieve >90% code coverage for the function.
```

### Error Handling Improvements

```markdown
Enhance the error handling in the upload command (cmd/upload.go) to include:

1. More specific error messages for different failure modes:
   - File not found
   - Permission denied
   - Bucket not found
   - Network timeout
   - API rate limits

2. Retry logic for transient failures:
   - Exponential backoff
   - Maximum retry attempts
   - Jitter to avoid thundering herd

3. Progress reporting:
   - Show upload percentage
   - Calculate transfer speed
   - Estimated time remaining

4. Validation:
   - File size limits
   - Bucket name validation
   - Object key validation
```

## Usage Prompts for Claude Code

### Common Operations

```markdown
Help me create a backup workflow using R2Go2:

1. Create a bucket named "daily-backups"
2. Upload all files from ./backups directory
3. Set lifecycle policy to delete files after 90 days
4. Generate a JSON report of the operation

Use R2Go2 commands and show the exact bash script I can use.
```

### Automation Scripts

```markdown
Create a bash script that:

1. Lists all R2 buckets using R2Go2 with JSON output
2. For each bucket, show:
   - Bucket name
   - Creation date
   - Estimated size (if available)
3. Generate a summary report in Markdown format

The script should handle errors gracefully and be suitable for daily cron job execution.
```

### CI/CD Integration

```markdown
Create a GitHub Actions workflow that uses R2Go2 to:

1. Deploy build artifacts to R2
2. Create a new bucket for each release version
3. Upload build files with proper object keys
4. Set lifecycle policies for old versions
5. Update a JSON manifest file with deployment information

Include proper secrets management and error handling.
```

## Debugging Prompts

### Troubleshooting Upload Issues

```markdown
I'm having issues uploading files to R2 using R2Go2. The command I'm running:

```bash
r2go2 upload my-bucket ./large-file.zip --key="backups/large-file.zip" --dry-run
```

Error message: "API authentication failed"

Help me debug this by:
1. Checking my environment variable setup
2. Verifying API token permissions
3. Testing API connectivity
4. Showing the exact curl commands R2Go2 would use
```

### Performance Optimization

```markdown
My R2Go2 uploads are slow. Help me optimize the upload performance by:

1. Analyzing current upload speed bottlenecks
2. Implementing concurrent uploads for multiple files
3. Adding chunked upload support for large files
4. Caching bucket information to reduce API calls
5. Implementing resume capability for interrupted uploads

Show the specific code changes needed in the upload command.
```

## Security Prompts

### Security Audit

```markdown
Conduct a security audit of R2Go2 focusing on:

1. API token handling and storage
2. Input validation and sanitization
3. File path traversal prevention
4. Bucket name validation
5. Error message information disclosure
6. Logging of sensitive data

Identify potential security vulnerabilities and suggest remediation steps.
```

### Access Control Implementation

```markdown
Add role-based access control to R2Go2:

1. Support for multiple API tokens with different permissions
2. Command-level permission checks
3. Bucket-level access restrictions
4. Audit logging for sensitive operations
5. Integration with Cloudflare's permission model

Show the implementation plan and code structure.
```

## Documentation Prompts

### API Documentation

```markdown
Generate comprehensive API documentation for R2Go2 including:

1. All command-line options and flags
2. Environment variables and configuration
3. JSON response schemas for each command
4. Error codes and meanings
5. Rate limiting information
6. Best practices and examples

Format as both Markdown for README and as JSON for API reference.
```

### Integration Examples

```markdown
Create integration examples for R2Go2 with popular tools:

1. Python script using subprocess to call R2Go2
2. Node.js wrapper for R2Go2 commands
3. Dockerfile for containerized R2Go2 usage
4. Kubernetes job specifications
5. Terraform configuration examples

Each example should include error handling and best practices.
```

---

These prompts are designed to help AI assistants understand R2Go2's architecture, contribute to its development, and help users troubleshoot common issues effectively.