---
ulid: 01M26JE08JAF29DTECA5DZPBME
title: 'R2: bucket policy get/set + expose library multipart as CLI verbs'
created: "2026-09-11T01:08:58.770118+04:00"
status: seed
source: agent
origin:
    session: 51
    trigger: Qwen corpus section 9 coverage diff
---

# R2: bucket policy get/set + expose library multipart as CLI verbs

cmd/ has no multipart verb though pkg/cosmoflare/multipart.go exists (create/upload-part/complete/abort/list-parts). Also r2 bucket policy get/set missing. Corpus section 9 r2 block.
