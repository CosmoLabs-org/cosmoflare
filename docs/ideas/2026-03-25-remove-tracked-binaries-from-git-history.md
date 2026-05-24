---
id: IDEA-032
legacy_id: IDEA-MN5GDX5C
title: Remove tracked binaries from git history
created: "2026-03-25T03:57:38.064826+01:00"
status: harvested
source: agent
origin:
    session: 17
    trigger: audit-infrastructure-repo-hygiene
tags:
    - audit
    - hygiene
---


# Remove tracked binaries from git history

simple-setup (4.8MB) and test-setup (4.8MB) are tracked in git, inflating every clone. git rm them and add to .gitignore. Also verify r2go2-enhanced and CosmoDev-R2Go2 (31MB total untracked) are covered by existing gitignore patterns.
