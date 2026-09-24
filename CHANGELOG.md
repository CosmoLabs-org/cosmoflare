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

## [0.31.0] - 2026-09-24

### Added
- Profile-prefix scoping declared at command definitions across the CLI (TASK-011)
- d1 push-sql: batched remote SQL push with retry, split recovery, resume (FEAT-018)
- Long-tail services — first CLI coverage: Logpush, Load Balancer pools/monitors, Waiting Room, Spectrum, Page Shield, Turnstile, Web Analytics (FEAT-037)
- Desktop: notification severity levels and light theme (FEAT-041)
- Structural safety defaults: every registry-destructive command runs dry unless --force; CLI paths derived from the live command tree (TASK-015)
- page-shield, turnstile, web-analytics CLIs + wave-3 registry (commit:4d62dd94)

### Fixed
- assert the SDK's PUT transport; decode-able delete stub (commit:74038865)
- salvage repairs — struct table, capture deadlock, validation creds (commit:b610f7d5)
- stateful stub — re-fetch after PUT returns the updated job (commit:f7f1048d)
- Today() honors the service clock — date-rollover time bomb (commit:b3bf236c)
- salvage repairs — helper name, JSON-escaping-safe assertions, registry entry (commit:f9d129d5)
- stub localStorage in test setup — jsdom ships none (commit:1bf2c0cc)
- register d1 parity in the command registry (commit:b786e50c)

## [0.30.0] - 2026-09-22

### Added
- named environment profiles part 2: resource-prefix scoping across d1/kv/r2/workers, d1 execute --local/--remote, dev --env reconciliation (FEAT-026)
- env profiles part 3: prefix scoping on bucket sub-resources and worker subcommands (FEAT-026)
- serve alert cycle: limits-snapshot 30m TTL cache + DNS usage 401/403 fail-fast (FEAT-014)
- cmd coverage wave: 8 test suites across bucket/worker/kv/wrangler/copy/durable-objects/terraform/alerts + CopyBuffer zero-chunk crash fix
- command registry spine wave 1: internal/cmdmanifest with r2 pilot + danger-stamped audit mutations (FEAT-020)
- first-run onboarding: bare-cosmoflare setup nudge + wizard points to status/doctor (FEAT-017)
- FEAT-020 wave 2: command registry covers worker/kv/d1/dns (58 entries, Qwen-verified permissions); destructive deletes run dry by default unless --force
- Tunnels management: tunnel CRUD, connector token, connections, cleanup (FEAT-035)
- Account management: members, roles, and audit-log query with NDJSON/CSV export (FEAT-036)
- Registrar operations: register, transfer, renew, auto-renew, lock/unlock, contacts, DNSSEC (FEAT-030)
- fEAT-036 — Account management + audit logs (commit:dda27057)

### Changed
- treat reader errors as terminal in all prompt steps (TASK-016) (commit:04555289)
- shared runGlobalsSnapshot helper (TASK-014) (commit:70578840)
- simplify pass on 8e77ac2..HEAD — 9 cleanups from 4 review agents (commit:c407fea9)
- /simplify pass — dead cache field, O(1) condition lookup, unit-from-registry, fixture dedup (commit:defea3e4)

### Fixed
- alert condition descriptor registry — one source for validation, errors, help (FEAT-015)
- Alerts: rules can be created and updated into a disabled state (TASK-013)
- Error classification: envelope auth failures map correctly instead of 502; one shared HTTP-status seam (TASK-012)
- carry API messages verbatim; array-shaped transfer stub (commit:2027daed)
- read cloudflare-go status structurally in ErrorStatus (commit:a4e2e48d)
- drop unused os import in audit command (commit:7069ed7d)
- drop duplicate test, bound prompt loop, dry-run before service (commit:fe7d3900)
- match indented JSON in probe report assertion (commit:10b81db9)
- make Execute help test hermetic — rebind rootCmd writers (commit:778ca50c)
- disabled-rule fixture writes YAML directly — Create force-enables (commit:f5f3fa9e)
- dedupe containsString, scope delete force flag to credential guards (commit:9a8b2f57)
- zero ChunkSize crashed CopyBuffer — plus cmd coverage tests (commit:70504c7f)
- repair salvaged interactive rewrite — stray paren, unused servers (commit:a7b7145d)
- replace salvaged self-recursive runFieldChecks with subtest loop (commit:134af9fc)
- drop dead prefix discard in runBucketUpdate (commit:72f25be8)
- complete salvaged wave — execLocalStmt compile fix + part-2 tests (commit:b866e2b7)

## [v0.29.0] - 2026-09-17

### Added
- redact-safe token retrieval (FEAT-029 part 1) (927da244)
- permission catalog pack + 10405 scope decode (FEAT-011) (c949271b)
- opt-in env-gated update check (COSMOFLARE_UPDATE_CHECK, default off) (FEAT-043)
- named environment profiles part 1: --env flag, profile plan_tier/resource_prefix, limits plan-tier awareness (FEAT-026)
- SSL custom hostnames (SaaS) CRUD + verification status verbs (FEAT-031)
- WAF lists CRUD + items add/replace + managed-rulesets update (FEAT-032)

## [v0.28.2] - 2026-09-16

### Added
- wire wave 2-3 groups + USAGE.md Workers section (FEAT-021) (a6e54045)
- worker types .d.ts generator (FEAT-021) (b32e4801)
- bindings list + live tail commands (FEAT-021) (9ea29030)
- workers.dev subdomain get/set (FEAT-021) (be2101b4)
- custom domain attach/detach/list (FEAT-021) (1dcbe2dd)
- deployments list/view/rollback + wave-1 wiring (FEAT-021) (326511a6)

### Fixed
- go install via proxy.golang.org failed for v0.28.0/v0.28.1 (emoji-named path in the tagged tree); directory removed, release gate now rejects non-ASCII paths (BUG-052) (b2d4eaf5)
- zero AccountID/APIToken in cron globals helper — full-suite order dependence (3282e614)
- route list flag check uses LocalFlags — cobra merges inherited persistent flags on root Execute, making Flags() order-dependent (6613958f)

## [0.28.1] - 2026-09-15

### Added
- typed daemon wire contract + TS codegen (FEAT-042) (commit:19449791)

### Changed
- split loadBuiltinThemes/createDefaultTutorials under funlen=80 (TASK-009) (commit:4c43a61b)
- split browser handleKey/renderRightPane under funlen=80 (TASK-009) (commit:9f4c82fe)
- split installer TUI god functions under funlen=80 (TASK-009) (commit:eb370472)
- split config/status/setup/mcp functions under funlen=80 (TASK-009) (commit:fb025d98)
- split 3 long fns under funlen=80 (TASK-009) (commit:fd0eee51)
- split cache/compare/watch functions under funlen=80 (TASK-009) (commit:e2553924)
- split file-table funcs for funlen (TASK-009) (commit:2f030734)

### Fixed
- dist archives carried ../ path entries (commit:bfd2aca5)

## [0.28.0] - 2026-09-14

### Added
- output presenter — pattern + first conversion (FEAT-040 slice) (commit:a6c7df10)
- parallel executor + unified transport policy (FEAT-038/039) (commit:6da6553b)

### Changed
- species-2 presenter conversion for compare.go (FEAT-040) (commit:69c67400)
- presenter conversion for knowledge_cmd (FEAT-040) (commit:cb7551fe)
- hand-convert 6 irregular species-2 stragglers (FEAT-040) (commit:89834823)
- presenter conversion for auth_permissions.go (FEAT-040) (commit:bd1b024c)
- presenter conversion for domains_get.go (FEAT-040) (commit:83f0926d)
- species-2 presenter conversion for bucket_lifecycle.go (FEAT-040) (commit:7bbd0e76)
- presenter conversion for bucket_policy.go (commit:17d8bbc6)
- presenter conversion for bucket_notifications (FEAT-040) (commit:b449051e)
- presenter conversion for d1_migrations (FEAT-040) (commit:6c37f981)
- species-2 presenter conversion for pages_deployment.go (FEAT-040) (commit:032dcb70)
- presenter conversion queue_send.go (FEAT-040) (commit:e91e48f2)
- presenter conversion for terraform.go (FEAT-040) (commit:0316d6a9)
- presenter conversion bucket_domain.go (FEAT-040) (commit:3c20cd10)
- finish species-2 conversion for email.go (FEAT-040) (commit:62506563)
- presenter species-1 sweep — 196 error branches collapsed (commit:e06d5130)

### Fixed
- JSON-mode errors now exit non-zero: --json failures print the parseable error envelope to stdout and return exit code 1 (previously exit 0 — breaking the deterministic exit-code contract for agent consumers). Diagnostics mirror to stderr.
- JSON-mode errors exit non-zero — envelope on stdout, exit 1 (commit:738623e7)

## [0.27.0] - 2026-09-13

### Fixed
- Sync engine data-loss fixes from 360-degree audit: direction-aware deletes for sync down --delete (BUG-040), exclude-protected deletes (BUG-041), timeout-free S3 data-plane client so large transfers survive past 30s (BUG-042), cancellation-safe multipart aborts (BUG-043). Also: auth rotate --revoke-old no longer revokes the new token (BUG-044), apply deletes gated behind --delete-unmanaged (BUG-049), KV duplicate-title ambiguity refused (BUG-048), --include sync filter implemented (BUG-051), desktop WCAG 4.1.3 live region (BUG-047), release archives no longer bundle docs/ (BUG-045), Dockerfile toolchain pinned (BUG-050).
- resolve all 12 critical bugs from the 2026-09-13 360-degree audit (commit:f29430a6)

## [0.26.0] - 2026-09-13

### Added
- D1 depth: migrations, time-travel restore, backups, export/import (FEAT-022) (FEAT-022)
- Queues producer surface: send, send-batch, DLQ, consumer management (FEAT-024) (FEAT-024)
- Rate-limit-aware shared client: Retry-After, backoff, retry flags (FEAT-025) (FEAT-025)
- Vectorize vector CRUD: upsert, get, delete, list vectors, namespaces (FEAT-027) (FEAT-027)
- R2 bucket policy get/set (FEAT-028) (FEAT-028)
- Domain fleet status matrix — all domains, all protection statuses, one view (FEAT-033) (FEAT-033)
- pages deployment view/retry/logs command group (commit:00dc4b04)
- pages domain command group (commit:a346ff64)
- pages env command group (commit:96119d28)
- Pages deployment retry and logs library (commit:95897f8b)
- Pages custom domains library (commit:b9da2795)
- Pages env vars and secrets library (commit:9f3fdd0e)
- do namespaces/objects/inspect commands (commit:1e9c078a)
- durable objects live-state service (commit:c702026f)
- file four coverage-domain features from pillar-A research (commit:d1858656)
- domain fleet status service — per-zone protection matrix (commit:0cfbedf1)
- promote FEAT-031/032; file domain fleet status matrix (commit:9fd247ec)
- d1 import — batched, resumable, TOOBIG split-retry (commit:994c19f0)
- statement-aware SQL splitter for D1 import (commit:89c1cdc6)
- d1 export command (commit:a11dbc66)
- d1 time-travel and export commands (commit:78c6a59c)
- d1 time-travel restore with quota pre-flight (commit:52552acc)
- rate-limit-aware REST client retries (commit:74718b24)
- d1 migrations commands (commit:4992a272)
- d1 migrations — create, list, apply (commit:836ee6a0)
- bucket policy commands (commit:be1360c8)
- vectorize vector commands (commit:870cf701)
- vectorize vector CRUD methods (commit:8578b871)
- queue consumer update/remove and dlq commands (commit:26f2743e)
- queue consumer update + DLQ configuration (commit:a131b286)
- auth permissions list command (commit:edf14c44)
- embedded Cloudflare API token permission manifest (commit:741e6bea)
- queue send and send-batch commands (commit:52333397)
- queue producer — Send and SendBatch service methods (commit:fee8bd4e)

### Changed
- consolidate not-found detectors into isNotFound (commit:faebd780)

## [0.25.0] - 2026-09-10

### Added
- decode, knowledge, ratelimit list/create commands
- ratelimit probe command + create --probe + traffic-class advisory
- knowledge layer: endpoint registry, plan caps, error decoding
- rate-limit traffic-class matrix with evidence sources
- RateLimitService with preflight validation and entrypoint PUT flow
- rate-limit trip prober with tripped/not-counted/inconclusive verdicts
- bounded trip prober with classify verdict engine (commit:fda50bd6)
- decode, knowledge list, ratelimit list/create (commit:23c9e0b6)
- add RateLimitService with preflight + entrypoint PUT flow (commit:3371e96c)
- route-blocking transport + central CF error decoding (commit:b9477f4c)
- ratelimit seed pack from MyCarGuide evidence (commit:de1ca108)
- pack types, embedded loader, route matching (commit:67814ce9)

### Changed
- consolidate review findings from /simplify pass (commit:c3f466ab)

### Fixed
- config delete now honors --dry-run (previously deleted the profile)
- webhook retries now send a full request body (previously empty on retry)
- honor dry-run in config delete + add config/cache tests (commit:de460543)
- rebuild request per retry attempt (commit:e76f3325)

## [0.24.0] - 2026-09-10

### Added
- Domain Center residuals — legacy pagerules merge + redirect-target attention check (FEAT-010) (FEAT-010)
- feat(doctor): redirect-target probe section — cmd supplies destinations, report carries results
- feat(cmd): domains stats --check-redirects probe pass
- feat(tui): domain detail pane renders redirect-issue badge when present
- feat(domains): merge legacy forwarding_url page rules via WithPageRules
- feat(domains): redirect-issue classification feeds needs-attention
- feat(domains): redirect prober with loop detection
- feat(redirects): map legacy forwarding_url page rules

### Fixed
- fix(kv): GetNamespace fetches directly via REST instead of list-scan
- direct-GET not-found typing for bare 404s; never fabricate from null result (commit:7297d911)

## [0.23.0] - 2026-09-09

### Added
- cosmoflare limits — plan-limit proximity view with alert feed
- cosmoflare limits — plan-limit proximity table and JSON (commit:c4f41c8a)
- limit-proximity metrics feed the evaluator (commit:cef13ad2)
- snapshot assembly with partial-failure semantics (commit:8b619673)
- live DNS usage endpoint with static fallback, zone plan legacy_id (commit:58f34664)
- workers plan resolution with subscriptions API and fallback chain (commit:7159245a)
- workers_plan project config field for limit tier fallback (commit:1bf62535)
- snapshot types, consumer interfaces, service constructor (commit:5e967efb)
- static limit tables and limitFor join function (commit:773de39d)

### Changed
- internal: the three REST services share one transport client (~200 duplicated lines consolidated), metrics --json collects analytics concurrently, zone auto-resolution moved into the library
- adopt shared restClient, fix inert serve wiring and condition registry (commit:ede3069c)
- shared restClient transport, ux.Confirm, library zone resolution (commit:eca12aa0)

### Fixed
- limits serve wiring, alert condition registry, and renderer labels fixed
- pin DNS usage endpoint path and fields to the live API (commit:d1f5026f)
- correct resolution-order doc comment — flag outranks config (commit:4fc6dff1)

## [0.22.0] - 2026-09-08

### Added
- bucket domain: new command group (attach/list/get/verify/update/detach) managing R2 bucket custom domains over the REST API, with zone auto-resolution, --json output, ownership/SSL activation polling, jurisdiction support, and optional min-TLS/cipher settings
- metrics: real usage analytics + reliable daemon snapshots — new AnalyticsService (GraphQL: zone HTTP, R2 storage/operations, Workers invocations, retention-validated windows); daemon publishes partial snapshots with per-source errors and populated profile instead of dropping on any failure; metrics --json gains windowed usage (--window, default 24h) with counts always printing
- alerts: evaluator closes the loop — enabled rules judged against live analytics (error-rate %, storage bytes, CPU-p99 latency, windowed failure counts) fire TriggerAlert through the serve bridge with per-rule cooldown; daemon evaluates every 5m, 'alerts check' runs one-shot from the CLI; cycles with missing data are skipped, never evaluated against zeros
- bucket lifecycle: get/set/clear R2 object lifecycle rules (expire by age/date, abort stale multipart uploads, transition to InfrequentAccess) with whole-config replace semantics and confirm-before-destruct
- bucket notifications: manage R2 event notification rules to Cloudflare Queues (list/create/get/delete, five exact action types plus object-create/object-delete groupings, prefix/suffix filters)
- manage R2 event notification rules to Queues (commit:461370a3)
- get, set, and clear R2 object lifecycle rules (commit:9e60bb6d)
- evaluator firing TriggerAlert from rules and live analytics (commit:7fd2d00b)
- partial snapshots with per-source errors, populated profile, usage analytics (commit:f9dd2b4f)
- GraphQL Analytics producers for zones, R2, and Workers (commit:ba0a51ea)
- attach, list, verify, update, and detach R2 bucket custom domains (commit:5291c9ce)

### Removed
- Removed pre-rename dead code: 4 cmd/*.go.disabled drafts, cmd_disabled/ (policy/restore/upload/webhook), and 4 internal/*_disabled packages (~7,800 lines). All superseded by live implementations or mock scaffolds; recoverable from git history. R2 lifecycle policies (no live equivalent yet) noted as a future roadmap candidate.

### Fixed
- sync: --checksum now falls back to size/mtime comparison for multipart-uploaded objects (their -N ETags can never match a content checksum); downloads restore the remote LastModified so files are no longer re-downloaded on every sync
- fall back to size/mtime for multipart ETags; restore remote mtime after download (commit:d1e141da)

## [0.21.0] - 2026-09-07

### Added
- Published first release with full binary distribution (v0.20.0: 5 binaries + checksums)
- map typed daemon errors to stable HTTP status + code (commit:d8f8f3c8)

### Changed
- Daemon REST errors map typed R2 errors to stable HTTP statuses (400/401/403/404/429/502) with {error, code} JSON bodies — desktop can distinguish user-fixable from transient failures

### Removed
- Purged 630 MB of dead blobs from git history (recovery patches, 12 old binaries) via git filter-repo; repo pack 485 MiB → 16.7 MiB

## [0.20.0] - 2026-09-06

### Added
- In-process event bus for real-time notifications (ROAD-080) (FEAT-008) (FEAT-008)
- MCP server now exposes the full CLI surface — 140+ tools auto-generated from the cobra tree, read-only by default, mutations behind an explicit allow switch
- bridge TriggerAlert to serve SSE notifications (FEAT-008) (commit:bc18c4c1)
- generate full-surface tools from the cobra tree with mutation gating (commit:164fa88b)

### Changed
- alert SSE fan-out fires before webhook retries; channel names exported as constants
- README regenerated from reality; local GoReleaser release config
- simplify alert bridge — honest lifetime, latency-first fan-out, channel constants (commit:df2fc3ce)

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