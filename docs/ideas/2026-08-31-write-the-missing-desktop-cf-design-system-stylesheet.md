---
ulid: 01M1AF055NY4WGFNENKPFKJD1G
id: IDEA-043
title: Write the missing desktop cf-* design system stylesheet
created: "2026-08-31T03:10:15.221421+04:00"
status: seed
source: agent
origin:
    session: 36
    trigger: audit-agent-17-design-taste
    file: docs/audit/2026-08-31-cosmoflare/agent-17-design-taste.md
tags:
    - audit
    - design
    - desktop
---

# Write the missing desktop cf-* design system stylesheet

desktop/src/styles.css is 15 lines defining none of the 30+ cf-* classes components reference; built CSS bundle is 167 bytes — the app ships as an unstyled stack. Implement: two-pane flex layout, cf-card-grid auto-fill grid, 44px nav targets, :focus-visible outlines, semantic color tokens (--bg/--surface/--text/--accent/--success/--danger), keep the 13.9:1 text pair. Pairs with the a11y pass (role=log, aria-label health state, h1).
