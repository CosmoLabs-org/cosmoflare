---
ulid: 01M4M3ZKZDTFEDT80PHRJNC5DY
title: In-app permissions guide — Ops tells the operator exactly what access it's missing and how to grant it
created: "2026-10-11T03:57:13.069474+04:00"
status: seed
source: human
origin:
    session: 74
    trigger: operator 2026-10-11 at session close
---

# Guides section — articles and educational tutorials inside Ops

Operator scope expansion (2026-10-11): a full GUIDES section in Ops — the home for important articles and educational tutorials for Cosmoflare Ops users. The permissions guide below is article #1.

## Article 1: the permissions guide — Ops tells the operator exactly what access it's missing and how to grant it

A drawer-routed 'Access & Permissions' view in the Ops app: one row per Cloudflare token permission the app benefits from, each with (a) what it unlocks, (b) why it's needed, (c) the exact dashboard path (My Profile -> API Tokens -> Edit), and (d) LIVE status where the app can detect absence — billing period source==='anchor' means Billing Read missing; absent email metrics means Email Routing Read missing; analytics-only worker profiles mean Workers Scripts Read missing; registrar fields absent on CF-registered domains mean Registrar Read missing. Green when present, amber with fix steps when not. Rows: Billing Read (BUG-056, FEAT-062 usage-v2, subscription-true periods), Email Routing Read (FEAT-059 email metrics + bounce alert class), Registrar Read (domain profiles: authoritative expiry, auto-renew state, transfer lock), API Tokens Read (FB-30 permissions preflight — commands check scopes before failing), Workers Scripts Read (richer worker profiles: bindings, last-deployed), Rulesets/WAF Read (FEAT-050 pre-launch verdict + audit corpus), D1 Read (D1 profile pages). Plus a separate card: the R2 API token (32-char key pair, R2 -> Manage API Tokens -> ~/.r2go2/config.yaml) that unlocks the CLI S3 data plane — a different token, not a scope. Companion to FB-30: preflight is the machine check, this guide is the human path. Content sourced from the session-2036 permission audit.
