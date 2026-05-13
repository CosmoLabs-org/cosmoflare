---
id: FB-948
title: ccs roadmap add ignores --category and --effort flags (inconsistent with update)
type: idea
status: pending
priority: medium
complexity: ""
from_project: CosmoDev-R2Go2
from_path: /Users/gabstudio/PROJECTS/CosmoDev-R2Go2
to_project: ClaudeCodeSetup
to_target: project
created: "2026-05-13T15:49:18.645233-03:00"
updated: "2026-05-13T15:49:18.645233-03:00"
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
  notes: ""
---

# FB-948: ccs roadmap add ignores --category and --effort flags (inconsistent with update)

## Problem
`ccs roadmap add` silently ignores --category and --effort flags, printing warnings like "Unknown flag ignored". The sibling command `ccs roadmap update` accepts both flags correctly. Users expect add and update to accept the same metadata flags.

## Current vs Expected

Current:
```
$ ccs roadmap add "Upload guardrails" --category security --effort medium
⚠️  Unknown flag --category ignored
⚠️  Unknown flag --effort ignored
✅ Created ROAD-034: Upload guardrails
```

Then must run a second command:
```
$ ccs roadmap update ROAD-034 --category security --effort medium
✅ Updated ROAD-034
```

Expected: `ccs roadmap add` should accept --category and --effort at creation time, same as update.

## Why It Matters
Every roadmap item needs category and effort. The two-step create-then-update pattern is friction that adds up. Users discover the inconsistency mid-workflow when the warning appears.

## Priority
Low. Workaround exists (create then update). But it's a polish issue that affects every new roadmap item.

## Reproduction
```
ccs roadmap add "Test item" --category core --effort small
# Observe warnings
ccs roadmap update ROAD-XXX --category core --effort small
# Works fine
```

## Affected Files
The `add` command handler in CCS needs the same flag registrations as the `update` command handler. Likely in the roadmap cobra command registration.

## Suggested Implementation
Add --category, --effort, --priority, and --tags flags to the `add` subcommand, mirroring the `update` subcommand's flag set. Apply them to the newly created item before writing to disk.

