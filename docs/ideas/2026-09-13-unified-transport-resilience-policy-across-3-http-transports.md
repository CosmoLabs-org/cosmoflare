---
ulid: 01M2CB44PXQFVMSJD8586FR211
title: Unified transport resilience policy across 3 HTTP transports
created: "2026-09-13T06:56:42.205417+04:00"
status: harvested
source: agent
origin:
    session: 53
    trigger: audit-agent-7-api-design
    file: docs/audit/2026-09-13-cosmoflare/agent-7-api-design.md
tags:
    - audit
    - api-design
promoted_to: FEAT-039
---

# Unified transport resilience policy across 3 HTTP transports

# Unified transport resilience policy across 3 HTTP transports

One timeout/retry/backoff policy across R2 SDK, restClient, cloudflare-go paths, configurable via ClientOption, documented once. Evidence: agent-7-api-design.md
