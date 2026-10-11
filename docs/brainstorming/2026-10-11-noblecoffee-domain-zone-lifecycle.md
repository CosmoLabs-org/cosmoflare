---
title: Domain & zone lifecycle controls on the Ops page
source: Noble.Coffee operator feedback (pasted 2026-10-11, session 2037)
related_ideas: 2026-10-11-ops-write-path-guarded-emergency-actions-in-the-ops-gui.md
related_issues: FEAT-pD5WR3L
status: proposed
created: "2026-10-11T20:05:00+04:00"
---

# Cosmoflare — domain & zone lifecycle controls on the Cloudflare ops page

## Context

We manage Cloudflare Pages/Workers projects and their custom domains across CosmoLabs projects (noble.coffee, cosmobee, baristabase, …). Today, taking a domain offline means clicking through the Cloudflare dashboard by hand — e.g. we just retired staging.noble.coffee by manually detaching it from its Pages project. We want this self-serve from our ops page.

## Goal

Add a Domains / Zones section to the Cloudflare ops page that lists, per account + project:

- every custom domain attached to our Pages and Workers projects (hostname, project, status, DNS/SSL state)
- every zone in the account (name, status, record count)

and gives two distinct operations with different safety levels:

1. **Take offline (reversible)** — detach a custom domain from its Pages/Workers project. The zone and DNS stay intact; re-attaching later must be one click. Use this for retiring a surface (staging.X, old preview hosts) without destroying anything.
2. **Remove (destructive, gated)** — delete a DNS zone. Not reversible: purges DNS records and any zone-level config. Requires a typed confirmation of the exact zone name, and must HARD-REFUSE apex domains of production sites (configurable allowlist of protected hostnames — noble.coffee, cosmolabs.org, etc.) unless an operator override flag is explicitly set.

## Requirements

- Confirmation flow: detach = one confirm; zone removal = typed name match + explicit "this cannot be undone" warning.
- Audit log: every action records operator, timestamp, target, and action type; list view on the page.
- Cloudflare API v4 (Pages custom domains, Workers custom domains, Zones endpoints) using the credentials/bindings this project already has — never log secrets.
- Batch view: "all domains attached to project X" so multi-domain projects can be reviewed at a glance.
- Errors from the API surface verbatim in the UI (no silent failures).

## Operator amendments (2026-10-11)

- Dry-run preview requirement WAIVED by the operator: features deploy immediately.

## Out of scope (for now)

- DNS record-level editing (zone-level only).
- Domain registration/transfers — only zones and attachments that already exist.

## Requested process

Start by exploring the existing ops page structure and auth/credential wiring, then propose the UI layout and API call plan before implementing.
