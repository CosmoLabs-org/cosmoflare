# Agent 10: Documentation

> Audit: 2026-08-31, cosmoflare v0.17.0, fresh mode. Report saved verbatim from agent output.

I have completed my investigation — 20+ docs read and cross-verified against Cobra command definitions, library constructors, git tags, and workflows. Here is my report.

## Documentation: 5/10

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| completeness | 6/10 | `docs/USAGE.md` (2388 lines) is genuinely comprehensive — 55+ command sections covering every implemented service, MCP, wrangler compat, library usage. But architecture/, api/, instructions/ are unfilled templates, zero ADRs, no CONTRIBUTING.md, no `examples/` dir, no library godoc. |
| accuracy | 4/10 | `docs/USAGE.md` is accurate (verified against code), but README — the front door — misstates project reality by 2.5 months: 8 shipped services marked "Planned", 2 incorrect quick-start commands, stale roadmap. Five tracked root docs still use the pre-rename "R2Go2"/"CosmoDev-R2Go2" identity with dead URLs. |
| onboarding | 5/10 | README quick start is good (1-command install, 2 env vars, ~5 min to first value). But GETTING_STARTED.md — the file a new dev opens by name — points at a dead repo URL, and TESTING-GUIDE.md builds a binary name (`R2Go2`) that no longer exists. No CONTRIBUTING.md, no issue/PR templates. |
| api_docs | 4/10 | The Go library is marketed as "the stable API surface" but has zero `// Package` comments across 54 non-test files, no `doc.go`, no example functions, and the primary interface comment still says "R2Go2 operations". No documented stability/versioning policy. CHANGELOG itself is well-kept (Keep-a-Changelog, 20 versions). |

### Critical Findings

1. **README service table and roadmap are 2.5 months stale** — `README.md:26-43` marks D1, Pages, Email Routing, Images, Stream, WAF/Firewall, Page Rules, and Workers AI as "Planned —", yet all are implemented with both command and library files (`cmd/d1.go`, `cmd/pages.go`, `cmd/email.go`, `cmd/images.go`, `cmd/stream.go`, `cmd/waf.go`, `cmd/firewall.go`, `cmd/pagerules.go`, `cmd/ai.go`, plus `queue.go`, `hyperdrive.go`, `vectorize.go`). `README.md:204-214` lists "Upcoming: MCP server, `cosmoflare status` dashboard, `cosmoflare diff`" — all shipped (`cmd/mcp.go`, `cmd/status.go`, `cmd/dashboard.go`, `cmd/diff.go`). A new developer reading only the README believes the product is a fraction of its actual size. The Project Structure section (`README.md:184-202`) lists 9 cmd files; there are 68.
   - **Severity**: high
   - **File**: `README.md:26-43`, `README.md:204-214`
   - **Fix**: Regenerate the Services table from `cmd/` + `pkg/cosmoflare/` file inventory (like CLAUDE.md already does correctly); delete the stale Roadmap "Upcoming" list in favor of a link to `docs/roadmap/`.

2. **README quick-start commands have wrong syntax** — `README.md:102` shows `cosmoflare kv put NAMESPACE_ID my-key "my-value"`, but the actual signature is `put [namespace-id] [key]` with the value supplied via `--value` or `--file` (`cmd/kv.go:95-102`). `README.md:98` shows `cosmoflare worker deploy --name my-worker --script worker.js`, but the real signature is positional `deploy [name]` with no `--name` flag (`cmd/worker.go:54`, examples at `:61`). The maintained `docs/USAGE.md:424-427` documents kv put correctly — the README is the drifted copy. First commands a new user copies fail.
   - **Severity**: high
   - **File**: `README.md:98`, `README.md:102`
   - **Fix**: Correct both to `cosmoflare kv put ns my-key --value="..."` and `cosmoflare worker deploy my-worker --script worker.js`. Add a CI lint that extracts code blocks from README and checks the first two tokens against Cobra `Use:` strings.

3. **GETTING_STARTED.md points at a dead repository URL** — last touched 2025-11-24 and still titled "Getting Started with R2Go2". Its one-command installers (`GETTING_STARTED.md:12-18`) fetch `https://raw.githubusercontent.com/CosmoLabs-org/CosmoDev-R2Go2/main/install.sh` — the pre-rename repo path. A new developer following this tracked, root-level guide gets a 404.
   - **Severity**: high
   - **File**: `GETTING_STARTED.md:12-18`
   - **Fix**: Either rewrite around `go install github.com/CosmoLabs-org/cosmoflare@latest` (the README path, which works — tags reach v0.17.0) or delete the file and fold anything unique into README.

4. **The "stable API surface" library has no package documentation** — `pkg/cosmoflare/` has 54 non-test files and zero `// Package` comments; no `doc.go`; no `example_test.go`; no `examples/` directory anywhere. The primary interface doc at `pkg/cosmoflare/client.go:14` reads `// R2Client is the primary interface for R2Go2 operations.` — the retired product name inside the most-read godoc comment in the library. `pkg.golangle.org`-style discovery is impossible; consumers have only the README's (accurate, verified) constructor snippets.
   - **Severity**: high
   - **File**: `pkg/cosmoflare/client.go:14` (and 53 sibling files)
   - **Fix**: Add `pkg/cosmoflare/doc.go` with a package overview (service list, client vs `*Service` patterns, config), a one-line package comment where a second package emerges, and `example_test.go` with runnable `Example*` functions for R2/DNS/KV. Replace "R2Go2" in client.go:14.

5. **Release workflow generates dead install instructions and mis-stamped versions** — `.github/workflows/release.yml:87` emits `go install github.com/CosmoLabs-org/CosmoDev-R2Go2@${VERSION}` into generated release notes (old module path), and `:62` sets ldflags `-X github.com/CosmoLabs-org/CosmoDev-R2Go2/cmd.AppVersion=...` — a package path that no longer exists, so the `-X` silently no-ops and shipped binaries may report `dev` (`cmd/root.go:22`) instead of the release version. Every published release note carries a broken install command.
   - **Severity**: high
   - **File**: `.github/workflows/release.yml:62`, `.github/workflows/release.yml:87`
   - **Fix**: Replace both occurrences of `CosmoDev-R2Go2` with `cosmoflare`; add a post-build assertion that the binary's `--version` output matches `GITHUB_REF_NAME`.

6. **Cluster of pre-rename artifacts committed at the repo root** — `BUILD-SUMMARY.md` ("R2Go2 Build Summary — Project Complete!"), `TESTING-GUIDE.md` ("R2Go2 Testing & Setup Guide", `go build -o R2Go2 .` — wrong binary name), `INTERACTIVE-SETUP-DEMO.md`, and an untracked root `USAGE.md` titled "R2Go2 Complete Usage Guide" (1368 lines, references `R2GO2_DEBUG`, and confusingly shadows the real `docs/USAGE.md`). Root-level doc sprawl makes the wrong doc easy to find and the right one hard to trust.
   - **Severity**: medium
   - **File**: `BUILD-SUMMARY.md:1`, `TESTING-GUIDE.md:1,7`, `USAGE.md` (root, untracked)
   - **Fix**: Move BUILD-SUMMARY/INTERACTIVE-SETUP-DEMO to `docs/history/` or delete; rewrite TESTING-GUIDE around `make build` / `go test ./pkg/... ./internal/...`; delete the untracked root USAGE.md.

7. **Template-only documentation directories and a broken link** — `docs/architecture/` contains only an unfilled README and a placeholder USAGE.md (its README cites example files like `ADR-001-database-choice.md` that don't exist; there are zero ADRs in the project). `docs/api/` holds one unfilled template. `docs/instructions/` is an empty template. `docs/FEATURES.md` is an untouched scaffold ("One-line description of the project", "Version: X.Y.Z", "[Company Name] — [Tagline]"). `docs/README.md:31` links `[SPEC.md](./SPEC.md)` which does not exist. For a 20+ service platform with a daemon, desktop app, and MCP server, there is no architecture document at all — the best architecture prose lives in `desktop/README.md` (current and good) but nothing covers the CLI/daemon/library split.
   - **Severity**: medium
   - **File**: `docs/README.md:31`, `docs/architecture/README.md`, `docs/FEATURES.md:1-4`
   - **Fix**: Either fill them or delete them — empty templates read as abandoned. Write one `docs/architecture/system-overview.md` (CLI → `pkg/cosmoflare` → CF API; `serve` daemon; MCP layer). Fix or remove the SPEC.md link.

8. **Undocumented environment variable affecting behavior** — `R2GO2_SKIP_FIRST_RUN` is referenced 15 times across `cmd/` and `internal/`, and `COSMOFLARE_NO_KEYCHAIN` 6 times; neither appears in README, and only `COSMOFLARE_NO_KEYCHAIN` appears once in `docs/USAGE.md`. `R2GO2_SKIP_FIRST_RUN` still carries the legacy prefix and controls first-run wizard behavior agents may need to suppress.
   - **Severity**: medium
   - **File**: `docs/USAGE.md:7-24` (Setup section — missing env var table)
   - **Fix**: Add an "Environment Variables" table to docs/USAGE.md covering `CLOUDFLARE_ACCOUNT_ID`, `CLOUDFLARE_API_TOKEN`, `COSMOFLARE_NO_KEYCHAIN`, `R2GO2_SKIP_FIRST_RUN`; consider renaming the latter with a compat alias.

9. **No contributor documentation** — no CONTRIBUTING.md anywhere, no `.github/ISSUE_TEMPLATE/`, no PR template, no code-of-conduct. Development conventions exist only in CLAUDE.md, which is an internal agent document, not a public contributor guide. For an MIT-licensed project seeking community adoption (stated goal in docs/PRODUCT-VISION.md), this blocks outside contributions.
   - **Severity**: medium
   - **File**: (missing at repo root / `.github/`)
   - **Fix**: Write a short CONTRIBUTING.md (build via `make build`, test via `go test ./pkg/... ./internal/...`, conventional commits, "every command needs `--json` + rich `--help`" rules already enforced by convention).

10. **docs/USAGE.md has no table of contents** — 2388 lines and 55+ `##` sections with no TOC or anchor index at the top; navigation requires full-text search. Minor given the per-command Cobra `--help` is excellent (verified rich `Long` + `Examples` in `cmd/kv.go:95-105`, `cmd/object.go:120-144` — the agent-first claim is genuinely honored in code).
    - **Severity**: low
    - **File**: `docs/USAGE.md:1-6`
    - **Fix**: Generate a linked TOC from the `##` headings (a 10-line script, can run in CI).

### Recommendations

- [ ] Rewrite README Services table + Roadmap from the actual `cmd/` inventory; fix the two broken quick-start commands (`kv put`, `worker deploy`) (effort: small)
- [ ] Add `pkg/cosmoflare/doc.go`, package comments, and `example_test.go` with runnable examples for the top 5 services; scrub "R2Go2" from client.go:14 (effort: medium)
- [ ] Fix `CosmoDev-R2Go2` module path in `.github/workflows/release.yml:62,87` and assert binary `--version` in CI (effort: small)
- [ ] Delete or rewrite GETTING_STARTED.md around the working `go install` path; remove root USAGE.md (untracked) and archive BUILD-SUMMARY.md / INTERACTIVE-SETUP-DEMO.md / TESTING-GUIDE.md (effort: small)
- [ ] Write CONTRIBUTING.md + minimal issue/PR templates (effort: medium)
- [ ] Add a CI doc-accuracy gate: extract fenced commands from README/docs and validate verb+subcommand against Cobra command tree (effort: medium)
- [ ] Fill or delete the template dirs (`docs/architecture/`, `docs/api/`, `docs/instructions/`, `docs/FEATURES.md`) and fix the `docs/SPEC.md` broken link; add one real system-overview architecture doc (effort: medium)
- [ ] Add an Environment Variables reference table to docs/USAGE.md including `R2GO2_SKIP_FIRST_RUN` (effort: small)
- [ ] Generate a TOC for docs/USAGE.md in CI (effort: small)

### Roadmap Suggestions

- **Documentation truth-align: README + root docs match v0.17.0 reality** — Regenerate the service table from code, fix broken quick-start syntax, purge pre-rename artifacts (priority: high, effort: small)
- **Go library godoc program** — doc.go, package comments, Example functions, and a documented stability/versioning policy for `pkg/cosmoflare` (priority: high, effort: medium)
- **Doc accuracy CI gate** — Validate documented CLI syntax against Cobra definitions so README drift like the kv/worker examples cannot recur (priority: medium, effort: medium)
- **Public contributor path** — CONTRIBUTING.md, issue templates, and a development-workflow guide derived from CLAUDE.md conventions (priority: medium, effort: small)
- **Architecture documentation** — One system-overview doc covering CLI/library/daemon/desktop/MCP layering and the first ADRs for decisions like the daemon bearer-token model (priority: medium, effort: medium)

**Strengths worth preserving**: `docs/USAGE.md` is a model command reference — current (updated 2026-07-01), syntax-exact against code, JSON output shapes included. `desktop/README.md` and `docs/PRODUCT-VISION.md` are current and well-written. CHANGELOG discipline is solid (Keep-a-Changelog, 20 tagged releases, git tags match). The Cobra `--help` layer is genuinely agent-first as claimed. The problem is everything except docs/USAGE.md drifted after the R2Go2→Cosmoflare rename and the June service-expansion wave.
