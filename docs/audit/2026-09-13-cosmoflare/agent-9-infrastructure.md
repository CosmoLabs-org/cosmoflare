# Agent 9 — Infrastructure Audit: cosmoflare v0.26.0

**Context**: Go CLI + Tauri desktop app managing Cloudflare. GitHub Actions CI is **permanently disabled by policy** (`.goreleaser.yaml:1-3`: "CI is permanently disabled on this repo (all workflows disabled_manually) — releases are ALWAYS cut from the maintainer machine"). This audit evaluates infrastructure against that reality: the local-machine release pipeline IS the deployment pipeline.

**Files read**: 22+ (Dockerfile, .dockerignore, all 3 workflows, .goreleaser.yaml, Makefile, cmd/terraform.go, pkg/cosmoflare/terraform.go, cmd/auth.go, internal/config/config.go, internal/keychain/keychain.go, internal/server/server.go, cmd/serve.go, cmd/root.go, install.sh, desktop/src-tauri/tauri.conf.json, desktop/scripts/build-sidecar.sh, cmd/installer_tui/main.go, pkg/cosmoflare/config.go, pkg/cosmoflare/export.go, tests/fixtures/testdata.go, .gitignore, SECURITY.md, transcript spot-checks).

## Infrastructure: 6/10

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| containerization | 5/10 | Multi-stage + non-root + static build done right, but toolchain drift (go 1.25 vs go.mod 1.26), unpinned `alpine:latest`, ~1.4GB build context from incomplete .dockerignore, stale `r2go2` identity, image not in release flow at all |
| orchestration | 4/10 | No k8s/helm/compose (N/A for CLI+desktop product). The daemon lifecycle that does exist (handshake, healthz, graceful shutdown, ctx-cancelled loops) is well built; Docker image has no run story (no HEALTHCHECK, CMD `--help`) |
| iac_quality | 7/10 | `cosmoflare terraform export/import-block` is a genuinely good IaC bridge: token as `sensitive` variable (never hardcoded), import blocks with `-generate-config-out` guidance, name sanitization, dry-run + JSON + tests. Docked for stale `~> 4.0` provider default (provider is at 5.x) and no state/backend guidance |
| security | 6/10 | Strong: keychain-first secrets with sentinel replacement + migration, 0600 atomic config writes, fail-closed SHA256 installer, SBOM, masked output, localhost daemon with 128-bit token auth. Weak: `auth rotate` is a shipped placeholder that always errors (plus a latent revoke-NEW-token bug), Tauri `csp: null`, plaintext fallback at rest, non-constant-time token compare, 93 committed conversation transcripts + GOrchestra artifacts generating scanner noise |

---

### Critical Findings

1. **Docker build toolchain drift — `golang:1.25-alpine` cannot build a `go 1.26` module deterministically** — The Dockerfile builder is `FROM golang:1.25-alpine` (Dockerfile:5) while `go.mod:3` declares `go 1.26` and the (dead) workflows set `GO_VERSION: "1.26"`. With Go's default `GOTOOLCHAIN=auto`, `make docker-build` (Makefile:336-340) silently downloads the 1.26 toolchain mid-build — nondeterministic, network-dependent, and it breaks outright under `GOTOOLCHAIN=local`. Nobody notices because the image is never built in the release flow (goreleaser ships bare binaries; `.goreleaser.yaml:61-62` has `release: disable: true` and no docker section).
   - **Severity**: high
   - **File**: `Dockerfile:5`, `go.mod:3`
   - **Fix**: Pin `FROM golang:1.26-alpine AS builder`, or better `FROM golang:1.26.X-alpine@sha256:...`. Decide whether the image ships at all (see Recommendation 1).

2. **`auth rotate` is a shipped placeholder that always fails, and its `--revoke-old` path would revoke the NEW token** — `generateNewToken` and `revokeOldToken` are stubs that unconditionally return errors (cmd/auth.go:479-491: "automatic token generation not implemented"). Every `cosmoflare auth rotate --profile=X` run dies at "Generating new API token...". Worse, there is a latent ordering bug: line 227 overwrites `profile.APIToken` with the new token, then line 239 calls `revokeOldToken(profile.APIToken)` — the argument is the NEW token. If rotation is ever implemented, `--revoke-old` would revoke the just-issued credential and leave the old one live. The command is advertised in help text and `auth status` output as a real security feature ("Rotate API tokens for enhanced security").
   - **Severity**: high
   - **File**: `cmd/auth.go:227-244`, `cmd/auth.go:479-491`
   - **Fix**: Capture `oldToken := profile.APIToken` before line 227 and pass that to `revokeOldToken`. Implement rotation via the Cloudflare API Token create/roll endpoints, or remove/hide the subcommand until it works.

3. **`.dockerignore` is incomplete — `COPY . .` ships a ~1.4GB build context** — `.dockerignore` excludes `docs/`, `build/`, `dist/`, `*_test.go` — but NOT `desktop/` (1.3GB, includes node_modules), `GOrchestra/` (7.3MB of committed session logs), or the stale root binaries `r2go2` (33MB), `cmd.test` (39MB), `r2go2-enhanced` (15MB), `r2go2-tui-installer` (12MB), `simple-setup`/`test-setup` (9.2MB). The file also carries duplicate entries (`.DS_Store` at lines 20 and 31, `.vscode/`/`.idea/` at 21-22 and 41-42) — a sign it has never been exercised. Measured with `du`: the unexcluded paths total ~1.4GB vs the ~5MB of source actually needed.
   - **Severity**: medium
   - **File**: `.dockerignore:1-50`
   - **Fix**: Add `desktop/`, `GOrchestra/`, `ClaudeDesktop/`, `ClaudeDesign/`, `r2go2*`, `cmd.test`, `*-setup`, `*.exe`, `.ccs*`, `plugins/`, dedupe entries.

4. **Dead CI workflows describe a pipeline that never runs — and encode a divergent release path** — `.github/workflows/ci.yml`, `test.yml`, and `release.yml` (3,700 bytes of cross-compile + release logic) are committed but permanently disabled. `release.yml:39-66` cross-compiles `cosmoflare-${OS}-${ARCH}` binaries and publishes via `softprops/action-gh-release` — a second, independent implementation of what `.goreleaser.yaml` + manual `gh release create` do locally. The two have already diverged historically (`.goreleaser.yaml:10-25` documents the v0.22.0 jq-asset-name failure that the workflow path does not handle: workflow uploads bare per-OS binaries; goreleaser docs warn `artifacts.json .name` collides). Neither path contains security scanning (no govulncheck, gosec, trivy, or CodeQL anywhere). Risk: anyone re-enabling Actions (or a future repo transfer) ships through an untested path with different asset semantics.
   - **Severity**: medium
   - **File**: `.github/workflows/release.yml:39-66`, `.github/workflows/ci.yml:1-92`
   - **Fix**: Either delete the workflows and keep a one-line README in `.github/` stating the no-CI policy + pointer to the local SOP, or strip them to release-notes-only. Add `govulncheck ./...` to `make release-prepare` so the one pipeline that runs has a vulnerability gate.

5. **Tauri desktop webview has no Content-Security-Policy (`"csp": null`)** — `desktop/src-tauri/tauri.conf.json:19-21` sets `"security": { "csp": null }`. The webview renders local React against a localhost daemon; a null CSP means no restriction on script/source loading if any view ever renders remote or user-influenced content (R2 object previews, error pages, notification bodies from webhook payloads). Tauri's own guidance is to always set a CSP.
   - **Severity**: medium
   - **File**: `desktop/src-tauri/tauri.conf.json:19-21`
   - **Fix**: Set an explicit CSP, e.g. `default-src 'self'; connect-src 'self' http://127.0.0.1:*; img-src 'self' data:; style-src 'self' 'unsafe-inline'`.

6. **Unpinned runtime base image** — Final stage is `FROM alpine:latest` (Dockerfile:36). `latest` is not reproducible and has no vulnerability floor; two builds a month apart get different base OSes.
   - **Severity**: medium
   - **File**: `Dockerfile:36`
   - **Fix**: Pin `alpine:3.21` (or digest). Optionally add a periodic base-image rebuild note to the release SOP.

7. **Stale `r2go2` identity across the container/Makefile surface** — The Dockerfile builds `-o r2go2` (Dockerfile:28-33), sets `ENV R2GO2_CONFIG_DIR` (:62), labels the image "R2Go2" (:71-77); the Makefile sets `DOCKER_IMAGE=r2go2` (Makefile:30) and titles itself "R2Go2 - Cloudflare R2 CLI Tool" (:1); `SECURITY.md` still points at the dead repo `CosmoLabs-org/CosmoDev-R2Go2` and instructs `./r2go2 --version`. Releases since the rename ship `cosmoflare-*` assets. A user following SECURITY.md hits a 404 repo; an operator running `make docker-build` gets an image whose binary answers to the alias, not the documented name.
   - **Severity**: low
   - **File**: `Dockerfile:28-77`, `Makefile:14,30`, `SECURITY.md:26-30`
   - **Fix**: One rename sweep: build `-o cosmoflare` (+ keep an `r2go2` symlink if the alias contract requires it), `COSMOFLARE_CONFIG_DIR`, `DOCKER_IMAGE=cosmoflare`, fix SECURITY.md URLs.

8. **Daemon token comparison is not constant-time** — `internal/server/server.go:148-151` compares the bearer token with `==` (twice: header and query forms). On localhost the timing side-channel is largely theoretical, but this is the auth gate for every endpoint including the credentials-touching REST surface, and the fix is one line.
   - **Severity**: low
   - **File**: `internal/server/server.go:148-151`
   - **Fix**: `subtle.ConstantTimeCompare([]byte(presented), []byte(s.cfg.Token)) == 1` (guard lengths first or accept the length leak, which is negligible for 128-bit hex).

9. **API token prompted with terminal echo** — `promptForAPIToken` (cmd/auth.go:417-422) reads the Cloudflare API token via `bufio.ReadString` with full echo: the secret lands on screen and in terminal scrollback. Also `--api-token` (cmd/root.go:136) and `--token` (cmd/auth.go:128) place the token in shell history and `ps` output. Standard CLI practice is silent read (`golang.org/x/term.ReadPassword`).
   - **Severity**: low
   - **File**: `cmd/auth.go:417-422`, `cmd/root.go:136`
   - **Fix**: Use `term.ReadPassword(int(os.Stdout.Fd()))` in the prompt path; document env-var/`--token-file`/stdin as the history-safe alternatives.

10. **93 committed conversation transcripts + GOrchestra session artifacts turn the security scanner into noise** — `git ls-files docs/conversation-transcripts/ | wc -l` = 93; GOrchestra/ holds committed agent `session.log`/`output.log` pairs (11K+ lines each) and recovery patches. All 13 "critical" AWS-key findings in the pre-computed scan (`_ctx/security.json`) trace to the AWS-documentation placeholder `AKIAIOSFODNN7EXAMPLE` — including in committed transcripts (`docs/conversation-transcripts/2026-05-16_024358_2cffd242.md:12110`) and fixtures (`tests/fixtures/testdata.go:17,27`, `internal/interactive/backup_restore_test.go:47`). None are real credentials, but a scanner that cries wolf 538 times guarantees a real leak gets skimmed past, and prior audit reports carrying the flagged strings are themselves committed, recursively feeding the scanner.
    - **Severity**: low
    - **File**: `docs/conversation-transcripts/` (93 files), `GOrchestra/`, `tests/fixtures/testdata.go:17`
    - **Fix**: Move transcripts/agent artifacts out of git (or a `docs/archive/` branch), and add an allowlist entry for `AKIAIOSFODNN7EXAMPLE` in the scanner config with a comment that it is the AWS docs placeholder.

11. **Plaintext secrets fallback at rest when the OS keychain is unavailable** — `storeSecrets` (internal/config/config.go:381-395) only moves `api_token`/`access_key`/`secret_key` into the keychain `if cm.secrets.Available()`. On headless Linux (no SecretService) or Windows without a backend, tokens persist in cleartext in `~/.cosmoflare/config.yaml` (0600, atomic write — good, but plaintext). This is a deliberate, conventional tradeoff (same posture as the AWS CLI), documented only in code.
    - **Severity**: low
    - **File**: `internal/config/config.go:381-395`, `internal/keychain/keychain.go:40-44`
    - **Fix**: Acceptable as-is; print a one-time warning on `auth login` when falling back to file storage ("token stored unencrypted at ~/.cosmoflare/config.yaml (no OS keychain)").

12. **Terraform provider version default is stale** — `terraformExportCmd` defaults `--provider-version "~> 4.0"` (cmd/terraform.go:84) while the command's own help advertises `"~> 5.0"` (line 35) and the Cloudflare provider has been 5.x for a long time. Generated `provider.tf` for a user who doesn't override pins them to a 4.x provider line.
    - **Severity**: low
    - **File**: `cmd/terraform.go:84`
    - **Fix**: Change default to `"~> 5.0"` (and add a test asserting the default matches the documented example).

13. **Raw `cmd.Start()` in installer TUI (ROAD-525 lint finding)** — `cmd/installer_tui/main.go:1280`: `_ = cmd.Start() // Fire and forget` for the browser-open helper. Benign for a URL opener, but the child is never `Wait()`ed (zombie until process exit) and it violates the project's own spawn policy (`daemon.SpawnAsync()`).
    - **Severity**: low
    - **File**: `cmd/installer_tui/main.go:1280`
    - **Fix**: `go func() { _ = cmd.Run() }()` or route through the project's daemon helper.

### Verified non-findings (evidence the scanner got wrong or things done right)

- **No hardcoded secrets in generated IaC** — `GenerateProviderHCL` (pkg/cosmoflare/terraform.go:398-423) emits `variable "cloudflare_api_token" { sensitive = true }` and `api_token = var.cloudflare_api_token`; account IDs are variables too. Export payloads (`pkg/cosmoflare/export.go:24-72`) contain resource data only — no token fields.
- **Installer is fail-closed** — `install.sh:128-200` downloads `checksums-sha256.txt`, refuses to install on missing file/missing entry/mismatch/no-tool ("Refusing to install an unverified binary (fail-closed policy)"), and verifies **before** the binary is ever chmod +x or executed.
- **Daemon transport security is properly scoped** — `serve` defaults to `127.0.0.1:0` (cmd/serve.go:66), generates a 128-bit `crypto/rand` token (:170-177), requires bearer auth on every endpoint including `/healthz` (internal/server/server.go:132-140), forces `COSMOFLARE_NO_KEYCHAIN=1` (:80), and the `?token=` query fallback is documented with its threat model (server.go:143-151).
- **Config writes are atomic and 0600** — tmp-file + chmod + rename (internal/config/config.go:122-142); keychain secrets are stored under sentinels with a `MigrateToKeychain` path (:410-432) and `testing.Testing()` isolation (internal/keychain/keychain.go:40-44) — the 2026-06-20 keychain-flood class of bug is engineered out.
- **`.cosmoflare.yaml` (project config) is credential-free by design** — `ProjectConfig` (pkg/cosmoflare/config.go:12-24) holds bucket/cache/guardrails/audit/mcp only; credentials live exclusively in the machine config + keychain. Committing it is safe.
- **SBOM + checksums ship with releases** — `make release-prepare` (Makefile:350) chains `clean deps test build-all dist checksums sbom`; sbom target emits CycloneDX + SPDX via syft (Makefile:170-180).
- **Sidecar cross-compile is disciplined** — `desktop/scripts/build-sidecar.sh`: `set -euo pipefail`, CGO disabled, `-trimpath -ldflags "-s -w"`, per-triple outputs wired as Tauri `beforeBuildCommand`.
- **Dead `.env.example` whitelist** — `.gitignore:136-137` has `!.env.example` but no such file exists anywhere; harmless, but either add the example (documenting `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID`, `R2_ENDPOINT`, AWS vars) or drop the whitelist.

### Recommendations

- [ ] Decide the Docker story: either wire `goreleaser`'s docker section to publish a scanned multi-arch image per release, or delete Dockerfile + Makefile docker targets — the current half-state rots silently (effort: medium)
- [ ] Fix the release-prepare pipeline gate: add `govulncheck ./...` (and optionally `gosec`) to `make release-prepare` so the only pipeline that runs has a vulnerability check (effort: small)
- [ ] Repair or remove `auth rotate`: implement Cloudflare token roll via API, fix the revoke-old ordering bug at cmd/auth.go:227-239 (effort: medium)
- [ ] Dockerfile hygiene pass: `golang:1.26-alpine`, pinned alpine, `-o cosmoflare`, HEALTHCHECK for `serve`, dedupe/extend .dockerignore (effort: small)
- [ ] Delete or neuter the three dead workflow files; replace with a policy pointer so the repo stops advertising a pipeline that cannot run (effort: small)
- [ ] Set a Tauri CSP in `desktop/src-tauri/tauri.conf.json` (effort: small)
- [ ] Silent token prompt via `x/term.ReadPassword`; add constant-time compare in server auth (effort: small)
- [ ] Bump `--provider-version` default to `~> 5.0` (effort: small)
- [ ] Create `.env.example` documenting the credential env vars (effort: small)
- [ ] Update SECURITY.md repo URLs + binary names; rename sweep in Makefile headers (effort: small)

### Roadmap Suggestions

- **Release supply-chain hardening** — cosign-sign release assets, govulncheck gate in release-prepare, scanner allowlist for AWS docs placeholders so real findings surface (priority: high, effort: medium)
- **Real credential lifecycle** — implement `auth rotate`/`revoke` against the Cloudflare API Tokens API with expiry warnings in `auth status` (priority: medium, effort: medium)
- **Container image or bust** — publish a maintained multi-arch image via goreleaser (with trivy scan + digest pinning) or remove the container surface entirely (priority: medium, effort: medium)
- **Repo de-noising** — move conversation transcripts and GOrchestra agent artifacts out of the tracked tree (coordinate with the 2026-09-07 history-purge constraints and sha-map) (priority: low, effort: medium)

```json:audit-result
{
  "agent": "infrastructure",
  "overall_score": 6,
  "sub_scores": {
    "containerization": 5,
    "orchestration": 4,
    "iac_quality": 7,
    "security": 6
  },
  "critical_findings": [
    {
      "title": "Docker builder toolchain drift: golang:1.25-alpine vs go.mod go 1.26",
      "severity": "high",
      "file": "Dockerfile:5",
      "fix": "Pin FROM golang:1.26-alpine (digest-pinned ideally); decide whether the image ships at all since goreleaser releases bare binaries only",
      "effort": "small"
    },
    {
      "title": "auth rotate is a shipped placeholder that always errors; --revoke-old would revoke the NEW token",
      "severity": "high",
      "file": "cmd/auth.go:479-491",
      "fix": "Implement rotation via Cloudflare API Token endpoints or remove the subcommand; capture oldToken before overwriting profile.APIToken (cmd/auth.go:227-239)",
      "effort": "medium"
    },
    {
      "title": ".dockerignore incomplete: COPY . . sends ~1.4GB build context (desktop/node_modules, 105MB stale binaries)",
      "severity": "medium",
      "file": ".dockerignore:1-50",
      "fix": "Exclude desktop/, GOrchestra/, r2go2*, cmd.test, *-setup, ClaudeDesktop/, ClaudeDesign/, plugins/; dedupe duplicate entries",
      "effort": "small"
    },
    {
      "title": "Dead CI workflows encode a divergent, untested release path with no security scanning",
      "severity": "medium",
      "file": ".github/workflows/release.yml:39-66",
      "fix": "Delete workflows (CI permanently disabled by policy) or reduce to notes-only; add govulncheck to make release-prepare so the live pipeline has a vulnerability gate",
      "effort": "small"
    },
    {
      "title": "Tauri desktop webview has no Content-Security-Policy (csp: null)",
      "severity": "medium",
      "file": "desktop/src-tauri/tauri.conf.json:19-21",
      "fix": "Set explicit CSP: default-src 'self'; connect-src 'self' http://127.0.0.1:*; img-src 'self' data:",
      "effort": "small"
    },
    {
      "title": "Unpinned runtime base image alpine:latest",
      "severity": "medium",
      "file": "Dockerfile:36",
      "fix": "Pin alpine:3.x or digest",
      "effort": "small"
    },
    {
      "title": "Non-constant-time daemon token comparison",
      "severity": "low",
      "file": "internal/server/server.go:148-151",
      "fix": "Use subtle.ConstantTimeCompare for the bearer and query token checks",
      "effort": "small"
    },
    {
      "title": "API token prompted with terminal echo; flags leak to shell history/ps",
      "severity": "low",
      "file": "cmd/auth.go:417-422",
      "fix": "Use golang.org/x/term.ReadPassword for silent input; document env-var as the history-safe path",
      "effort": "small"
    },
    {
      "title": "93 committed transcripts + GOrchestra artifacts make the secret scanner 538-finding noise",
      "severity": "low",
      "file": "docs/conversation-transcripts/",
      "fix": "Move transcripts/agent logs out of the tracked tree; allowlist AKIAIOSFODNN7EXAMPLE (AWS docs placeholder) in scanner config",
      "effort": "medium"
    },
    {
      "title": "Stale r2go2 identity across Dockerfile, Makefile, SECURITY.md (dead repo URL CosmoDev-R2Go2)",
      "severity": "low",
      "file": "Makefile:30",
      "fix": "Rename sweep to cosmoflare; fix SECURITY.md URLs and binary names",
      "effort": "small"
    },
    {
      "title": "Terraform export provider version default stale (~> 4.0 vs provider 5.x)",
      "severity": "low",
      "file": "cmd/terraform.go:84",
      "fix": "Default to ~> 5.0, matching the command's own help example",
      "effort": "small"
    },
    {
      "title": "Plaintext secrets fallback when OS keychain unavailable (no warning)",
      "severity": "low",
      "file": "internal/config/config.go:381-395",
      "fix": "Print one-time warning on auth login when storing unencrypted to config.yaml",
      "effort": "small"
    },
    {
      "title": "Raw cmd.Start() in installer TUI violates project spawn policy (ROAD-525)",
      "severity": "low",
      "file": "cmd/installer_tui/main.go:1280",
      "fix": "Use cmd.Run() in a goroutine or daemon.SpawnAsync()",
      "effort": "small"
    }
  ],
  "recommendations": [
    {
      "action": "Decide the Docker story: publish a scanned multi-arch image via goreleaser docker section, or delete the container surface",
      "effort": "medium",
      "priority": "medium"
    },
    {
      "action": "Add govulncheck (and optionally gosec) to make release-prepare — the only pipeline that runs",
      "effort": "small",
      "priority": "high"
    },
    {
      "action": "Implement real auth rotate via Cloudflare API and fix the revoke-old token ordering bug",
      "effort": "medium",
      "priority": "medium"
    },
    {
      "action": "Dockerfile hygiene: golang:1.26-alpine, pinned alpine, -o cosmoflare, HEALTHCHECK, extended .dockerignore",
      "effort": "small",
      "priority": "medium"
    },
    {
      "action": "Delete or neuter dead workflow files; leave a policy pointer",
      "effort": "small",
      "priority": "low"
    },
    {
      "action": "Set Tauri CSP; constant-time token compare; silent token prompt",
      "effort": "small",
      "priority": "medium"
    },
    {
      "action": "Create .env.example (gitignore whitelist already expects it)",
      "effort": "small",
      "priority": "low"
    },
    {
      "action": "Bump terraform --provider-version default to ~> 5.0",
      "effort": "small",
      "priority": "low"
    }
  ],
  "roadmap_suggestions": [
    {
      "title": "Release supply-chain hardening",
      "description": "cosign-sign release assets, govulncheck gate in release-prepare, scanner allowlist for AWS docs placeholders so real findings surface",
      "priority": "high",
      "effort": "medium"
    },
    {
      "title": "Real credential lifecycle",
      "description": "Implement auth rotate/revoke against the Cloudflare API Tokens API with expiry warnings in auth status",
      "priority": "medium",
      "effort": "medium"
    },
    {
      "title": "Container image or bust",
      "description": "Either publish a maintained multi-arch image via goreleaser with trivy scanning and digest pinning, or remove the Dockerfile and Makefile docker targets entirely",
      "priority": "medium",
      "effort": "medium"
    },
    {
      "title": "Repo de-noising",
      "description": "Move conversation transcripts and GOrchestra agent artifacts out of the tracked tree; coordinate with history-purge constraints",
      "priority": "low",
      "effort": "medium"
    }
  ]
}
```
