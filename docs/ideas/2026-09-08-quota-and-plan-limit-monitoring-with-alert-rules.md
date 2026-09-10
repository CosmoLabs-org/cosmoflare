---
ulid: 01M207F075KTAPE32TAPCEZ725
id: IDEA-052
id_assigned_at: 2026-09-10T16:45:25.445483+04:00
title: Quota and plan-limit monitoring with alert rules
created: "2026-09-08T14:01:50.565881+04:00"
status: withered
source: human
origin:
    session: 43
tags:
    - observability
    - alerts
resolution:
    reason: implemented
    date: "2026-09-10T02:07:09.559365+04:00"
---

# Quota and plan-limit monitoring with alert rules

# Quota and plan-limit monitoring with alert rules

Surface 'how close am I to my plan limits' — requests, bandwidth, R2 storage/Class-A/Class-B operations, Workers invocation limits, KV/DO caps where the API exposes them — as a first-class 'cosmoflare limits' view plus alert rules (feeds TriggerAlert via newServeAlertBridge: warn at 80%, page at 95%). Part of the local control plane direction (ROAD-090) but standalone: needs a quota/usage producer independent of full GraphQL analytics.
