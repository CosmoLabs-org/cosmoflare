---
project: CosmoDev-R2Go2
version: 0.2.2
date: 2026-03-07
previous: 0.2.1
slug: fixes-and-improvements
title: "fixes-and-improvements Release"
---

# CosmoDev-R2Go2 v0.2.2 Release Notes

**Release Date**: March 7, 2026

**Previous**: v0.2.1

## Overview

This release brings 1 bug fix, and 2 improvements.

## Highlights

- Fixed 38 go vet issues across interactive/cli packages
- Comprehensive test coverage: API 88.2%/TUI 35.9%/interactive 6.6%
- Updated CI/CD to Go 1.25/1.26 with latest GitHub Actions

## Bug Fixes

- resolve 38 go vet issues across interactive and cli packages (commit:eb0c3b84)

## Improvements

- Consolidate formatBytes (6x) and maskAccountID (3x) into shared internal/utils package
- consolidate formatBytes (6x) and maskAccountID (3x) into internal/utils (commit:bec5f132)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 11 |
| Files changed | 78 |
| Bug fixes | 1 |

---
_Full changelog: CHANGELOG.md_
