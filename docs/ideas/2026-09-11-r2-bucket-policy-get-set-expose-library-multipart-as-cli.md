---
ulid: 01M26JE08JAF29DTECA5DZPBME
title: 'R2: bucket policy get/set + expose library multipart as CLI verbs'
created: "2026-09-11T01:08:58.770118+04:00"
status: withered
source: agent
origin:
    session: 51
    trigger: Qwen corpus section 9 coverage diff
resolution:
    reason: implemented
    date: "2026-09-12T03:01:50.21755+04:00"
    ref: FEAT-028
    note: Policy get/set shipped as FEAT-028 (merge 7f48107). Manual multipart verbs deliberately descoped — re-seeded separately.
---

# R2: bucket policy get/set + expose library multipart as CLI verbs

# R2: bucket policy get/set + expose library multipart as CLI verbs

cmd/ has no multipart verb though pkg/cosmoflare/multipart.go exists (create/upload-part/complete/abort/list-parts). Also r2 bucket policy get/set missing. Corpus section 9 r2 block.
