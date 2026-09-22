# TASK-011 wave 2 — prefixedResourceArgs conversion: worker rest group

Repo: /Users/gabstudio/PROJECTS/cosmoflare. The mechanism already exists (TASK-011 pilot, merged): read `cmd/prefixed_args.go` first, and `cmd/dns.go` for a converted reference group.

## Files you own (ONLY these)

- `cmd/worker_routes.go`
- `cmd/worker_secrets.go`
- `cmd/worker_subdomain.go`
- `cmd/worker_tail.go`
- `cmd/worker_types.go`

## Conversion rule (per site)

For every `applyResourcePrefix(...)` call in these files:
1. Positional args[0] sites: set the cobra command's `Args:` field to `prefixedResourceArgs(<current validation>)` (preserve the CURRENT validation exactly; a command with no Args field gets `prefixedResourceArgs(nil)` if it takes a name in args[0]) and remove the wrap in the runner.
2. Flag-sourced names (worker routes --script is a known one): replace the manual wrap with `prefixedFlag(cmd, "script")` at the top of the runner.
3. JUDGMENT: only convert NAME sites. IDs (UUIDs), non-first positions, or already-correct prefix handling — LEAVE, and note in the commit body.
4. Do NOT touch applyResourcePrefix itself, tests, or any file outside your list.

## Verify

`go build ./... && go vet ./cmd/ && go test ./cmd/ -count=1 -timeout 420s` — all green.

## Commit

`refactor(worker): declare resource-prefix scoping for routes/secrets/tail/types (TASK-011)` — conventional, no AI attribution. List any skipped sites + reason in the body.
