---
project: CosmoDev-R2Go2
version: 0.4.0
date: 2026-05-13
previous: 0.3.1
slug: features-and-fixes
title: "features-and-fixes Release"
---

# CosmoDev-R2Go2 v0.4.0 Release Notes

**Release Date**: May 13, 2026

**Previous**: v0.3.1

## Overview

This release brings 2 new features, and 1 bug fix.

## Highlights

Add multipart upload for large files with auto-threshold at 100MB. Add structured JSON output to all 10 remaining commands. Add terminal progress bars for uploads and downloads. Create integration test suite with 6 tests against live R2.

## What's New

- add --json output to all commands and wire progress bars (commit:fc69600b)
- implement multipart upload for large files (commit:48a6ec8b)

## Bug Fixes

- use composite ETag from CompleteMultipartUpload output (commit:e84e13ed)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 7 |
| Files changed | 49 |
| New features | 2 |
| Bug fixes | 1 |

---
_Full changelog: CHANGELOG.md_
