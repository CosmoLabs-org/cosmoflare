---
project: cosmoflare
version: 0.26.0
date: 2026-09-13
previous: 0.25.0
slug: features-and-improvements
title: "features-and-improvements Release"
---

# cosmoflare v0.26.0 Release Notes

**Release Date**: September 13, 2026

**Previous**: v0.25.0

## Overview

This release brings 34 new features, and 1 improvement.

## Highlights

v0.26.0 turns a week of Cloudflare research into product surface. The D1 depth pack lands migrations with in-database state tracking, time-travel restore with quota pre-flight, streaming SQL export, and batched import with statement-aware splitting plus TOOBIG split-retry. Queues gain the full producer surface (send, send-batch, DLQ configuration, consumer management) and Pages gains env vars, secrets, domains, and deployment operations. Durable Objects live-state inspection ships as the first CLI coverage anywhere. Account-wide awareness arrives with a 76-family token permission manifest (auth permissions list), a rate-limit-aware REST client honoring Retry-After, and the domain fleet status matrix (doctor --all) showing every zone's SSL, DNSSEC, and protection posture in one scriptable view. Rounding out: vectorize vector CRUD, R2 bucket policy get/set, and a consolidated structured not-found detector.

## What's New

- D1 depth: migrations, time-travel restore, backups, export/import (FEAT-022) (FEAT-022)
- Queues producer surface: send, send-batch, DLQ, consumer management (FEAT-024) (FEAT-024)
- Rate-limit-aware shared client: Retry-After, backoff, retry flags (FEAT-025) (FEAT-025)
- Vectorize vector CRUD: upsert, get, delete, list vectors, namespaces (FEAT-027) (FEAT-027)
- R2 bucket policy get/set (FEAT-028) (FEAT-028)
- Domain fleet status matrix — all domains, all protection statuses, one view (FEAT-033) (FEAT-033)
- pages deployment view/retry/logs command group (commit:00dc4b04)
- pages domain command group (commit:a346ff64)
- pages env command group (commit:96119d28)
- Pages deployment retry and logs library (commit:95897f8b)
- Pages custom domains library (commit:b9da2795)
- Pages env vars and secrets library (commit:9f3fdd0e)
- do namespaces/objects/inspect commands (commit:1e9c078a)
- durable objects live-state service (commit:c702026f)
- file four coverage-domain features from pillar-A research (commit:d1858656)
- domain fleet status service — per-zone protection matrix (commit:0cfbedf1)
- promote FEAT-031/032; file domain fleet status matrix (commit:9fd247ec)
- d1 import — batched, resumable, TOOBIG split-retry (commit:994c19f0)
- statement-aware SQL splitter for D1 import (commit:89c1cdc6)
- d1 export command (commit:a11dbc66)
- d1 time-travel and export commands (commit:78c6a59c)
- d1 time-travel restore with quota pre-flight (commit:52552acc)
- rate-limit-aware REST client retries (commit:74718b24)
- d1 migrations commands (commit:4992a272)
- d1 migrations — create, list, apply (commit:836ee6a0)
- bucket policy commands (commit:be1360c8)
- vectorize vector commands (commit:870cf701)
- vectorize vector CRUD methods (commit:8578b871)
- queue consumer update/remove and dlq commands (commit:26f2743e)
- queue consumer update + DLQ configuration (commit:a131b286)
- auth permissions list command (commit:edf14c44)
- embedded Cloudflare API token permission manifest (commit:741e6bea)
- queue send and send-batch commands (commit:52333397)
- queue producer — Send and SendBatch service methods (commit:fee8bd4e)

## Improvements

- consolidate not-found detectors into isNotFound (commit:faebd780)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 131 |
| Files changed | 126 |
| New features | 34 |

---
_Full changelog: CHANGELOG.md_
