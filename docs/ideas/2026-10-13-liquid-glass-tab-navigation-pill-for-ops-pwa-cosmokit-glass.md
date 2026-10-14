---
ulid: 01M4V3PVHWKZ2T7ZMS4YZ9Q6WR
id: IDEA-094
id_assigned_at: 2026-10-14T20:11:32.626364+04:00
title: Liquid glass tab navigation pill for Ops PWA + CosmoKit glass-surface module
created: "2026-10-13T21:07:06.940326+04:00"
status: harvested
source: human
origin:
    session: 77
promoted_to: FEAT-pCTEKWX
---

# Liquid glass tab navigation pill for Ops PWA + CosmoKit glass-surface module

# Liquid glass tab navigation pill for Ops PWA + CosmoKit glass-surface module



This idea proposes building a tab bar for the Ops PWA where the active tab indicator is a translucent "liquid glass" pill — think Apple's Liquid Glass design language: frosted, light-refracting material that slides smoothly between tabs to mark the current section. Rather than hardcoding that effect once, the glass rendering itself (backdrop blur, translucency, specular edge highlights) would be extracted into a reusable `glass-surface` module in CosmoKit, so the same material could later back cards, modals, and toolbars. It's useful because it modernizes the PWA with a native-app feel while centralizing the notoriously fiddly parts — `backdrop-filter` performance, browser fallbacks, stacking-context bugs — into one tested primitive instead of copy-pasted glassmorphism CSS.

If you share the current tab component, I can sketch exactly where the pill logic would slot in.
