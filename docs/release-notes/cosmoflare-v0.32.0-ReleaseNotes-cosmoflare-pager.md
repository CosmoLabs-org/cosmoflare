---
project: cosmoflare
type: release
version: 0.32.0
source: .version-registry.json
date: 2026-10-07T19:18:19.3353+04:00
slug: cosmoflare-pager
title: "cosmoflare-pager Release"
---

# cosmoflare v0.32.0 Release Notes

**Release Date**: October 7, 2026

## Overview

This release brings 4 new features.

## Highlights

The Cosmoflare Pager ships: zero-infrastructure Web Push alerting from the Go core to an installable PWA, with ack/snooze parity on the desktop dashboard (ADR-001 mobile strategy — one Go core, PWA-first, gomobile for future native). The permission catalog doubles to 157 families from the verified Qwen dataset, the command registry reaches enforced completeness, and the knowledge endpoint registry now guards every Cloudflare API call.

## What's New

- FEAT-011: ingest Qwen permission dataset — catalog grows 76 to 157 token-permission families with endpoint unlocks, spectrum/web-analytics permission sparseness filled in the command registry
- FEAT-020: command registry complete — permission columns filled for all API commands, NoPermsRequired marker for verified-none endpoints, invariant test enforcing completeness
- FEAT-044: knowledge.Transport wired at the control-plane chokepoint — every CF API call now gets the endpoint registry; WithHTTPClient overrides wrapped (wrap-always); R2 data plane stays registry-free
- FEAT-045: Cosmoflare Pager v1 — Web Push (VAPID) zero-infra alert delivery: alertspush package (keypair/subscription store, 2KB severity payload, dispatch with expiry pruning), alerts watch + push CLI, installable static PWA with ack/snooze history, desktop notifications parity, ADR-001 mobile strategy

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 56 |
| Files changed | 185 |
| New features | 4 |

---
_Full changelog: CHANGELOG.md_
