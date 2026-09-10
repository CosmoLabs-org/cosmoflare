---
project: cosmoflare
version: 0.12.0
date: 2026-05-30
previous: 0.11.0
slug: features-and-fixes-and-improvements
title: "features-and-fixes-and-improvements Release"
---

# CosmoDev-R2Go2 v0.12.0 Release Notes

**Release Date**: May 30, 2026

**Previous**: v0.11.0

## Overview

This release brings 25 new features, 1 bug fix, and 2 improvements.

## Highlights

14 new Cloudflare service commands including dev, images, hyperdrive, vectorize, sync, and apply. Renamed binary and package from r2go2 to cosmoflare. Added GitHub Actions CI/CD for cross-platform builds. 170+ new tests bringing total to 2,794.

## What's New

- cosmoflare dev — local development proxy server with hot-reload
- cosmoflare logs --follow — real-time log tailing with level filtering
- cosmoflare init — interactive project scaffolding with framework detection
- cosmoflare images — Cloudflare Images upload, variants, delivery
- cosmoflare hyperdrive — database connection pooling management
- cosmoflare diff — compare local config against live Cloudflare state
- cosmoflare cost — monthly cost estimation across R2, Workers, KV
- cosmoflare watch — auto-sync local directory to R2 on file changes
- cosmoflare vectorize — vector database for AI workloads
- cosmoflare sync — rsync-like directory synchronization with R2
- cosmoflare templates — project scaffolding with 5 built-in templates
- cosmoflare apply — declarative config reconciliation against live state
- GitHub Actions CI/CD workflows for cross-platform builds and releases
- implement cosmoflare apply command (ROAD-056) (commit:fdfa0635)
- add cosmoflare templates command (ROAD-059) (commit:27b5d249)
- implement cosmoflare sync command for local-R2 directory synchronization (commit:2be0f07d)
- implement Cloudflare Vectorize service (ROAD-048) (commit:0b3bd5dc)
- add cosmoflare watch command for auto-syncing local directories to R2 (commit:8479f9d9)
- add cosmoflare cost command for monthly cost estimation (ROAD-061) (commit:9b23e914)
- add cosmoflare diff command (ROAD-052) (commit:299002a1)
- implement Cloudflare Hyperdrive service (ROAD-046) (commit:27667177)
- implement Cloudflare Images service (ROAD-044) (commit:b3e6e2fc)
- add cosmoflare init project scaffolding (ROAD-057) (commit:9a4f2bca)
- add logs --follow for real-time log tailing (ROAD-051) (commit:589e7bd1)
- add cosmoflare dev local proxy server (ROAD-058) (commit:5cdcbcce)

## Bug Fixes

- use DefValue instead of GetString in waf flag default test (commit:cc02a21c)

## Improvements

- Renamed binary from r2go2 to cosmoflare, pkg/r2go2 to pkg/cosmoflare
- rename to Cosmoflare — binary, package, and branding (commit:cc30242f)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 56 |
| Files changed | 401 |
| New features | 25 |
| Bug fixes | 1 |

---
_Full changelog: CHANGELOG.md_
