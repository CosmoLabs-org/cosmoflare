---
title: Service Interfaces for Testability
issue: TASK-002
status: implemented
created: 2026-05-18T14:30:00-03:00
deliverables:
  - id: BR-01
    title: "pkg/cosmoflare/interfaces.go with 12 service interfaces"
  - id: BR-02
    title: "Compile-time satisfaction checks for all 12 services"
  - id: BR-03
    title: "Tests verifying interface definitions compile"
---

# Service Interfaces for Testability (TASK-002)

**Goal**: Extract interfaces from existing concrete service types to enable mock testing and future provider swapping.

**Date**: 2026-05-18
**Issue**: TASK-002
**Status**: Design complete

---

## Current State

The pkg/cosmoflare package has two patterns:

1. **R2Client** (already an interface in `client.go:19-47`) — covers bucket CRUD, object CRUD, upload/download, copy, presign. Implemented by `client` struct. **Already testable via interface.**

2. **Service structs** (concrete, not interfaces) — WorkerService, KVService, DNSService, ZoneService, SSLService, CacheService, CORSService, D1Service, EmailService, FirewallService, HealthcheckService, DomainService, DoctorService, PageRuleService, WAFService, PagesService, QueueService. **NOT mockable.**

## Design Decision

### Q1: Where should interfaces live?

**Decision: In `pkg/cosmoflare/` alongside implementations**, not `internal/api/`. Rationale:
- `pkg/cosmoflare/` is the public importable API surface
- Go convention: interfaces live where they're consumed, but for a library, co-location is standard
- `internal/api/` doesn't exist as a package (the tests that import it are broken)
- A new file `pkg/cosmoflare/interfaces.go` keeps all service contracts in one discoverable place

### Q2: Interface naming convention?

**Decision: Service name matches struct minus "Service" suffix, or use Go convention of adding "-er".**

Actually, Go convention is to name the interface after the behavior. But for consistency with R2Client which is already the pattern, use the same `<Domain>Service` pattern for interfaces and `<domain>Service` (lowercase) for the concrete impl. But that clashes with the existing exported struct names.

**Better approach**: Define interfaces with the same name as the current structs, rename the concrete structs to add `impl` suffix. This is the minimal-disruption pattern since all public API consumers use the constructor functions (NewWorkerService, etc.) which already return pointers — change them to return the interface type.

**Simplest approach (chosen)**: Don't rename anything. Add interfaces in `interfaces.go` with a different name: `WorkerServiceAPI`, `KVServiceAPI`, etc. Constructors keep returning concrete types. Consumers who need mockability import the interface type. Zero breaking changes.

**Wait — even simpler**: Go's implicit interface satisfaction means we don't need to change ANYTHING about the structs. Just define the interfaces. Any mock that implements the same methods satisfies the interface. The CMD layer and tests can accept the interface type where needed.

### Q3: Which services get interfaces?

**All non-stub services** (12 total):
- WorkerService (6 methods)
- KVService (8 methods)
- DNSService (5 methods)
- ZoneService (5 methods)
- SSLService (5 methods)
- CacheService (6 methods)
- CORSService (3 methods)
- FirewallService (5 methods)
- WAFService (8 methods)
- HealthcheckService (5 methods)
- DomainService (3 methods)
- DoctorService (5 methods)

Not yet covered by interfaces (have full implementations, not stubs): D1Service (5 methods), EmailService (14 methods), PageRuleService (5 methods), PagesService (7 methods), QueueService (8 methods). These should be added in a follow-up.

### Q4: What about the Cloudflare API dependency?

Not in scope. The `cloudflare.API` and `s3.Client` dependencies inside the structs are implementation details. The interfaces abstract at the service level, not the transport level.

## Implementation Plan

### Phase 1: interfaces.go (one file, zero breaking changes)

1. Create `pkg/cosmoflare/interfaces.go`
2. Define one interface per service, matching the public method signatures of each concrete struct
3. Add a compile-time check per service: `var _ WorkerServicer = (*WorkerService)(nil)`

### Phase 2: Wire into cmd/ layer (separate PR)

1. Change cmd/ handlers to accept interface types instead of concrete pointers
2. This enables injecting mocks for cmd-level integration tests (ROAD-017)

### Phase 3: Generate mocks (separate PR)

1. Use `mockgen` or hand-write test doubles
2. Add cmd-level tests with mocked services

## Deliverables

- BR-01: `pkg/cosmoflare/interfaces.go` with 12 service interfaces
- BR-02: Compile-time satisfaction checks for all 12
- BR-03: Tests verifying interface definitions compile

## Naming Convention

```go
// Interface names: add -er suffix to avoid collision with struct names
type WorkerServicer interface { ... }
type KVServicer interface { ... }
type DNSServicer interface { ... }
// etc.

// Compile-time checks
var _ WorkerServicer = (*WorkerService)(nil)
var _ KVServicer = (*KVService)(nil)
```

The `-er` suffix follows Go convention (Reader, Writer, Stringer) and avoids any naming collision with the concrete types.
