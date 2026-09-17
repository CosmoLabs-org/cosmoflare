---
project: cosmoflare
version: 0.29.0
date: 2026-09-17
slug: features
title: "features Release"
---

# cosmoflare v0.29.0 Release Notes

**Release Date**: September 17, 2026

## Overview

This release brings 6 new features.

## Highlights

Service-depth wave across four surfaces. SSL for SaaS: custom hostnames CRUD with ownership-verification status verbs (FEAT-031). WAF: lists CRUD, item add/replace, and managed-rulesets update (FEAT-032). Named environment profiles, part 1: the `--env` flag selects a profile with `plan_tier`/`resource_prefix`, and account limits honor the plan tier (FEAT-026). Auth tooling: redact-safe token retrieval (FEAT-029 part 1) and a permission catalog knowledge pack that decodes API-token scopes, including error 10405 scope decode (FEAT-011). The update check is opt-in through `COSMOFLARE_UPDATE_CHECK` and stays off by default (FEAT-043).

## What's New

- redact-safe token retrieval (FEAT-029 part 1) (927da244)
- permission catalog pack + 10405 scope decode (FEAT-011) (c949271b)
- opt-in env-gated update check (COSMOFLARE_UPDATE_CHECK, default off) (FEAT-043)
- named environment profiles part 1: --env flag, profile plan_tier/resource_prefix, limits plan-tier awareness (FEAT-026)
- SSL custom hostnames (SaaS) CRUD + verification status verbs (FEAT-031)
- WAF lists CRUD + items add/replace + managed-rulesets update (FEAT-032)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 35 |
| Files changed | 68 |
| New features | 6 |

---
_Full changelog: CHANGELOG.md_
