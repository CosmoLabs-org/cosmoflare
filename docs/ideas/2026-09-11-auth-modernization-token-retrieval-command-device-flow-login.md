---
ulid: 01M26TFKXVCZ40ZJVYC61FFH2H
title: 'Auth modernization: token retrieval command + device-flow login'
created: "2026-09-11T03:29:40.283637+04:00"
status: seed
source: agent
origin:
    session: 51
    trigger: 'Qwen docs-references corpus: two 2025-2026 wrangler auth changelogs'
---

# Auth modernization: token retrieval command + device-flow login

Two small auth upgrades from the docs-references corpus: (1) 'cosmoflare auth token' — retrieve current credentials for use by other tools (wrangler added wrangler auth token 2025-12-18, changelog cited in qwen-results-docs-references.md); (2) 'auth login --device' — OAuth 2.0 Device Authorization Grant, no local callback server (wrangler added 2026-08-04). Both fit FEAT-017 first-run wizard; file as feature when picked up.
