---
ulid: 01M1AEZJ5B6PK573DTVHMRV04D
id: IDEA-040
title: Paginate sync ListRemoteObjects to stop --delete data destruction
created: "2026-08-31T03:09:55.755283+04:00"
status: seed
source: agent
origin:
    session: 36
    trigger: audit-agent-2-core-logic
    file: docs/audit/2026-08-31-cosmoflare/agent-2-core-logic.md
tags:
    - audit
    - core-logic
---

# Paginate sync ListRemoteObjects to stop --delete data destruction

cmd/sync.go:428 lists maxKeys=1000 once and discards NextToken. Beyond 1000 objects, sync down --delete treats remote objects as absent and DELETES matching local files; sync up re-uploads everything past 1000 every run. Fix: loop on result.NextToken as continuationToken. Test with fake backend returning 2 pages.
