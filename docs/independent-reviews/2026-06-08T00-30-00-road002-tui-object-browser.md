---
reviewed_file: "docs/brainstorming/2026-06-07-road002-tui-object-browser.md"
reviewed_at: 2026-06-08T00:30:00-03:00
mode: single
findings_total: 7
findings_critical: 2
findings_major: 3
findings_minor: 2
fixes_applied: 6
dimensions_checked: 10
reviewer: "opus"
---

# Independent Review — `2026-06-07-road002-tui-object-browser.md`

## Summary

This brainstorming document designs a split-pane TUI object browser for cosmoflare (ROAD-002), building on the basic flat list shipped in ROAD-020. The design is architecturally sound — a self-contained BrowserModel sub-model with well-defined boundaries, sensible narrow/wide mode switching, and prefix-based folder navigation. However, the document suffered from a pattern of **interface specification drift**: earlier code blocks showed one method signature while later prose described a different one, and multiple library gaps were understated.

The two critical findings were both about library extension scope: (1) the `FetchObjects` method showed contradictory signatures — `page int` in the code block but `continuationToken string` in the prose — and the R2Client `ListObjects` method has no continuation token parameter at all, making token-based pagination impossible without a library change the spec didn't acknowledge; (2) the `CommonPrefixes` extraction was described as a "~5 line" change but the S3 SDK returns `[]types.CommonPrefix` (a struct with a `.Prefix` field), not `[]string` — requiring careful extraction and also the continuation token extension.

The spec's strengths are notable: clear scope boundaries with ROAD-002's Out of Scope section, the decision to drop `StorageClass` rather than extend `HeadResult`, the two-keypress delete pattern reuse, and good error path coverage for delete/HeadObject (added in a prior review round). The narrow-mode single-pane design is elegant. After fixes, the spec is implementation-ready.

## Findings

| # | Dimension | Finding | Severity | Action |
|---|-----------|---------|----------|--------|
| 1 | Contradiction (D4) | `FetchObjects` shown with `prefix string, page int` in code block but described as `prefix, continuationToken string` in prose below. Two conflicting signatures in the same section. | Critical | Fixed: unified to token-based signature throughout |
| 2 | Impossible operations (D2) | R2Client `ListObjects` has no `ContinuationToken` input parameter. Token-based pagination can't work without extending the library interface AND implementation — a larger change than acknowledged (~15 lines, not ~5). | Critical | Fixed: added "Library gap — ContinuationToken" section documenting the full scope |
| 3 | Assumption surfacing (D5) | S3 SDK `CommonPrefixes` is `[]types.CommonPrefix` (struct with `.Prefix` field), not `[]string`. Extraction requires `aws.ToString(cp.Prefix)` per entry. | Major | Fixed: noted SDK type in the CommonPrefixes gap section |
| 4 | Missing error paths (D3) | No handling specified for: (a) `Enter` on bucket in left pane when object fetch fails, (b) `Backspace` at root prefix in narrow mode (visibility swap not stated). | Major | Fixed: added both error/behavior paths to Object Actions section |
| 5 | Structural gaps (D1) | `SectionObjectList` removal has 18 references across source + test files. Migration path for renumbering section constants and updating tests not mentioned. | Major | Fixed: added note about 18 references and renumbering in File Changes table |
| 6 | Frontmatter/chain (D10) | Chain trace: `origin: ROAD-002` is a roadmap ID not an issue ID (ccs doc-review reports gap). No plan_ref yet (expected — plan not written). Roadmap still `captured` status. | Minor | Note: plan_ref will auto-link when plan is written. Roadmap status update deferred to implementation. |
| 7 | Scope (D7) | Scope is well-controlled. Out of Scope section is explicit. No creep detected. | Minor | Confirmed sound: clear boundary with ROAD-002 deliverables, no feature bleed. |

## Frontmatter Assessment

| Field | Status | Note |
|-------|--------|------|
| title | ✅ | Present |
| created | ✅ | ISO8601 with timezone |
| status | ✅ | approved |
| roadmap | ✅ | ROAD-002 |
| origin | ⚠️ | ROAD-002 (roadmap ID, not issue ID — ccs chain trace flags this) |
| deliverables | ✅ | BR-01 through BR-08 present |
| last_reviewed | ✅ | Updated to current review timestamp |

## Changes Made

- Unified `FetchObjects` signature to token-based: `(ctx, bucket, prefix, continuationToken string)` — removed contradictory `page int` code block
- Added `NextToken string` and `HasMore bool` to `ObjectListing` type, removed `Total int`
- Added "Library gap — ContinuationToken" paragraph documenting R2Client interface extension need (~15 lines)
- Updated CommonPrefixes gap note to document S3 SDK type extraction (`[]types.CommonPrefix` → `aws.ToString(cp.Prefix)`)
- Added error paths: bucket selection failure keeps focus on left pane, backspace at root in narrow mode swaps visible pane
- Added `SectionObjectList` removal scope note (18 references, renumbering needed) to File Changes table
- Added `pkg/cosmoflare/client.go` to File Changes table for interface extension
- Updated review stamp with current timestamp and report reference
