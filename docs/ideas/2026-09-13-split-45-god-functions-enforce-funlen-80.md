---
ulid: 01M2CB3YME8QDR7HE12EHKQFT1
title: Split 45 god functions; enforce funlen=80
created: "2026-09-13T06:56:35.982137+04:00"
status: withered
source: agent
origin:
    session: 53
    trigger: audit-agent-1-code-quality
    file: docs/audit/2026-09-13-cosmoflare/agent-1-code-quality.md
tags:
    - audit
    - code-quality
promoted_to: TASK-009
resolution:
    reason: implemented
    date: "2026-09-15T20:17:37.54708+04:00"
    ref: TASK-009
    note: Resolved via TASK-009 (closed)
---

# Split 45 god functions; enforce funlen=80

# Split 45 god functions; enforce funlen=80

# Split 45 god functions; enforce funlen=80

45 functions over 80 lines, worst 238 (internal/tui/update.go handleKeyMsg). Split runObjectPut/handleKeyMsg/loadBuiltinThemes into per-path helpers; adopt golangci-lint funlen=80. Evidence: agent-1-code-quality.md
