---
ulid: 01M2CB42GA2Z0NKQCBJK8G86GZ
title: Typed daemon wire contract + TS codegen
created: "2026-09-13T06:56:39.946418+04:00"
status: withered
source: agent
origin:
    session: 53
    trigger: audit-agent-7-api-design
    file: docs/audit/2026-09-13-cosmoflare/agent-7-api-design.md
tags:
    - audit
    - api-design
promoted_to: FEAT-042
resolution:
    reason: implemented
    date: "2026-09-15T22:12:59.769103+04:00"
    ref: FEAT-042
    note: Resolved via FEAT-042 (closed)
---

# Typed daemon wire contract + TS codegen

# Typed daemon wire contract + TS codegen

# Typed daemon wire contract + TS codegen

ServeSource returns any; TS client casts unchecked (rest.go:24). Return concrete Go structs and generate TypeScript types for desktop/src/api, drift-checked. Evidence: agent-7-api-design.md
