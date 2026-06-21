---
project: cosmoflare
version: 0.17.0
date: 2026-06-21
previous: 0.16.0
slug: features-and-fixes
title: "features-and-fixes Release"
---

# cosmoflare v0.17.0 Release Notes

**Release Date**: June 21, 2026

**Previous**: v0.16.0

## Overview

This release brings 9 new features, and 6 bug fixes.

## Highlights

Cosmoflare Desktop v1 + cosmoflare serve daemon: new local HTTP+SSE daemon (token auth; REST read endpoints for accounts/zones/R2/Workers/KV; SSE /events; two-tier systems+cloudflare health) backing a Tauri v2 desktop app with a multi-account read-only dashboard and real-time notifications. Includes desktop reliability fixes: daemon no longer orphaned on app quit, notifications persist across tabs, accounts errors surfaced, SSE-driven health.

## What's New

- emit notifications on cloudflare_online transitions (BR-07) (commit:3cd093a1)
- real-time notifications panel (P-08) (commit:be5ee69f)
- multi-account read-only dashboard (P-07) (commit:0b82d646)
- app shell, API/SSE clients, health indicators (P-06) (commit:4302a56e)
- supervise cosmoflare daemon lifecycle from Rust (P-05) (commit:b4d0af3d)
- scaffold Tauri v2 app with sidecar config (P-04) (commit:e7401cd7)
- add SSE /events (metrics/notifications/status) + health (P-03) (commit:b142676a)
- add REST read endpoints over existing services (P-02) (commit:308788d2)
- add cosmoflare serve daemon skeleton with token auth + /healthz (P-01) (commit:cae32ba5)

## Bug Fixes

- independent review of ROAD-080 brainplan — 2 issues (commit:ccaf5a06)
- persist notifications across tabs, harden accounts + SSE health (commit:21cf37f0)
- kill daemon on app exit + make set_child atomic (commit:e3321ea0)
- config init expects .cosmoflare not legacy .r2go2 (commit:c1ed9307)
- kill sidecar child on failed handshake/health (leak hygiene) (commit:886d6f9f)
- provide QueryClientProvider in App (runtime crash on Dashboard) (commit:24130280)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 41 |
| Files changed | 117 |
| New features | 9 |
| Bug fixes | 6 |

---
_Full changelog: CHANGELOG.md_
