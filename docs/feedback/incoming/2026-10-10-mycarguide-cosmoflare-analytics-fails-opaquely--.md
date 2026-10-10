---
ulid: 01M4HJJQYMZZAA0D98T2R22EFG
id: FB-25
title: '`cosmoflare analytics` fails opaquely + --json mode dumps usage text after the JSON. Investigate and fix.'
type: bug
status: implemented
priority: medium
complexity: ""
from_project: MyCarGuide
from_path: /Users/gabstudio/PROJECTS/MyCarGuide
to_project: cosmoflare
to_target: project
created: "2026-10-10T04:14:36.50006+04:00"
updated: "2026-10-10T19:36:33.851724+04:00"
suggested_conversion: bug
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
  notes: 'All three halves verified fixed 2026-10-10 against current build: R2 data-plane failure -> BUG-059/3fbc2bc (fail-fast with setup guidance, live-reproduced); --json stdout purity + error cause -> BUG-058-canonical/b2640bb (stdout parses as pure JSON, exit 1, structured error envelope); ''(assumed)'' perms language -> b2640bb (auth status shows real scope handling).'
---

# FB-pR22EFG: `cosmoflare analytics` fails opaquely + --json mode dumps usage text after the JSON. Investigate and fix.

**What happened (exact repro, 2026-10-10):**
1. `cosmoflare analytics --bucket=mycarguide` → `Error: failed to list objects: cosmoflare: ListObjects: failed to list objects` — the same string double-wrapped, no HTTP status, no endpoint, no cause. Meanwhile `cosmoflare auth status` reports "Connection successful" and the same CLOUDFLARE_API_TOKEN works fine against KV REST and GraphQL Analytics APIs for this account (287a...6990). Probably an R2 ListObjects permission gap or bucket-scoping issue — but nothing in the output lets you tell, which is the bug. Bonus irony: cosmoflare HAS a `decode` command for Cloudflare error codes; its own errors don't use it or link to it.
2. `cosmoflare analytics --bucket=mycarguide --json` → stdout carries valid JSON (`{"success":false,"error":"..."}`) and THEN non-JSON `Error: ...` + the full usage/flags dump on stdout. This breaks `| jq` and every scripted consumer: parse fails after the JSON object. JSON mode contract: JSON only on stdout; error text and usage go to stderr (or into the JSON envelope).
3. Related: `cosmoflare auth status` prints "Token permissions: R2 management (assumed)" — it assumes instead of reading the token's actual scopes (CF exposes token verification). An "assumed" permission model is exactly how failure #1 stayed invisible until invoked.

**Why it matters:** analytics is the entry point for the usage-audit workflow (MyCarGuide's CF review relied on it); a hard failure with no cause sends the operator to hand-rolled curl, which is what cosmoflare exists to replace. The --json violation breaks CI/agent consumers fleet-wide.

**Proposed fix:** surface the underlying cause (HTTP status + CF error code + endpoint) in ListObjects failures, run them through `cosmoflare decode`, and suggest the likely permission scope; enforce strict JSON-on-stdout in --json mode (diagnostics → stderr); replace "(assumed)" with real token-scope reads, or drop the line.

**Priority:** high — feature completely broken with misleading health signal.


## Recurrence (ingest duplicate)

- twin ulid: 01M4HJJQYMZZAA0D98T2R22EFG
- twin path: /Users/gabstudio/PROJECTS/cosmoflare/docs/feedback/incoming/2026-10-10-mycarguide-fb-pr22efg.md
- folded: 2026-10-10T19:35:21+04:00
