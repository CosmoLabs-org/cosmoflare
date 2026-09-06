---
project: cosmoflare
version: 0.21.0
date: 2026-09-07
previous: 0.20.0
slug: features-and-improvements
title: "features-and-improvements Release"
---

# cosmoflare v0.21.0 Release Notes

**Release Date**: September 7, 2026

**Previous**: v0.20.0

## Overview

This release brings 2 new features, and 2 improvements.

## Highlights

Daemon error contract v2 maps typed errors to 400/401/403/404/429/502 with {error, code} bodies; first release shipped full binary distribution (v0.20.0, 5 platforms); git history purged of 630 MB dead blobs

## What's New

- Published first release with full binary distribution (v0.20.0: 5 binaries + checksums)
- map typed daemon errors to stable HTTP status + code (commit:d8f8f3c8)

## Improvements

- Daemon REST errors map typed R2 errors to stable HTTP statuses (400/401/403/404/429/502) with {error, code} JSON bodies — desktop can distinguish user-fixable from transient failures

## Removed

- Purged 630 MB of dead blobs from git history (recovery patches, 12 old binaries) via git filter-repo; repo pack 485 MiB → 16.7 MiB

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 19 |
| Files changed | 92 |
| New features | 2 |

---
_Full changelog: CHANGELOG.md_
