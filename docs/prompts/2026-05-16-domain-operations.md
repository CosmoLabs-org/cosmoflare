---
brainstorm_ref: docs/brainstorming/2026-05-16-domain-operations.md
branch: master
covers_brainstorm_deliverables:
    - BR-01
    - BR-02
    - BR-03
    - BR-04
    - BR-05
    - BR-06
    - BR-07
    - BR-08
covers_plan_deliverables:
    - P-01
    - P-02
    - P-03
    - P-04
    - P-05
    - P-06
    - P-07
    - P-08
created: "2026-05-16"
id: P-2026-05-16-domain-operations
plan_ref: docs/planning-mode/2026-05-16-domain-operations.md
priority: medium
requires_reading:
    - docs/brainstorming/2026-05-16-domain-operations.md
    - docs/planning-mode/2026-05-16-domain-operations.md
schema_version: 1
status: PENDING
title: Domain Operations Center — Full Implementation
---

# Domain Operations Center — Full Implementation

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/brainstorming/2026-05-16-domain-operations.md`** — design rationale and decision history.
2. **`docs/planning-mode/2026-05-16-domain-operations.md`** — implementation plan with deliverables and file scope.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## Context

Cosmoflare has 11 service commands implemented (R2, Workers, KV, DNS, Zones, SSL, Cache, Page Rules, WAF, Email Routing, D1). This session adds the "domain operations center" — the ability to see all domains at a glance and deep-diagnose any one of them. Two new top-level commands: `cosmoflare domains` (overview with pagination/filtering for 50+ domain portfolios) and `cosmoflare doctor` (4-probe diagnostics with actionable fix suggestions).

Design spec: `docs/brainstorming/2026-05-16-domain-operations.md`
Implementation plan: `docs/planning-mode/2026-05-16-domain-operations.md`

## Execution Strategy

**Wave 1 (parallel):** G-01 (Healthcheck API) + G-02/G-03/G-04/G-05 (diagnostic probes in doctor.go)
**Wave 2:** G-06 (DomainService — depends on ZoneService + health enrichment)
**Wave 3 (parallel):** G-07 (domains CLI) + G-08 (doctor CLI)

## Goals

### [ ] G-01 Healthcheck library (pkg/r2go2/healthcheck.go)
Covers P-01.

### [ ] G-02 DNS propagation probe (pkg/r2go2/doctor.go)
Covers P-02.

### [ ] G-03 SSL certificate probe (pkg/r2go2/doctor.go)
Covers P-03.

### [ ] G-04 HTTP response probe (pkg/r2go2/doctor.go)
Covers P-04.

### [ ] G-05 Nameserver consistency probe (pkg/r2go2/doctor.go)
Covers P-05.

### [ ] G-06 Domain overview service (pkg/r2go2/domains.go)
Covers P-06.

### [ ] G-07 CLI — cosmoflare domains command (cmd/domains.go)
Covers P-07.

### [ ] G-08 CLI — cosmoflare doctor command (cmd/doctor.go)
Covers P-08.

## Related

- Brainstorm: `docs/brainstorming/2026-05-16-domain-operations.md`
- Plan: `docs/planning-mode/2026-05-16-domain-operations.md`
