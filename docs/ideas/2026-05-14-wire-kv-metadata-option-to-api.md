---
id: IDEA-018
title: Wire KV metadata option to API
created: "2026-05-14T05:28:11.092326-03:00"
status: seed
source: human
origin:
    session: 2027
---

# Wire KV metadata option to API

kvConfig.metadata is accepted via WithKVMetadata but never passed to the cloudflare-go WriteWorkersKVEntry API call. Either wire it or remove the option to avoid misleading users.
