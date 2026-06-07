---
project: cosmoflare
version: 0.14.0
date: 2026-06-07
previous: 0.13.0
slug: features-and-fixes-and-improvements
title: "features-and-fixes-and-improvements Release"
---

# cosmoflare v0.14.0 Release Notes

**Release Date**: June 7, 2026

**Previous**: v0.13.0

## Overview

This release brings 9 new features, 6 bug fixes, and 2 improvements.

## Highlights

Real-time metrics dashboard, dev server webhooks, S3 migration re-enable, and Bubble Tea interactive models. 12,500 lines of tests from 19 parallel Sonnet agents; all pre-existing test failures fixed.

## What's New

- cosmoflare metrics — real-time TUI dashboard with live R2/Workers/KV stats, --interval flag, --json mode (ROAD-060)
- cosmoflare dev --notify — webhook notifications on dev server start/stop/error events (ROAD-072)
- cosmoflare migrate from-s3 — re-enabled S3-to-R2 migration command (ROAD-019)
- Bubble Tea interactive prompts — text input, list select, and confirm models replace hand-rolled stdin loop (ROAD-008)
- integration test suite with httptest mock servers for R2, Workers, and KV CRUD operations (ROAD-017)
- ~9,300 lines of unit tests across cmd/ and pkg/cosmoflare/ via 18 parallel Sonnet agents — covers command structure, flags, helpers, constructors, and validation
- re-enable S3-to-R2 migration command (ROAD-019) (commit:2033f576)
- add webhook notifications for dev server lifecycle (ROAD-072) (commit:9c691c8d)
- add real-time TUI metrics dashboard (ROAD-060) (commit:6fc9299a)

## Bug Fixes

- internal/migration/s3.go — resolved 11 build errors (missing helpers, AWS SDK call, pointer dereference)
- config.Save() viper .tmp extension bug — replaced with yaml.Marshal for reliable config persistence
- pre-existing test failures in internal/interactive and internal/webhook from r2go2→cosmoflare rename
- correct bucket update and palette render assertions (commit:bbc61391)
- resolve all pre-existing test failures in interactive and config (commit:0f6cd715)
- resolve 11 build errors in s3.go (commit:3ab8302a)

## Improvements

- USAGE.md — replaced 226 r2go2 references with cosmoflare across all CLI examples
- migrate prompts to Bubble Tea models (ROAD-008) (commit:d6963f24)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 20 |
| Files changed | 75 |
| New features | 9 |
| Bug fixes | 6 |

---
_Full changelog: CHANGELOG.md_
