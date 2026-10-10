---
ulid: 01M4JE04NXMYBP29W9QH130P9K
id: FB-30
title: Permissions preflight — check token scopes BEFORE operations fail with "request is not authorized"
type: feature
status: pending
priority: medium
complexity: ""
from_project: MyCarGuide
from_path: /Users/gabstudio/PROJECTS/MyCarGuide
to_project: cosmoflare
to_target: project
created: "2026-10-10T12:13:47.069933+04:00"
updated: "2026-10-10T12:13:47.069933+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 19
suggested_workflow: []
response:
  acknowledged: null
  acknowledged_by: null
  started: null
  implemented: null
  rejected: null
  rejection_reason: null
  notes: ""
---

# FB-p130P9K: Permissions preflight — check token scopes BEFORE operations fail with "request is not authorized"

**What happened:** creating a zone Cache Rule via the Rulesets API returned `{"message":"request is not authorized"}` with zero indication of WHICH scope was missing. The same token happily serves KV REST, GraphQL Analytics, D1 insights, and Pages deploys — so the failure was invisible until the exact call failed. `cosmoflare auth status` prints "Token permissions: R2 management (assumed)" — it ASSUMES instead of reading actual scopes, so it could not have warned us. Both the Cache Rule and the WAF rule (BUG-011 follow-ups) are blocked on this blind spot until the token is edited by hand in the dashboard.

**Why it matters:** every CosmoLabs project will eventually need zone-level config (cache rules, WAF, transforms). A platform wrapper whose health check says "Connection successful" while the operation-critical scopes are absent trains operators to discover permission gaps one failed API call at a time — the worst possible place, mid-task.

**Proposed:** (1) read REAL token scopes (Cloudflare's token-verification endpoint) and print them in `auth status` — drop "(assumed)"; (2) a scope requirement map per command family (rulesets:edit for cache/WAF rules, kv per namespace ops, d1 read/write, r2 list…) surfaced as `cosmoflare auth preflight <command>`; (3) before any write-class command, check the map and print a one-line "missing scope: Zone:Rulesets:Edit" instead of letting the API reject it.

**Repro/evidence:** MyCarGuide 2026-10-10: `POST /zones/<id>/rulesets/phases/http_request_cache_settings/entrypoint` → "request is not authorized"; same token succeeds on /zones, /client/v4/graphql, KV REST, wrangler d1 insights/execute --remote, pages deploy. Companion: FB-pR22EFG (opaque analytics failure — same root: unknown scopes).

**Priority:** high — unblocks the entire zone-config surface (cache rules are the #1 CF efficiency lever per the corpus FB).


## Recurrence (ingest duplicate)

- twin ulid: 01M4JE04NXMYBP29W9QH130P9K
- twin path: /Users/gabstudio/PROJECTS/cosmoflare/docs/feedback/incoming/2026-10-10-mycarguide-permissions-preflight--check-token-sc.md
- folded: 2026-10-10T19:35:21+04:00
