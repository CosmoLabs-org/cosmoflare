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

This release brings 11 new features, and 7 bug fixes.

## Highlights

Cosmoflare Desktop v1 + cosmoflare serve daemon: new local HTTP+SSE daemon (token auth; REST read endpoints for accounts/zones/R2/Workers/KV; SSE /events; two-tier systems+cloudflare health) backing a Tauri v2 desktop app with a multi-account read-only dashboard and real-time notifications. Includes desktop reliability fixes: daemon no longer orphaned on app quit, notifications persist across tabs, accounts errors surfaced, SSE-driven health.

## What's New

- # FEAT-007: Cosmoflare Desktop (Tauri) v1 — daemon + dashboard + notifications

**Type**: feature
**Status**: closed
**Created**: 2026-06-20

## Description

Implement v1 of the Cosmoflare desktop app (ROAD-063): a Tauri shell that spawns/supervises a local 'cosmoflare serve' daemon (HTTP+SSE over existing services) and renders a read-only multi-account dashboard with real-time notifications. Cross-platform via Rust-owned daemon lifecycle. Credentials via config flow, never the OS keychain. Design: docs/brainstorming/2026-06-20-cosmoflare-desktop.md. Plan: docs/planning-mode/2026-06-20-cosmoflare-desktop.md. GLM Wave-1 manifest: docs/prompts/2026-06-20-cosmoflare-desktop-glm-tasks.yaml.
- cosmoflare serve: local HTTP+SSE daemon (token auth, REST read endpoints, SSE /events, two-tier health) backing the desktop app
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

- Desktop: fix daemon orphan on app quit, notifications reset on tab switch, accounts 401 swallow, and stale health indicators
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
| Commits | 40 |
| Files changed | 116 |
| New features | 11 |
| Bug fixes | 7 |

---
_Full changelog: CHANGELOG.md_
