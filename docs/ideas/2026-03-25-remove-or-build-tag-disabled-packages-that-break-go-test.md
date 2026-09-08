---
id: IDEA-030
title: Remove or build-tag disabled packages that break go test
created: "2026-03-25T03:57:33.113067+01:00"
status: withered
source: agent
origin:
    session: 17
    trigger: audit-code-quality-tech-debt
tags:
    - audit
    - cleanup
resolution:
    reason: implemented
    date: "2026-09-08T15:08:11.636692+04:00"
    note: 2026-09-08 sweep removed all disabled packages/files; build+vet clean
---

# Remove or build-tag disabled packages that break go test

# Remove or build-tag disabled packages that break go test

6 packages with _disabled suffix or in migration/ fail to build: cmd_disabled, analytics_disabled, api_disabled, domain_disabled, migration, migration_disabled. They pollute go test ./... output. Either delete them, use Go build tags, or move to a separate module.
