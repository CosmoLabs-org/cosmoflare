---
reviewed_file: "docs/brainstorming/2026-06-08-road007-s3-migration-resume.md"
reviewed_at: 2026-06-08T03:00:00-03:00
mode: single
findings_total: 5
findings_critical: 0
findings_major: 3
findings_minor: 2
fixes_applied: 3
dimensions_checked: 10
reviewer: "opus"
---

# Independent Review — `2026-06-08-road007-s3-migration-resume.md`

## Summary

This brainstorming document designs a real S3-to-R2 migration engine with checkpoint-based resume for cosmoflare (ROAD-007). The spec replaces an existing simulated transfer loop with a worker pool, streaming transfers, and persistent checkpointing in `~/.cosmoflare/migrations/`. The architecture is solid — clean separation between checkpoint management, worker pool, and orchestration, with well-thought-out decisions on failure handling (retry 3x + skip) and concurrency (goroutine pool with channel).

The dominant pattern of issues was **missing robustness paths**: the spec originally didn't address checkpoint file corruption (crash during periodic flush), didn't specify how workers receive both the S3 and R2 clients (function signature gap), and used a fixed 5-minute per-object timeout that would be insufficient for multi-GB objects. A secondary issue was that the R2 `Upload` method does NOT auto-switch to multipart for large objects — the spec claimed this was handled internally but the library has separate `Upload` and `MultipartUpload` methods.

The spec's strengths are considerable: the checkpoint hash scheme (deterministic from bucket+filter), the periodic flush strategy (every 10 objects or 30s — not every object), the graceful SIGINT design using an atomic bool instead of context cancellation (lets in-flight transfers complete), and the clear "stale checkpoint" warning when resuming without `--resume`. These demonstrate practical experience with large-scale data migration patterns. After fixes, the spec is implementation-ready.

## Findings

| # | Dimension | Finding | Severity | Action |
|---|-----------|---------|----------|--------|
| 1 | Missing error paths (D3) | No handling for checkpoint file corruption. `saveCheckpoint` could crash mid-write (power loss, disk full). `loadCheckpoint` would then fail on corrupt JSON with no recovery path. | Major | Fixed: added atomic write (temp file + rename) for save, and graceful fallback (treat as no checkpoint + log warning) for corrupt load |
| 2 | Interface mismatches (D8) | Worker function signature not specified — workers need both the S3 client and R2 client but the spec only described them pulling from a channel. No clarity on how clients are passed. | Major | Fixed: added explicit `transferWorker` function signature with both clients as parameters |
| 3 | Assumptions (D5) | Spec says "R2 Upload handles multipart internally" in Out of Scope. Codebase verification: `Upload` and `MultipartUpload` are separate methods in `pkg/cosmoflare/upload.go`. Objects >100MB would fail or be slow without explicit multipart handling. | Major | Fixed: added size check in worker — use `MultipartUpload` for objects >100MB |
| 4 | Feasibility (D6) | Fixed 5-minute per-object timeout. A 10GB object at 10MB/s takes ~17 minutes. The timeout would kill the transfer prematurely. | Minor | Fixed: changed to size-scaled timeout: `max(5 minutes, size / 1MB/s)` |
| 5 | Frontmatter/chain (D10) | Chain trace: `origin: ROAD-007` is a roadmap ID, not an issue ID. FEAT-004 is linked to ROAD-007 but already `done`. No plan_ref (expected — plan not yet written). Timestamps ISO8601. | Minor | Noted: FEAT-004 is the comparison tool (already shipped), not the migration resume. Origin is correct as ROAD-007 since no dedicated issue exists for resume support. |

## Frontmatter Assessment

| Field | Status | Note |
|-------|--------|------|
| title | ✅ | Present |
| created | ✅ | ISO8601 with timezone |
| status | ✅ | approved |
| roadmap | ✅ | ROAD-007 |
| origin | ⚠️ | ROAD-007 (roadmap ID, not issue ID — no dedicated issue) |
| deliverables | ✅ | BR-01 through BR-07 present |
| last_reviewed | ✅ | Stamped during this review |

## Changes Made

- Added explicit `transferWorker` function signature showing both `s3Client` and `r2Client` parameters, plus `wg *sync.WaitGroup`
- Added size-based routing: objects >100MB use `r2Client.MultipartUpload` instead of `Upload`
- Changed per-object timeout from fixed 5 minutes to `max(5 minutes, size / 1MB/s)` for large object support
- Added checkpoint corruption safety: atomic write via temp file + rename for `saveCheckpoint`, graceful fallback for corrupt JSON on `loadCheckpoint`
