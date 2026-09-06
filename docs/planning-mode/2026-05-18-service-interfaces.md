---
brainstorm: docs/brainstorming/2026-05-18-service-interfaces.md
completed: "2026-05-24T00:00:00-03:00"
created: "2026-05-18T14:30:00-03:00"
deliverables:
    - id: P-01
      title: pkg/r2go2/interfaces.go — 12 service interfaces with compile-time checks
    - id: P-02
      title: pkg/r2go2/interfaces_test.go — verification tests
    - id: P-03
      title: All existing tests pass (zero regressions)
goals_completed: 4
goals_total: 4
issue: TASK-002
last_review_content_hash: eda32fd7f192522b09f1629414ee73dfc9d7c4431c417c4017902467d3802d07
last_review_findings: 0
last_review_ref: docs/planning-mode/2026-05-18-service-interfaces.md
last_reviewed: "2026-09-06T20:14:03.956218+04:00"
related_prompts: []
requires_reading: []
schema_version: 1
status: COMPLETED
tags: []
title: Service Interfaces for Testability
---

# Plan: Service Interfaces for Testability (TASK-002)

**Date**: 2026-05-18
**Issue**: TASK-002
**Brainstorm**: docs/brainstorming/2026-05-18-service-interfaces.md
**Scope**: Phase 1 only (interfaces.go + compile checks, zero breaking changes)

---

## Steps

### 1. Extract method signatures from all 12 service structs

Read each service file and extract public method signatures:
- `pkg/r2go2/worker.go` — WorkerService methods
- `pkg/r2go2/kv.go` — KVService methods
- `pkg/r2go2/dns.go` — DNSService methods
- `pkg/r2go2/zone.go` — ZoneService methods
- `pkg/r2go2/ssl.go` — SSLService methods
- `pkg/r2go2/cloudflare_cache.go` — CacheService methods
- `pkg/r2go2/cors.go` — CORSService methods
- `pkg/r2go2/firewall.go` — FirewallService methods
- `pkg/r2go2/waf.go` — WAFService methods
- `pkg/r2go2/healthcheck.go` — HealthcheckService methods
- `pkg/r2go2/domains.go` — DomainService methods
- `pkg/r2go2/doctor.go` — DoctorService methods

### 2. Create `pkg/r2go2/interfaces.go`

- One interface per service using `-er` suffix naming
- Method signatures must match exactly (including parameter names for documentation)
- Add compile-time satisfaction checks: `var _ Interfacer = (*ConcreteType)(nil)`

### 3. Create `pkg/r2go2/interfaces_test.go`

- Verify all compile-time checks pass
- Test that interface types are assignable from constructor return values

### 4. Run tests

- `go build ./pkg/r2go2/` — must compile
- `go vet ./pkg/r2go2/` — must pass
- `go test ./pkg/r2go2/` — existing tests must still pass

## Deliverables

- P-01: `pkg/r2go2/interfaces.go` — 12 service interfaces with compile-time checks
- P-02: `pkg/r2go2/interfaces_test.go` — verification tests
- P-03: All existing tests pass (zero regressions)

## Acceptance Criteria

- [x] Every public method on each service struct appears in the corresponding interface
- [x] Compile-time checks prevent accidental method signature drift
- [x] Zero changes to existing code (no renames, no import changes)
- [x] `go test ./pkg/...` passes
