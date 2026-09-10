---
project: CosmoDev-R2Go2
version: 0.8.0
date: 2026-05-16
previous: 0.7.1
slug: features-and-fixes-and-improvements
title: "features-and-fixes-and-improvements Release"
---

# CosmoDev-R2Go2 v0.8.0 Release Notes

**Release Date**: May 16, 2026

**Previous**: v0.7.1

## Overview

This release brings 11 new features, 1 bug fix, and 3 improvements.

## Highlights

Phase 4+5: 8 new Cloudflare services (DNS, Zones, SSL, Cache, Page Rules, WAF, Email Routing, D1). Cosmoflare rebrand with mission + constitution. 11 service command groups total. UnitTesting merge with coverage improvements across 7 packages.

## What's New

- Add DNS Records service (ROAD-035)
- Add Zone Management service (ROAD-036)
- Add SSL/TLS Management service (ROAD-037)
- Add Cache Management service (ROAD-038)
- Add Page Rules service (ROAD-039)
- Add WAF/Firewall service (ROAD-040)
- Add Email Routing service (ROAD-041)
- Add D1 Database service (ROAD-042)
- Shell completion already implemented (ROAD-055)
- add Page Rules, WAF, Email Routing, D1 services (commit:85e86b00)
- add DNS, Zone, SSL/TLS, and Cache services (commit:ff3e694b)

## Bug Fixes

- resolve race conditions in Execute/waitForCompletion (commit:b3e5a7f7)

## Improvements

- Cosmoflare rebrand — README, PRODUCT-VISION constitution, binary alias
- Test coverage improvements across 7 internal packages (UnitTesting merge)
- replace all fmt.Scanln with InputReader injection (commit:6ff75d7d)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 32 |
| Files changed | 131 |
| New features | 11 |
| Bug fixes | 1 |

---
_Full changelog: CHANGELOG.md_
