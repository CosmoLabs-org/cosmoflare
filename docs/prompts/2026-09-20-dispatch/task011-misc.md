# TASK-011 wave 2 — prefixedResourceArgs conversion: misc group

Repo: /Users/gabstudio/PROJECTS/cosmoflare. The mechanism already exists (TASK-011 pilot, merged): read `cmd/prefixed_args.go` first, and `cmd/dns.go` for a converted reference group.

## Files you own (ONLY these)

- `cmd/d1.go`
- `cmd/envscope.go`
- `cmd/kv.go`
- `cmd/sync.go`
- `cmd/watch.go`

## Conversion rule (per site)

For every `applyResourcePrefix(...)` call in these files:
1. Positional args[0] sites where args[0] is a resource NAME: set the cobra command's `Args:` field to `prefixedResourceArgs(<current validation>)` (preserve the CURRENT validation exactly; a command with no Args field gets `prefixedResourceArgs(nil)` if it takes a name in args[0]) and remove the wrap in the runner.
2. JUDGMENT — these files are name/ID mixed: d1.go and kv.go take DATABASE/NAMESPACE IDs (UUIDs) in some positions — an ID site must NOT be prefixed; if the existing call prefixes an ID, that is a latent bug: leave the behavior EXACTLY as-is, note it in the commit body, do not fix here. sync.go and watch.go take bucket NAMES — those convert.
3. Do NOT touch applyResourcePrefix itself, tests, or any file outside your list.

## Verify

`go build ./... && go vet ./cmd/ && go test ./cmd/ -count=1 -timeout 420s` — all green.

## Commit

`refactor(cmd): declare resource-prefix scoping for sync/watch/misc runners (TASK-011)` — conventional, no AI attribution. List any skipped/flagged sites + reason in the body.
