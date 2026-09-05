---
ulid: 01M1AF1KXMS0BWCBN4EDXJRDM6
id: IDEA-049
title: Restore the binary release channel and ship first GitHub Release
created: "2026-08-31T03:11:03.092324+04:00"
status: withered
source: agent
origin:
    session: 36
    trigger: audit-agent-5-distribution
    file: docs/audit/2026-08-31-cosmoflare/agent-5-distribution.md
tags:
    - audit
    - distribution
    - release
resolution:
    reason: implemented
    date: "2026-09-05T21:35:00.832744+04:00"
---

# Restore the binary release channel and ship first GitHub Release

# Restore the binary release channel and ship first GitHub Release

Zero GitHub Releases exist despite 21 tags; release workflow failed 18/18 runs since March (root cause: internal/interactive races). Fix sequence: races -> ldflags rename sweep -> installer rewrite -> tag v0.18.0 with a release smoke gate (download assets, verify checksums + --version). Release notes exist in docs/release-notes/ for releases that never shipped.
