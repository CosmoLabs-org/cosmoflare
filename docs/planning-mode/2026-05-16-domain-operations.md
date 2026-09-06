---
branch: master
created: "2026-05-16T00:00:00-03:00"
goals_completed: 0
goals_total: 8
origin: "/brainplan"
priority: high
status: PLANNED
tags:
  - domains
  - doctor
  - health
  - diagnostics
title: "Domain Operations Center — Implementation Plan"
schema_version: 1
deliverables:
  - P-01: Healthcheck library (pkg/r2go2/healthcheck.go)
  - P-02: DNS propagation probe (pkg/r2go2/doctor.go)
  - P-03: SSL certificate probe (pkg/r2go2/doctor.go)
  - P-04: HTTP response probe (pkg/r2go2/doctor.go)
  - P-05: Nameserver consistency probe (pkg/r2go2/doctor.go)
  - P-06: Domain overview service (pkg/r2go2/domains.go)
  - P-07: CLI — cosmoflare domains command (cmd/domains.go)
  - P-08: CLI — cosmoflare doctor command (cmd/doctor.go)
requires_reading:
  - docs/brainstorming/2026-05-16-domain-operations.md
  - pkg/r2go2/zone.go
  - pkg/r2go2/dns.go
  - pkg/r2go2/ssl.go
  - cmd/zone.go
  - cmd/dns.go
---

# Domain Operations Center — Implementation Plan

**Date**: 2026-05-16
**Design spec**: `docs/brainstorming/2026-05-16-domain-operations.md`
**Roadmap**: ROAD-050, ROAD-054

## Execution Model

**3 waves of parallel agents:**
1. Wave 1: P-01 (Healthcheck API) + P-02/P-03/P-04/P-05 (4 diagnostic probes — single file)
2. Wave 2: P-06 (Domain overview service) — depends on Zone + health enrichment
3. Wave 3: P-07 + P-08 (CLI commands) — depends on library layer

## File Scope

### New Files
- `pkg/r2go2/healthcheck.go` — HealthcheckService wrapping Cloudflare API
- `pkg/r2go2/doctor.go` — DoctorService with 4 diagnostic probes
- `pkg/r2go2/domains.go` — DomainService (zone listing + health enrichment)
- `cmd/domains.go` — cosmoflare domains CLI
- `cmd/doctor.go` — cosmoflare doctor CLI

### Test Files
- `pkg/r2go2/healthcheck_test.go`
- `pkg/r2go2/doctor_test.go`
- `pkg/r2go2/domains_test.go`

### Existing Files (read-only reference)
- `pkg/r2go2/zone.go` — ZoneService (List, Get, GetSettings)
- `pkg/r2go2/dns.go` — DNSService (List with filters)
- `pkg/r2go2/ssl.go` — SSLService (GetSSL, GetVerification, GetSettings)

## Implementation Steps

### P-01: Healthcheck Library
**Model**: opus
**Files**: `pkg/r2go2/healthcheck.go`

Wrap Cloudflare Healthcheck API:
- Types: `Healthcheck` (ID, Name, Address, Type, Status, Interval, Timeout, Retries, Suspended)
- `HealthcheckService` struct with `cf *cloudflare.API` + `zoneID string`
- Methods: `List(ctx)`, `Get(ctx, id)`, `Create(ctx, ...)`, `Update(ctx, id, ...)`, `Delete(ctx, id)`
- cloudflare-go methods: `Healthchecks`, `Healthcheck`, `CreateHealthcheck`, `UpdateHealthcheck`, `DeleteHealthcheck`

### P-02: DNS Propagation Probe
**Model**: opus
**Files**: `pkg/r2go2/doctor.go`

```go
type DNSProbeResult struct {
    Resolver string
    Records  map[string][]string // type -> values
    Latency  time.Duration
    Error    error
}

func (d *DoctorService) CheckDNSPropagation(ctx context.Context, domain string) ([]DNSProbeResult, error)
```

Query 6 public resolvers (8.8.8.8, 8.8.4.4, 1.1.1.1, 1.0.0.1, 208.67.222.222, 9.9.9.9).
For each: resolve A, AAAA, MX, NS. Use `net.Resolver` with custom dialer.
Return results + consistency flag.

### P-03: SSL Certificate Probe
**Model**: opus (same file as P-02)

```go
type SSLProbeResult struct {
    Valid       bool
    Issuer      string
    Subject     string
    NotBefore   time.Time
    NotAfter    time.Time
    DaysLeft    int
    TLSVersion  string
    ChainLength int
    HSTS        bool
    Error       error
}

func (d *DoctorService) CheckSSL(ctx context.Context, domain string) (*SSLProbeResult, error)
```

Use `crypto/tls.DialWithDialer` → inspect `ConnectionState().PeerCertificates`.
Then HTTP GET to check HSTS header.

### P-04: HTTP Response Probe
**Model**: opus (same file as P-02)

```go
type HTTPProbeResult struct {
    StatusCode    int
    ResponseTime  time.Duration
    RedirectChain []string
    CloudflareRay string
    Server        string
    Error         error
}

func (d *DoctorService) CheckHTTP(ctx context.Context, domain string) (*HTTPProbeResult, error)
```

HTTP client with `CheckRedirect` callback to capture redirect chain.
Check `cf-ray` header to verify Cloudflare proxy.
Timeout: 10s.

### P-05: Nameserver Consistency Probe
**Model**: opus (same file as P-02)

```go
type NSProbeResult struct {
    Expected    []string
    Actual      []string
    Match       bool
    DNSSEC      bool
    SOA         string
    Error       error
}

func (d *DoctorService) CheckNameservers(ctx context.Context, domain string, expected []string) (*NSProbeResult, error)
```

Query NS records from public resolver. Compare with Cloudflare-assigned NS.
Check for DNSSEC by querying DNSKEY record.

### P-06: Domain Overview Service
**Model**: opus
**Files**: `pkg/r2go2/domains.go`

```go
type DomainStatus struct {
    Zone        *Zone
    DNSStatus   string // "ok", "warn", "err"
    SSLStatus   string // "valid", "expiring", "expired", "none"
    HealthStatus string // "up", "down", "unknown"
    RecordCount int
    NSStatus    string // "cloudflare", "external", "mismatch"
}

type DomainListOptions struct {
    Page     int
    PerPage  int
    Filter   string // "active", "paused", etc.
    Name     string // substring/glob filter
    Sort     string // "name", "status", "records"
}

type DomainService struct {
    zones *ZoneService
    ssl   *SSLService
    dns   *DNSService
    doctor *DoctorService
}

func (s *DomainService) List(ctx, opts DomainListOptions) ([]*DomainStatus, *Pagination, error)
func (s *DomainService) GetDetail(ctx, zoneID string) (*DomainDetail, error)
```

Enriches zone list with SSL status, DNS record count, and optional health probe.

### P-07: CLI — cosmoflare domains
**Model**: opus
**Files**: `cmd/domains.go`

```
cosmoflare domains [--detail] [--filter=active] [--name="*.com"] [--page=1] [--per-page=50] [--sort=name] [--json]
```

Table output with status indicators. Detail mode shows per-domain cards.
Self-register: `func init() { rootCmd.AddCommand(domainsCmd) }`

### P-08: CLI — cosmoflare doctor
**Model**: opus
**Files**: `cmd/doctor.go`

```
cosmoflare doctor <domain-or-zone-id> [--fix] [--json]
cosmoflare doctor --all [--fix] [--json]
```

Runs all 4 probes, prints results with pass/warn/fail indicators.
`--fix` appends actionable cosmoflare commands for each issue found.
Self-register: `func init() { rootCmd.AddCommand(doctorCmd) }`

## Verification

After all steps complete:
```bash
go build ./...
go test ./pkg/r2go2/ -v
go vet ./pkg/r2go2/ ./cmd/
./build/r2go2 domains --help
./build/r2go2 doctor --help
```
