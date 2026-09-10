---
project: CosmoDev-R2Go2
version: 0.7.1
date: 2026-05-16
previous: 0.7.0
slug: fixes
title: "fixes Release"
---

# CosmoDev-R2Go2 v0.7.1 Release Notes

**Release Date**: May 16, 2026

**Previous**: v0.7.0

## Overview

This release brings 3 bug fixes.

## Highlights

Fix config mapstructure tags preventing profile reload. Add 33 TUI render tests (35.9% to 86.1% coverage). Add 12 upload integration tests. Expand roadmap with 31 Cloudflare service items for Cosmoflare vision.

## Bug Fixes

- resolve duplicate status field in phase4 prompt frontmatter (commit:4ad3fea4)
- add mapstructure tags to Profile struct (commit:9619ec09)
- guard TestNewClientValidation against env vars (commit:b2962328)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 13 |
| Files changed | 73 |
| Bug fixes | 3 |

---
_Full changelog: CHANGELOG.md_
