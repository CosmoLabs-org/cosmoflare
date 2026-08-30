---
ulid: 01M1AEZD1QGNRNZD6NNPND1F9J
id: IDEA-039
title: Fix multipart short-read silent corruption in upload engine
created: "2026-08-31T03:09:50.519566+04:00"
status: seed
source: agent
origin:
    session: 36
    trigger: audit-agent-2-core-logic
    file: docs/audit/2026-08-31-cosmoflare/agent-2-core-logic.md
tags:
    - audit
    - core-logic
---

# Fix multipart short-read silent corruption in upload engine

MultipartUpload and ResumableMultipartUpload accept short reads as success: io.ReadFull EOF before uploadedBytes==size continues the loop, uploading empty parts and completing a truncated object with NO error. Fix: abort with validationError on EOF/ErrUnexpectedEOF before size is reached, in both pkg/cosmoflare/upload.go:207 and multipart.go:493. Add regression test with a reader shorter than declared size.
