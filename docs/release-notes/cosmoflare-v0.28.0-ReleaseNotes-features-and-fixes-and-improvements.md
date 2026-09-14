---
project: cosmoflare
version: 0.28.0
date: 2026-09-14
previous: 0.27.0
slug: features-and-fixes-and-improvements
title: "features-and-fixes-and-improvements Release"
---

# cosmoflare v0.28.0 Release Notes

**Release Date**: September 14, 2026

**Previous**: v0.27.0

## Overview

This release brings 2 new features, 2 bug fixes, and 15 improvements.

## Highlights

Presenter sweep (FEAT-040): ~390 if-JSONOutput branches collapsed across 55 cmd files; parallel sync --concurrency (FEAT-038); unified transport policy (FEAT-039); JSON errors exit 1; pkg.go.dev landing text fixed

## What's New

- output presenter — pattern + first conversion (FEAT-040 slice) (commit:a6c7df10)
- parallel executor + unified transport policy (FEAT-038/039) (commit:6da6553b)

## Bug Fixes

- JSON-mode errors now exit non-zero: --json failures print the parseable error envelope to stdout and return exit code 1 (previously exit 0 — breaking the deterministic exit-code contract for agent consumers). Diagnostics mirror to stderr.
- JSON-mode errors exit non-zero — envelope on stdout, exit 1 (commit:738623e7)

## Improvements

- species-2 presenter conversion for compare.go (FEAT-040) (commit:69c67400)
- presenter conversion for knowledge_cmd (FEAT-040) (commit:cb7551fe)
- hand-convert 6 irregular species-2 stragglers (FEAT-040) (commit:89834823)
- presenter conversion for auth_permissions.go (FEAT-040) (commit:bd1b024c)
- presenter conversion for domains_get.go (FEAT-040) (commit:83f0926d)
- species-2 presenter conversion for bucket_lifecycle.go (FEAT-040) (commit:7bbd0e76)
- presenter conversion for bucket_policy.go (commit:17d8bbc6)
- presenter conversion for bucket_notifications (FEAT-040) (commit:b449051e)
- presenter conversion for d1_migrations (FEAT-040) (commit:6c37f981)
- species-2 presenter conversion for pages_deployment.go (FEAT-040) (commit:032dcb70)
- presenter conversion queue_send.go (FEAT-040) (commit:e91e48f2)
- presenter conversion for terraform.go (FEAT-040) (commit:0316d6a9)
- presenter conversion bucket_domain.go (FEAT-040) (commit:3c20cd10)
- finish species-2 conversion for email.go (FEAT-040) (commit:62506563)
- presenter species-1 sweep — 196 error branches collapsed (commit:e06d5130)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 144 |
| Files changed | 143 |
| New features | 2 |
| Bug fixes | 2 |

---
_Full changelog: CHANGELOG.md_
