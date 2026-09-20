# TASK-016 — Wizard EOF contract: honor ReadLine errors instead of discarding

Repo: /Users/gabstudio/PROJECTS/cosmoflare.

## Problem

`internal/interactive/input.go` `ReadLine()` returns `(string, error)` with an EOF error on closed stdin, but `internal/interactive/setup.go` call sites (~lines 157, 190, 223, 240, 288, 295 — verify current line numbers) discard the error: `response, _ := w.Input.ReadLine()`. Result: three prompt loops with three different EOF/empty behaviors (Step1 err-exit, Step2 unbounded empty-retry, Step3 err + 3-empty bound). The next prompt author copies whichever they find.

## Fix

1. Add a small helper on SetupWizard:

```go
// ask prints the prompt and reads one line. A reader error (EOF, closed
// stdin) is terminal: scripted and agent callers never answer prompts, so
// the wizard returns the error instead of looping on empty reads. The
// bounded-empty-retry policy stays ONLY where product intent wants a human
// to retry (Step 3's account ID).
func (w *SetupWizard) ask(prompt string) (string, error)
```

2. Convert every `response, _ := w.Input.ReadLine()` site in setup.go to `ask` (or explicit error handling where a bare read is genuinely correct — justify in a comment if so). On error the enclosing step returns a meaningful error (the callers already have error paths — Step1 shows the pattern).
3. Keep the Step 3 three-empty bound (merged fix from batch-005 redispatch) — it becomes the ONLY empty-retry policy in the file.
4. Tests in `internal/interactive/setup_test.go` (follow existing patterns; `input_test.go` has a mockReader): per-step closed-stdin test — each step that prompts must return an error, not loop. Use a short test deadline to fail fast if any loop survives.

## Files

`internal/interactive/setup.go`, `internal/interactive/setup_test.go` (and `input.go` only if the helper belongs there).

## Verify

`go test ./internal/interactive/ -count=1 -timeout 120s` green; `go vet ./internal/interactive/` clean. The existing `TestSetupWizardAccountInfo_ManualFallbackWithEmptyToken` in cmd/setup_wizard_test.go must stay green unchanged (empty values on closed stdin via the swallowed wrapper error).

## Commit

`refactor(wizard): treat reader errors as terminal across all prompt steps (TASK-016)` — conventional, no AI attribution.
