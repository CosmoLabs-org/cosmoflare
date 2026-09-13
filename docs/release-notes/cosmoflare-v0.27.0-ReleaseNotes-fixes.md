---
project: cosmoflare
version: 0.27.0
date: 2026-09-13
previous: 0.26.0
slug: fixes
title: "fixes Release"
---

# cosmoflare v0.27.0 Release Notes

**Release Date**: September 13, 2026

**Previous**: v0.26.0

## Overview

This release brings 2 bug fixes.

## Highlights

Sync-engine correctness hardening from the 2026-09-13 360-degree audit: direction-aware down-sync deletes, exclude-protected deletions, timeout-free S3 data plane, cancellation-safe multipart aborts. Auth rotate revoke-ordering fix, apply gated behind --delete-unmanaged, KV duplicate-title refusal, --include implemented, desktop WCAG 4.1.3 live region, govulncheck release gate with 4 dependency security upgrades (x/text, x/net, aws eventstream, aws s3).

## Bug Fixes

- Sync engine data-loss fixes from 360-degree audit: direction-aware deletes for sync down --delete (BUG-040), exclude-protected deletes (BUG-041), timeout-free S3 data-plane client so large transfers survive past 30s (BUG-042), cancellation-safe multipart aborts (BUG-043). Also: auth rotate --revoke-old no longer revokes the new token (BUG-044), apply deletes gated behind --delete-unmanaged (BUG-049), KV duplicate-title ambiguity refused (BUG-048), --include sync filter implemented (BUG-051), desktop WCAG 4.1.3 live region (BUG-047), release archives no longer bundle docs/ (BUG-045), Dockerfile toolchain pinned (BUG-050).
- resolve all 12 critical bugs from the 2026-09-13 360-degree audit (commit:f29430a6)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 39 |
| Files changed | 157 |
| Bug fixes | 2 |

---
_Full changelog: CHANGELOG.md_
