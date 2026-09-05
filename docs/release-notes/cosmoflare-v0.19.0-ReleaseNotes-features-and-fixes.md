---
project: cosmoflare
version: 0.19.0
date: 2026-09-05
previous: 0.18.0
slug: features-and-fixes
title: "features-and-fixes Release"
---

# cosmoflare v0.19.0 Release Notes

**Release Date**: September 5, 2026

**Previous**: v0.18.0

## Overview

This release brings 3 new features, and 11 bug fixes.

## Highlights

First GitHub Release ever shipped: the 18-attempt-failed release pipeline is unblocked (data races fixed, version stamping repaired, installer checksums added). Carries 13 of the 14 fixes from the 2026-08-31 comprehensive audit: silent multipart corruption aborted, sync pagination stops --delete data destruction, watcher never deletes unreadable paths, guardrails now enforced on all upload paths with default sensitive-file excludes, --json config errors emit the agent envelope, client transport options wired (timeout finally reaches both SDKs), profile resolution implemented, dead options removed, and the desktop app gains its full cf-* design system with WCAG live regions and an axe regression gate.

## What's New

- bridge SSE metrics channel to React Query cache (commit:be8637b2)
- wire MetricsProducer with --metrics-interval flag (commit:7f954756)
- add MetricsProducer with subscriber gating and delta detection (commit:4a7bdd24)

## Bug Fixes

- verify downloaded binary checksums before install (commit:ef7f8ed6)
- wire transport options, implement profile resolution, drop no-op options (commit:3b21556b)
- repoint version stamping and installers at cosmoflare (commit:342cf8e8)
- eliminate shared-state data races (BUG-030) (commit:9606ac6e)
- implement cf-* stylesheet, visible health state, and ARIA live regions (commit:5de362d6)
- enforce CheckUpload on upload paths and default-exclude sensitive files (commit:5ac6e976)
- abort multipart upload when reader is shorter than declared size (commit:3e826448)
- never report deletions for paths that failed to scan (commit:b105502c)
- emit JSON error envelope when config validation fails in --json mode (commit:eeee5864)
- paginate remote object listing across NextToken pages (commit:06c0718e)
- resolve three v1.1 review bugs (BUG-022, BUG-023, BUG-024) (commit:4f501e05)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 93 |
| Files changed | 226 |
| New features | 3 |
| Bug fixes | 11 |

---
_Full changelog: CHANGELOG.md_
