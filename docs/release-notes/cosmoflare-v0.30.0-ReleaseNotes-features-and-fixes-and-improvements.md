---
project: cosmoflare
version: 0.30.0
date: 2026-09-22
previous: 0.29.0
slug: features-and-fixes-and-improvements
title: "features-and-fixes-and-improvements Release"
---

# cosmoflare v0.30.0 Release Notes

**Release Date**: September 22, 2026

**Previous**: v0.29.0

## Overview

This release brings 11 new features, 16 bug fixes, and 4 improvements.

## Highlights

Three command-registry waves land the FEAT-020 metadata spine across the full CLI surface (r2, workers, kv, d1, dns, email, waf, ssl, cache, hyperdrive, alerts, tunnel, account), with destructive deletes now defaulting to dry-run unless --force. New platform depth: Cloudflare Tunnels management (FEAT-035), account members/roles and audit-log query with NDJSON/CSV export (FEAT-036), and full registrar operations — register, transfer, renew, lock/unlock, contacts, DNSSEC (FEAT-030). Fixes: alert rules can now be created and updated into a disabled state (TASK-013); HTTP-status classification unified into one seam so envelope auth failures map correctly instead of 502 (TASK-012). Coverage batches lifted cmd-package coverage from 60.2% to 67.5% with two real bugs found and fixed along the way.

## What's New

- named environment profiles part 2: resource-prefix scoping across d1/kv/r2/workers, d1 execute --local/--remote, dev --env reconciliation (FEAT-026)
- env profiles part 3: prefix scoping on bucket sub-resources and worker subcommands (FEAT-026)
- serve alert cycle: limits-snapshot 30m TTL cache + DNS usage 401/403 fail-fast (FEAT-014)
- cmd coverage wave: 8 test suites across bucket/worker/kv/wrangler/copy/durable-objects/terraform/alerts + CopyBuffer zero-chunk crash fix
- command registry spine wave 1: internal/cmdmanifest with r2 pilot + danger-stamped audit mutations (FEAT-020)
- first-run onboarding: bare-cosmoflare setup nudge + wizard points to status/doctor (FEAT-017)
- FEAT-020 wave 2: command registry covers worker/kv/d1/dns (58 entries, Qwen-verified permissions); destructive deletes run dry by default unless --force
- Tunnels management: tunnel CRUD, connector token, connections, cleanup (FEAT-035)
- Account management: members, roles, and audit-log query with NDJSON/CSV export (FEAT-036)
- Registrar operations: register, transfer, renew, auto-renew, lock/unlock, contacts, DNSSEC (FEAT-030)
- fEAT-036 — Account management + audit logs (commit:dda27057)

## Bug Fixes

- alert condition descriptor registry — one source for validation, errors, help (FEAT-015)
- Alerts: rules can be created and updated into a disabled state (TASK-013)
- Error classification: envelope auth failures map correctly instead of 502; one shared HTTP-status seam (TASK-012)
- carry API messages verbatim; array-shaped transfer stub (commit:2027daed)
- read cloudflare-go status structurally in ErrorStatus (commit:a4e2e48d)
- drop unused os import in audit command (commit:7069ed7d)
- drop duplicate test, bound prompt loop, dry-run before service (commit:fe7d3900)
- match indented JSON in probe report assertion (commit:10b81db9)
- make Execute help test hermetic — rebind rootCmd writers (commit:778ca50c)
- disabled-rule fixture writes YAML directly — Create force-enables (commit:f5f3fa9e)
- dedupe containsString, scope delete force flag to credential guards (commit:9a8b2f57)
- zero ChunkSize crashed CopyBuffer — plus cmd coverage tests (commit:70504c7f)
- repair salvaged interactive rewrite — stray paren, unused servers (commit:a7b7145d)
- replace salvaged self-recursive runFieldChecks with subtest loop (commit:134af9fc)
- drop dead prefix discard in runBucketUpdate (commit:72f25be8)
- complete salvaged wave — execLocalStmt compile fix + part-2 tests (commit:b866e2b7)

## Improvements

- treat reader errors as terminal in all prompt steps (TASK-016) (commit:04555289)
- shared runGlobalsSnapshot helper (TASK-014) (commit:70578840)
- simplify pass on 8e77ac2..HEAD — 9 cleanups from 4 review agents (commit:c407fea9)
- /simplify pass — dead cache field, O(1) condition lookup, unit-from-registry, fixture dedup (commit:defea3e4)

## Breaking Changes

> _None in this release_

## Upgrade Instructions

No breaking changes in this release. Standard upgrade applies.

## Stats

| Metric | Value |
|--------|-------|
| Commits | 113 |
| Files changed | 257 |
| New features | 11 |
| Bug fixes | 16 |

---
_Full changelog: CHANGELOG.md_
