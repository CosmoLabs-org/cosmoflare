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
updated: "2026-05-18T04:11:21.352281-03:00"
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
  notes: Cloudflare SDK (v0.116.0) has no GetWorkersKVNamespace method — only List/Create/Delete/Update. The list-and-scan approach is the only option. Marking as SDK limitation, not a code bug. Can revisit if Cloudflare adds a direct get endpoint.
---

# FB-001: GetNamespace does O(n) list scan

KVService.GetNamespace lists all namespaces then scans for matching ID. Should use direct API call or at minimum document the O(n) behavior for large accounts.

