---
project: CosmoDev-R2Go2
version: 0.11.0
date: 2026-05-27
previous: 0.10.0
slug: features-and-fixes
title: "features-and-fixes Release"
---

# CosmoDev-R2Go2 v0.11.0 Release Notes

**Release Date**: May 27, 2026

**Previous**: v0.10.0

## Overview

This release brings 5 new features, and 8 bug fixes.

## Highlights

Full Cloudflare Pages and Queues service implementations with CLI commands and tests. Six audit Phase 0 bug fixes including multipart sort, NoSuchKey classification, atomic config write, and install.sh case.

## What's New

- # FEAT-002: Local HTTP dev server proxying to R2

**Type**: feature
**Status**: closed
**Created**: 2026-02-26

## Description

r2go2 serve command starts a local HTTP server that proxies requests to an R2 bucket for local testing without deploying
- Cloudflare Pages service — project and deployment management (ROAD-043)
- Cloudflare Queues service — queue and consumer management (ROAD-033)
- implement Cloudflare Queues service and CLI commands (commit:b6b62fbe)
- implement Cloudflare Pages service and CLI commands (commit:a4e9309f)

## Bug Fixes

- Sort multipart upload parts before assembly (data corruption prevention)
- Classify S3 NoSuchKey as ErrNotFound sentinel error
- Atomic config file write prevents TOCTOU race
- Fix install.sh binary name case for Linux
- Fix release.yml asset path containing slashes
- Remove bucket update no-op — return clear unsupported error
- sort multipart parts and classify NoSuchKey as ErrNotFound (commit:43198d07)
- audit phase 0 — install.sh case, release paths, atomic config write, bucket update no-op (commit:258e9d5a)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 35 |
| Files changed | 129 |
| New features | 5 |
| Bug fixes | 8 |

---
_Full changelog: CHANGELOG.md_
