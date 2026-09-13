---
ulid: 01M2CB4PMD5883M5EQ0ZQZPEQS
title: SSE announcement bus (WCAG 4.1.3)
created: "2026-09-13T06:57:00.55725+04:00"
status: seed
source: agent
origin:
    session: 53
    trigger: audit-agent-18-accessibility
    file: docs/audit/2026-09-13-cosmoflare/agent-18-accessibility.md
tags:
    - audit
    - design
---

# SSE announcement bus (WCAG 4.1.3)

Notifications silent on non-Notifications tabs — live regions unmount (App.tsx:152). One always-mounted aria-live=polite region at App level narrating arrivals + health transitions. Evidence: agent-18-accessibility.md
