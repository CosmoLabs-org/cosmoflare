---
ulid: 01M1AF0ARSE9ZMS46G8C3QKDP7
id: IDEA-044
title: Auto-generate MCP tools from the cobra command tree
created: "2026-08-31T03:10:20.953926+04:00"
status: withered
source: agent
origin:
    session: 36
    trigger: audit-agent-4-competitive
    file: docs/audit/2026-08-31-cosmoflare/agent-4-competitive.md
tags:
    - audit
    - competitive
    - mcp
resolution:
    reason: implemented
    date: "2026-09-04T03:35:49.941255+04:00"
---

# Auto-generate MCP tools from the cobra command tree

# Auto-generate MCP tools from the cobra command tree

MCP server registers 8 tools vs 366 CLI commands (2% of own surface) at pkg/cosmoflare/mcp.go:286-335. Cloudflare's official MCP exposes 2,500 endpoints. Auto-generate tool schemas (name, flags, --json output) from the cobra tree; expose read-only wholesale, gate mutations behind guardrails. The only maintainable path to parity.
