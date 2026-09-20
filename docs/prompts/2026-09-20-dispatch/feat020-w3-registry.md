# FEAT-020 wave 3 — cmdmanifest registry: email, waf, ssl, cache, hyperdrive, alerts

You are extending the command registry in /Users/gabstudio/PROJECTS/cosmoflare (Go, module github.com/CosmoLabs-org/cosmoflare).

## Files you own (edit ONLY these two)

- `internal/cmdmanifest/data.go` — append entries
- `cmd/manifest_tree_test.go` — extend the group lists

## What to do

1. In `cmd/manifest_tree_test.go`, add `"email", "waf", "ssl", "cache", "hyperdrive", "alerts"` to the group lists in `collectCobraPaths` calls, and extend the group switch/cases in `TestManifestCLIPathsResolveAgainstCobraTree` and `TestCobraTreeLeavesAllRegistered` so the wave-3 groups are verified the same way worker/kv/d1/dns are.
2. Run `DUMP_TREE=1 go test ./cmd/ -run TestDumpCobraTree -v` to enumerate the live leaf paths under those six groups. One registry entry per leaf — no more, no fewer.
3. In `internal/cmdmanifest/data.go`, append one `Command` per leaf under a `// --- <group> (wave 3; permissions from the Qwen dataset) ---` section header, copying the exact field layout and formatting of the wave-2 blocks (worker/kv/d1/dns at the end of the file):
   - `ID` dotted (`email.rule.create` style), `CLIPath` from the dump.
   - `WranglerEquivalent` when a close wrangler command exists (must start with `"wrangler "`), else omit the field (zero value `""`).
   - `Service` tag = group name (`email`, `waf`, `ssl`, `cache`, `hyperdrive`, `alerts`); `Scope` = resource granularity; `Verb` one of `read|write|delete|list`.
   - `APIOps`: read the corresponding `pkg/cosmoflare/` service file (email.go, waf.go or waf_lists.go, ssl.go, cloudflare_cache.go, hyperdrive.go, alerts.go) and record the endpoints those methods hit (`METHOD /path` with `{account_id}`/`{zone_id}` placeholders). When the library wraps the cloudflare-go SDK, use the documented Cloudflare API path for that operation.
   - `Permissions`: from the verified token-UI names in `docs/research/2026-09-17-cf-perms-next-waves-tiers/qwen-results.md` — grep the YAML for the matching group (e.g. Email Routing Addresses, SSL and Certificates, Cache Purge, Hyperdrive, Notifications/Alerts). Use the exact spelling from the dataset. When the dataset has no matching group, leave Permissions EMPTY with a one-line comment `// permission pending FEAT-011 dataset` — never guess a name.
   - `DangerLevel` low|medium|high (reads/lists low; writes medium; irreversible container deletes high). `Destructive: true` ONLY for high-danger irreversible deletes (destructive ⇒ high is test-enforced). `Trackable: true`.
   - `Limits`, `LocalChecks`, `APIChecks`, `RateLimit`: omit unless you can ground them in the library code — sparse is correct, fabricated is not.
4. Verify: `go test ./internal/cmdmanifest/ ./cmd/ -run 'Manifest|CobraTree' -v` — all green. The tree tests prove both directions: every registered path exists in the cobra tree, every wave-3 leaf is registered.

## Rules

- NEVER edit files outside the two listed.
- gofmt clean (`gofmt -l internal/cmdmanifest/ cmd/` empty).
- Commit with `git commit -m "feat(manifest): FEAT-020 wave 3 — email/waf/ssl/cache/hyperdrive/alerts groups"` (conventional, no AI attribution).
