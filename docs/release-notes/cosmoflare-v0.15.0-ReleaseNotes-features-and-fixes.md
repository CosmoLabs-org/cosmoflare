---
project: cosmoflare
version: 0.15.0
date: 2026-06-09
previous: 0.14.0
slug: features-and-fixes
title: "features-and-fixes Release"
---

# cosmoflare v0.15.0 Release Notes

**Release Date**: June 9, 2026

**Previous**: v0.14.0

## Overview

This release brings 13 new features, and 1 bug fix.

## Highlights

Dashboard TUI with live monitoring, split-pane object browser, and S3 migration with resume. Keychain credential storage for secure config management.

## What's New

- Add OS keychain credential storage with automatic fallback to file-based config (FEAT-005)
- Add real-time TUI dashboard with live monitoring, bucket CRUD, and object browsing (ROAD-020)
- Add split-pane TUI object browser with folder navigation and object management (ROAD-002)
- Add real S3-to-R2 migration with resume support, concurrent transfers, and ETag verification (ROAD-007)
- replace simulated loop with real orchestrator and verify mode (ROAD-007 P-03/P-04) (commit:e96ef905)
- add checkpoint persistence and worker pool (ROAD-007 P-01/P-02) (commit:3f36fe40)
- integrate BrowserModel into dashboard, remove SectionObjectList (ROAD-002 P-06) (commit:4ba6f1d9)
- add BrowserModel with split-pane layout and folder navigation (ROAD-002 P-03/P-04/P-05) (commit:03d744ce)
- extend ListObjects and DataSource for browser (ROAD-002 P-01/P-02) (commit:17944372)
- add --interval flag, absorb metrics TUI into dashboard (ROAD-020 P-08) (commit:2ee51362)
- wire live data, monitoring, objects, and bucket CRUD into dashboard (ROAD-020 P-03/P-04/P-05/P-06/P-07) (commit:ae0b1b93)
- add DataSource interface with API and null backends (ROAD-020 P-01/P-02) (commit:78a655f8)
- add OS keychain credential storage (FEAT-005) (commit:5d5fd13a)

## Bug Fixes

- remove orphaned internal/api test files and fix TestNewClientValidation env dependency (commit:15752349)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 24 |
| Files changed | 70 |
| New features | 13 |
| Bug fixes | 1 |

---
_Full changelog: CHANGELOG.md_
