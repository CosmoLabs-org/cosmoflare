---
id: IDEA-MN5GDTBT
title: Remove or build-tag disabled packages that break go test
created: "2026-03-25T03:57:33.113067+01:00"
status: seed
source: agent
origin:
    session: 17
    trigger: audit-code-quality-tech-debt
tags:
    - audit
    - cleanup
---

# Remove or build-tag disabled packages that break go test

6 packages with _disabled suffix or in migration/ fail to build: cmd_disabled, analytics_disabled, api_disabled, domain_disabled, migration, migration_disabled. They pollute go test ./... output. Either delete them, use Go build tags, or move to a separate module.
