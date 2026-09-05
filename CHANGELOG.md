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

## [0.19.0] - 2026-09-05

### Added
- bridge SSE metrics channel to React Query cache (commit:be8637b2)
- wire MetricsProducer with --metrics-interval flag (commit:7f954756)
- add MetricsProducer with subscriber gating and delta detection (commit:4a7bdd24)

### Fixed
- verify downloaded binary checksums before install (commit:ef7f8ed6)
- wire transport options, implement profile resolution, drop no-op options (commit:3b21556b)
- repoint version stamping and installers at cosmoflare (commit:342cf8e8)
- eliminate shared-state data races (BUG-030) (commit:9606ac6e)
- implement cf-* stylesheet, visible health state, and ARIA live regions (commit:5de362d6)
- enforce CheckUpload on upload paths and default-exclude sensitive files (commit:5ac6e976)
- abort multipart upload when reader is shorter than declared size (commit:3e826448)
- never report deletions for paths that failed to scan (commit:b105502c)
- emit JSON error envelope when config validation fails in --json mode (commit:eeee5864)
- paginate remote object listing across NextToken pages (commit:06c0718e)
- resolve three v1.1 review bugs (BUG-022, BUG-023, BUG-024) (commit:4f501e05)

## [0.18.0] - 2026-09-05

### Added
- bridge SSE metrics channel to React Query cache (commit:be8637b2)
- wire MetricsProducer with --metrics-interval flag (commit:7f954756)
- add MetricsProducer with subscriber gating and delta detection (commit:4a7bdd24)

### Fixed
- verify downloaded binary checksums before install (commit:ef7f8ed6)
- wire transport options, implement profile resolution, drop no-op options (commit:3b21556b)
- repoint version stamping and installers at cosmoflare (commit:342cf8e8)
- eliminate shared-state data races (BUG-030) (commit:9606ac6e)
- implement cf-* stylesheet, visible health state, and ARIA live regions (commit:5de362d6)
- enforce CheckUpload on upload paths and default-exclude sensitive files (commit:5ac6e976)
- abort multipart upload when reader is shorter than declared size (commit:3e826448)
- never report deletions for paths that failed to scan (commit:b105502c)
- emit JSON error envelope when config validation fails in --json mode (commit:eeee5864)
- paginate remote object listing across NextToken pages (commit:06c0718e)
- resolve three v1.1 review bugs (BUG-022, BUG-023, BUG-024) (commit:4f501e05)

## [0.17.0] - 2026-06-21

### Added
- emit notifications on cloudflare_online transitions (BR-07) (commit:3cd093a1)
- real-time notifications panel (P-08) (commit:be5ee69f)
- multi-account read-only dashboard (P-07) (commit:0b82d646)
- app shell, API/SSE clients, health indicators (P-06) (commit:4302a56e)
- supervise cosmoflare daemon lifecycle from Rust (P-05) (commit:b4d0af3d)
- scaffold Tauri v2 app with sidecar config (P-04) (commit:e7401cd7)
- add SSE /events (metrics/notifications/status) + health (P-03) (commit:b142676a)
- add REST read endpoints over existing services (P-02) (commit:308788d2)
- add cosmoflare serve daemon skeleton with token auth + /healthz (P-01) (commit:cae32ba5)

### Fixed
- independent review of ROAD-080 brainplan — 2 issues (commit:ccaf5a06)
- persist notifications across tabs, harden accounts + SSE health (commit:21cf37f0)
- kill daemon on app exit + make set_child atomic (commit:e3321ea0)
- config init expects .cosmoflare not legacy .r2go2 (commit:c1ed9307)
- kill sidecar child on failed handshake/health (leak hygiene) (commit:886d6f9f)
- provide QueryClientProvider in App (runtime crash on Dashboard) (commit:24130280)

## [0.16.0] - 2026-06-20

### Added
- Domain Management Center — TUI dashboard, redirect visibility, registrar overlay (FEAT-006) (FEAT-006)
- add DomainBrowserModel + quick-add redirect + domains tui command (P-06/P-07) (commit:691e4598)
- add domains command tree (get/stats/ns/redirects) with service factory (P-04) (commit:bea72bb6)
- add redirects command group for modern Redirect Rules (P-05) (commit:5eda1243)
- add NewRedirectServiceFromCreds + NewRegistrarServiceFromCreds (Wave C seam) (commit:403f08b9)
- enrich DomainDetail with redirects+registrar, add SummarizeDomains (P-03) (commit:aaec93b6)
- add RedirectService List/Create/Delete for modern Redirect Rules (P-01) (commit:556c8ca7)
- add RegistrarService registration overlay (P-02) (commit:5dffe87c)

### Fixed
- Removed 7 orphaned R2Go2-era test files that broke 'go test ./...' compilation (BUG-021)
- Keychain layer no longer probes the macOS keychain under tests or when COSMOFLARE_NO_KEYCHAIN=1, eliminating spurious security dialogs
- independent review of Cosmoflare Desktop brainplan — 8 findings (commit:9bea55d6)
- never use OS keychain under test or when disabled (no `security` noise) (commit:da335a3d)
- remove 7 orphaned R2Go2-era test files breaking compilation (BUG-021) (commit:19b61398)
- unique IDs via atomic counter, not UnixNano (BUG-020) (commit:07017e82)

## [0.15.0] - 2026-06-09

### Added
- Add OS keychain credential storage with automatic fallback to file-based config (FEAT-005)
- Add real-time TUI dashboard with live monitoring, bucket CRUD, and object browsing (ROAD-020)
- Add split-pane TUI object browser with folder navigation and object management (ROAD-002)
- Add real S3-to-R2 migration with resume support, concurrent transfers, and ETag verification (ROAD-007)
- replace simulated loop with real orchestrator and verify mode (ROAD-007 P-03/P-04) (commit:e96ef905)
- add checkpoint persistence and worker pool (ROAD-007 P-01/P-02) (commit:3f36fe40)
- integrate BrowserModel into dashboard, remove SectionObjectList (ROAD-002 P-06) (commit:4ba6f1d9)
- add BrowserModel with split-pane layout and folder navigation (ROAD-002 P-03/P-04/P-05) (commit:03d744ce)
- extend ListObjects and DataSource for browser (ROAD-002 P-01/P-02) (commit:17944372)
- add --interval flag, absorb metrics TUI into dashboard (ROAD-020 P-08) (commit:2ee51362)
- wire live data, monitoring, objects, and bucket CRUD into dashboard (ROAD-020 P-03/P-04/P-05/P-06/P-07) (commit:ae0b1b93)
- add DataSource interface with API and null backends (ROAD-020 P-01/P-02) (commit:78a655f8)
- add OS keychain credential storage (FEAT-005) (commit:5d5fd13a)

### Fixed
- remove orphaned internal/api test files and fix TestNewClientValidation env dependency (commit:15752349)

## [0.14.0] - 2026-06-07

### Added
- cosmoflare metrics — real-time TUI dashboard with live R2/Workers/KV stats, --interval flag, --json mode (ROAD-060)
- cosmoflare dev --notify — webhook notifications on dev server start/stop/error events (ROAD-072)
- cosmoflare migrate from-s3 — re-enabled S3-to-R2 migration command (ROAD-019)
- Bubble Tea interactive prompts — text input, list select, and confirm models replace hand-rolled stdin loop (ROAD-008)
- integration test suite with httptest mock servers for R2, Workers, and KV CRUD operations (ROAD-017)
- ~9,300 lines of unit tests across cmd/ and pkg/cosmoflare/ via 18 parallel Sonnet agents — covers command structure, flags, helpers, constructors, and validation
- re-enable S3-to-R2 migration command (ROAD-019) (commit:2033f576)
- add webhook notifications for dev server lifecycle (ROAD-072) (commit:9c691c8d)
- add real-time TUI metrics dashboard (ROAD-060) (commit:6fc9299a)

### Changed
- USAGE.md — replaced 226 r2go2 references with cosmoflare across all CLI examples
- migrate prompts to Bubble Tea models (ROAD-008) (commit:d6963f24)

### Fixed
- internal/migration/s3.go — resolved 11 build errors (missing helpers, AWS SDK call, pointer dereference)
- config.Save() viper .tmp extension bug — replaced with yaml.Marshal for reliable config persistence
- pre-existing test failures in internal/interactive and internal/webhook from r2go2→cosmoflare rename
- correct bucket update and palette render assertions (commit:bbc61391)
- resolve all pre-existing test failures in interactive and config (commit:0f6cd715)
- resolve 11 build errors in s3.go (commit:3ab8302a)

## [0.13.0] - 2026-06-06

### Added
- cosmoflare alerts — alert rules for error rates, storage limits, failures
- cosmoflare terraform — generate .tf files from live Cloudflare state
- cosmoflare validate — config validation against Cloudflare API constraints
- cosmoflare account — multi-account switching
- cosmoflare audit — CLI mutation audit logging with search and export
- cosmoflare wrangler — import wrangler.toml compatibility layer
- cosmoflare mcp — MCP tool server for AI agent integration
- cosmoflare plugin — community extension system with install/remove/run
- cosmoflare stream — video upload, live inputs, signed tokens
- cosmoflare ai — Workers AI inference and AI Gateway management
- cosmoflare export/import — full account config backup and restore
- Resumable multipart uploads with state tracking and progress
- add alert rules service and CLI commands (ROAD-062) (commit:9023cf22)
- add terraform export and import-block commands (ROAD-071) (commit:948e46e7)
- add config validation command (ROAD-070) (commit:59f8e486)
- add multi-account switching for cosmoflare (ROAD-069) (commit:86b2ec15)
- add CLI mutation audit logging (ROAD-068) (commit:4cb9c8e1)
- add wrangler.toml compatibility layer (ROAD-067) (commit:154d7a5c)
- add MCP server for AI agent integration (ROAD-066) (commit:96b7f090)
- add community plugin system for cosmoflare extensions (ROAD-065) (commit:dcccbfcf)
- add resumable multipart uploads with state tracking (ROAD-013) (commit:d1b4c52d)
- implement Cloudflare Stream video service (ROAD-045) (commit:11a1f6df)
- add Workers AI and AI Gateway service (ROAD-049) (commit:aeb056a4)
- add import/export commands for full account config backup (ROAD-053) (commit:a4372749)

### Changed
- GitHub repo renamed from CosmoLabs-org/r2go2 to CosmoLabs-org/cosmoflare
- Go module path updated to github.com/CosmoLabs-org/cosmoflare
- complete GitHub repo rename to CosmoLabs-org/cosmoflare (P-02) (commit:e216d623)

### Fixed
- update r2go2→cosmoflare references in webhook and palette tests (commit:d74d24a4)

## [0.12.0] - 2026-05-30

### Added
- cosmoflare dev — local development proxy server with hot-reload
- cosmoflare logs --follow — real-time log tailing with level filtering
- cosmoflare init — interactive project scaffolding with framework detection
- cosmoflare images — Cloudflare Images upload, variants, delivery
- cosmoflare hyperdrive — database connection pooling management
- cosmoflare diff — compare local config against live Cloudflare state
- cosmoflare cost — monthly cost estimation across R2, Workers, KV
- cosmoflare watch — auto-sync local directory to R2 on file changes
- cosmoflare vectorize — vector database for AI workloads
- cosmoflare sync — rsync-like directory synchronization with R2
- cosmoflare templates — project scaffolding with 5 built-in templates
- cosmoflare apply — declarative config reconciliation against live state
- GitHub Actions CI/CD workflows for cross-platform builds and releases
- implement cosmoflare apply command (ROAD-056) (commit:fdfa0635)
- add cosmoflare templates command (ROAD-059) (commit:27b5d249)
- implement cosmoflare sync command for local-R2 directory synchronization (commit:2be0f07d)
- implement Cloudflare Vectorize service (ROAD-048) (commit:0b3bd5dc)
- add cosmoflare watch command for auto-syncing local directories to R2 (commit:8479f9d9)
- add cosmoflare cost command for monthly cost estimation (ROAD-061) (commit:9b23e914)
- add cosmoflare diff command (ROAD-052) (commit:299002a1)
- implement Cloudflare Hyperdrive service (ROAD-046) (commit:27667177)
- implement Cloudflare Images service (ROAD-044) (commit:b3e6e2fc)
- add cosmoflare init project scaffolding (ROAD-057) (commit:9a4f2bca)
- add logs --follow for real-time log tailing (ROAD-051) (commit:589e7bd1)
- add cosmoflare dev local proxy server (ROAD-058) (commit:5cdcbcce)

### Changed
- Renamed binary from r2go2 to cosmoflare, pkg/r2go2 to pkg/cosmoflare
- rename to Cosmoflare — binary, package, and branding (commit:cc30242f)

### Fixed
- use DefValue instead of GetString in waf flag default test (commit:cc02a21c)

## [0.11.0] - 2026-05-27

### Added
- # FEAT-002: Local HTTP dev server proxying to R2

**Type**: feature
**Status**: closed
**Created**: 2026-02-26

## Description

r2go2 serve command starts a local HTTP server that proxies requests to an R2 bucket for local testing without deploying
- Cloudflare Pages service — project and deployment management (ROAD-043)
- Cloudflare Queues service — queue and consumer management (ROAD-033)
- implement Cloudflare Queues service and CLI commands (commit:b6b62fbe)
- implement Cloudflare Pages service and CLI commands (commit:a4e9309f)

### Fixed
- Sort multipart upload parts before assembly (data corruption prevention)
- Classify S3 NoSuchKey as ErrNotFound sentinel error
- Atomic config file write prevents TOCTOU race
- Fix install.sh binary name case for Linux
- Fix release.yml asset path containing slashes
- Remove bucket update no-op — return clear unsupported error
- sort multipart parts and classify NoSuchKey as ErrNotFound (commit:43198d07)
- audit phase 0 — install.sh case, release paths, atomic config write, bucket update no-op (commit:258e9d5a)

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