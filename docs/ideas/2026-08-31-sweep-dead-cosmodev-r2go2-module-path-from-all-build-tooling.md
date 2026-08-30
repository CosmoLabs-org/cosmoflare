---
ulid: 01M1AEZQEP36Q652YQHVFAQ20A
id: IDEA-041
title: Sweep dead CosmoDev-R2Go2 module path from all build tooling
created: "2026-08-31T03:10:01.174494+04:00"
status: seed
source: agent
origin:
    session: 36
    trigger: audit-agent-5-distribution
    file: docs/audit/2026-08-31-cosmoflare/agent-5-distribution.md
tags:
    - audit
    - distribution
---

# Sweep dead CosmoDev-R2Go2 module path from all build tooling

go.mod is cosmoflare but ldflags inject github.com/CosmoLabs-org/CosmoDev-R2Go2/cmd.AppVersion in Makefile:19, Dockerfile:30-32, release.yml:62-64 — Go -X silently no-ops on nonexistent symbols so every binary reports version dev. Also release.yml:87 go install path and all 3 installer scripts point at the dead repo. Fix all sites in one sweep; define the ldflags string once; add post-build --version assertion.
