# TASK-011 wave 2 — prefixedResourceArgs conversion: bucket group

Repo: /Users/gabstudio/PROJECTS/cosmoflare. The mechanism already exists (TASK-011 pilot, merged): read `cmd/prefixed_args.go` first, and `cmd/dns.go` for a converted reference group.

## Files you own (ONLY these)

- `cmd/bucket.go`
- `cmd/bucket_domain.go`
- `cmd/bucket_lifecycle.go`
- `cmd/bucket_notifications.go`
- `cmd/bucket_policy.go`

## Conversion rule (per site)

For every `applyResourcePrefix(...)` call in these files:
1. Positional args[0] sites: set the cobra command's `Args:` field to `prefixedResourceArgs(<current validation>)` (preserve the CURRENT validation exactly; a command with no Args field gets `prefixedResourceArgs(nil)` if it takes a name in args[0]) and remove the wrap in the runner. Note bucket_domain.go prefixes args[0] (bucket) while args[1] (domain) passes through — only args[0] converts.
2. Flag-sourced names: replace the manual wrap with `prefixedFlag(cmd, "<flag>")` at the top of the runner.
3. JUDGMENT: only convert NAME sites (bucket names). IDs, non-first positions, spec-file contents — LEAVE, and note in the commit body.
4. Do NOT touch applyResourcePrefix itself, tests, or any file outside your list.

## Verify

`go build ./... && go vet ./cmd/ && go test ./cmd/ -count=1 -timeout 420s` — all green.

## Commit

`refactor(bucket): declare resource-prefix scoping at command definitions (TASK-011)` — conventional, no AI attribution. List any skipped sites + reason in the body.
