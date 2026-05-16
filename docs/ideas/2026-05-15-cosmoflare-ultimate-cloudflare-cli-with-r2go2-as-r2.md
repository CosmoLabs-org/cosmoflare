---
id: IDEA-020
title: Cosmoflare — ultimate Cloudflare CLI with R2Go2 as R2 component
created: "2026-05-15T23:47:45.081526-03:00"
status: seed
source: human
origin:
    session: 2027
---

# Cosmoflare — ultimate Cloudflare CLI with R2Go2 as R2 component

## Vision

R2Go2 is evolving into **Cosmoflare** — the ultimate open-source CLI for managing the entire Cloudflare developer platform. R2Go2 becomes the R2 storage component within a larger umbrella.

## Platform Coverage

- **R2** (storage) — R2Go2, the core component (implemented)
- **Workers** (compute) — deploy, list, logs, bindings (implemented)
- **KV** (key-value) — namespaces, get/put/delete (implemented)
- **D1** (SQL database) — create, query, migrate (stub)
- **Pages** (static hosting) — deploy, list, custom domains (stub)
- **Queues** (message queues) — create, send, consume (stub)
- **Domains** — DNS management, domain registration (planned)
- **DNS** — record management, zone configuration (planned)

## Key Points

- Single binary, single config (`.r2go2.yaml` → eventually `.cosmoflare.yaml`)
- Agent-friendly CLI (AI tools use it from terminal)
- Go library importable by any project
- CCS integration (`ccs r2` → `ccs cosmoflare`)

## Next Steps

1. Audit current command structure for rebrand feasibility
2. Plan migration path: binary alias (`cosmoflare` → `r2go2`), config migration
3. Implement remaining platform stubs (D1, Pages, Queues, DNS, Domains)
