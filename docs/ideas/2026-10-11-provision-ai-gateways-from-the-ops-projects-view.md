---
ulid: 01M4M4P1SG7CYQTA3FK5AAQTRX
title: Provision AI Gateways from the Ops Projects view
created: "2026-10-11T04:09:28.112089+04:00"
status: seed
source: human
origin:
    session: 75
---

# Provision AI Gateways from the Ops Projects view

From a project's profile in Ops, create/attach a Cloudflare AI Gateway (first-class API) and link it to that project's workers, so per-project model usage, caching and rate limits (already collected by the AI Gateway collector, BR-02) can be governed per project. Same write-path prerequisites as the emergency-actions idea: write-scoped token, confirmation, audit log.
