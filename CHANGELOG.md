# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Enhanced CLI Features**: Professional-grade file operations with real-time monitoring
- **Progress Monitoring System**: Real-time progress bars with speed and ETA calculation
- **Batch Operation Manager**: Concurrent processing with queue management and statistics
- **Smart Error Handling**: Intelligent retry logic with exponential backoff and contextual suggestions
- **Interactive Confirmations**: Safe operations with detailed previews and user feedback
- **Enhanced Copy Command**: Advanced file copying with resume capability and integrity verification
- **Performance Optimization**: Adaptive chunking, parallel processing, and memory-efficient streaming
- **User Experience Enhancements**: Professional output formatting and comprehensive statistics

### Improved
- **Documentation**: Comprehensive usage guide and enhanced CLI features documentation
- **Architecture**: Clean separation of concerns with dedicated packages for CLI enhancements
- **Error Recovery**: Robust error classification and recovery strategies

### Technical
- **8 new Go packages** for enhanced functionality (`internal/cli/*`)
- **1000+ lines** of production-ready Go code
- **Thread-safe operations** with proper goroutine coordination
- **Memory-efficient streaming** for large file operations

## [0.10.0] - 2026-05-24

### Added
- # FEAT-001: TUI command palette with fuzzy search

**Type**: feature
**Status**: closed
**Plan**: docs/planning-mode/2026-05-18-tui-command-palette.md
**Created**: 2026-02-26

## Description

Ctrl+P fuzzy search across all available commands, context-aware, like VS Code/lazygit
- comprehensive 360° project audit — 70.8/100 (Mature) (commit:390657f9)
- add command palette with fuzzy search (FEAT-001) (commit:14a1e739)
- add service interfaces for testability (TASK-002) (commit:decbba0f)

### Fixed
- correct number shortcut tests + file 3 bugs + brainplan FEAT-001/TASK-002 (commit:8b527f9a)
- wire metadata and TTL options into Put() (commit:43201691)
- use local kvForce flag instead of shared workerForce variable (commit:2e00505e)
- render progress bar during download transfer (commit:22411727)

## [0.9.0] - 2026-05-18

### Added
- implement CORS management via Transform Rules (ROAD-016) (commit:c6bce930)
- add cosmoflare status dashboard command (ROAD-050) (commit:b6483fb5)
- add FirewallService and CLI command (commit:239bbe62)
- add EmailService and CLI command (commit:eec97a47)
- add PageRuleService and CLI command (commit:6345c93c)
- add cosmoflare domains CLI command (commit:f62228a6)
- add cosmoflare doctor CLI command (commit:5b5d8d10)
- add DomainService with overview, detail, health enrichment, and formatting (commit:b64f7a87)
- add DoctorService with 4 diagnostic probes (commit:643c3bcd)
- add DomainService with overview, detail, and health enrichment (commit:6ea48be3)
- add HealthcheckService wrapping Cloudflare Healthcheck API (commit:79104da9)

### Changed
- remove dead collectResults() method and update tests (commit:a4996784)

### Fixed
- GlowingText empty-string panic and easeInOutCubic output overflow (commit:c9a9d60b)
- prevent panics from overflow and divide-by-zero in progress bars (commit:8fe8b641)
- race condition in Execute() — goroutine lifecycle restructure (commit:08e6a5eb)
- fetch-before-update in CLI to preserve unmodified rule fields (commit:732af9de)
- address review issues — pointer Verified, flag-changed guards, test fix (commit:4f1053b8)
- restore JSON parsing for action values, fix priority docs (commit:602bc6d9)
- correct exit code documentation in --help (commit:1064b818)
- address review nits — comment accuracy, SOA→PrimaryNS rename (commit:caa824ab)
- address review issues — JSON duration types, DNS consistency, probe errors (commit:b57927a9)
- fetch-then-merge Update to avoid zero-value overwrites (commit:785fe4a4)

## [0.8.0] - 2026-05-16

### Added
- Add DNS Records service (ROAD-035)
- Add Zone Management service (ROAD-036)
- Add SSL/TLS Management service (ROAD-037)
- Add Cache Management service (ROAD-038)
- Add Page Rules service (ROAD-039)
- Add WAF/Firewall service (ROAD-040)
- Add Email Routing service (ROAD-041)
- Add D1 Database service (ROAD-042)
- Shell completion already implemented (ROAD-055)
- add Page Rules, WAF, Email Routing, D1 services (commit:85e86b00)
- add DNS, Zone, SSL/TLS, and Cache services (commit:ff3e694b)

### Changed
- Cosmoflare rebrand — README, PRODUCT-VISION constitution, binary alias
- Test coverage improvements across 7 internal packages (UnitTesting merge)
- replace all fmt.Scanln with InputReader injection (commit:6ff75d7d)

### Fixed
- resolve race conditions in Execute/waitForCompletion (commit:b3e5a7f7)

## [0.7.1] - 2026-05-16

### Fixed
- resolve duplicate status field in phase4 prompt frontmatter (commit:4ad3fea4)
- add mapstructure tags to Profile struct (commit:9619ec09)
- guard TestNewClientValidation against env vars (commit:b2962328)

## [0.7.0] - 2026-05-14

### Added
- Pre-signed URL generation for temporary download access
- Pipe/stdin support: upload from stdin, download to stdout
- Bucket comparison tool (r2go2 compare)
- Analytics command with per-bucket breakdown
- add presign, pipe support, compare, analytics commands (commit:9dc2417f)
- add pre-signed URL generation (commit:f1186400)

## [0.5.0] - 2026-05-14

### Added
- Workers and KV services (Phase 3)
- Shell completions with dynamic bucket names
- Retry logic with exponential backoff
- improve shell completions with dynamic bucket names (commit:d4df080e)
- add retry logic with exponential backoff (commit:466516e1)
- add Workers and KV services (Phase 3) (commit:551a07de)

## [0.4.0] - 2026-05-13

### Added
- add --json output to all commands and wire progress bars (commit:fc69600b)
- implement multipart upload for large files (commit:48a6ec8b)

### Fixed
- use composite ETag from CompleteMultipartUpload output (commit:e84e13ed)

## [0.3.1] - 2026-05-08

### Fixed
- update Go version, remove mock API client, add LICENSE (commit:d5c3cbe4)
- resolve 6 CLI bugs across bucket, object, and root commands (commit:5f54356c)

## [0.3.0] - 2026-05-08

### Added
- Extract public Go library into pkg/r2go2/ with real S3/Cloudflare API wiring
- extract public library into pkg/r2go2/ (commit:d5352b38)
- add animation suppression for faster tests (commit:6155b287)
- add InputReader interface for testability (commit:6dbf526b)

### Changed
- Wire CLI commands to pkg/r2go2 library replacing internal/api stubs
- wire CLI commands to pkg/r2go2 library (commit:a15ece85)

### Fixed
- correct binary name and add CCS integration (commit:386a8a4a)
- resolve 5 critical bugs from codebase audit (commit:36c16c53)

## [0.2.2] - 2026-03-07

### Changed
- Consolidate formatBytes (6x) and maskAccountID (3x) into shared internal/utils package
- consolidate formatBytes (6x) and maskAccountID (3x) into internal/utils (commit:bec5f132)

### Fixed
- resolve 38 go vet issues across interactive and cli packages (commit:eb0c3b84)

## [0.2.1] - 2026-03-02

### Added
- Upgrade project to CCS 2026.02 standards (migrations, USAGE.md, YAML frontmatter)
- Comprehensive codebase analysis with 4 Opus agents (architecture, API, TUI, testing)
- Initialize issues system with 6 bugs, 4 features, 3 tasks
- Initialize roadmap system with 8 strategic items (ROAD-000 through ROAD-007)
- initialize issue and roadmap tracking from codebase analysis (commit:896b979a)
- complete TUI installer with wizard handlers and user-local install (commit:3b963938)

### Changed
- extract S3API interface for testability (commit:c4a90c86)

### Fixed
- allow docs/sessions/ files in gitignore (commit:0d71e97d)

## [0.1.0] - 2025-11-24

### Added
- Initial release of R2Go2 CLI tool
- Universal versioning system setup
- Git hooks for automated build tracking and changelog generation

<!-- generated by git-cliff -->