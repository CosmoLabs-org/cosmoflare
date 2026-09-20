# FEAT-020 wave 4 — cmdmanifest registry: tunnel + account groups

Repo: /Users/gabstudio/PROJECTS/cosmoflare.

## Files you own (ONLY these two)

- `internal/cmdmanifest/data.go`
- `cmd/manifest_tree_test.go`

## Task

Same procedure as wave 3 (see the wave-3 block at the end of data.go for the format):

1. Add `"tunnel", "account"` to the group lists in `collectCobraPaths` calls and the group switches in `TestManifestCLIPathsResolveAgainstCobraTree` / `TestCobraTreeLeavesAllRegistered` in cmd/manifest_tree_test.go.
2. `DUMP_TREE=1 go test ./cmd/ -run TestDumpCobraTree -v` — one entry per live leaf under `tunnel` and `account`.
3. Append entries to data.go:
   - tunnel group: permissions account `Cloudflare Tunnel: Read` (reads) / `Cloudflare Tunnel: Edit` (writes) — verified names from docs/research/2026-09-17-cf-perms-next-waves-tiers/qwen-results.md (grep "Cloudflare Tunnel"); endpoints from pkg/cosmoflare/tunnel.go. `tunnel delete` is destructive+high (irreversible, cascade).
   - account group: member/role/audit-log commands; endpoints from pkg/cosmoflare/account_members.go and account_audit.go; permissions from the dataset (grep "Members" and "Audit Logs"). `account member remove` is destructive+high.
4. `go test ./internal/cmdmanifest/ ./cmd/ -run 'Manifest|CobraTree' -count=1 -v` green; `gofmt -l internal/cmdmanifest/ cmd/` empty.

## Commit

`feat(manifest): FEAT-020 wave 4 — tunnel + account groups` — conventional, no AI attribution.
