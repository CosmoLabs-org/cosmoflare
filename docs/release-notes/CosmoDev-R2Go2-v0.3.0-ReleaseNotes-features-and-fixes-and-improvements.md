---
project: CosmoDev-R2Go2
version: 0.3.0
date: 2026-05-08
previous: 0.2.2
slug: features-and-fixes-and-improvements
title: "features-and-fixes-and-improvements Release"
---

# CosmoDev-R2Go2 v0.3.0 Release Notes

**Release Date**: May 8, 2026

**Previous**: v0.2.2

## Overview

This release brings 4 new features, 2 bug fixes, and 2 improvements.

## Highlights

New public Go library (pkg/r2go2/) with real S3/Cloudflare API wiring. CLI commands migrated from internal/api stubs to new library. Added 5-tier cache policy engine, upload guardrails, and JSONL audit logging.

## What's New

- Extract public Go library into pkg/r2go2/ with real S3/Cloudflare API wiring
- extract public library into pkg/r2go2/ (commit:d5352b38)
- add animation suppression for faster tests (commit:6155b287)
- add InputReader interface for testability (commit:6dbf526b)

## Bug Fixes

- correct binary name and add CCS integration (commit:386a8a4a)
- resolve 5 critical bugs from codebase audit (commit:36c16c53)

## Improvements

- Wire CLI commands to pkg/r2go2 library replacing internal/api stubs
- wire CLI commands to pkg/r2go2 library (commit:a15ece85)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 23 |
| Files changed | 222 |
| New features | 4 |
| Bug fixes | 2 |

---
_Full changelog: CHANGELOG.md_
