---
ulid: 01M2CB3W6DSRSX0K9R16JQ74WW
title: Output presenter interface to collapse 618 JSONOutput branches
created: "2026-09-13T06:56:33.485218+04:00"
status: seed
source: agent
origin:
    session: 53
    trigger: audit-agent-1-code-quality
    file: docs/audit/2026-09-13-cosmoflare/agent-1-code-quality.md
tags:
    - audit
    - code-quality
---

# Output presenter interface to collapse 618 JSONOutput branches

618 inline if-JSONOutput presentation branches across cmd/. Extract a presenter interface (JSON/table implementations built once per run) called unconditionally from handlers. Large effort, do per-command-group. Evidence: agent-1-code-quality.md
