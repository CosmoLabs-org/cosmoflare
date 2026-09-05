# Agent 9: Infrastructure

> Audit: 2026-08-31, cosmoflare v0.17.0, fresh mode. Report saved verbatim from agent output.

I have all the evidence I need. Compiling the report.

## Infrastructure: 4/10

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| containerization | 4/10 | Multi-stage + non-root done correctly, but the image cannot build at all: `.dockerignore` excludes `go.mod`/`go.sum`, and ldflags inject a dead module path |
| orchestration | 2/10 | No compose/K8s/Helm anywhere; the `cosmoflare serve` daemon has no deployment artifact (unit file, manifest) with limits or probes |
| iac_quality | 3/10 | The tool *exports* Terraform (nice feature), but the project itself has zero IaC, a stale provider pin default, and a 994-line disabled CI/CD generator |
| security | 4/10 | Good gitignore hygiene and daemon auth design, but no pipeline scanning, an installer that downloads unverified binaries from a dead repo, and plaintext profile secrets |

### Critical Findings

1. **Docker build is broken — `.dockerignore` excludes `go.mod` and `go.sum`** — The ignore file contains `*.mod` and `*.sum` patterns, which match the root `go.mod`/`go.sum`. The Dockerfile's dependency layer `COPY go.mod go.sum ./` (Dockerfile:14) fails with "not found" because those files never enter the build context. Nobody noticed because no CI job ever builds the image. The `docker-build` Makefile target (Makefile:337-340) is therefore dead on arrival.
   - **Severity**: high
   - **File**: `.dockerignore:29-30`
   - **Fix**: Delete the `*.mod` / `*.sum` entries (they were probably meant for `vendor/` leftovers). Add a CI job that runs `docker build` on every push so breakage is caught.

2. **All version stamping is dead — ldflags target the old module path** — The module was renamed to `github.com/CosmoLabs-org/cosmoflare` (go.mod:1), and the version variables live in `cmd` (cmd/root.go:22-24: `AppVersion`, `BuildTime`, `GitCommit`). But every build injects into `github.com/CosmoLabs-org/CosmoDev-R2Go2/cmd.*`:
     - Makefile:19 `LDFLAGS=-ldflags "-X github.com/CosmoLabs-org/CosmoDev-R2Go2/cmd.AppVersion=..."`
     - Dockerfile:30-32 (same three `-X` flags)
     - release.yml:62-64 (same, in the tagged-release build)
     Go's `-X` silently ignores non-existent symbols, so every released binary reports `version dev`, `commit unknown`. `cosmoflare --version` lies in all five release artifacts.
   - **Severity**: high
   - **File**: `Makefile:19`, `Dockerfile:30-32`, `.github/workflows/release.yml:62-64`
   - **Fix**: Replace `CosmoDev-R2Go2/cmd` with `cosmoflare/cmd` in all three sites. Define the canonical ldflags string once (a `scripts/version.sh` or a Make variable imported by the workflow) — three copies of the same string is how this drifted.

3. **Release notes publish a dead install path** — release.yml:87 emits `go install github.com/CosmoLabs-org/CosmoDev-R2Go2@${VERSION}` into every GitHub release. The actual repo is `CosmoLabs-org/cosmoflare` (verified via `git remote -v`). Every user who copies the documented install command gets a 404.
   - **Severity**: high
   - **File**: `.github/workflows/release.yml:87`
   - **Fix**: Use `github.com/CosmoLabs-org/cosmoflare@${VERSION}` (and note `go install` needs the module path, which works since `main.go` is at repo root).

4. **Installer downloads from a dead repo with no checksum verification** — install.sh:16 sets `REPO="CosmoLabs-org/CosmoDev-R2Go2"`; the version probe and download URL (install.sh:111-131) both hit that stale repo. Even against the right repo, `download_binary()` does a bare `curl -L -o` with no sha256 check — despite the release workflow publishing `checksums-sha256.txt` (release.yml:70-74) precisely for this. This is a supply-chain gap: a MITM or hijacked release would install unverified code.
   - **Severity**: high
   - **File**: `install.sh:16,111-131`
   - **Fix**: Point `REPO` at `CosmoLabs-org/cosmoflare`, rename the binary to `cosmoflare`, download `checksums-sha256.txt` and verify with `sha256sum -c` (or `shasum -a 256` on macOS) before `chmod +x`.

5. **`make dist` ships 76 MB of internal session transcripts in public archives** — The dist target tars `../docs/` into every release archive (Makefile:95-97). `docs/` is 76 MB, of which `docs/sessions/` is 67 MB and `docs/conversation-transcripts/` is 6.6 MB of full AI-session logs. Anyone running `make release-prepare` and uploading the output publishes internal project management data. The CI release workflow doesn't use this target (it only uploads binaries), so the leak path is the documented Makefile flow.
   - **Severity**: medium
   - **File**: `Makefile:95-97`
   - **Fix**: Ship only `README.md` + `LICENSE` in archives, or a curated `docs/USAGE.md`. Add `docs/sessions/`, `docs/conversation-transcripts/` to a dist-time exclusion list.

6. **No security scanning anywhere in the pipeline** — The three workflows (ci.yml, test.yml, release.yml) run vet, golangci-lint, tests, and a release build. There is no CodeQL, no gosec, no trivy/grype image scan, no `go vet`-adjacent dependency audit (`govulncheck`), and no dependabot.yml or CODEOWNERS (verified absent in `.github/`). golangci-lint is run with `version: latest` (ci.yml:31), so lint results are non-reproducible and a new linter release can break CI overnight. This project wraps the entire Cloudflare API and stores credentials — it is exactly the kind of code that needs `govulncheck` in CI.
   - **Severity**: medium
   - **File**: `.github/workflows/ci.yml:28-32`
   - **Fix**: Pin the golangci-lint version (e.g. `v1.64.x`). Add a `security` job: `golang.org/x/vuln/cmd/govulncheck ./...` + `aquasecurity/trivy-action` on the Docker image (which also forces CI to build the image, fixing finding 1's detection gap).

7. **Duplicate `test-integration` Makefile targets — one dead, one interactive** — The target is defined twice: Makefile:200-204 runs `go test -tags=integration ./pkg/r2go2/` (a package that no longer exists — it's `pkg/cosmoflare/` now), and Makefile:448-457 overrides it with a version that prompts `read -p "Continue? (y/N)"`, which hangs forever in any non-tty/CI context. Both declare `.PHONY`. This is classic rename drift: the project moved from r2go2 → cosmoflare and the Makefile kept both eras.
   - **Severity**: medium
   - **File**: `Makefile:200-204,448-457`
   - **Fix**: Delete the first definition; fix the package path in the survivor; gate the confirmation prompt on `[ -t 0 ]` or move it behind a `CONFIRM=1` flag.

8. **Profile credentials persisted as plaintext YAML; keychain module exists but is not wired to profile storage** — `ProfileConfig` carries `APIToken`, `AccessKey`, `SecretKey` (pkg/cosmoflare/config.go:53-60) and `MachineConfig.Save()` writes them verbatim to `~/.r2go2/config.yaml` (config.go:139-161). The chmod 0600 (config.go:159) is correct, but the values are plaintext on disk and included in any home-dir backup/sync. Meanwhile `internal/keychain/` implements a proper secret store with OS-keychain backend plus tests — it's only consumed by `internal/config/config.go`, not by the profile save path. Note the project memory explicitly forbids triggering macOS keychain probing automatically; the fix should be opt-in (`--secure` flag on `cosmoflare auth login` / account switch), storing only a keychain reference in the YAML.
   - **Severity**: medium
   - **File**: `pkg/cosmoflare/config.go:139-161`
   - **Fix**: Add an opt-in flag on profile creation that writes the token via `internal/keychain` and stores `api_token: keychain://<profile>` in the YAML; keep 0600 plaintext as the default fallback for CI/headless use.

9. **Docker builder image lags the module's Go directive** — Dockerfile:5 uses `golang:1.25-alpine` while go.mod:3 declares `go 1.26` (CI correctly uses `GO_VERSION: "1.26"`, ci.yml:10). The 1.25 toolchain only proceeds by auto-downloading the 1.26 toolchain at build time — non-hermetic, slower, and breaks in air-gapped/locked-down registries. Also `alpine:latest` (Dockerfile:36) as the final base is unpinned, and `GOARCH=amd64` is hardcoded (no arm64 image despite `linux/arm64` being in the Makefile's cross-compile matrix, Makefile:26).
   - **Severity**: medium
   - **File**: `Dockerfile:5,28,36`
   - **Fix**: Use `golang:1.26-alpine` (or derive from the go.mod version in CI), pin `alpine:3.21@sha256:...` by digest, and add a buildx multi-platform job if the image is published.

### Additional observations (not scored as findings)

- **Orchestration absence is mostly justified** — this is a CLI plus a localhost daemon; `cosmoflare serve` binds `127.0.0.1:0` with a random bearer token (cmd/serve.go:60-62), which is the correct secure default. But nothing (compose file, systemd unit, k8s manifest) documents how to run the daemon long-lived, so the first user who does it will invent their own deployment. One example compose file with resource limits and a healthcheck against the daemon's token-protected endpoint would close this.
- **`test.yml` is fully redundant** — its single PR job duplicates the PR-path of ci.yml's lint+test (both trigger on `pull_request` to master). Every PR pays for the same tests twice. Consolidate into ci.yml.
- **CI test job runs the suite twice** — the coverage step (ci.yml:58-61) re-runs all tests without `-race` instead of reusing the race run's profile; coverage output is printed but never uploaded as an artifact or to Codecov.
- **Terraform export feature is solid design** — `cosmoflare terraform export` generates per-service `.tf` files with a pinned provider constraint (`--provider-version` default `~> 4.0`, cmd/terraform.go:83) and import blocks for adoption. Worth bumping the default to `~> 5.0` when the provider major is validated.
- **`cmd/cicd.go.disabled`** (994 lines) — a disabled CI/CD-pipeline generator. Either revive it behind the new module path or delete it; 994 lines of dead code that still references the old identity is drift bait.
- **Repo hygiene**: `git count-objects` reports a 419 MiB pack. `simple-setup` and `test-setup` (4.8 MB binaries each) are still tracked even though `.gitignore` lists them (added before the rule). `GOrchestra/` holds 234 MB of session recovery patches, and the security scanner already flags example AWS keys inside those committed patches. This slows every CI checkout.
- **Docker identity drift**: image is named `r2go2` (Makefile:30), builds binary `r2go2`, sets `R2GO2_CONFIG_DIR`, while the product is `cosmoflare`. The Makefile header, help text, DEB/RPM packaging, and `install-brew` (which misuses `brew install --cask` on a CLI binary, Makefile:119) all still say R2Go2.
- **No `.env.example`** — `.gitignore:145-146` whitelists `!.env.example`, but the file doesn't exist. The project documents `CLOUDFLARE_API_TOKEN`/`CLOUDFLARE_ACCOUNT_ID` as the credential interface; a one-line example file is the expected convention.

### Recommendations

- [ ] Fix `.dockerignore` (`*.mod`/`*.sum` removal) and add a `docker build` CI job so the image is continuously buildable (effort: small)
- [ ] Replace the dead `CosmoDev-R2Go2` module path in ldflags across Makefile, Dockerfile, release.yml, and fix the `go install` line and install.sh `REPO` — single rename-drift sweep (effort: small)
- [ ] Add checksum verification to install.sh and pin the golangci-lint version in ci.yml (effort: small)
- [ ] Add a `security` CI job: `govulncheck` + trivy image scan + dependabot.yml (effort: medium)
- [ ] Remove `docs/` from `make dist` archives; ship README/LICENSE/USAGE.md only (effort: small)
- [ ] Delete the duplicate `test-integration` target, fix its package path, and make its prompt non-interactive-safe (effort: small)
- [ ] Wire `internal/keychain` into profile save behind an opt-in flag; keep plaintext-0600 as headless default (effort: medium)
- [ ] Consolidate test.yml into ci.yml and upload coverage artifacts (effort: small)
- [ ] Rebrand Makefile/Dockerfile/packaging from r2go2 to cosmoflare; drop or fix `install-brew` (effort: medium)
- [ ] Remove tracked 4.8 MB binaries (`simple-setup`, `test-setup`) and consider excluding `GOrchestra/` recovery patches from the public repo (effort: medium)

### Roadmap Suggestions

- **Supply-chain hardening: reproducible, verified releases** — Pin all tool versions in CI, verify installer checksums, add SBOM generation to the release workflow (the `make sbom` target exists but is never run in CI), and sign release artifacts (effort: medium, priority: high)
- **Docker image publishing pipeline** — Multi-arch buildx (amd64+arm64), digest-pinned bases, trivy scan gate, and push to ghcr.io on tags; currently the image exists but is unbuildable and unpublished (effort: medium, priority: medium)
- **Deployment story for `cosmoflare serve`** — Example docker-compose + systemd unit + k8s manifest with resource limits and health checks against the token-protected endpoints, so the daemon has a supported long-running deployment path (effort: medium, priority: low)
- **Rename-debt cleanup: r2go2 → cosmoflare** — One sweep covering Makefile, Dockerfile, DEB/RPM specs, install scripts, `~/.r2go2` config dir (with migration), and the disabled `cicd.go.disabled`; the dead `CosmoDev-R2Go2` path has already caused silent breakage in three build paths (effort: large, priority: high)
