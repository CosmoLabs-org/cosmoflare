---
id: IDEA-MN5GE7TC
title: Fix PersistentPreRun to skip API validation for non-API commands
created: "2026-03-25T03:57:51.888221+01:00"
status: seed
source: agent
origin:
    session: 17
    trigger: audit-core-logic-bugs
tags:
    - audit
    - bug
---

# Fix PersistentPreRun to skip API validation for non-API commands

BUG-002: rootCmd.PersistentPreRun calls validateEnvironment() which requires CLOUDFLARE_API_TOKEN for ALL commands including help, version, completion, setup, config. First-time users cannot even see help output. Fix by checking cmd.Name() or moving validation to individual command PreRun.
