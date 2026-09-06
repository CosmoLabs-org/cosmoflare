---
branch: master
created: "2026-05-16T10:00:00-03:00"
deliverables:
    - BR-01: domains command with health-indicator table, pagination, filtering
    - BR-02: doctor command with 4-probe diagnostic engine
    - BR-03: DNS propagation checker (multi-resolver)
    - BR-04: SSL certificate chain validator
    - BR-05: HTTP response prober with redirect chain tracing
    - BR-06: Nameserver consistency checker with DNSSEC
    - BR-07: Fix suggestion engine (actionable cosmoflare commands)
    - BR-08: Cloudflare Healthcheck API integration
last_review_content_hash: 16488e5b0d0839ffc5fc88b89b41a78e2789c30f1c9e528ac156792fa31c215f
last_review_findings: 0
last_review_ref: docs/brainstorming/2026-05-16-domain-operations.md
last_reviewed: "2026-09-06T20:14:03.523187+04:00"
origin: /brainplan
schema_version: 1
status: APPROVED
tags:
    - domains
    - doctor
    - health
    - dns-propagation
    - ssl
    - nameservers
    - monitoring
title: Domain Operations Center — domains + doctor commands
---

# Domain Operations Center — domains + doctor commands

**Date**: 2026-05-16
**Status**: APPROVED
**Related roadmap**: ROAD-050 (cosmoflare status), ROAD-054 (cosmoflare doctor)

## Problem

Managing 50+ domains on Cloudflare requires constant visibility: are nameservers correct? Has DNS propagated? Are SSL certs valid? Are sites actually responding? Today this requires the Cloudflare dashboard, multiple browser tabs, or third-party tools. Cosmoflare should be the single command to answer "is my stuff working?"

## Design Decisions

### Command Structure

Two top-level commands, complementary:

```
cosmoflare domains                        # Overview: all zones with health indicators
cosmoflare domains --detail               # Per-domain cards with full info
cosmoflare domains --filter=active        # Filter by status
cosmoflare domains --name="*.com"         # Filter by name pattern
cosmoflare domains --page=2 --per-page=25 # Pagination (default: 50)
cosmoflare domains --json                 # Machine output

cosmoflare doctor example.com             # Deep diagnostics for one domain
cosmoflare doctor example.com --json      # Machine output
cosmoflare doctor example.com --fix       # Show fix commands
cosmoflare doctor --all                   # Run doctor on all domains (slow)
```

### domains Command — Overview Table

Default output (compact table, like `kubectl get pods`):

```
DOMAIN              STATUS    DNS    SSL     HEALTH   RECORDS  NS
example.com         active    ok     valid   up       42       cloudflare
myapp.io            active    warn   valid   up       12       cloudflare
staging.dev         active    ok     expiring down    8        cloudflare
oldsite.net         paused    --     --      --       3        external
```

Status indicators:
- DNS: `ok` (resolving correctly) / `warn` (propagation issues) / `err` (not resolving)
- SSL: `valid` / `expiring` (<30 days) / `expired` / `none`
- Health: `up` (HTTP 200) / `down` (non-200 or timeout) / `--` (not checked)
- NS: `cloudflare` / `external` / `mismatch`

`--detail` mode shows per-domain cards:
```
example.com (active)
  Nameservers: anna.ns.cloudflare.com, bob.ns.cloudflare.com
  Records:     42 (12 A, 8 CNAME, 6 MX, 16 TXT)
  SSL:         Full (strict) — expires 2026-09-15 (122 days)
  Health:      200 OK (142ms) — last checked: just now
```

### doctor Command — Deep Diagnostics

Runs 4 diagnostic probes on a single domain:

#### Probe 1: DNS Propagation
Query multiple public resolvers to verify records have propagated:
- Google (8.8.8.8, 8.8.4.4)
- Cloudflare (1.1.1.1, 1.0.0.1)
- OpenDNS (208.67.222.222)
- Quad9 (9.9.9.9)

For each resolver: query A, AAAA, MX, NS records. Compare results.
Flag: inconsistent responses across resolvers = propagation issue.

#### Probe 2: SSL Certificate Chain
Connect to the domain via TLS and inspect:
- Certificate validity dates (expiry warning if <30 days)
- Chain completeness (root → intermediate → leaf)
- TLS version negotiated (flag TLS 1.0/1.1 as deprecated)
- HSTS header presence
- Compare with Cloudflare's reported SSL mode

#### Probe 3: HTTP Response
Make actual HTTP/HTTPS requests:
- Check response status code
- Measure response time
- Follow redirect chains (flag loops, excessive redirects)
- Check for mixed content (HTTP resources on HTTPS page)
- Verify the response comes through Cloudflare (check cf-ray header)

#### Probe 4: Nameserver Consistency
- Compare assigned NS from Cloudflare with what DNS returns
- Check DNSSEC status
- Detect NS delegation issues
- Verify SOA record

### Fix Suggestion Engine

When `--fix` is passed (or issues found), doctor prints actionable commands:

```
ISSUES FOUND:

  DNS: Missing A record for www.example.com
  FIX: cosmoflare dns create ZONE_ID --type=A --name=www --content=YOUR_IP

  SSL: TLS minimum version is 1.0 (deprecated)
  FIX: cosmoflare ssl update ZONE_ID --min-tls=1.2

  NS: Nameservers don't match Cloudflare assignment
  FIX: Update nameservers at your registrar to:
       anna.ns.cloudflare.com
       bob.ns.cloudflare.com
```

### Health Check Integration

Use Cloudflare's Healthcheck API (already available in cloudflare-go v0.116.0):
- `Healthchecks(ctx, zoneID)` — list existing health checks
- `CreateHealthcheck(ctx, zoneID, healthcheck)` — create new
- `Healthcheck(ctx, zoneID, id)` — get status

`cosmoflare doctor` reads existing Cloudflare healthchecks AND performs its own active probes. The Cloudflare healthchecks give historical data; our probes give real-time from the user's network.

### Pagination & Filtering

All list operations paginate by default:
- `--page N` (default 1)
- `--per-page N` (default 50)
- `--filter status=active` or `--filter status=paused`
- `--name "pattern"` (substring match or glob)
- `--sort name|status|records` (default: name)

For `--json` output, pagination metadata is included:
```json
{
  "domains": [...],
  "pagination": {"page": 1, "per_page": 50, "total": 127, "total_pages": 3}
}
```

## Architecture

### Library Layer (`pkg/cosmoflare/`)

Files (implemented):
- `domains.go` — `DomainService` wrapping ZoneService + health enrichment
- `doctor.go` — `DoctorService` with the 4 diagnostic probes
- `healthcheck.go` — `HealthcheckService` wrapping Cloudflare Healthcheck API

The diagnostic probes use Go's standard library:
- `net` package for DNS resolution (custom resolver addresses)
- `crypto/tls` for certificate inspection
- `net/http` for HTTP probing

### CLI Layer (`cmd/`)

Files (implemented):
- `domains.go` — `cosmoflare domains` with table/detail/json output
- `doctor.go` — `cosmoflare doctor` with diagnostic runner + fix suggestions

### Dependencies

No new external dependencies. All probes use Go stdlib:
- `net.Resolver` with custom DNS server addresses
- `crypto/tls.Dial` for cert inspection
- `net/http.Client` with timeouts for HTTP probes

## Not In Scope (This Phase)

- Continuous monitoring / alerting (future: ROAD-062)
- Historical health data storage
- Push notifications for domain issues (future: mobile app)
- Load balancer / traffic steering management
- Custom hostname management (SaaS providers)
