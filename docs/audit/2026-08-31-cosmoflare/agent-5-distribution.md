# Agent 5: Distribution

> Audit: 2026-08-31, cosmoflare v0.17.0, fresh mode. Report saved verbatim from agent output.

I have all the evidence needed. Compiling the final report.

## Distribution: 3/10

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| ci_cd | 3/10 | Well-designed workflows (3-OS matrix, lint, race tests, build verification) but master CI is red on 100% of recorded runs; two distinct root causes unaddressed for months; no desktop CI |
| packaging | 2/10 | Cross-compile matrix for 5 platforms exists and works, but zero release artifacts have ever shipped; all 4 installer scripts point to a dead repo; version stamping broken in every build path |
| deployment | 2/10 | Tag-triggered release workflow is structurally complete (build, checksums, notes, publish) but failed 18/18 runs since 2026-03; 0 GitHub Releases exist in repo history |
| multi_platform | 6/10 | Verified clean cross-compile (CGO off, darwin/arm64 from linux tested exit 0), CI tests on ubuntu/macos/windows; gaps: no package-manager channels, desktop sidecar committed for macOS-arm64 only |
| developer_experience | 5/10 | Strong error messages with troubleshooting tips, 2388-line USAGE.md, 2-command time-to-first-value; but every scripted install channel is broken and shipped binaries would report version "dev" |

### Critical Findings

1. **Release pipeline has never succeeded — zero GitHub Releases exist** — All 18 Release workflow runs since 2026-03-07 failed (`gh run list --workflow=Release`: 18/18 `failure`); `gh api repos/CosmoLabs-org/cosmoflare/releases` returns length 0. Tags v0.10.0–v0.17.0 are pushed but each release aborted at the "Run tests before release" step. The primary distribution channel (binary downloads) has delivered nothing, ever. Hand-written release notes exist for v0.15–v0.17 in `docs/release-notes/` — they document releases that never shipped.
   - **Severity**: high
   - **File**: `.github/workflows/release.yml:31`
   - **Fix**: Fix the test failures (finding 2), then tag v0.18.0 and verify artifacts land.

2. **Data races in `internal/interactive` block both CI and Release** — `go test -race ./internal/interactive/` fails: `TestEncryptDecryptRoundtrip`, `TestEncryptEmptyData`, `TestDecryptWrongPassword`, `TestReadPasswordWithReader` report "race detected during execution of test" (reproduced locally; races only fire across tests, indicating package-level shared state). The Release workflow's test gate (`release.yml:31`) and CI's test jobs (`ci.yml:55`) both run `-race`, so every pipeline run dies here. Master CI is red on all 7 recorded runs back to 2026-05-30.
   - **Severity**: high
   - **File**: `internal/interactive/` (crypto/setup helpers shared across 520 tests, only 6 use `t.Parallel`)
   - **Fix**: Find the package-level mutable state (key cache/terminal state), guard with mutex or move to per-test instances.

3. **Version injection uses a dead module path — every binary reports "dev"** — `go.mod:1` declares `module github.com/CosmoLabs-org/cosmoflare`, but `release.yml:62-64` and `Makefile:19` inject `-X github.com/CosmoLabs-org/CosmoDev-R2Go2/cmd.AppVersion=...` (the pre-rename repo). The linker silently ignores `-X` for nonexistent symbols. Empirically verified: building with the `CosmoDev-R2Go2` path yields `Cosmoflare version dev`; building with the correct `cosmoflare` path yields the injected version. Even a green release pipeline would ship binaries that all claim to be "dev", breaking version support, `--json` version fields, and update checks.
   - **Severity**: high
   - **File**: `.github/workflows/release.yml:62-64`, `Makefile:19`
   - **Fix**: Replace `CosmoDev-R2Go2` with `cosmoflare` in both ldflags strings; add a post-build assertion that `./cosmoflare --version` matches `${VERSION}` in the workflow.

4. **All four installer scripts download from a dead repo** — `install.sh:19` sets `REPO="CosmoLabs-org/CosmoDev-R2Go2"` and `BINARY_NAME="r2go2"`; the download URL at `install.sh:136` becomes `github.com/CosmoLabs-org/CosmoDev-R2Go2/releases/download/v$VERSION/r2go2-$OS-$ARCH`. That repo does not exist, and even if it did, released assets are named `cosmoflare-$OS-$ARCH` (release.yml:54). Same dead-repo bug in `install.ps1:11-12` (Windows users get nothing) and `install-menu.sh:159,178` — which also uses a malformed API URL (`https://api.github.com/CosmoLabs-org/...` missing `/repos/`). Anyone running the documented `curl | bash` install gets a 404.
   - **Severity**: high
   - **File**: `install.sh:19-20,136`, `install.ps1:11-12`, `install-menu.sh:159,178`
   - **Fix**: Point all three scripts at `CosmoLabs-org/cosmoflare`, binary name `cosmoflare`, install as `~/.local/bin/cosmoflare` with an optional `r2go2` symlink.

5. **Release notes instruct users to run a broken install command** — `release.yml:87` generates: `go install github.com/CosmoLabs-org/CosmoDev-R2Go2@${VERSION}` — the wrong module path; the command fails for every user who follows the published notes. (README.md:50 has the correct path, making this an internal inconsistency.)
   - **Severity**: medium
   - **File**: `.github/workflows/release.yml:87`
   - **Fix**: Change to `go install github.com/CosmoLabs-org/cosmoflare@${VERSION}`.

6. **golangci-lint job cannot lint Go 1.26 — config error on every run** — `ci.yml:28-32` uses `golangci-lint-action@v6` with `version: latest`, which resolves v1.64.8 (built with go1.24). CI log: `Error: can't load config: the Go language version (go1.24) used to build golangci-lint is lower than the targeted Go version (1.26)`, exit 3. The repo has no `.golangci.yml`, so the "latest" action version is the only control — and it is not actually latest.
   - **Severity**: medium
   - **File**: `.github/workflows/ci.yml:28-32`
   - **Fix**: Pin `version: v2.x` (golangci-lint v2 supports Go 1.26) or install via `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`; commit a minimal `.golangci.yml`.

7. **Dockerfile is orphaned and broken** — `Dockerfile:6` builds `FROM golang:1.25-alpine` but `go.mod:3` requires `go 1.26`, so the image build fails with "go.mod requires go >= 1.26". No CI job, Makefile target, or publish workflow references Docker at all, despite `DOCKER_IMAGE`/`DOCKER_TAG` variables defined at `Makefile:27-28`.
   - **Severity**: medium
   - **File**: `Dockerfile:6`
   - **Fix**: Bump to `golang:1.26-alpine`, add `make docker` target, publish to GHCR in the release workflow.

8. **Desktop tier is unshippable beyond macOS-arm64 and drifts one version behind** — `desktop/src-tauri/tauri.conf.json:4` pins `0.16.0` while the CLI is 0.17.0 (`.version-registry.json:7`). The sidecar `desktop/src-tauri/binaries/cosmoflare-aarch64-apple-darwin` (25.7MB) is committed for one platform only — `targets: "all"` means Windows/Linux bundles will fail at missing sidecar triplets. No updater/signing config, no desktop CI, no desktop release workflow.
   - **Severity**: medium
   - **File**: `desktop/src-tauri/tauri.conf.json:4`, `desktop/src-tauri/binaries/`
   - **Fix**: Build sidecars for all 3 targets in CI (never commit), derive desktop version from `.version-registry.json`, add a desktop release job with signing.

9. **Makefile still packages the dead R2Go2 product** — `BINARY_NAME=r2go2` throughout: `make install` installs `r2go2` (not `cosmoflare`), the RPM spec URL points at the dead repo (`Makefile:155`), the deb package is named `r2go2`, and `make install-brew` runs `brew install --cask build/r2go2` (`Makefile:115-122`) which is invalid cask usage and always fails. `make test-integration` (`Makefile:203`) targets `./pkg/r2go2/`, a directory that no longer exists. `make dist` embeds `../docs/` (525 doc files, including internal session transcripts) into user-facing tarballs.
   - **Severity**: medium
   - **File**: `Makefile:14,115-122,155,203`
   - **Fix**: Rename to `cosmoflare` primary binary, delete or repair `install-brew`, fix the integration path, ship only README/LICENSE/USAGE.md in archives.

10. **Repo carries committed binaries and 419 MiB of history** — `simple-setup` and `test-setup` (4.8MB each, Nov 2025) are git-tracked; the working tree holds `r2go2` (34MB), `cmd.test` (40MB), `r2go2-enhanced` (16MB), `r2go2-tui-installer` (12MB) untracked; pack size is 418.90 MiB. `git clone` cost for an MIT community CLI is a distribution disincentive.
    - **Severity**: low
    - **File**: repo root (`simple-setup`, `test-setup` tracked)
    - **Fix**: Remove tracked binaries, add build outputs to `.gitignore`, consider history cleanup before wider adoption.

11. **Public library errors carry stale `r2go2:` branding** — `pkg/cosmoflare/errors.go:12-19` renders `r2go2: %s: bucket=%s ...` for every error in the package imported as `cosmoflare`. Agents and library consumers see a different product name than the one they installed.
    - **Severity**: low
    - **File**: `pkg/cosmoflare/errors.go:12-19`
    - **Fix**: Change prefix to `cosmoflare:` (breaking for string matchers — do it before 1.0).

### Recommendations

- [ ] Fix data races in `internal/interactive` (run `go test -race ./internal/interactive/` locally, isolate the shared package state, add mutex or per-test instances) (effort: medium)
- [ ] Correct the ldflags module path in `release.yml:62-64` and `Makefile:19` from `CosmoDev-R2Go2` to `cosmoflare`; add a `./cosmoflare --version` assertion step to the release workflow (effort: small)
- [ ] Rewrite `install.sh`, `install.ps1`, `install-menu.sh` to target `CosmoLabs-org/cosmoflare` with `cosmoflare` binary names; test each on a clean machine (effort: small)
- [ ] Pin golangci-lint to a v2 release in `ci.yml` and commit `.golangci.yml`; delete the redundant `test.yml` (duplicates ci.yml's PR path) (effort: small)
- [ ] Adopt GoReleaser (`.goreleaser.yml`) to replace the hand-rolled cross-compile loop — gains archives, Homebrew tap autoupdate, SBOM, and snapshot builds for free (effort: medium)
- [ ] Publish a Homebrew tap (`homebrew-tap` repo with formula) and a Scoop manifest — the two highest-leverage channels for a Go CLI (effort: medium)
- [ ] Fix the Dockerfile Go version and publish a GHCR image in the release workflow (effort: small)
- [ ] Replace the committed desktop sidecar with a CI build step producing all three platform sidecars; sync desktop version to `.version-registry.json` (effort: medium)
- [ ] Add a release smoke check: after `softprops/action-gh-release`, run `gh release download` + `sha256sum -c` + `--version` verification so a silent empty release can never happen again (effort: small)

### Roadmap Suggestions

- **Restore the binary release channel** — Fix races, ldflags path, and installer scripts; ship v0.18.0 with verified checksums and a version-assertion gate. The project currently has no working binary distribution at all. (priority: high, effort: medium)
- **Homebrew + Scoop distribution** — GoReleaser-driven tap and bucket manifests so macOS/Windows users install without a Go toolchain. (priority: high, effort: medium)
- **Desktop (Tauri) release pipeline** — CI-built sidecars for all targets, code signing, updater feed, version sync with CLI registry. Required for the paid tier of the 3-tier product vision. (priority: medium, effort: large)
- **Linux package channels** — Publish the existing deb/rpm Makefile targets to a PPA/repo or attach them to GitHub Releases with install docs. (priority: medium, effort: medium)
- **Repo slimming** — Remove committed binaries and stop tracking build outputs; keep clone size competitive for an OSS CLI. (priority: low, effort: small)
