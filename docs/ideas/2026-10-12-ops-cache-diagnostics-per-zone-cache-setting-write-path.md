---
ulid: 01M4PR0F7XVYQ3J6ZFB3AWVJHK
id: IDEA-092
id_assigned_at: 2026-10-14T20:11:32.626364+04:00
title: Ops cache diagnostics per zone + cache-setting write path
created: "2026-10-12T04:25:41.373881+04:00"
status: harvested
source: human
origin:
    session: 76
promoted_to: FEAT-pT9Y6FM
---

# Ops cache diagnostics per zone + cache-setting write path

# Ops cache diagnostics per zone + cache-setting write path

**Operator context (2026-10-12):** asked whether cache-miss alerts are fixable in code or Cloudflare settings, and whether the Ops app can handle it. Real numbers from KV summary (illuminate.am, 24h): 1,944 total requests, 70% dynamic, only 139 cache-eligible, ~62 miss+expired/day — the 50% miss alert rides a tiny base; it is threshold noise, not an origin problem. Concrete shape: (1) per-zone cache breakdown in Ops (hit/miss/dynamic/expired/bypass/none — summary.ts already collects it); (2) feed/push detail carries absolute numbers ("46 misses of 139 eligible"), never a bare percentage; (3) later, write path for cache settings (Smart Tiered Cache toggle, Cache Rules authoring) following the lifecycle CF_API_TOKEN_WRITE + audit pattern — verify exact API scopes before building.



**What it means:** This idea combines two pieces of cache tooling — (1) operational diagnostics that expose per-zone cache state (hit/miss rates, cached objects, effective TTLs, purge history) so an on-call engineer can answer "why is this zone serving stale content?" in seconds rather than guessing, and (2) a dedicated write path for cache settings, meaning configuration changes flow through a single validated, auditable code path instead of ad-hoc updates.

**Why it's useful:** Most cache incidents come from one of two failure modes — unclear current state ("what's actually cached?") or untracked configuration drift ("who changed this TTL, and when?"). Diagnostics address the first; a hardened write path (with validation, versioning, and audit logging) addresses the second, so you can trace any anomalous cache behavior back to either runtime state or a specific settings change.
