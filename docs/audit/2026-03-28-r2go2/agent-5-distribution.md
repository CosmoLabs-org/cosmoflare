# Distribution Audit: CosmoDev-R2Go2

## 1. Packaging (Score: 72/100)

**Dockerfile:**
- Properly uses multi-stage build (golang:1.25-alpine -> alpine:latest)
- Creates non-root user (r2go2:r2go2, UID/GID 1001) -- good security practice
- CGO_ENABLED=0 for static binary, strips debug symbols (-w -s)
- ca-certificates and tzdata included
- OCI labels present
- **Issue**: Hardcodes GOARCH=amd64 instead of using TARGETARCH
- **Issue**: Uses alpine:latest instead of pinned tag

**Makefile:**
- 30+ targets including build, test, lint, dist, deb, rpm, sbom, docker, version management
- Cross-compilation for 7 platforms
- **Critical**: linux/armv7 is not a valid GOARCH value (should be arm with GOARM=7)
- **Issue**: install-brew target uses incorrect brew install --cask syntax
- **Issue**: sha256sum is Linux-specific; macOS uses shasum -a 256

**go.mod/go.sum:**
- Clean version pinning to Go 1.25.3
- No replace or retract directives
- 167 lines in go.sum -- reasonable dependency footprint

## 2. CI/CD (Score: 55/100)

**build.yml:**
- Triggers on push/PR to master and main
- Jobs: test, lint (golangci-lint), security-scan (gosec), build (7-platform matrix), docker, sbom
- Module caching, Codecov upload
- **Issue**: GOOS/GOARCH env vars set to full platform string before shell override

**test-suite.yml:**
- **BUG-003 confirmed**: Uses Go 1.26 in matrix -- does not exist
- Multi-OS: ubuntu, macos, windows
- Quality gate with coverage thresholds (70% overall, 80% critical paths)
- **Issue**: Codecov conditional on go-version 1.26 -- never uploads

**release.yml:**
- Tag-triggered, auto-generated release notes
- 7-platform builds with SHA256 checksums
- DEB/RPM packages, Homebrew tap auto-update
- **Critical**: Uses deprecated actions/create-release@v1 and actions/upload-release-asset@v1
- **Issue**: Missing LICENSE file and docs/docker-hub.md referenced in workflow

## 3. Multi-Platform (Score: 68/100)

- 7 GOOS/GOARCH combinations (linux/armv7 invalid)
- Cross-platform test suite (295 lines) covering OS detection, paths, permissions
- CI runs on 3 OS matrix
- CGO_ENABLED=0 everywhere ensures static portability
- **Weakness**: Command execution tests are pure placeholders

## 4. Install Experience (Score: 62/100)

**install.sh:**
- Detects OS and arch, supports curl and wget
- Auto-modifies shell RC files for PATH
- **Critical**: Binary name mismatch (R2Go2 uppercase vs r2go2 lowercase in releases) -- download 404
- **Issue**: No checksum verification despite release producing .sha256 files
- **Issue**: Modifies RC files without asking

**Setup wizard:**
- Full interactive with --quiet mode for CI
- Profile support, backup/restore
- Installer TUI is a separate binary with no build target

## Summary

| Category | Score |
|----------|-------|
| Packaging | 72 |
| CI/CD | 55 |
| Multi-Platform | 68 |
| Install Experience | 62 |
| **Overall** | **64** |

Ambitious distribution infrastructure with several broken components. Install script has binary name mismatch, CI uses non-existent Go version, deprecated GitHub Actions, and missing referenced files.
