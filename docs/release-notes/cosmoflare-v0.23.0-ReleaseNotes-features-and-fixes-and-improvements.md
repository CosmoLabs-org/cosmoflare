---
project: cosmoflare
version: 0.23.0
date: 2026-09-09
previous: 0.22.0
slug: features-and-fixes-and-improvements
title: "features-and-fixes-and-improvements Release"
---

# cosmoflare v0.23.0 Release Notes

**Release Date**: September 9, 2026

**Previous**: v0.22.0

## Overview

This release brings 9 new features, 3 bug fixes, and 3 improvements.

## Highlights

cosmoflare limits — plan-limit proximity view (Workers/R2/Zones/DNS quota) with live DNS quota API, partial-failure snapshots, workers_plan config, alert-evaluator feed (3 new conditions); Domain Center residuals brainplan; serve wiring + condition registry + renderer fixes

## What's New

- cosmoflare limits — plan-limit proximity view with alert feed
- cosmoflare limits — plan-limit proximity table and JSON (commit:c4f41c8a)
- limit-proximity metrics feed the evaluator (commit:cef13ad2)
- snapshot assembly with partial-failure semantics (commit:8b619673)
- live DNS usage endpoint with static fallback, zone plan legacy_id (commit:58f34664)
- workers plan resolution with subscriptions API and fallback chain (commit:7159245a)
- workers_plan project config field for limit tier fallback (commit:1bf62535)
- snapshot types, consumer interfaces, service constructor (commit:5e967efb)
- static limit tables and limitFor join function (commit:773de39d)

## Bug Fixes

- limits serve wiring, alert condition registry, and renderer labels fixed
- pin DNS usage endpoint path and fields to the live API (commit:d1f5026f)
- correct resolution-order doc comment — flag outranks config (commit:4fc6dff1)

## Improvements

- internal: the three REST services share one transport client (~200 duplicated lines consolidated), metrics --json collects analytics concurrently, zone auto-resolution moved into the library
- adopt shared restClient, fix inert serve wiring and condition registry (commit:ede3069c)
- shared restClient transport, ux.Confirm, library zone resolution (commit:eca12aa0)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 72 |
| Files changed | 59 |
| New features | 9 |
| Bug fixes | 3 |

---
_Full changelog: CHANGELOG.md_
