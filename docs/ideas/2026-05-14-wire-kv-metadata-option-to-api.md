---
id: IDEA-018
title: Wire KV metadata option to API
created: "2026-05-14T05:28:11.092326-03:00"
status: withered
source: human
origin:
    session: 2027
resolution:
    reason: implemented
    date: "2026-09-08T15:08:11.722579+04:00"
    note: WithKVMetadata exists (kv.go:37)
---

# Wire KV metadata option to API

# Wire KV metadata option to API

kvConfig.metadata is accepted via WithKVMetadata but never passed to the cloudflare-go WriteWorkersKVEntry API call. Either wire it or remove the option to avoid misleading users.
