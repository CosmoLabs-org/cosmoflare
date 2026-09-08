---
ulid: 01M207FKQ0G5X7FWMP023KCR92
title: R2 event notifications — bucket webhooks
created: "2026-09-08T14:02:10.528613+04:00"
status: withered
source: human
origin:
    session: 43
tags:
    - r2
    - events
resolution:
    reason: implemented
    date: "2026-09-08T16:31:29.762155+04:00"
    note: 'Landed 2026-09-08 merge fc1bc81: BucketNotificationService + bucket notifications list/create/get/delete'
---

# R2 event notifications — bucket webhooks

# R2 event notifications — bucket webhooks

R2 can emit event notifications (object created/deleted) to queues/webhooks/Hyperloop targets; cosmoflare has no coverage. Add service + 'cosmoflare bucket notifications' commands (create/list/delete rules, target configuration). High value for the agent-first positioning: agents reacting to storage events without polling. The dead cmd_disabled/webhook.go draft targeted the pre-daemon design — build fresh against the current R2 notifications API.
