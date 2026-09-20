# FEAT-030 — Registrar operations depth: register, transfer, renew, lock/unlock, contacts, DNSSEC

Repo: /Users/gabstudio/PROJECTS/cosmoflare. This is the registrar piece of FEAT-006 Domain Center's remaining waves. RegistrarService currently ships ONLY List (`pkg/cosmoflare/registrar.go`) — FEAT-006 Wave A was list-only.

## Files you own

- `pkg/cosmoflare/registrar.go` — extend the existing service
- `pkg/cosmoflare/registrar_test.go` — extend
- `cmd/domains_registrar.go` — NEW command file (register onto `domainsCmd`'s group the way domains_ns.go does — READ that file first for the registration pattern and the `newDomainService` seam usage)
- `cmd/domains_registrar_run_test.go` — NEW tests

## Operations (verified endpoint shapes — Cloudflare Registrar API; use cloudflare-go's typed resources where they exist, as worker_cron.go does)

- `POST /accounts/{account_id}/registrar/domains` — register/transfer-in a domain (body: name) — transfers have a pending state to poll
- `GET /accounts/{account_id}/registrar/domains/{domain_name}` — get status/details
- `POST /accounts/{account_id}/registrar/domains/{domain_name}/cancel_transfer` — cancel a pending transfer
- `POST /accounts/{account_id}/registrar/domains/{domain_name}/renew` — renew (auto-renew is a PATCH on the domain: `auto_renew: bool`)
- Lock: `PATCH`/`PUT` on the domain resource with `locked: bool` (transfer_lock)
- Contacts: `PUT /accounts/{account_id}/registrar/domains/{domain_name}/contacts` (registrant/billing/admin/tech) — accept a `RegistrarContacts` struct
- DNSSEC: `GET/PUT .../dnssec` (status + DS records; enabling returns DS data that must be added at the parent — encode as a pre-flight validation note in the command output, not a blocker)

Permissions: Registrar API uses standard Account/Billing scopes (dataset notes the exact UI group lacks a dedicated category — use `Permissions{Account: []string{"Registrar Domains"}}` ONLY if the Qwen dataset contains that exact name; grep `docs/research/2026-09-17-cf-perms-next-waves-tiers/qwen-results.md` for "Registrar" and use what is there, else leave a `// permission pending FEAT-011 dataset` comment).

## Sequencing rules → pre-flight validations (encode these)

- transfer REQUIRES the domain to be unlocked at the current registrar — validate/warn before initiating
- DNSSEC disable REQUIRES DS records removed at the parent first — warn
- renew only when the domain is in a renewable window — surface API errors verbatim

## Commands

`cosmoflare domains registrar list|get|register|transfer|cancel-transfer|renew|auto-renew <bool>|lock <domain>|unlock <domain>|contacts get|contacts update|dnssec get|dnssec enable|dnssec disable`. Conventions per CLAUDE.md: rich Long help with examples, --json via outPayload/outResult/outErr (pattern: cmd/hyperdrive.go), agent-readable errors. `transfer` and `register` are cost-incurring mutations — DangerLevel medium in help text tone, dry-run respect via the standard `DryRun` flag like cmd/domains.go's list does. NO registry entries (wave 4 owns data.go concurrently).

## Tests

Construction error paths (missing account/token), missing-creds runner guards per subcommand, JSON envelope shape, sequencing warnings (unlock-before-transfer, DS-before-disable) on a stubbed service. Follow cmd/doctor_run_test.go creds-guard + capturePrint patterns.

## Verify

`go build ./...`; `go vet ./cmd/ ./pkg/cosmoflare/`; `go test ./cmd/ ./pkg/cosmoflare/ -count=1 -timeout 420s` green.

## Commit

`feat(registrar): full registrar ops — register, transfer, renew, lock, contacts, DNSSEC (FEAT-030)` — conventional, no AI attribution.
