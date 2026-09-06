---
branch: master
created: "2026-05-16T00:00:00-03:00"
deliverables:
    - P-01: Module path rename from CosmoDev-R2Go2 to cosmoflare
    - P-02: GitHub repo redirect setup
    - P-03: Downstream migration guide
goals_completed: 0
goals_total: 3
last_review_content_hash: 86d1a3267a081fa9b7779ecf13b1b704cf44936a1833533c5543a19e25595e8b
last_review_findings: 0
last_review_ref: docs/planning-mode/2026-05-16-cosmoflare-module-rename.md
last_reviewed: "2026-09-06T20:14:03.8533+04:00"
origin: manual
priority: medium
schema_version: 1
status: COMPLETED
tags:
    - rebrand
    - module-path
    - breaking-change
title: Cosmoflare Module Path Rename
---

# Cosmoflare Module Path Rename

**Date**: 2026-05-16
**Status**: PLANNED (not yet scheduled)
**Priority**: Medium — deferred until downstream consumers exist

## Current State

- Module path: `github.com/CosmoLabs-org/CosmoDev-R2Go2`
- Binary name: `r2go2` (primary), `cosmoflare` (symlink alias)
- Go proxy: Published as CosmoDev-R2Go2

## Target State

- Module path: `github.com/CosmoLabs-org/cosmoflare`
- Binary name: `cosmoflare` (primary), `r2go2` (backward-compatible alias)
- GitHub: Repo renamed, old URL redirects

## Why Defer

1. No known downstream `go get` consumers yet — the rename is breaking
2. Go proxy caches the old path indefinitely — renaming creates a new module identity
3. The `r2go2` binary still works regardless of module path
4. Better to time this with v1.0.0 when the API surface stabilizes
5. Phase 4-6 services still in progress — premature to freeze the API

## Plan

### P-01: Module Path Rename
1. Create or rename repo to `github.com/CosmoLabs-org/cosmoflare`
2. Update `go.mod`: `module github.com/CosmoLabs-org/cosmoflare`
3. Update all internal imports across cmd/, pkg/, internal/, tests/
4. Add `retract` directive to old module for the final version
5. Tag new version under new module path

### P-02: GitHub Repo Redirect
1. Rename repo (GitHub auto-redirects old URLs)
2. Update CI badges, install instructions, documentation
3. Update go.sum references in CosmoLabs projects

### P-03: Downstream Migration Guide
1. Write migration guide: old import path to new
2. Keep old module published with retract notice for 6 months

## Prerequisites

- [ ] Stable API surface (no breaking changes to `pkg/r2go2/`)
- [ ] At least one external consumer
- [ ] v1.0.0 release candidate
- [ ] Phase 4-6 services implemented
