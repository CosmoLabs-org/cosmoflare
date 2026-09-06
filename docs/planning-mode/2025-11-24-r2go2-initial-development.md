---
branch: r2go2-initial-development
completed: "2026-03-07T00:00:00-03:00"
created: "2025-11-24T00:00:00-03:00"
goals_completed: 16
goals_total: 16
origin: migrated by ccs prompts migrate
priority: medium
status: COMPLETED
deliverables:
  - P-01: Initial R2Go2 CLI and library implementation
title: R2Go2 Initial Development Planning
---

# R2Go2 Initial Development Planning

**Date**: 2025-11-24
**Project**: R2Go2 - Cloudflare R2 CLI Tool
**Company**: CosmoLabs
**Status**: Planning Phase

## Project Overview

R2Go2 is a production-ready CLI tool for managing Cloudflare R2 buckets. Built with Go 1.21+, Cobra for CLI commands, and Cloudflare's official Go SDK.

## Core Requirements

### Commands
- `create <bucket-name>`: Create new R2 bucket
- `list`: List all buckets with names and creation dates
- `delete <bucket-name>`: Delete bucket (with confirmation)
- `upload <bucket> <local-file> --key=<object-key>`: Upload file to bucket
- `policy <bucket> set --days=30`: Set lifecycle policy for object deletion

### Technical Requirements
- **Authentication**: CLOUDFLARE_API_TOKEN environment variable
- **Endpoint**: https://api.cloudflare.com/client/v4/accounts/{account_id}/r2/buckets
- **Account ID**: From CLOUDFLARE_ACCOUNT_ID env var or prompt
- **Error Handling**: Graceful exits with usage tips
- **JSON Output**: --json flag for scripting
- **Global Flags**: --account-id, --dry-run

### Architecture
- **Structure**: cmd/root.go, internal/api, main.go
- **Testing**: Unit tests for core functions
- **Build**: Makefile for cross-compilation

## Implementation Plan

### Phase 1: Project Setup ✅
- [x] Initialize Go module with dependencies
- [x] Set up universal versioning system
- [x] Create project documentation structure

### Phase 2: Core Implementation
- [ ] Initialize go.mod with required dependencies
- [ ] Create main.go entry point
- [ ] Implement cmd/root.go with Cobra setup
- [ ] Create internal/api package for R2 operations

### Phase 3: Command Implementation
- [ ] cmd/create.go - Bucket creation
- [ ] cmd/list.go - Bucket listing
- [ ] cmd/delete.go - Bucket deletion
- [ ] cmd/upload.go - File uploads
- [ ] cmd/policy.go - Lifecycle policies

### Phase 4: Testing & Documentation
- [ ] Unit tests for core functions
- [ ] Makefile for building
- [ ] Comprehensive README
- [ ] Integration testing

## Dependencies

```go
// Required packages
github.com/spf13/cobra         // CLI framework
github.com/cloudflare/cloudflare-go  // Cloudflare SDK
github.com/spf13/viper         // Configuration management
github.com/stretchr/testify    // Testing framework
```

## Key Design Decisions

### Authentication Strategy
- Environment variables for API token and account ID
- Fallback to interactive prompts if not set
- Secure credential handling (no logging of tokens)

### Error Handling
- Structured error responses with clear messages
- Graceful degradation for network issues
- User-friendly error messages with next steps

### Output Format
- Human-readable default output
- Structured JSON for scripting (--json flag)
- Color-coded output with emojis for better UX

### CLI Design
- Follow Cobra best practices
- Consistent flag patterns across commands
- Help-driven design with comprehensive usage information

## Testing Strategy

### Unit Tests
- Test R2 API wrapper functions
- Mock Cloudflare SDK responses
- Error scenario testing
- Input validation testing

### Integration Tests
- End-to-end command testing
- Real Cloudflare R2 API testing (with test account)
- File upload/download testing

### Performance Tests
- Large file upload handling
- Concurrent operations
- Memory usage optimization

## Security Considerations

- API token security (no logging, secure storage)
- Input validation to prevent injection
- File path validation for uploads
- Bucket name validation (R2 naming rules)

## Future Enhancements

### Version 2.0 Features
- Batch operations for multiple files
- Sync functionality (local ↔ R2)
- Advanced lifecycle policies
- Bucket metrics and statistics
- Configuration file support

### CLI Enhancements
- Auto-completion support
- Interactive mode
- Progress bars for uploads
- Color themes

## Release Plan

### v0.1.0 (Initial Release)
- Core bucket operations
- File upload functionality
- Basic lifecycle policies
- JSON output support

### v0.2.0 (Enhanced Features)
- Batch operations
- Improved error handling
- Performance optimizations
- Extended testing coverage

## Success Metrics

- **Usability**: Intuitive command structure
- **Reliability**: Comprehensive error handling
- **Performance**: Efficient API usage
- **Maintainability**: Clean, testable code
- **Documentation**: Complete usage guides

---

*Last updated: 2025-11-24*