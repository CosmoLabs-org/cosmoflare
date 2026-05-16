---
branch: master
created: "2026-05-16"
goals_completed: 0
goals_total: 5
origin: manual
priority: high
related_prompts:
    - docs/prompts/2026-05-16-phase4-cloudflare-services.md
started: "2026-05-16"
status: PLANNED
tags:
    - cloudflare
    - dns
    - zones
    - ssl
    - cache
    - parallel-agents
title: Phase 4 — Cloudflare Core Services (DNS, Zones, SSL, Cache)
schema_version: 1
deliverables:
    - P-01: DNS Records library + CLI
    - P-02: Zone management library + CLI
    - P-03: SSL/TLS management library + CLI
    - P-04: Cache management library + CLI
    - P-05: Unit tests for all 4 services
requires_reading:
    - pkg/r2go2/worker.go
    - pkg/r2go2/kv.go
    - cmd/worker.go
    - cmd/kv.go
    - pkg/r2go2/errors.go
    - pkg/r2go2/types.go
    - cmd/root.go
---

# Phase 4 — Cloudflare Core Services (DNS, Zones, SSL, Cache)

**Date**: 2026-05-16
**Branch**: master
**Context**: R2Go2 v0.7.0 has R2 storage, Workers, and KV implemented. Now adding the 4 highest-priority Cloudflare services that unlock every Cloudflare user, not just developers.

## Execution Model

**5 parallel Opus agents** — each builds one service module (library + CLI). After all complete, **parallel GLM agents** run unit tests.

### Agent Dispatch Plan

| Agent | Model | Deliverable | Files to Create |
|-------|-------|-------------|-----------------|
| Agent A | opus | P-01: DNS Records | `pkg/r2go2/dns.go`, `cmd/dns.go` |
| Agent B | opus | P-02: Zones | `pkg/r2go2/zone.go`, `cmd/zone.go` |
| Agent C | opus | P-03: SSL/TLS | `pkg/r2go2/ssl.go`, `cmd/ssl.go` |
| Agent D | opus | P-04: Cache | `pkg/r2go2/cache.go`, `cmd/cache.go` |
| Agent E | opus | Wire all 4 into root.go + register cobra commands | `cmd/root.go` (edit) |

### Test Dispatch (after agents A-D merge)

| Test Agent | Model | Scope |
|------------|-------|-------|
| Test A | glm-turbo | `pkg/r2go2/dns_test.go` |
| Test B | glm-turbo | `pkg/r2go2/zone_test.go` |
| Test C | glm-turbo | `pkg/r2go2/ssl_test.go` |
| Test D | glm-turbo | `pkg/r2go2/cache_test.go` |

## Reference Pattern

All services follow the same pattern as Workers (`pkg/r2go2/worker.go`) and KV (`pkg/r2go2/kv.go`):

1. **Types** — struct definitions with json tags
2. **Service struct** — holds `*cloudflare.API` + `accountID` (for account-scoped) or `zoneID` (for zone-scoped)
3. **Constructor** — `New{Service}FromCreds(accountID, apiToken)` convenience + `New{Service}(api, accountID)`
4. **Functional options** — `With{Option}` pattern for optional params
5. **CRUD methods** — Create, List/Get, Update, Delete with validation + error wrapping
6. **CLI commands** — cobra commands with `--json` support, `--help` with examples

## cloudflare-go v0.116.0 API Methods

DNS Records use zone-scoped API (`cloudflare.ZoneIdentifier(zoneID)`):
- `CreateDNSRecord(ctx, rc, params)` → DNSRecord
- `ListDNSRecords(ctx, rc, params)` → []DNSRecord
- `GetDNSRecord(ctx, rc, recordID)` → DNSRecord
- `UpdateDNSRecord(ctx, rc, params)` → DNSRecord
- `DeleteDNSRecord(ctx, rc, recordID)` → error

Zones use account-scoped API (`cloudflare.AccountIdentifier(accountID)`):
- `CreateZone(ctx, params)` → Zone
- `ListZones(ctx, params)` → []Zone
- `ZoneDetails(ctx, zoneID)` → Zone
- `EditZone(ctx, zoneID, params)` → Zone
- `DeleteZone(ctx, zoneID)` → error

SSL/TLS use zone-scoped API:
- `GetSSL(ctx, zoneID)` → SSLSetting
- `EditSSL(ctx, zoneID, params)` → SSLSetting
- `GetSSLVerification(ctx, zoneID)` → SSLVerification
- Zone settings for min_tls_version, always_use_https, etc.

Cache use zone-scoped API:
- `PurgeCache(ctx, rc, params)` → CachePurgeResponse
- `ZoneCacheSettings(ctx, zoneID)` → CacheSetting
- `UpdateZoneCacheSettings(ctx, zoneID, params)` → CacheSetting

## Goals

### [ ] P-01: DNS Records Library + CLI (Agent A — opus)

**Library** (`pkg/r2go2/dns.go`):
- Types: `DNSRecord` (ID, Type, Name, Content, TTL, Proxied, Priority, Comment)
- `DNSService` struct with `*cloudflare.API` + `zoneID`
- `NewDNSServiceFromCreds(zoneID, apiToken)` — note: DNS is zone-scoped, needs zone ID
- Methods: `Create`, `List` (with type/name filters), `Get`, `Update`, `Delete`
- Functional options: `WithDNSProxied`, `WithDNSTTL`, `WithDNSPriority`, `WithDNSComment`

**CLI** (`cmd/dns.go`):
- `r2go2 dns create <zone-id> --type=A --name=sub --content=1.2.3.4 --proxied`
- `r2go2 dns list <zone-id> [--type=CNAME] [--name=www]`
- `r2go2 dns get <zone-id> <record-id>`
- `r2go2 dns update <zone-id> <record-id> --content=5.6.7.8`
- `r2go2 dns delete <zone-id> <record-id>`
- All with `--json`, detailed `--help`, examples

**Model**: opus — requires reading cloudflare-go DNS API, designing type mappings

### [ ] P-02: Zone Management Library + CLI (Agent B — opus)

**Library** (`pkg/r2go2/zone.go`):
- Types: `Zone` (ID, Name, Status, Type, Nameservers, Plan, CreatedOn, ModifiedOn)
- `ZoneService` struct with `*cloudflare.API` + `accountID`
- `NewZoneServiceFromCreds(accountID, apiToken)`
- Methods: `Create`, `List` (with name/status filters), `Get`, `Update`, `Delete`
- Extra: `GetSettings` (returns zone-level settings), `GetHealthcheck`

**CLI** (`cmd/zone.go`):
- `r2go2 zone create example.com --type=full`
- `r2go2 zone list [--status=active]`
- `r2go2 zone get <zone-id>`
- `r2go2 zone settings <zone-id>`
- `r2go2 zone delete <zone-id> --force`

**Model**: opus — zone API is foundational, other services depend on zone IDs

### [ ] P-03: SSL/TLS Management Library + CLI (Agent C — opus)

**Library** (`pkg/r2go2/ssl.go`):
- Types: `SSLCertificate` (ID, Type, Status, ExpiresOn, UploadedOn), `SSLSettings`
- `SSLService` struct with `*cloudflare.API` + `zoneID`
- Methods: `GetSSL`, `EditSSL`, `GetVerification`, `GetSettings`, `UpdateSettings`
- Settings: minimum_tls_version, always_use_https, automatic_https_rewrites, ssl_recommender

**CLI** (`cmd/ssl.go`):
- `r2go2 ssl status <zone-id>`
- `r2go2 ssl settings <zone-id>`
- `r2go2 ssl update <zone-id> --minimum-tls=1.2 --always-https`
- `r2go2 ssl verify <zone-id>`

**Model**: opus — SSL settings are security-critical, needs careful error handling

### [ ] P-04: Cache Management Library + CLI (Agent D — opus)

**Library** (`pkg/r2go2/cache.go`):
- Types: `CacheSettings` (TTL, DevelopmentMode), `CachePurgeResult`
- `CacheService` struct with `*cloudflare.API` + `zoneID`
- Methods: `PurgeAll`, `PurgeByURLs`, `PurgeByTags`, `PurgeByHosts`, `GetSettings`, `UpdateSettings`
- Functional options: `WithCacheTTL`, `WithDevMode`

**CLI** (`cmd/cache.go`):
- `r2go2 cache purge <zone-id> --all`
- `r2go2 cache purge <zone-id> --url=https://example.com/style.css`
- `r2go2 cache purge <zone-id> --tag=static`
- `r2go2 cache settings <zone-id>`
- `r2go2 cache settings <zone-id> --ttl=3600 --dev-mode`

**Model**: opus — cache operations are destructive (purge), needs confirmation flags

### [ ] P-05: Wire Commands + Register in root.go (Agent E — opus)

**Edit** `cmd/root.go`:
- Add `dnsCmd`, `zoneCmd`, `sslCmd`, `cacheCmd` to root command
- Ensure PersistentPreRun works for new commands (API token validation)
- Add `cmd/dns.go`, `cmd/zone.go`, `cmd/ssl.go`, `cmd/cache.go` imports

**Verify**:
- `go build -o /dev/null .` passes
- `r2go2 dns --help`, `r2go2 zone --help`, etc. work
- `go vet ./...` passes (excluding pre-existing issues)

**Model**: opus — needs to understand root.go structure, cobra registration pattern

## After Merge — Test Phase

Once all 5 agents complete and merge:

1. Run `go build ./...` to verify compilation
2. Dispatch 4 parallel GLM-turbo test agents:
   - Each creates `pkg/r2go2/{service}_test.go`
   - Tests: constructor validation, CRUD with mocked cloudflare.API, error wrapping, edge cases
   - Follow pattern from `pkg/r2go2/worker_test.go` and `pkg/r2go2/kv_test.go`
3. Final verification: `go test ./pkg/r2go2/ -v -count=1`

## Merge Order

1. Agent E (root.go wiring) depends on A-D completing first — wait for A-D, then run E
2. Actually: A-D can create their cobra commands with local registration. Agent E only needs to add them to rootCmd.
3. Alternative: each agent creates standalone commands that self-register via `init()`. Then no Agent E needed.

**Recommended**: Each agent creates files with `func init() { rootCmd.AddCommand(xxxCmd) }` pattern. No separate Agent E. This means 4 parallel agents, not 5.

## Commands for Next Session

```bash
# Phase 1: Dispatch 4 parallel Opus agents
/orchestra auto "Build Phase 4 Cloudflare services: DNS Records, Zones, SSL/TLS, Cache"

# Phase 2: After merge, dispatch 4 parallel GLM test agents
/orchestra auto "Write unit tests for Phase 4: DNS, Zone, SSL, Cache services"
```
