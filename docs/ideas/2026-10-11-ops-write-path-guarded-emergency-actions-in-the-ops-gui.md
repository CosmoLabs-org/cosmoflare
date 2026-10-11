---
ulid: 01M4M4NSW2MV1QMYQQ6JCRWMKE
title: 'Ops write-path: guarded emergency actions in the Ops GUI'
created: "2026-10-11T04:09:20.002271+04:00"
status: seed
source: human
origin:
    session: 75
---

# Ops write-path: guarded emergency actions in the Ops GUI

Kill-switch operations from the Ops GUI, audit-logged: stop traffic to a worker (remove routes + disable cron triggers), pause queue consumers. Requires a second write-scoped CF token held by the ops worker, an explicit confirmation step per action, and every action logged to the cosmoflare audit trail. Cloudflare has no literal pause for Workers/D1, but route/cron removal stops compute exposure effectively.
