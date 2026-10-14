---
ulid: 01M4KFCD5TRA6F9Z541W88YTCH
id: IDEA-085
id_assigned_at: 2026-10-14T20:11:32.626364+04:00
title: Profile pages for every unique entity + future control actions
created: "2026-10-10T21:57:11.99457+04:00"
status: harvested
source: human
origin:
    session: 74
    trigger: operator 2026-10-10 session 2036
promoted_to: ROAD-p7QJ0RX
---

# Profile pages for every unique entity + future control actions

# Profile pages for every unique entity + future control actions

Every unique entry in Ops gets its own profile page with properly organized information: workers (shipped), domains (shipped), D1 databases (needed — rows read/written/read-queries/rows-per-query from the summary row, plus schema/size when D1 storage datasets are wired), zones, KV namespaces, DO namespaces. Future direction (explicitly deferred): control actions on profiles — stop/pause buttons, drain, scale-to-zero — turning Ops from read-only monitoring into an operations console. Explore per-service control surfaces and their API prerequisites (most need additional token scopes; some actions are destructive and need confirmation UX).
