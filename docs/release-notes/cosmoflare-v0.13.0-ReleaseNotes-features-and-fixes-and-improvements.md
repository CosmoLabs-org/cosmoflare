---
project: CosmoDev-R2Go2
version: 0.13.0
date: 2026-06-06
previous: 0.12.0
slug: features-and-fixes-and-improvements
title: "features-and-fixes-and-improvements Release"
---

# CosmoDev-R2Go2 v0.13.0 Release Notes

**Release Date**: June 6, 2026

**Previous**: v0.12.0

## Overview

This release brings 24 new features, 1 bug fix, and 3 improvements.

## Highlights

12 new commands: MCP server, Workers AI, Stream video, multipart uploads, plugin system, export/import, wrangler compat, audit log, multi-account, config validation, Terraform export, alert rules. GitHub repo renamed to CosmoLabs-org/cosmoflare with full module path migration.

## What's New

- cosmoflare alerts — alert rules for error rates, storage limits, failures
- cosmoflare terraform — generate .tf files from live Cloudflare state
- cosmoflare validate — config validation against Cloudflare API constraints
- cosmoflare account — multi-account switching
- cosmoflare audit — CLI mutation audit logging with search and export
- cosmoflare wrangler — import wrangler.toml compatibility layer
- cosmoflare mcp — MCP tool server for AI agent integration
- cosmoflare plugin — community extension system with install/remove/run
- cosmoflare stream — video upload, live inputs, signed tokens
- cosmoflare ai — Workers AI inference and AI Gateway management
- cosmoflare export/import — full account config backup and restore
- Resumable multipart uploads with state tracking and progress
- add alert rules service and CLI commands (ROAD-062) (commit:9023cf22)
- add terraform export and import-block commands (ROAD-071) (commit:948e46e7)
- add config validation command (ROAD-070) (commit:59f8e486)
- add multi-account switching for cosmoflare (ROAD-069) (commit:86b2ec15)
- add CLI mutation audit logging (ROAD-068) (commit:4cb9c8e1)
- add wrangler.toml compatibility layer (ROAD-067) (commit:154d7a5c)
- add MCP server for AI agent integration (ROAD-066) (commit:96b7f090)
- add community plugin system for cosmoflare extensions (ROAD-065) (commit:dcccbfcf)
- add resumable multipart uploads with state tracking (ROAD-013) (commit:d1b4c52d)
- implement Cloudflare Stream video service (ROAD-045) (commit:11a1f6df)
- add Workers AI and AI Gateway service (ROAD-049) (commit:aeb056a4)
- add import/export commands for full account config backup (ROAD-053) (commit:a4372749)

## Bug Fixes

- update r2go2→cosmoflare references in webhook and palette tests (commit:d74d24a4)

## Improvements

- GitHub repo renamed from CosmoLabs-org/r2go2 to CosmoLabs-org/cosmoflare
- Go module path updated to github.com/CosmoLabs-org/cosmoflare
- complete GitHub repo rename to CosmoLabs-org/cosmoflare (P-02) (commit:e216d623)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 21 |
| Files changed | 257 |
| New features | 24 |
| Bug fixes | 1 |

---
_Full changelog: CHANGELOG.md_
