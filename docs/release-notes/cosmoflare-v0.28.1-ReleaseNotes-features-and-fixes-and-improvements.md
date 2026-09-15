---
project: cosmoflare
version: 0.28.1
date: 2026-09-15
previous: 0.28.0
slug: features-and-fixes-and-improvements
title: "features-and-fixes-and-improvements Release"
---

# cosmoflare v0.28.1 Release Notes

**Release Date**: September 15, 2026

**Previous**: v0.28.0

## Overview

This release brings 1 new feature, 1 bug fix, and 7 improvements.

## Highlights

Patch release: release archives repackaged with root-level files (v0.28.0 tarballs carried ../ path entries that tar and Homebrew refuse). All 85 god functions split under funlen=80 with golangci enforcement adopted. Daemon wire contract now typed end-to-end with generated TypeScript types and a release drift gate; cmd/ test coverage raised from 47% to 60%.

## What's New

- typed daemon wire contract + TS codegen (FEAT-042) (commit:19449791)

## Bug Fixes

- dist archives carried ../ path entries (commit:bfd2aca5)

## Improvements

- split loadBuiltinThemes/createDefaultTutorials under funlen=80 (TASK-009) (commit:4c43a61b)
- split browser handleKey/renderRightPane under funlen=80 (TASK-009) (commit:9f4c82fe)
- split installer TUI god functions under funlen=80 (TASK-009) (commit:eb370472)
- split config/status/setup/mcp functions under funlen=80 (TASK-009) (commit:fb025d98)
- split 3 long fns under funlen=80 (TASK-009) (commit:fd0eee51)
- split cache/compare/watch functions under funlen=80 (TASK-009) (commit:e2553924)
- split file-table funcs for funlen (TASK-009) (commit:2f030734)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 82 |
| Files changed | 121 |
| New features | 1 |
| Bug fixes | 1 |

---
_Full changelog: CHANGELOG.md_
