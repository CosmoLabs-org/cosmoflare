---
project: cosmoflare
version: 0.25.0
date: 2026-09-10
previous: 0.24.0
slug: features-and-fixes-and-improvements
title: "features-and-fixes-and-improvements Release"
---

# cosmoflare v0.25.0 Release Notes

**Release Date**: September 10, 2026

**Previous**: v0.24.0

## Overview

This release brings 12 new features, 4 bug fixes, and 1 improvement.

## Highlights

Rate-limit traffic-class matrix + live trip probe (FEAT-013); knowledge layer hardening (idempotent decode, stdlib reuse); fixes: webhook retry body, dry-run config delete; ~3,300 test lines across 8 packages

## What's New

- decode, knowledge, ratelimit list/create commands
- ratelimit probe command + create --probe + traffic-class advisory
- knowledge layer: endpoint registry, plan caps, error decoding
- rate-limit traffic-class matrix with evidence sources
- RateLimitService with preflight validation and entrypoint PUT flow
- rate-limit trip prober with tripped/not-counted/inconclusive verdicts
- bounded trip prober with classify verdict engine (commit:fda50bd6)
- decode, knowledge list, ratelimit list/create (commit:23c9e0b6)
- add RateLimitService with preflight + entrypoint PUT flow (commit:3371e96c)
- route-blocking transport + central CF error decoding (commit:b9477f4c)
- ratelimit seed pack from MyCarGuide evidence (commit:de1ca108)
- pack types, embedded loader, route matching (commit:67814ce9)

## Bug Fixes

- config delete now honors --dry-run (previously deleted the profile)
- webhook retries now send a full request body (previously empty on retry)
- honor dry-run in config delete + add config/cache tests (commit:de460543)
- rebuild request per retry attempt (commit:e76f3325)

## Improvements

- consolidate review findings from /simplify pass (commit:c3f466ab)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 114 |
| Files changed | 150 |
| New features | 12 |
| Bug fixes | 4 |

---
_Full changelog: CHANGELOG.md_
