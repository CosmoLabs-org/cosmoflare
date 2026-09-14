---
ulid: 01M2CB3W6DSRSX0K9R16JQ74WW
title: Output presenter interface to collapse 618 JSONOutput branches
created: "2026-09-13T06:56:33.485218+04:00"
status: withered
source: agent
origin:
    session: 53
    trigger: audit-agent-1-code-quality
    file: docs/audit/2026-09-13-cosmoflare/agent-1-code-quality.md
tags:
    - audit
    - code-quality
promoted_to: FEAT-040
resolution:
    reason: implemented
    date: "2026-09-14T22:31:36.832629+04:00"
    ref: FEAT-040
    note: Resolved via FEAT-040 (complete)
---

# Output presenter interface to collapse 618 JSONOutput branches

# Output presenter interface to collapse 618 JSONOutput branches

# Output presenter interface to collapse 618 JSONOutput branches

618 inline if-JSONOutput presentation branches across cmd/. Extract a presenter interface (JSON/table implementations built once per run) called unconditionally from handlers. Large effort, do per-command-group. Evidence: agent-1-code-quality.md
