---
project: CosmoDev-R2Go2
version: 0.3.1
date: 2026-05-08
previous: 0.3.0
slug: fixes
title: "fixes Release"
---

# CosmoDev-R2Go2 v0.3.1 Release Notes

**Release Date**: May 8, 2026

**Previous**: v0.3.0

## Overview

This release brings 2 bug fixes.

## Highlights

Fix PersistentPreRun blocking non-API commands. Correct Go version from non-existent 1.25.3 to 1.26. Add real YAML parsing to bucket import. Replace fake string matching with proper glob and regex in object search. Delete 1,728 lines of mock API client. Remove tracked binaries and add MIT LICENSE.

## Bug Fixes

- update Go version, remove mock API client, add LICENSE (commit:d5c3cbe4)
- resolve 6 CLI bugs across bucket, object, and root commands (commit:5f54356c)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 4 |
| Files changed | 34 |
| Bug fixes | 2 |

---
_Full changelog: CHANGELOG.md_
