---
ulid: 01M1AEZWX7A7BWXVHGBEZ4YNBB
id: IDEA-042
title: Wire guardrails CheckUpload into mutation paths
created: "2026-08-31T03:10:06.759697+04:00"
status: withered
source: agent
origin:
    session: 36
    trigger: audit-agent-2-core-logic
    file: docs/audit/2026-08-31-cosmoflare/agent-2-core-logic.md
tags:
    - audit
    - core-logic
    - security
resolution:
    reason: implemented
    date: "2026-09-04T03:35:49.839015+04:00"
---

# Wire guardrails CheckUpload into mutation paths

# Wire guardrails CheckUpload into mutation paths

GuardrailChecker (bucket allowlist, max_file_size, blocked_keys) has zero production callers — parsed and tested but never enforced. A .cosmoflare.yaml with blocked_keys:[.env] uploads .env anywhere unimpeded. Wire CheckUpload into client.Upload/MultipartUpload or at minimum cmd/sync.go and cmd/watch.go; default-exclude .git/, .env*, .DS_Store. This is the defensible agent-safety niche per the competitive analysis.
