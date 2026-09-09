---
project: cosmoflare
version: 0.24.0
date: 2026-09-10
previous: 0.23.0
slug: features-and-fixes
title: "features-and-fixes Release"
---

# cosmoflare v0.24.0 Release Notes

**Release Date**: September 10, 2026

**Previous**: v0.23.0

## Overview

This release brings 8 new features, and 2 bug fixes.

## Highlights

Domain Center residuals shipped: legacy Page-Rule forwarding_url merge (WithPageRules + RedirectRule.Source), RedirectProber with loop detection, redirect-issue attention criterion, domains stats --check-redirects, doctor redirect-target section, TUI redirect badge; KV GetNamespace now one direct API call

## What's New

- Domain Center residuals — legacy pagerules merge + redirect-target attention check (FEAT-010) (FEAT-010)
- feat(doctor): redirect-target probe section — cmd supplies destinations, report carries results
- feat(cmd): domains stats --check-redirects probe pass
- feat(tui): domain detail pane renders redirect-issue badge when present
- feat(domains): merge legacy forwarding_url page rules via WithPageRules
- feat(domains): redirect-issue classification feeds needs-attention
- feat(domains): redirect prober with loop detection
- feat(redirects): map legacy forwarding_url page rules

## Bug Fixes

- fix(kv): GetNamespace fetches directly via REST instead of list-scan
- direct-GET not-found typing for bare 404s; never fabricate from null result (commit:7297d911)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 56 |
| Files changed | 59 |
| New features | 8 |
| Bug fixes | 2 |

---
_Full changelog: CHANGELOG.md_
