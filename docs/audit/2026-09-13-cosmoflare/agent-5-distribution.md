# Agent 5: Distribution — 5/10

Audit date: 2026-09-13. Scope: CI/CD, packaging, release process, multi-platform coverage, CLI developer experience (absorbed from retired Agent 3). Policy context honored: GitHub Actions are permanently disabled by project policy; releases are cut from the maintainer machine via goreleaser + `gh release create`. CI scores are therefore judged against the *local* release discipline, not against "does GitHub Actions exist".

**Sub-scores**:

| Dimension | Score | Notes |
|-----------|-------|-------|
| ci_cd | 4/10 | No automated pipeline anywhere (policy-decided), local gates (ccs smoke, verify-worktree, smokesig) partially compensate; dead workflow files remain in-repo and contradict the canonical flow |
| packaging | 5/10 | Solid goreleaser v2 config (5 targets, ldflags stamping, sha256 checksums, fail-closed installers) but no package-manager channels, no SBOM/signing in the real path, and a second divergent Makefile pipeline that would ship secrets |
| deployment | 4/10 | Release runbook exists and is battle-documented, but the final publish step silently failed for 4 consecutive releases (v0.23.0–v0.26.0 tagged+pushed, no GitHub release objects); no guard detects this |
| multi_platform | 6/10 | CLI covers darwin/linux amd64+arm64 and windows/amd64 with CGO-free static builds and OS-specific installers; gaps: windows/arm64, linux/armv7 (installer accepts it, nothing builds it), desktop sidecar lacks linux/arm64, mobile tier not started |
| developer_experience | 6/10 | Fast time-to-first-value (~3 commands), actionable errors with fix hints, --json everywhere, 2923-line USAGE.md; dragged down by broken install doc URL, R2Go2/cosmoflare branding split, and alias not shipped by installer |

### Critical Findings

1. **Four consecutive tagged releases were never published to GitHub** — Tags v0.23.0, v0.24.0, v0.25.0, v0.26.0 are pushed to origin (verified via `git ls-remote`), but `gh api repos/CosmoLabs-org/cosmoflare/releases/tags/<tag>` returns 404 for all four — not even drafts. Latest published release is v0.22.0 (2026-09-08). Consequence: version split-brain. `install.sh` resolves "latest release" via the GitHub API (install.sh:112) and installs v0.22.0, while `go install ...@latest` resolves the tag and builds v0.26.0. Binary users are 4 minor versions behind and have no way to know. `dist/upload/` currently holds the complete staged v0.26.0 asset set (5 binaries + checksums-sha256.txt) that was never uploaded. The release process has no completion guard, so this failed silently four times.
   - **Severity**: high
   - **File**: `.goreleaser.yaml:22` (the `gh release create` runbook step that was skipped)
   - **Fix**: Publish the staged assets for v0.26.0 (`gh release create v0.26.0 dist/checksums-sha256.txt dist/upload/* --repo CosmoLabs-org/cosmoflare --notes-file docs/release-notes/cosmoflare-v0.26.0-ReleaseNotes-features-and-improvements.md --latest`); decide backfill for v0.23–v0.25 (rebuild from tags or document the gap). Then add a post-release smoke check: after tag push, assert `gh release view <tag>` succeeds.

2. **`make dist` tars the entire `docs/` tree into release archives — including files containing real credentials** — Makefile:95-98 bundles `../docs/` into every platform tarball/zip. `docs/` contains conversation transcripts that the 2026-09-13 security scan flags for embedded AWS Access Key IDs (e.g. `docs/conversation-transcripts/2026-05-16_024358_2cffd242.md:12110`), GOrchestra agent session logs, and internal audit reports. Anyone following `make help`'s "Release: release-prepare" path instead of the goreleaser runbook publishes secrets in the artifact. The Makefile release path and the goreleaser path are both live and advertised; nothing marks one canonical.
   - **Severity**: high
   - **File**: `Makefile:97` (`tar -czf $(BINARY_NAME)-$(VERSION)-$$os-$$arch.tar.gz $$output_name ../README.md ../LICENSE ../docs/`)
   - **Fix**: Remove `../docs/` from the dist target immediately (a binary needs at most README + LICENSE). Longer term, delete the Makefile dist/checksums/release-prepare path entirely — goreleaser is the real pipeline — or gate it behind a `ci-disabled-legacy` comment block.

3. **GETTING_STARTED.md's headline install one-liner points at a dead repository** — Line 15: `curl -fsSL https://raw.githubusercontent.com/CosmoLabs-org/CosmoDev-R2Go2/main/install.sh | bash` (and line 20 for install.ps1). The repo was renamed to `cosmoflare` (and history was purged 2026-09-07); CosmoDev-R2Go2 no longer exists at that path, so every new user following the onboarding doc gets a 404 from `curl -fsSL` (which, unlike plain `-f`-less curl, at least exits non-zero). The working installers (`install.sh`, `install.ps1`) in this repo are never linked from README.md either — its Install section offers only `go install` and source builds.
   - **Severity**: high
   - **File**: `GETTING_STARTED.md:15`
   - **Fix**: Point both URLs at `CosmoLabs-org/cosmoflare/main/install.sh`; add the one-liner (and the PowerShell equivalent) to README.md's Install section; consider archiving GETTING_STARTED.md if superseded by USAGE.md.

4. **Dead CI workflows remain armed and contradict the canonical release flow** — `.github/workflows/ci.yml`, `release.yml`, `test.yml` are tracked but permanently disabled at the GitHub level. If ever re-enabled, `release.yml` would race the local process and produce a *different* release: it cross-compiles with `AppVersion=${GITHUB_REF_NAME}` (release.yml:37,62) which embeds `v0.26.0` — the goreleaser flow embeds `0.26.0` (`.goreleaser.yaml:47` uses `{{ .Version }}`) — so binaries from the two paths report different version strings for the same tag. ci.yml/test.yml also signal to outside contributors that PRs get tested, which is false. Keeping dead-but-correct-looking automation is a re-enable accident waiting to happen.
   - **Severity**: medium
   - **File**: `.github/workflows/release.yml:37`
   - **Fix**: Delete the three workflow files (policy is permanent), or replace each with a one-line stub: "CI intentionally disabled — releases are cut locally, see .goreleaser.yaml header." Document the policy in README's Development section.

5. **Two divergent release pipelines with different binary names, asset names, and platform sets** — The Makefile is still branded R2Go2 end-to-end: `BINARY_NAME=r2go2` (Makefile:14), builds `r2go2` and symlinks `cosmoflare` (Makefile:49 — inverted: goreleaser ships `cosmoflare` and the docs call `r2go2` the alias). Its `build-all` covers 7 platforms including linux/armv7 and windows/arm64 (Makefile:26) which goreleaser does not build; its dist target produces `r2go2-vX.Y.Z-os-arch.tar.gz` archives while the real release publishes bare `cosmoflare-os-arch` binaries. `make install` installs only `r2go2`; `make deb` hardcodes `Architecture: amd64` regardless of host.
   - **Severity**: medium
   - **File**: `Makefile:14`
   - **Fix**: Rebase the Makefile on `BINARY_NAME=cosmoflare` with an `r2go2` symlink alias (matching Makefile:49's intent but the right way round), align PLATFORMS with `.goreleaser.yaml` goos/goarch, and drop the duplicate dist path per finding 2.

6. **No package-manager distribution channel exists** — No Homebrew formula anywhere (`find *.rb` — empty), no scoop manifest, no winget, no chocolatey; the Docker image is built and tagged locally only (`docker-build`, Makefile:337-340 — no push target, no registry). The only touchless install paths are the unpromoted `install.sh` and `go install`. For an open-source CLI aiming at community adoption, brew/scoop are table stakes. The Makefile's `install-brew` target is broken besides: `brew install --cask build/r2go2` (Makefile:119) passes a binary path where a cask name/path-to-cask-rb is required — it cannot work as written.
   - **Severity**: medium
   - **File**: `Makefile:119`
   - **Fix**: Remove `install-brew`; create `CosmoLabs-org/homebrew-tap` with a `cosmoflare` formula pulling `cosmoflare-<os>-<arch>` + checksums from GitHub releases (the bare-binary + checksums-sha256.txt asset layout already fits a formula perfectly); consider a GitHub Action-free `tap bump` step appended to the local release runbook.

7. **install.sh accepts linux/armv7, which no pipeline builds** — `detect_os_arch` maps `armv7l` → `armv7` (install.sh:95-97), then downloads `cosmoflare-linux-armv7` — an asset goreleaser never produces (`.goreleaser.yaml:40-44` builds only amd64/arm64). On a 32-bit ARM box the installer fails at download with a 404 *after* the checksums step succeeds, an avoidable dead end. (The divergent Makefile builds armv7 at Makefile:26 but its r2go2-named artifacts are never published.)
   - **Severity**: medium
   - **File**: `install.sh:95`
   - **Fix**: Either drop `armv7l` from the case statement with a clear "unsupported — build from source" message, or add `goarch: arm` + `goarm: [7]` to the goreleaser build matrix.

8. **Desktop tier: version skew, unsigned, no updater, and sidecar platform gap** — `desktop/src-tauri/tauri.conf.json:4` pins version `0.16.0` against CLI `0.26.0` with no stated mapping; the desktop README defers code-signing/notarization/auto-update to "v2" (desktop/README.md:129) and `tauri.conf.json` confirms: no `macos.signingIdentity`, no updater artifacts, `"csp": null`. The sidecar script builds 4 triples (build-sidecar.sh:19-23) — linux/arm64 is missing even though the CLI ships it, so linux-arm64 desktop builds would bundle no sidecar. As a paid-tier product this is pre-release, which is legitimate today, but there is no release process at all for the desktop app (no tagging, notes, or signing runbook).
   - **Severity**: medium
   - **File**: `desktop/src-tauri/tauri.conf.json:4`
   - **Fix**: Add a `linux/arm64` target to build-sidecar.sh (pure Go, trivially cross-compiles); document a desktop release SOP (version mapping to CLI, notarization steps for macOS, signing for Windows) before the paid tier ships; enable `createUpdaterArtifacts` when the updater lands.

9. **Stale R2Go2 branding and metadata across the distribution surface** — `.version-registry.json:5` still describes the project as "R2Go2 - A production-ready CLI tool for managing Cloudflare R2 buckets" (project is now the full Cloudflare platform); install.sh is titled "R2Go2 Installation Script" and configures `~/.config/r2go2` (install.sh:23) while installing the `cosmoflare` binary; CHANGELOG.md carries an "Unreleased" section describing r2go2-era work ("8 new Go packages", `internal/cli/*`) positioned above `[0.26.0]`, which will mislead the next release-notes pass. The documented "r2go2 backward-compatible alias" (README.md:22) is only honored by source builds — goreleaser and install.sh ship no `r2go2`.
   - **Severity**: low
   - **File**: `.version-registry.json:5`
   - **Fix**: Update registry description and install.sh header/CONFIG_DIR (with migration from `~/.config/r2go2`); have install.sh optionally symlink `r2go2 -> cosmoflare` to honor the alias promise; move the stale Unreleased changelog entries into the released section they belong to.

Non-finding observations: `cmd/installer_tui/main.go:1280` uses raw `cmd.Start()` (ROAD-525 lint hit) — it is a fire-and-forget URL opener, low orphan risk, but it is the same pattern BUG-124 banned; the embedded version-stamp assertion in the goreleaser runbook (".goreleaser.yaml:8") is good practice and matches the CI gate concept.

### Blocking Issues for Next Release

1. Publish v0.26.0 (staged in `dist/upload/`) — until the four-release gap is resolved, every binary install serves v0.22.0 (finding 1).
2. Remove `../docs/` from the Makefile dist target before anyone reaches for `make release-prepare` (finding 2).
3. Fix the GETTING_STARTED.md install URLs (finding 3) — it is the doc a new user hits first.
4. Update CHANGELOG's Unreleased section so v0.27.0 notes are generated from real content (finding 9).

### Recommendations

- [ ] Publish the staged v0.26.0 assets and add a `tag ↔ release` consistency check to the release smoke (`gh release view vX.Y.Z` must succeed before the release is called done) (effort: small)
- [ ] Strip `../docs/` from Makefile dist archives, and delete the duplicate dist/checksums/release-prepare path in favor of the goreleaser runbook (effort: small)
- [ ] Fix GETTING_STARTED.md URLs; add curl|bash and PowerShell one-liners to README.md's Install section (effort: small)
- [ ] Delete or stub the three disabled workflow files so the repo stops advertising CI that policy forbids (effort: small)
- [ ] Drop armv7 from install.sh detection or build it in goreleaser; decide windows/arm64 explicitly (effort: small)
- [ ] Rebrand Makefile to cosmoflare-primary with r2go2 symlink, align PLATFORMS with goreleaser (effort: medium)
- [ ] Create a Homebrew tap (cosmoflare formula from release assets) and remove the broken install-brew target; add Scoop/winget manifests for Windows (effort: medium)
- [ ] Sign checksums-sha256.txt with cosign keyless and generate an SBOM in the goreleaser flow (syft exists as a Makefile target but is not part of the real release path) (effort: medium)
- [ ] Desktop: add linux/arm64 sidecar triple; write the desktop release SOP (version mapping, notarization, updater artifacts) before the paid tier (effort: large)

### Platform Expansion Roadmap

Current CLI release matrix: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64 (all CGO_ENABLED=0, bare binaries + sha256). Desktop matrix: darwin amd64/arm64, windows/amd64, linux/amd64 (Tauri bundles, unsigned). Mobile: none.

Expansion order, by user impact:
1. **Homebrew (macOS/Linux)** — highest ROI for an open-source CLI; asset layout already compatible (priority: high, effort: medium)
2. **Scoop + winget (Windows)** — covers the PowerShell audience install.ps1 currently serves manually (priority: medium, effort: medium)
3. **ghcr.io Docker image** — Dockerfile already exists, needs a build+push step and a usage contract (CLI-in-container for CI usage) (priority: medium, effort: small)
4. **windows/arm64 + linux/armv7** — decide both directions explicitly; either ship them in goreleaser or reject them cleanly in installers (priority: low, effort: small)
5. **Desktop signing + updater** — notarized DMG (macOS), signed MSI (Windows), Tauri updater artifacts; prerequisite for the paid tier (priority: high before paid launch, effort: large)
6. **Mobile tier (React Native + Expo)** — distribution via App Store/Play; no groundwork exists yet beyond the product-vision doc (priority: low, effort: large)

### Roadmap Suggestions

- **Release publication guard** — Post-tag smoke that fails loudly when a pushed tag has no GitHub release after N minutes; prevents a fifth silent gap (priority: high, effort: small)
- **Package-manager coverage wave** — Homebrew tap + Scoop + winget + ghcr.io image, all driven from the existing checksums-sha256.txt asset contract (priority: high, effort: medium)
- **Signed, attested releases** — cosign-sign the checksums file and attach an SBOM to each release; makes the fail-closed installer verification chain end at a trusted root (priority: medium, effort: medium)
- **Single-pipeline cleanup** — Retire the r2go2-branded Makefile release path and the dead workflow files; one name, one asset vocabulary, one runbook (priority: medium, effort: small)
- **Desktop distribution v2** — linux/arm64 sidecar, signing/notarization, updater artifacts, and a desktop release SOP with version mapping to the CLI (priority: medium, effort: large)
- **Mobile distribution groundwork** — Expo/EAS build profiles and store metadata skeleton per the 3-tier product vision (priority: low, effort: large)

```json:audit-result
{
  "agent": "distribution",
  "overall_score": 5,
  "sub_scores": {
    "ci_cd": 4,
    "packaging": 5,
    "deployment": 4,
    "multi_platform": 6,
    "developer_experience": 6
  },
  "critical_findings": [
    {
      "title": "Four consecutive tagged releases never published to GitHub",
      "severity": "high",
      "file": ".goreleaser.yaml:22",
      "fix": "Publish staged v0.26.0 assets via gh release create; backfill or document v0.23-v0.25; add a post-release guard asserting gh release view <tag> succeeds",
      "effort": "small"
    },
    {
      "title": "make dist tars docs/ (containing transcripts with AWS keys) into release archives",
      "severity": "high",
      "file": "Makefile:97",
      "fix": "Remove ../docs/ from the dist target; retire the duplicate Makefile release path in favor of goreleaser",
      "effort": "small"
    },
    {
      "title": "GETTING_STARTED.md install one-liner points at dead CosmoDev-R2Go2 repo",
      "severity": "high",
      "file": "GETTING_STARTED.md:15",
      "fix": "Repoint to CosmoLabs-org/cosmoflare/main/install.sh and surface the one-liner in README.md",
      "effort": "small"
    },
    {
      "title": "Dead CI workflows remain armed and contradict the local release flow (version string diverges: v0.26.0 vs 0.26.0)",
      "severity": "medium",
      "file": ".github/workflows/release.yml:37",
      "fix": "Delete or stub the three workflow files per the permanent no-CI policy",
      "effort": "small"
    },
    {
      "title": "Divergent r2go2-branded Makefile release pipeline (different binary name, asset names, platforms)",
      "severity": "medium",
      "file": "Makefile:14",
      "fix": "Rebase Makefile on BINARY_NAME=cosmoflare with r2go2 symlink; align PLATFORMS with .goreleaser.yaml",
      "effort": "medium"
    },
    {
      "title": "No package-manager distribution (no Homebrew/Scoop/winget; broken install-brew target; Docker never pushed)",
      "severity": "medium",
      "file": "Makefile:119",
      "fix": "Create CosmoLabs-org/homebrew-tap with a formula driven by release assets; remove install-brew; add Scoop/winget manifests",
      "effort": "medium"
    },
    {
      "title": "install.sh accepts linux/armv7 which no build pipeline produces (download 404)",
      "severity": "medium",
      "file": "install.sh:95",
      "fix": "Drop armv7l from detection with an unsupported message, or add goarm 7 to the goreleaser matrix",
      "effort": "small"
    },
    {
      "title": "Desktop tier: version skew (0.16.0 vs CLI 0.26.0), unsigned, no updater, sidecar lacks linux/arm64",
      "severity": "medium",
      "file": "desktop/src-tauri/tauri.conf.json:4",
      "fix": "Add linux/arm64 sidecar triple; write desktop release SOP with version mapping, notarization, and updater artifacts before paid tier",
      "effort": "large"
    },
    {
      "title": "Stale R2Go2 branding across distribution metadata (registry description, install.sh, CHANGELOG Unreleased; r2go2 alias not shipped)",
      "severity": "low",
      "file": ".version-registry.json:5",
      "fix": "Update description/branding, migrate config dir, have install.sh symlink r2go2, and reconcile the stale Unreleased changelog",
      "effort": "small"
    }
  ],
  "recommendations": [
    {
      "action": "Publish staged v0.26.0 assets and add tag-to-release consistency smoke",
      "effort": "small",
      "priority": "high"
    },
    {
      "action": "Remove ../docs/ from Makefile dist archives and retire the duplicate release path",
      "effort": "small",
      "priority": "high"
    },
    {
      "action": "Fix GETTING_STARTED.md install URLs and add one-liners to README",
      "effort": "small",
      "priority": "high"
    },
    {
      "action": "Delete or stub disabled GitHub workflow files",
      "effort": "small",
      "priority": "medium"
    },
    {
      "action": "Create Homebrew tap formula; add Scoop/winget manifests",
      "effort": "medium",
      "priority": "high"
    },
    {
      "action": "cosign-sign checksums and attach SBOM to each release",
      "effort": "medium",
      "priority": "medium"
    },
    {
      "action": "Rebrand Makefile to cosmoflare-primary with r2go2 symlink; align platform matrix",
      "effort": "medium",
      "priority": "medium"
    },
    {
      "action": "Desktop: add linux/arm64 sidecar and release SOP before paid tier",
      "effort": "large",
      "priority": "medium"
    }
  ],
  "roadmap_suggestions": [
    {
      "title": "Release publication guard",
      "description": "Smoke check that fails when a pushed tag has no GitHub release; prevents silent four-release gaps",
      "priority": "high",
      "effort": "small"
    },
    {
      "title": "Package-manager coverage wave",
      "description": "Homebrew tap + Scoop + winget + ghcr.io image driven from the checksums-sha256.txt asset contract",
      "priority": "high",
      "effort": "medium"
    },
    {
      "title": "Signed, attested releases",
      "description": "cosign-sign the checksums file and attach an SBOM so the fail-closed installer chain ends at a trusted root",
      "priority": "medium",
      "effort": "medium"
    },
    {
      "title": "Single-pipeline cleanup",
      "description": "Retire r2go2-branded Makefile release path and dead workflows; one name, one asset vocabulary, one runbook",
      "priority": "medium",
      "effort": "small"
    },
    {
      "title": "Desktop distribution v2",
      "description": "Signing, notarization, updater artifacts, linux/arm64 sidecar, and desktop release SOP with CLI version mapping",
      "priority": "medium",
      "effort": "large"
    },
    {
      "title": "Mobile distribution groundwork",
      "description": "Expo/EAS build profiles and store metadata skeleton per the 3-tier product vision",
      "priority": "low",
      "effort": "large"
    }
  ]
}
```
