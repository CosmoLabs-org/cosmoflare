---
ulid: 01M2CB46XQ33M11PCNQ6FCJZS2
title: govulncheck gate in release-prepare
created: "2026-09-13T06:56:44.471747+04:00"
status: seed
source: agent
origin:
    session: 53
    trigger: audit-agent-9-infrastructure
    file: docs/audit/2026-09-13-cosmoflare/agent-9-infrastructure.md
tags:
    - audit
    - infrastructure
---

# govulncheck gate in release-prepare

No security scanning anywhere in the live (local) release pipeline. Add govulncheck (optionally gosec) to make release-prepare — the only pipeline that runs. Evidence: agent-9-infrastructure.md
