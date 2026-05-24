---
project: CosmoDev-R2Go2
version: 0.10.0
date: 2026-05-24
previous: 0.9.0
slug: features-and-fixes
title: "features-and-fixes Release"
---

# CosmoDev-R2Go2 v0.10.0 Release Notes

**Release Date**: May 24, 2026

**Previous**: v0.9.0

## Overview

This release brings 4 new features, and 4 bug fixes.

## Highlights

v0.10.0 adds a VS Code-style Ctrl+P command palette with fuzzy search to the TUI dashboard (FEAT-001), 12 service interfaces for mock testing (TASK-002), and fixes 3 bugs: download progress bar rendering, shared CLI flag isolation, and KV metadata/TTL wiring via bulk endpoint fallback. Includes a comprehensive 360° audit scoring 70.8/100 with 10 critical findings documented.

## What's New

- # FEAT-001: TUI command palette with fuzzy search

**Type**: feature
**Status**: closed
**Plan**: docs/planning-mode/2026-05-18-tui-command-palette.md
**Created**: 2026-02-26

## Description

Ctrl+P fuzzy search across all available commands, context-aware, like VS Code/lazygit
- comprehensive 360° project audit — 70.8/100 (Mature) (commit:390657f9)
- add command palette with fuzzy search (FEAT-001) (commit:14a1e739)
- add service interfaces for testability (TASK-002) (commit:decbba0f)

## Bug Fixes

- correct number shortcut tests + file 3 bugs + brainplan FEAT-001/TASK-002 (commit:8b527f9a)
- wire metadata and TTL options into Put() (commit:43201691)
- use local kvForce flag instead of shared workerForce variable (commit:2e00505e)
- render progress bar during download transfer (commit:22411727)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 58 |
| Files changed | 176 |
| New features | 4 |
| Bug fixes | 4 |

---
_Full changelog: CHANGELOG.md_
