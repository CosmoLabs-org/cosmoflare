---
ulid: 01M207FPGWWC35YBHF2WND4NX5
title: R2 lifecycle policies
created: "2026-09-08T14:02:13.404247+04:00"
status: seed
source: human
origin:
    session: 43
tags:
    - r2
    - storage
---

# R2 lifecycle policies

R2 bucket lifecycle rules (expire objects by prefix/age, abort incomplete multipart uploads). Flagged during the 2026-09-08 dead-code sweep: cmd_disabled/policy.go was a 2.9KB pre-rename draft, deleted; rebuild fresh. Service + 'cosmoflare bucket policy' commands; fits GuardrailChecker-adjacent safety story (data-deletion operations get confirm + audit trail).
