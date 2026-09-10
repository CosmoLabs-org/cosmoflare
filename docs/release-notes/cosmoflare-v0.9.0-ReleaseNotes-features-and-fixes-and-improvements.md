---
project: CosmoDev-R2Go2
version: 0.9.0
date: 2026-05-18
previous: 0.8.0
slug: features-and-fixes-and-improvements
title: "features-and-fixes-and-improvements Release"
---

# CosmoDev-R2Go2 v0.9.0 Release Notes

**Release Date**: May 18, 2026

**Previous**: v0.8.0

## Overview

This release brings 11 new features, 10 bug fixes, and 1 improvement.

## Highlights

v0.9.0 delivers CORS management via Cloudflare Transform Rules (ROAD-016) and the cosmoflare status dashboard command (ROAD-050), plus a major push on test coverage. pkg/r2go2 went from 44.8% to 70.3%, cmd/ from 12.2% to 50%+, TUI hit 98.4%, and interactive reached 92.2%. The batch manager's dead collectResults() code was cleaned up. With coverage targets met across all packages, this release marks the project as release-ready for the Cosmoflare ecosystem.

## What's New

- implement CORS management via Transform Rules (ROAD-016) (commit:c6bce930)
- add cosmoflare status dashboard command (ROAD-050) (commit:b6483fb5)
- add FirewallService and CLI command (commit:239bbe62)
- add EmailService and CLI command (commit:eec97a47)
- add PageRuleService and CLI command (commit:6345c93c)
- add cosmoflare domains CLI command (commit:f62228a6)
- add cosmoflare doctor CLI command (commit:5b5d8d10)
- add DomainService with overview, detail, health enrichment, and formatting (commit:b64f7a87)
- add DoctorService with 4 diagnostic probes (commit:643c3bcd)
- add DomainService with overview, detail, and health enrichment (commit:6ea48be3)
- add HealthcheckService wrapping Cloudflare Healthcheck API (commit:79104da9)

## Bug Fixes

- GlowingText empty-string panic and easeInOutCubic output overflow (commit:c9a9d60b)
- prevent panics from overflow and divide-by-zero in progress bars (commit:8fe8b641)
- race condition in Execute() — goroutine lifecycle restructure (commit:08e6a5eb)
- fetch-before-update in CLI to preserve unmodified rule fields (commit:732af9de)
- address review issues — pointer Verified, flag-changed guards, test fix (commit:4f1053b8)
- restore JSON parsing for action values, fix priority docs (commit:602bc6d9)
- correct exit code documentation in --help (commit:1064b818)
- address review nits — comment accuracy, SOA→PrimaryNS rename (commit:caa824ab)
- address review issues — JSON duration types, DNS consistency, probe errors (commit:b57927a9)
- fetch-then-merge Update to avoid zero-value overwrites (commit:785fe4a4)

## Improvements

- remove dead collectResults() method and update tests (commit:a4996784)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 172 |
| Files changed | 386 |
| New features | 11 |
| Bug fixes | 10 |

---
_Full changelog: CHANGELOG.md_
