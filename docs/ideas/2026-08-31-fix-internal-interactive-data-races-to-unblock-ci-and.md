---
ulid: 01M1AF0FTBVH2ZA3V4Q3EG24S7
id: IDEA-045
title: Fix internal/interactive data races to unblock CI and Release
created: "2026-08-31T03:10:26.123053+04:00"
status: seed
source: agent
origin:
    session: 36
    trigger: audit-agent-5-distribution
    file: docs/audit/2026-08-31-cosmoflare/agent-5-distribution.md
tags:
    - audit
    - distribution
    - ci
---

# Fix internal/interactive data races to unblock CI and Release

go test -race ./internal/interactive/ fails (TestEncryptDecryptRoundtrip et al) — package-level shared state across 520 tests. This fails the -race gate in both ci.yml:55 and release.yml:31, making master CI red and the release pipeline 18/18 failed. Isolate the shared state (key cache/terminal state), mutex or per-test it.
