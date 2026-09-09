---
ulid: 01M207EXMKWTJHWQ2B86MKWADZ
title: First-run setup wizard — one token, whole platform
created: "2026-09-08T14:01:47.92376+04:00"
status: harvested
source: human
origin:
    session: 43
tags:
    - onboarding
    - ux
promoted_to: FEAT-017
---

# First-run setup wizard — one token, whole platform

# First-run setup wizard — one token, whole platform

From the MCP discussion: cosmoflare needs zero MCP and zero multi-step config — one API token covers REST + R2 S3 endpoints. Make that frictionless: 'cosmoflare setup' (or extend 'account') runs a wizard — ask/validate token, resolve account ID, verify with TestConnection, write .cosmoflare.yaml, offer doctor check — done in under a minute. Success bar: a new user goes from install to 'cosmoflare status' showing live account state with ONE command and ONE credential. Why: the official cf CLI + MCP stack needs plugin installs, OAuth per server, and restarts; our moat is exactly this simplicity.
