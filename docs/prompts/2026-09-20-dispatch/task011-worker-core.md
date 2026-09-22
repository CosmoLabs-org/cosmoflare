# TASK-011 wave 2 — prefixedResourceArgs conversion: worker core group

Repo: /Users/gabstudio/PROJECTS/cosmoflare. The mechanism already exists (TASK-011 pilot, merged): read `cmd/prefixed_args.go` first, and `cmd/dns.go` for a converted reference group.

## Files you own (ONLY these)

- `cmd/worker.go`
- `cmd/worker_bindings.go`
- `cmd/worker_cron.go`
- `cmd/worker_deployments.go`
- `cmd/worker_domains.go`

## Conversion rule (per site)

For every `applyResourcePrefix(args[0])` call in these files:
1. Set the cobra command's `Args:` field to `prefixedResourceArgs(<whatever validation it has today>)` — preserve the CURRENT validation exactly (e.g. `cobra.ExactArgs(1)` → `prefixedResourceArgs(cobra.ExactArgs(1))`; a command with no Args field gets `prefixedResourceArgs(nil)` if it takes a name in args[0], otherwise leave it alone).
2. Remove the `applyResourcePrefix` wrap in the runner — read `args[0]` bare.
3. JUDGMENT: only convert sites where args[0] is a resource NAME (worker name, cron schedule context, etc.). If a site prefixes an ID (UUID), a non-first position, or a flag value — LEAVE IT, and note it in the commit body. Flag-sourced names use `prefixedFlag(cmd, "flagname")` instead.
4. Do NOT touch applyResourcePrefix itself, tests, or any file outside your list.

## Verify

`go build ./... && go vet ./cmd/ && go test ./cmd/ -count=1 -timeout 420s` — all green.

## Commit

`refactor(worker): declare resource-prefix scoping at command definitions (TASK-011)` — conventional, no AI attribution. List any skipped sites + reason in the body.
