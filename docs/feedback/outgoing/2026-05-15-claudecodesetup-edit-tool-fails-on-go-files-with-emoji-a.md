---
id: FB-959
title: Edit tool fails on Go files with emoji and tab indentation
type: idea
status: pending
priority: medium
complexity: ""
from_project: CosmoDev-R2Go2
from_path: /Users/gabstudio/PROJECTS/CosmoDev-R2Go2
to_project: ClaudeCodeSetup
to_target: project
created: "2026-05-15T00:00:19.643887-03:00"
updated: "2026-05-15T00:00:19.643887-03:00"
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

# FB-959: Edit tool fails on Go files with emoji and tab indentation

## Problem
The Edit tool consistently fails to match strings in Go source files that contain Unicode emoji characters (like the emoji used in printInfo/printSuccess calls) or files that use tab indentation with specific patterns. The error is always 'String to replace not found in file' even when the exact string is visible in Read output.

## Current vs Expected
Read output shows:
  printInfo("⬇️  Downloading object: %s/%s", bucketName, objectKey)

Edit attempt:
  old_string: printInfo("⬇️  Downloading object: %s/%s", bucketName, objectKey)
  Result: String to replace not found in file

Workaround: Used Python scripts with open()/replace() which matched the same string perfectly.

Expected: Edit tool should handle Unicode emoji and tab-indented Go files the same way Python's string matching does.

## Why It Matters
Go CLI tools commonly use emoji in user-facing output. This session required 5+ Python/sed workarounds for edits on cmd/object.go, adding significant friction and token cost. Each failed Edit attempt wastes a tool call.

## Priority
Medium — workarounds exist but they're slower and add context noise. This will recur in any Go CLI project with emoji output.

## Reproduction
1. Create a Go file with tab indentation and emoji in string literals
2. Use Read to view the file
3. Copy exact text from Read output into Edit old_string
4. Edit fails with 'not found'
5. Same string matches fine via Python open().replace()

## Affected Files
The Edit tool implementation — likely a Unicode normalization or whitespace normalization issue in the string matching logic.

## Suggested Implementation
Investigate whether the Edit tool normalizes Unicode or strips invisible characters before matching. Compare against Python's exact byte matching which works correctly. The issue may be NFC vs NFD normalization of emoji characters, or tab-to-space conversion in the matching path.

