# TASK-011 (pilot) — cobra Args-level resource-prefix mechanism

Repo: /Users/gabstudio/PROJECTS/cosmoflare.

## Scope: PILOT ONLY

The full refactor covers ~50 `applyResourcePrefix(args[0])` sites across 21 files. This dispatch lands the MECHANISM plus ONE converted group as the pilot. Do NOT convert the other groups.

## Mechanism

1. New file `cmd/prefixed_args.go`:

```go
// prefixedResourceArgs wraps a cobra positional-args validator so the
// first argument is treated as a profile-prefixed resource name: the
// prefix is applied (idempotently) before the wrapped validator runs.
// Runners then read args[0] bare — the convention is declared once at
// command definition instead of remembered per runner.
func prefixedResourceArgs(wrapped cobra.PositionalArgs) cobra.PositionalArgs

// prefixedFlag applies the profile prefix to a name sourced from a flag
// rather than a positional (e.g. worker routes --script): call it at the
// top of the runner, before any use of the flag value.
func prefixedFlag(cmd *cobra.Command, name string) string
```

2. Convert the **dns** group as the pilot (cmd/dns.go — every runner that calls `applyResourcePrefix`): set `Args: prefixedResourceArgs(cobra.MinimumNArgs(1))`-style on the dns subcommands (preserve each command's CURRENT arg validation semantics — check what Args they have today, including the implicit default), remove the `applyResourcePrefix(args[0])` calls from the dns runners, keep `applyResourcePrefix` itself exported for the unconverted groups.
3. Coverage test `cmd/prefixed_args_test.go`: with an active profile prefix set, a dns command invocation rewrites args[0]; without a profile, args pass through unchanged; the wrapped validator still enforces arity; idempotent on already-prefixed names. Follow existing cmd test patterns (creds-guard style, direct runner calls).

## Files

`cmd/prefixed_args.go` (new), `cmd/prefixed_args_test.go` (new), `cmd/dns.go`. NOTHING else — other groups convert in a later wave.

## Verify

`go build ./...`, `go vet ./cmd/`, `go test ./cmd/ -count=1 -timeout 300s` green. The tree-walk tests must stay green (leaf paths unchanged).

## Commit

`refactor(cmd): prefixedResourceArgs mechanism + dns group pilot (TASK-011)` — conventional, no AI attribution.
