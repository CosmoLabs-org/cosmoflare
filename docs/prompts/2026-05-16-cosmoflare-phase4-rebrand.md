---
branch: master
status: COMPLETED
created: "2026-05-16"
goals_completed: 0
goals_total: 8
origin: manual
priority: high
related_prompts:
    - docs/planning-mode/2026-05-16-phase4-cloudflare-services.md
started: "2026-05-16"
status: PENDING
tags:
    - cloudflare
    - dns
    - zones
    - ssl
    - cache
    - rebrand
    - cosmoflare
    - parallel-agents
title: Cosmoflare Phase 4 — Core Services + Rebrand
schema_version: 1
deliverables:
    - P-01: DNS Records library + CLI (ROAD-035)
    - P-02: Zone management library + CLI (ROAD-036)
    - P-03: SSL/TLS management library + CLI (ROAD-037)
    - P-04: Cache management library + CLI (ROAD-038)
    - P-05: Unit tests for all 4 services (parallel GLM agents)
    - P-06: Cosmoflare rebrand — binary alias, README, docs, module path plan
    - P-07: Shell completion scaffold (ROAD-055)
    - P-08: Commit all work + update roadmap status
requires_reading:
    - docs/planning-mode/2026-05-16-phase4-cloudflare-services.md
    - docs/PRODUCT-VISION.md
    - pkg/r2go2/worker.go
    - pkg/r2go2/kv.go
    - cmd/worker.go
    - cmd/kv.go
    - pkg/r2go2/errors.go
    - pkg/r2go2/types.go
    - cmd/root.go
    - CLAUDE.md
---

# Cosmoflare Phase 4 — Core Services + Rebrand

**Date**: 2026-05-16
**Branch**: master
**Context**: R2Go2 v0.7.0 has R2 storage, Workers, and KV. This session adds the 4 highest-priority Cloudflare services (DNS, Zones, SSL, Cache), rebrands to Cosmoflare, and scaffolds shell completion. All implementation work dispatched to parallel Opus agents. All test work dispatched to parallel GLM agents.

**Product vision**: See `docs/PRODUCT-VISION.md` — Cosmoflare is the ultimate open-source Cloudflare CLI, with a future React Native mobile app as the paid tier.

## Execution Model

**4 parallel Opus agents for implementation** → merge → **4 parallel GLM agents for tests** → **1 Opus agent for rebrand + polish** → commit.

## Goals

### [ ] P-01: DNS Records Library + CLI (ROAD-035)
**Model**: opus (dispatch via isolated worktree)
**Files**: `pkg/r2go2/dns.go`, `cmd/dns.go`

DNS record management. Zone-scoped (needs zone ID, not account ID).

Library (`pkg/r2go2/dns.go`):
- Types: `DNSRecord` (ID, Type, Name, Content, TTL, Proxied, Priority, Comment, ZoneID)
- `DNSService` struct — holds `*cloudflare.API` + `zoneID`
- `NewDNSServiceFromCreds(zoneID, apiToken)` constructor
- `NewDNSService(api *cloudflare.API, zoneID string)` constructor
- Functional options: `WithDNSProxied(bool)`, `WithDNSTTL(int)`, `WithDNSPriority(int)`, `WithDNSComment(string)`
- Methods:
  - `Create(ctx, recordType, name, content string, opts ...DNSOption) (*DNSRecord, error)`
  - `List(ctx, opts ...DNSListOption) ([]*DNSRecord, error)` — filter by type, name, content
  - `Get(ctx, recordID string) (*DNSRecord, error)`
  - `Update(ctx, recordID string, opts ...DNSOption) (*DNSRecord, error)`
  - `Delete(ctx, recordID string) error`
- Error wrapping using pattern from `errors.go`: `validationError`, `newError`, `notFound`
- Uses `cloudflare.ZoneIdentifier(zoneID)` for all API calls

CLI (`cmd/dns.go`):
- Subcommands: `create`, `list`, `get`, `update`, `delete`
- All require `zone-id` as first argument
- All support `--json` flag
- All have detailed `--help` with examples
- Self-register via `func init() { rootCmd.AddCommand(dnsCmd) }`

### [ ] P-02: Zone Management Library + CLI (ROAD-036)
**Model**: opus (dispatch via isolated worktree)
**Files**: `pkg/r2go2/zone.go`, `cmd/zone.go`

Zone listing and management. Account-scoped.

Library (`pkg/r2go2/zone.go`):
- Types: `Zone` (ID, Name, Status, Type, Nameservers, Plan, CreatedOn, Paused, VanityNameservers)
- `ZoneService` struct — holds `*cloudflare.API` + `accountID`
- `NewZoneServiceFromCreds(accountID, apiToken)` constructor
- Methods: `Create`, `List` (filter by name, status, type), `Get`, `Delete`, `GetSettings`
- Settings returns zone-level settings as a map or typed struct

CLI (`cmd/zone.go`):
- Subcommands: `create`, `list`, `get`, `settings`, `delete`
- `r2go2 zone list --json`, `r2go2 zone get <zone-id>`, etc.

### [ ] P-03: SSL/TLS Management Library + CLI (ROAD-037)
**Model**: opus (dispatch via isolated worktree)
**Files**: `pkg/r2go2/ssl.go`, `cmd/ssl.go`

SSL/TLS certificate and settings management. Zone-scoped.

Library (`pkg/r2go2/ssl.go`):
- Types: `SSLCertificate`, `SSLSettings` (MinTLSVersion, AlwaysUseHTTPS, AutomaticHTTPSRewrites, OCSPStapling)
- `SSLService` struct — holds `*cloudflare.API` + `zoneID`
- Methods: `GetSSL`, `EditSSL`, `GetVerification`, `GetSettings`, `UpdateSettings`

CLI (`cmd/ssl.go`):
- Subcommands: `status`, `settings`, `update`, `verify`
- `r2go2 ssl status <zone-id>`, `r2go2 ssl update <zone-id> --min-tls=1.2 --always-https`

### [ ] P-04: Cache Management Library + CLI (ROAD-038)
**Model**: opus (dispatch via isolated worktree)
**Files**: `pkg/r2go2/cache.go`, `cmd/cache.go`

Cache management with purge operations. Zone-scoped. Destructive operations need `--force`.

Library (`pkg/r2go2/cache.go`):
- Types: `CacheSettings` (TTL, DevelopmentMode), `CachePurgeResult`
- `CacheService` struct — holds `*cloudflare.API` + `zoneID`
- Methods: `PurgeAll`, `PurgeByURLs`, `PurgeByTags`, `PurgeByHosts`, `GetSettings`, `UpdateSettings`
- Functional options: `WithCacheTTL(int)`, `WithDevMode(bool)`

CLI (`cmd/cache.go`):
- Subcommands: `purge`, `settings`
- `r2go2 cache purge <zone-id> --all` (requires --force)
- `r2go2 cache purge <zone-id> --url=https://...` or `--tag=static` or `--host=example.com`
- `r2go2 cache settings <zone-id>`, `r2go2 cache settings <zone-id> --ttl=3600`

### [ ] P-05: Unit Tests for All 4 Services
**Model**: glm-turbo (parallel dispatch after P-01 through P-04 merge)
**Files**: `pkg/r2go2/dns_test.go`, `pkg/r2go2/zone_test.go`, `pkg/r2go2/ssl_test.go`, `pkg/r2go2/cache_test.go`

Each test file covers:
1. Constructor validation (missing zoneID, missing apiToken, nil API client)
2. CRUD validation (empty names, missing IDs)
3. Error wrapping assertions (correct error types, messages contain context)
4. Edge cases (special characters in DNS names, large record sets, concurrent operations)

Follow pattern from `pkg/r2go2/worker_test.go` and `pkg/r2go2/kv_test.go`.

### [ ] P-06: Cosmoflare Rebrand
**Model**: opus (single agent after tests pass)

Rebrand the project while maintaining backward compatibility:
1. **README.md** — Update title, description, installation, examples. Mention `r2go2` as alias.
2. **CLAUDE.md** — Already updated, verify consistency.
3. **docs/USAGE.md** — Add note that `cosmoflare` is the primary binary, `r2go2` still works.
4. **Binary alias** — Update Makefile/build to produce both `cosmoflare` and `r2go2` binaries (or symlink).
5. **go.mod** — Do NOT rename module path yet (breaking for downstream). Document plan in `docs/planning-mode/` for future rename to `github.com/CosmoLabs-org/cosmoflare`.
6. **New commands** — All new Phase 4 commands should document both `cosmoflare dns list` and `r2go2 dns list` in help text.

### [ ] P-07: Shell Completion Scaffold (ROAD-055)
**Model**: opus (small task, can combine with P-06 or standalone)

Add shell completion generation:
- `r2go2 completion bash > /etc/bash_completion.d/r2go2`
- `r2go2 completion zsh > "${fpath[1]}/_r2go2"`
- `r2go2 completion fish > ~/.config/fish/completions/r2go2.fish`
- Cobra has built-in `genBashCompletion`, `genZshCompletion`, `genFishCompletion` — just wire them up.
- Add `completion` subcommand to rootCmd.
- Document in USAGE.md.

### [ ] P-08: Commit + Roadmap Update
**Model**: opus (session orchestrator)

After all work is merged and tested:
1. Run `go build ./...` and `go test ./pkg/r2go2/ -v` to verify
2. Run `go vet ./...` to check for issues
3. Commit with semantic messages (feat, test, docs, chore)
4. Update ROAD-035, ROAD-036, ROAD-037, ROAD-038 status to completed
5. Update ROAD-055 status to completed
6. Update `.version-registry.json` if appropriate

## Reference Implementation Pattern

All services follow the Worker/KV pattern (see `requires_reading` files):

```
pkg/r2go2/{service}.go:
  - Types ({Service}, {Service}Settings, {Service}Option)
  - Service struct { cf *cloudflare.API, accountID/zoneID string }
  - New{Service}FromCreds(creds) → *Service, error
  - CRUD methods with validation + cloudflare-go API calls
  - Error wrapping via validationError/newError/notFound

cmd/{service}.go:
  - cobra.Command with Use, Short, Long (includes examples)
  - Subcommands for each CRUD operation
  - Flags for all options
  - run{Service}{Operation} functions that create service + call method + format output
  - func init() { rootCmd.AddCommand(xxxCmd) }
```

## Session Management — NON-NEGOTIABLE

### TaskList Tracking
Every P-01 through P-08 gets a TaskCreate BEFORE any code runs. Mark `in_progress` when starting, `completed` when done. The user sees a live spinner — keep it accurate.

### Parallel Agent Execution
This session is designed for MAXIMUM parallelism:
1. **Phase 1 (implementation)**: Dispatch all 4 Opus agents (P-01 through P-04) simultaneously via `/orchestra auto` or individual `ccs glm-agent exec --model opus` calls. Do NOT run them sequentially.
2. **Phase 2 (testing)**: After ALL implementation agents merge, dispatch 4 GLM-turbo test agents (P-05) simultaneously.
3. **Phase 3 (polish)**: P-06 (rebrand) + P-07 (shell completion) can run in parallel if independent. P-08 (commit) runs last.

### Roadmap Updates — As You Go
After each goal completes and merges:
- `ccs roadmap update ROAD-035 --status completed` (or appropriate status)
- Update `goals_completed` in this prompt's frontmatter
- Do NOT batch all roadmap updates at the end — update as each deliverable ships

### Commit Strategy
- **After Phase 1 (implementation)**: Run `/commit-all` to commit the 4 new service modules. Take a breather, verify the build.
- **After Phase 2 (tests)**: Run `/commit-all` to commit test files.
- **After Phase 3 (rebrand + polish)**: Run `/commit-all` for the final batch.
- **Do NOT wait until the very end** to commit everything in one blob. Commit at each natural checkpoint so work is never lost.
- Each commit should use `ccs commit-batch` with semantic messages: `feat(dns)`, `feat(zone)`, `test(dns)`, `docs(cosmoflare)`, etc.

### Session Flow
```
1. /run-continuation 2026-05-16-cosmoflare-phase4-rebrand
2. TaskCreate for P-01 through P-08 (8 tasks)
3. Read all requires_reading files
4. Dispatch P-01, P-02, P-03, P-04 in parallel (4 Opus agents)
5. Review + merge each agent as it completes
6. /commit-all (implementation checkpoint)
7. Update ROAD-035/036/037/038 status
8. Dispatch P-05 tests in parallel (4 GLM-turbo agents)
9. Review + merge test agents
10. /commit-all (test checkpoint)
11. P-06 (rebrand) + P-07 (shell completion) — parallel or sequential
12. /commit-all (final checkpoint)
13. P-08: verify build, update remaining roadmap items, final commit
```

## Important Notes

- **cloudflare-go v0.116.0** — check API method signatures against this version
- **DNS/SSL/Cache are zone-scoped** — they take `zoneID`, not `accountID`. Use `cloudflare.ZoneIdentifier(zoneID)`.
- **Zones are account-scoped** — use `cloudflare.AccountIdentifier(accountID)`.
- **Cache purge is destructive** — `--force` flag required for `--all` purge, similar to `bucket delete --force`
- **Backward compatible** — `r2go2` binary must still work after rebrand. `cosmoflare` is the new name, `r2go2` is the alias.
