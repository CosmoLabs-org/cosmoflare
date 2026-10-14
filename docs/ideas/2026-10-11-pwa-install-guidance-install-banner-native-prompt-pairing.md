---
ulid: 01M4M6QQT5P5KYGPMHM3QNFTQW
id: IDEA-091
id_assigned_at: 2026-10-14T20:11:32.626364+04:00
title: 'PWA install guidance: install banner, native prompt, pairing-as-funnel'
created: "2026-10-11T04:45:20.58184+04:00"
status: harvested
source: human
origin:
    session: 75
promoted_to: FEAT-pXNQ5GW
---

# PWA install guidance: install banner, native prompt, pairing-as-funnel

# PWA install guidance: install banner, native prompt, pairing-as-funnel

The Ops pager is already a full PWA (manifest standalone + maskable icons + sw.js + apple-touch-icon) but nothing advertises installability. Build the advertising layer: (1) install-state detection via matchMedia('(display-mode: standalone)') / navigator.standalone — guidance hides itself once installed; (2) Android/desktop: capture beforeinstallprompt and surface an Install button that fires prompt() (native dialog); (3) iOS: dismissible step card 'Share -> Add to Home Screen -> open from Home Screen', shown only on iOS Safari when not standalone; (4) Pairing view becomes the funnel: step 1 Install, step 2 Allow notifications, step 3 Paired — push value is the install reason; (5) banner appears after first successful data render, once per viewer, dismissal remembered. iOS 16.4+ push requires the installed PWA, which is why G-01 pairing depends on this.
