---
id: FB-001
title: GetNamespace does O(n) list scan
type: idea
status: implemented
priority: medium
complexity: ""
from_project: CosmoDev-R2Go2
from_path: /Users/gabstudio/PROJECTS/CosmoDev-R2Go2
to_project: CosmoDev-R2Go2
to_target: self
created: "2026-05-14T05:28:12.546155-03:00"
updated: "2026-09-09T19:51:33.924892+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 2027
suggested_workflow: []
response:
  acknowledged: null
  acknowledged_by: null
  started: null
  implemented: null
  rejected: null
  rejection_reason: null
  notes: Fixed 2026-09-09: kv.go GetNamespace now issues one direct REST GET via the credential-built restClient (BUG-039). Not-found keeps R2NotFoundError; auth errors pass through unlabeled. Full pkg/cosmoflare suite green.
---

# FB-001: GetNamespace does O(n) list scan

KVService.GetNamespace lists all namespaces then scans for matching ID. Should use direct API call or at minimum document the O(n) behavior for large accounts.


---

**Update (2026-09-09):**

Reopened 2026-09-09: status 'implemented' was premature — kv.go GetNamespace still list-scans. Tracked as issue above (direct conversion blocked: item is addressed to pre-rename 'CosmoDev-R2Go2').
