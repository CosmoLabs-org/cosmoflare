---
project: cosmoflare
version: 0.20.0
date: 2026-09-06
previous: 0.19.0
slug: features-and-improvements
title: "features-and-improvements Release"
---

# cosmoflare v0.20.0 Release Notes

**Release Date**: September 6, 2026

**Previous**: v0.19.0

## Overview

This release brings 4 new features, and 3 improvements.

## Highlights

MCP full surface (140+ auto-generated tools, mutation gating) · alert→SSE bridge for the desktop daemon (FEAT-008) · README regenerated from reality · local GoReleaser release pipeline

## What's New

- In-process event bus for real-time notifications (ROAD-080) (FEAT-008) (FEAT-008)
- MCP server now exposes the full CLI surface — 140+ tools auto-generated from the cobra tree, read-only by default, mutations behind an explicit allow switch
- bridge TriggerAlert to serve SSE notifications (FEAT-008) (commit:bc18c4c1)
- generate full-surface tools from the cobra tree with mutation gating (commit:164fa88b)

## Improvements

- alert SSE fan-out fires before webhook retries; channel names exported as constants
- README regenerated from reality; local GoReleaser release config
- simplify alert bridge — honest lifetime, latency-first fan-out, channel constants (commit:df2fc3ce)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 51 |
| Files changed | 629 |
| New features | 4 |

---
_Full changelog: CHANGELOG.md_
