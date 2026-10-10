---
created: "2026-10-06T19:19:33+04:00"
goals_completed: 0
goals_total: 0
origin: migrated by ccs prompts migrate
priority: medium
related_prompts: []
requires_reading: []
schema_version: 1
status: PENDING
tags: []
title: FEAT-020 completion — fill the last sparse permission rows in cmdmanifest
---

# FEAT-020 completion — fill the last sparse permission rows in cmdmanifest

## Context

`internal/cmdmanifest/data.go` is the per-command registry (FEAT-020). A
query of `cmdmanifest.Load()` shows 17 of 207 entries with empty
Permissions. Classification (verified 2026-09-30 by the session):

- 3 × `worker.domain.{list,attach,detach}` — call the real CF Workers
  domains API (`GET/POST/DELETE /accounts/{account_id}/workers/domains...`).
  These need their permission column filled.
- 1 × `account.verify` — calls `GET /user/tokens/verify`, which accepts ANY
  valid token regardless of scopes. Needs an explicit "verified none
  required" marker, not a guessed permission.
- 13 × (`alerts.*` 8 rows, `account.{list,add,switch,remove,current}` 5
  rows) — local-only commands (no CF API calls; accounts live in
  `~/.cosmoflare/accounts.yaml`, alerts rules are local). Empty permissions
  is CORRECT for these; they must stay untouched.

The permission-name source of truth is
`pkg/cosmoflare/permdata/permissions.json` (157 families, FEAT-011 Qwen
dataset). The Workers-domains permission family there is
`account.workers_scripts` ("Workers Scripts"). Before filling the three
rows, confirm the Workers domains API permission against Cloudflare's docs
page for the endpoint (developers.cloudflare.com — Workers domains API /
"Workers Scripts" permission). If docs name a different/more specific
permission, use the docs name and say so in the commit body.

## Exact changes

All in this repo, package `internal/cmdmanifest`:

1. `manifest.go` — add one field to the `Command` struct:

   ```go
   NoPermsRequired bool // true = verified that no token permission gates this command's API calls
   ```

   Update the package doc comment's permissions sentence to cover the new
   semantics: empty Permissions + NoPermsRequired=false means "not yet
   authored"; NoPermsRequired=true means "verified none required".

2. `data.go` — fill exactly 4 entries:
   - `worker.domain.list`: `Permissions: Permissions{Account: []string{"Workers Scripts"}}`
     (read verbs need the read-access form if the catalog distinguishes —
     permissions in this registry are family names, follow the existing
     rows' style, e.g. r2 rows use "Workers R2 Storage").
   - `worker.domain.attach` and `worker.domain.detach`: same account
     permission as above.
   - `account.verify`: `NoPermsRequired: true`.
   - Do NOT touch the 13 local-only rows.

3. `manifest_test.go` — add tests FIRST (red), then implement:

   - `TestPermissionsCompleteForAPICommands`: every entry with
     `len(APIOps) > 0` must have at least one permission across
     Account/Zone/User OR `NoPermsRequired == true`. Local-only entries
     (no APIOps) are exempt.
   - `TestWorkerDomainPermissionsFilled`: the three `worker.domain.*`
     entries declare exactly one account permission whose name matches the
     family name in `pkg/cosmoflare/permdata` for `account.workers_scripts`
     (read the name from the JSON via a relative-path read or hardcode with
     a comment citing permdata id `account.workers_scripts`).
   - `TestAccountVerifyNoPermsRequired`: `account.verify` has
     `NoPermsRequired == true`.

## Verification (mandatory, in order)

```bash
go test ./internal/cmdmanifest/ -run 'TestPermissionsCompleteForAPICommands|TestWorkerDomainPermissionsFilled|TestAccountVerifyNoPermsRequired'   # must FAIL before data.go/manifest.go changes, PASS after
go test ./internal/cmdmanifest/    # full package green
go vet ./internal/cmdmanifest/
```

Commit (conventional, body required) e.g.:
`feat(cmdmanifest): fill worker-domain permissions + NoPermsRequired marker (FEAT-020)`
Body: state which permission you filled, the docs URL you verified against,
and that the 13 local-only rows are intentionally untouched.

## Out of scope

- Any change to the 13 local-only rows.
- Any change outside `internal/cmdmanifest/`.
- Renaming existing fields.
