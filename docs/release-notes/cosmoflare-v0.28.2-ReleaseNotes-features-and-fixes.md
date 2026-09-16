---
project: cosmoflare
version: 0.28.2
date: 2026-09-16
slug: features-and-fixes
title: "features-and-fixes Release"
---

# cosmoflare v0.28.2 Release Notes

**Release Date**: September 16, 2026

## Overview

This release brings 6 new features, and 3 bug fixes.

## Highlights

Patch release restoring Go module distribution: an emoji-named directory committed into v0.28.0/v0.28.1 made proxy.golang.org refuse both versions' zips, pinning `go install` to v0.27.0 — the path is gone and the release gate now rejects non-ASCII tree paths (BUG-052). FEAT-021 adds 29 Workers commands (deployments with rollback, custom domains, workers.dev subdomain, secrets, versions, routes, bindings, live tail) plus a worker-types `.d.ts` generator.

## What's New

- wire wave 2-3 groups + USAGE.md Workers section (FEAT-021) (a6e54045)
- worker types .d.ts generator (FEAT-021) (b32e4801)
- bindings list + live tail commands (FEAT-021) (9ea29030)
- workers.dev subdomain get/set (FEAT-021) (be2101b4)
- custom domain attach/detach/list (FEAT-021) (1dcbe2dd)
- deployments list/view/rollback + wave-1 wiring (FEAT-021) (326511a6)

## Bug Fixes

- go install via proxy.golang.org failed for v0.28.0/v0.28.1 (emoji-named path in the tagged tree); directory removed, release gate now rejects non-ASCII paths (BUG-052) (b2d4eaf5)
- zero AccountID/APIToken in cron globals helper — full-suite order dependence (3282e614)
- route list flag check uses LocalFlags — cobra merges inherited persistent flags on root Execute, making Flags() order-dependent (6613958f)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 39 |
| Files changed | 58 |
| New features | 6 |
| Bug fixes | 3 |

---
_Full changelog: CHANGELOG.md_
