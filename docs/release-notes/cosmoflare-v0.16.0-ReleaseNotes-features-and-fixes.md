---
project: cosmoflare
version: 0.16.0
date: 2026-06-20
previous: 0.15.0
slug: features-and-fixes
title: "features-and-fixes Release"
---

# cosmoflare v0.16.0 Release Notes

**Release Date**: June 20, 2026

**Previous**: v0.15.0

## Overview

This release brings 8 new features, and 6 bug fixes.

## Highlights

FEAT-006 Domain Management Center (RedirectService, RegistrarService, domains + redirects CLI, TUI browser); BUG-021 test-compilation fix; BUG-020 webhook ID collisions; keychain dialog-flood fix

## What's New

- Domain Management Center — TUI dashboard, redirect visibility, registrar overlay (FEAT-006) (FEAT-006)
- add DomainBrowserModel + quick-add redirect + domains tui command (P-06/P-07) (commit:691e4598)
- add domains command tree (get/stats/ns/redirects) with service factory (P-04) (commit:bea72bb6)
- add redirects command group for modern Redirect Rules (P-05) (commit:5eda1243)
- add NewRedirectServiceFromCreds + NewRegistrarServiceFromCreds (Wave C seam) (commit:403f08b9)
- enrich DomainDetail with redirects+registrar, add SummarizeDomains (P-03) (commit:aaec93b6)
- add RedirectService List/Create/Delete for modern Redirect Rules (P-01) (commit:556c8ca7)
- add RegistrarService registration overlay (P-02) (commit:5dffe87c)

## Bug Fixes

- Removed 7 orphaned R2Go2-era test files that broke 'go test ./...' compilation (BUG-021)
- Keychain layer no longer probes the macOS keychain under tests or when COSMOFLARE_NO_KEYCHAIN=1, eliminating spurious security dialogs
- independent review of Cosmoflare Desktop brainplan — 8 findings (commit:9bea55d6)
- never use OS keychain under test or when disabled (no `security` noise) (commit:da335a3d)
- remove 7 orphaned R2Go2-era test files breaking compilation (BUG-021) (commit:19b61398)
- unique IDs via atomic counter, not UnixNano (BUG-020) (commit:07017e82)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 36 |
| Files changed | 117 |
| New features | 8 |
| Bug fixes | 6 |

---
_Full changelog: CHANGELOG.md_
