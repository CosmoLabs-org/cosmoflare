---
id: IDEA-037
title: Desktop dashboard is not live — metrics SSE channel has no daemon producer
created: "2026-06-21T03:02:12.326639-03:00"
status: harvested
source: agent
origin:
    session: 18
    trigger: worktree-end reflection
    file: internal/server/sse.go
tags:
    - desktop
    - v1.1
    - notifications
---


# Desktop dashboard is not live — metrics SSE channel has no daemon producer

The frontend useDaemonSSE hook subscribes to a 'metrics' channel, but the Go daemon never Publish('metrics', ...). Dashboard cards hydrate from REST and only refetch on profile switch (React Query key change), so counts go stale until the user switches accounts. v1.1: either add a daemon metrics poll loop that Publish('metrics', deltas) every Ns, or set a refetchInterval on the dashboard's useQuery calls.
