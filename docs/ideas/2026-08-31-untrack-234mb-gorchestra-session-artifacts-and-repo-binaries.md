---
ulid: 01M1AF0N2K0DR3RQXMVDCHEA0E
id: IDEA-046
title: Untrack 234MB GOrchestra session artifacts and repo binaries
created: "2026-08-31T03:10:31.507685+04:00"
status: seed
source: agent
origin:
    session: 36
    trigger: audit-agent-1-code-quality
    file: docs/audit/2026-08-31-cosmoflare/agent-1-code-quality.md
tags:
    - audit
    - hygiene
---

# Untrack 234MB GOrchestra session artifacts and repo binaries

GOrchestra/sessions/ holds 81 tracked recovery.patch files (some 665K lines, 234MB) with AWS example keys flagged critical by security scans, masking real findings. Also tracked: build/r2go2 36MB, .github/.DS_Store, simple-setup/test-setup binaries; 5x42MB patches not gitignored. git rm --cached + gitignore GOrchestra/sessions/; archive outside repo.
