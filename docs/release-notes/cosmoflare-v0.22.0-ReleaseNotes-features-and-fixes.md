---
project: cosmoflare
version: 0.22.0
date: 2026-09-08
previous: 0.21.0
slug: features-and-fixes
title: "features-and-fixes Release"
---

# cosmoflare v0.22.0 Release Notes

**Release Date**: September 8, 2026

**Previous**: v0.21.0

## Overview

This release brings 11 new features, 2 bug fixes, and 1 improvement.

## Highlights

This release completes the R2 surface and closes the observability loop. Sync correctness fixes end perpetual re-transfers (multipart ETag fallback, download mtime restore). Three new R2 capabilities ship: bucket custom domains (attach/verify/detach), object lifecycle rules (expire, abort multipart, InfrequentAccess transitions), and event notifications to Queues. The metrics stack goes live: a GraphQL Analytics producer layer feeds reliable daemon snapshots (partial with per-source errors, byte-based deltas) and a full alerting loop — rules evaluated against live analytics fire notifications with cooldown dedup. Six thousand lines of pre-rename dead code are removed after a port review.

## What's New

- bucket domain: new command group (attach/list/get/verify/update/detach) managing R2 bucket custom domains over the REST API, with zone auto-resolution, --json output, ownership/SSL activation polling, jurisdiction support, and optional min-TLS/cipher settings
- metrics: real usage analytics + reliable daemon snapshots — new AnalyticsService (GraphQL: zone HTTP, R2 storage/operations, Workers invocations, retention-validated windows); daemon publishes partial snapshots with per-source errors and populated profile instead of dropping on any failure; metrics --json gains windowed usage (--window, default 24h) with counts always printing
- alerts: evaluator closes the loop — enabled rules judged against live analytics (error-rate %, storage bytes, CPU-p99 latency, windowed failure counts) fire TriggerAlert through the serve bridge with per-rule cooldown; daemon evaluates every 5m, 'alerts check' runs one-shot from the CLI; cycles with missing data are skipped, never evaluated against zeros
- bucket lifecycle: get/set/clear R2 object lifecycle rules (expire by age/date, abort stale multipart uploads, transition to InfrequentAccess) with whole-config replace semantics and confirm-before-destruct
- bucket notifications: manage R2 event notification rules to Cloudflare Queues (list/create/get/delete, five exact action types plus object-create/object-delete groupings, prefix/suffix filters)
- manage R2 event notification rules to Queues (commit:461370a3)
- get, set, and clear R2 object lifecycle rules (commit:9e60bb6d)
- evaluator firing TriggerAlert from rules and live analytics (commit:7fd2d00b)
- partial snapshots with per-source errors, populated profile, usage analytics (commit:f9dd2b4f)
- GraphQL Analytics producers for zones, R2, and Workers (commit:ba0a51ea)
- attach, list, verify, update, and detach R2 bucket custom domains (commit:5291c9ce)

## Bug Fixes

- sync: --checksum now falls back to size/mtime comparison for multipart-uploaded objects (their -N ETags can never match a content checksum); downloads restore the remote LastModified so files are no longer re-downloaded on every sync
- fall back to size/mtime for multipart ETags; restore remote mtime after download (commit:d1e141da)

## Removed

- Removed pre-rename dead code: 4 cmd/*.go.disabled drafts, cmd_disabled/ (policy/restore/upload/webhook), and 4 internal/*_disabled packages (~7,800 lines). All superseded by live implementations or mock scaffolds; recoverable from git history. R2 lifecycle policies (no live equivalent yet) noted as a future roadmap candidate.

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 56 |
| Files changed | 79 |
| New features | 11 |
| Bug fixes | 2 |

---
_Full changelog: CHANGELOG.md_
