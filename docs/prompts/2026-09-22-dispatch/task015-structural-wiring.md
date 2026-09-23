# TASK-015 — Structural dry-run/audit wiring: derive paths from cobra, wrap destructive RunEs once

Repo: /Users/gabstudio/PROJECTS/cosmoflare. Follow-up to TASK-011: dry-run/audit wiring is still opt-in per RunE body with hand-typed path literals (`[]string{"r2","bucket","delete"}` etc.). This dispatch makes coverage structural.

## Files you own

- `cmd/destructive.go` — NEW: the two mechanisms below
- `cmd/destructive_test.go` — NEW
- Every file with a hand-typed destructiveDryRun/auditMutation literal: `cmd/bucket.go`, `cmd/d1.go`, `cmd/kv.go`, `cmd/worker.go`, `cmd/waf_lists.go`, `cmd/tunnel.go`, `cmd/loadbalancer.go`, `cmd/logpush.go` (grep to confirm the full set)
- `cmd/dryrun_wire.go` — destructiveDryRun may delegate to the new mechanism; keep it compiling
- Registration init()s of every registry-destructive command

## Mechanism 1 — derived CLI path

```go
// cliPathOf derives the command's manifest path from the live cobra tree:
// CommandPath() minus the root name. A rename in the tree flows through
// automatically — no hand-typed literals to forget.
func cliPathOf(cmd *cobra.Command) []string
```

Replace EVERY `[]string{"...", ...}` literal passed to `destructiveDryRun` or `auditMutation` with `cliPathOf(cmd)` (runners already receive cmd). cmd/audit_wire.go's cliPath parameter stays — callers now pass derived paths.

## Mechanism 2 — registration wrapper

```go
// withDestructiveDefaults wraps a mutating runner so registry-flagged
// destructive commands execute dry unless --force. When the registry marks
// the command destructive and the operator passed neither --force nor
// --dry-run, the wrapper runs the runner with DryRun temporarily true —
// the runner's existing "if DryRun" branch prints its own preview, so a
// newly registered destructive command gets safe defaults with ZERO
// runner-side wiring to remember. Non-destructive and already-dry runs
// pass through untouched.
func withDestructiveDefaults(cmd *cobra.Command, runE func(*cobra.Command, []string) error) func(*cobra.Command, []string) error
```

- Force detection: `cmd.Flags().GetBool("force")` — every delete command already names its flag "force"; treat a missing flag as force=false.
- Apply at registration: every command whose registry entry has `Destructive: true` — grep internal/cmdmanifest/data.go for `Destructive: true` and wrap exactly those commands' RunE in their init() (bucket delete, worker delete, d1 delete, kv namespace delete, tunnel delete, lb pool delete, lb monitor delete, logpush job delete — verify against the registry, do not trust this list).
- Then REMOVE the now-redundant `destructiveDryRun(...)` calls inside those runners — their plain `if DryRun { ... }` branches handle everything. Keep each runner's force flag and any confirmation prompts for NON-registry-destructive commands (dns delete keeps its prompt exactly as is — registry says non-destructive).

## Tests (cmd/destructive_test.go)

- cliPathOf: returns the manifest path for a known command ("bucket delete" → ["r2","bucket","delete"]... actually the manifest path for r2 bucket delete is r2/bucket/delete — assert against cmdmanifest.Load().ResolveCLI(cliPathOf(cmd)) returning ok)
- withDestructiveDefaults table: destructive+no flags → runner observes DryRun=true; destructive+force → false; destructive+explicit DryRun → true; non-destructive command → runner observes the flag as passed
- A registry sweep test: every registry-destructive command's RunE is wrapped — walk rootCmd, for each registry-destructive command assert the wrapper is in effect (e.g. marker behavior: invoking with empty creds yields a DRY RUN notice, not a service error)

## Verify

`go build ./... && go vet ./cmd/ && go test ./cmd/ ./internal/cmdmanifest/ -count=1 -timeout 420s` green. Existing dry-run tests (TestRunBucketDelete_*, TestRunD1Delete_DryRunDefault, tunnel/lb/logpush delete tests) must stay green UNCHANGED — they pin the same user-visible behavior through the new mechanism.

## Commit

`refactor(cmd): structural destructive defaults + derived CLI paths (TASK-015)` — conventional, no AI attribution.
