# FEAT-018 P0 — `cosmoflare pages deploy --verify`: deploy + poll + verify in one command

Repo: /Users/gabstudio/PROJECTS/cosmoflare. Evidence: the deploy+verify loop ran 3× in one day by hand (build stamp → wrangler pages deploy → 6+ manual curls).

## Files you own

- `cmd/pages_deploy_verify.go` — NEW (registers flags onto the EXISTING `pages deploy` command and wraps/extending its runner — READ cmd/pages.go first; if pages deploy lives there, add ONLY a new file with an init() that appends flags via `pagesDeployCmd.Flags()` and a new `--verify` code path; do NOT restructure pages.go)
- `cmd/pages_deploy_verify_test.go` — NEW

## Command contract

`cosmoflare pages deploy DIR [--project NAME] --verify URLSPEC [--build-id STR]`

- `--verify "path1,path2"` (or repeatable) — AFTER deploy, poll the deployment URL until live (GET the deployment URL; the Pages deploy response carries the URL — reuse the existing pages deploy service call in cmd/pages.go / pkg/cosmoflare/pages.go), then GET each path and check HTTP 200 (non-2xx or connection error = FAIL).
- `--build-id STR` — if the environment var PUBLIC_BUILD_ID is not set and --build-id is empty, default to `git rev-parse --short HEAD` output (exec git, ignore error → empty).
- Output: pass/fail TABLE in human mode (path, status, ms); `--json` envelope: {deployed_url, build_id, checks:[{path,status,ok,ms}]}. NON-ZERO exit on any failed check — this is the agent-verifiable contract.
- Poll: up to 10 attempts, 5s apart, first successful 200 on ANY verify path (or the deployment URL when no paths given) marks live; timeout = FAIL with clear message.
- Respect DryRun: print what would run, skip network.

## Tests (pure logic + guards, no network)

- URL list parsing (dedupe, strip empty, relative paths kept as-is)
- poll decision as a pure function: attempt count + status → {keepPolling, live, failed}
- pass/fail table renderer on a fixed check set; JSON envelope shape
- build-id fallback chain: flag > env > git (stub the git call behind a package var)
- missing-creds guard; DryRun short-circuit

Follow cmd/dns_test.go + doctor_run_test.go patterns (creds-guard, capturePrint). If the existing pages deploy runner is not seam-friendly, test your new helpers directly and guard-test the command entry.

## Verify

`go build ./... && go vet ./cmd/ && go test ./cmd/ -count=1 -timeout 420s` green.

## Commit

`feat(pages): deploy --verify — poll-until-live plus route pass/fail table (FEAT-018)` — conventional, no AI attribution.
