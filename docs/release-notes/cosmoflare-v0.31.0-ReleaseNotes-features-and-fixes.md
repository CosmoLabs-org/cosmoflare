---
project: cosmoflare
version: 0.31.0
date: 2026-09-24T17:02:15.981985+04:00
previous: 0.30.0
slug: features-and-fixes
title: "features-and-fixes Release"
---

# cosmoflare v0.31.0 Release Notes

**Release Date**: September 24, 2026

**Previous**: v0.30.0

## Overview

This release brings 6 new features, and 7 bug fixes.

## Highlights

Seven services gain first-of-its-kind CLI coverage: Logpush job management with destination ownership validation, Load Balancer pools and monitors, Waiting Room, Spectrum, Page Shield, Turnstile, and Web Analytics (FEAT-037). Production data workflows get d1 push-sql — statement-aware batched push with retry, split-recovery, and kill-safe resume (FEAT-018). Safety becomes structural: all seventeen registry-destructive commands preview dry unless --force, and CLI paths derive from the live command tree so renames flow through (TASK-015); profile-prefix scoping is declared once at command definition instead of remembered per runner (TASK-011). The desktop tier gains notification severity levels and a light theme (FEAT-041).

## What's New

- Profile-prefix scoping declared at command definitions across the CLI (TASK-011)
- d1 push-sql: batched remote SQL push with retry, split recovery, resume (FEAT-018)
- Long-tail services — first CLI coverage: Logpush, Load Balancer pools/monitors, Waiting Room, Spectrum, Page Shield, Turnstile, Web Analytics (FEAT-037)
- Desktop: notification severity levels and light theme (FEAT-041)
- Structural safety defaults: every registry-destructive command runs dry unless --force; CLI paths derived from the live command tree (TASK-015)
- page-shield, turnstile, web-analytics CLIs + wave-3 registry (commit:4d62dd94)

## Bug Fixes

- assert the SDK's PUT transport; decode-able delete stub (commit:74038865)
- salvage repairs — struct table, capture deadlock, validation creds (commit:b610f7d5)
- stateful stub — re-fetch after PUT returns the updated job (commit:f7f1048d)
- Today() honors the service clock — date-rollover time bomb (commit:b3bf236c)
- salvage repairs — helper name, JSON-escaping-safe assertions, registry entry (commit:f9d129d5)
- stub localStorage in test setup — jsdom ships none (commit:1bf2c0cc)
- register d1 parity in the command registry (commit:b786e50c)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 57 |
| Files changed | 136 |
| New features | 6 |
| Bug fixes | 7 |

---
_Full changelog: CHANGELOG.md_
