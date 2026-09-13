---
ulid: 01M2CB4CWXAM3CFRBGS7A1YQFC
title: Remote MCP transport (HTTP, token-authed) + tools/list pagination
created: "2026-09-13T06:56:50.589173+04:00"
status: withered
source: agent
origin:
    session: 53
    trigger: audit-agent-4-competitive
    file: docs/audit/2026-09-13-cosmoflare/agent-4-competitive.md
tags:
    - audit
    - competitive
resolution:
    reason: duplicate
    date: "2026-09-13T17:05:13.016264+04:00"
    note: covered by ROAD-097 (remote MCP transport) — filed from this idea during the 2026-09-13 audit
---

# Remote MCP transport (HTTP, token-authed) + tools/list pagination

# Remote MCP transport (HTTP, token-authed) + tools/list pagination

MCP is stdio-only; official cloudflare/mcp (828 stars, remote OAuth) attacks the differentiator. Add streamable HTTP transport reusing serve daemon + tools/list cursor pagination; market fail-closed gating. Evidence: agent-4-competitive.md
