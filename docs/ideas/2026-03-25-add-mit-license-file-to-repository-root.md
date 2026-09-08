---
id: IDEA-031
title: Add MIT LICENSE file to repository root
created: "2026-03-25T03:57:35.395226+01:00"
status: withered
source: agent
origin:
    session: 17
    trigger: audit-distribution-legal
tags:
    - audit
    - legal
resolution:
    reason: implemented
    date: "2026-09-08T15:08:11.597791+04:00"
    note: LICENSE ships MIT (CLAUDE.md license field)
---

# Add MIT LICENSE file to repository root

# Add MIT LICENSE file to repository root

All source files reference MIT license, Dockerfile has MIT label, but no actual LICENSE file exists. This is a legal requirement for open source distribution. Copy standard MIT template and set copyright to CosmoLabs.
