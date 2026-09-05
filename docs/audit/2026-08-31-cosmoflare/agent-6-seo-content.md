# Agent 6: SEO/Content

> Audit: 2026-08-31, cosmoflare v0.17.0, fresh mode. Report saved verbatim from agent output.

Investigation complete. I have read 12+ files, queried GitHub/PkgGoDev/CosmoLabs.org, and checked the live search landscape. Full report follows.

## SEO/Content: 2/10

Cosmoflare has no public discoverability surface. The repository is private. The library 404s on pkg.go.dev. The organization site lists no products. The README — the only front door — is three months stale and undersells the shipped product by half. The content engine itself is real: 365 markdown files, a 2388-line usage guide, 19 released versions in the changelog. None of it is reachable by any search engine or any developer outside the org.

**Sub-scores**:

| Dimension | Score | Notes |
|-----------|-------|-------|
| meta_tags | 1/10 | Only HTML in the project is `desktop/index.html:3-7` — a title tag and viewport, nothing else. No description, no Open Graph, no favicon. No site exists to carry meta tags. |
| structured_data | 0/10 | No JSON-LD, no sitemap.xml, no robots.txt anywhere in the repo. Capability does not exist. |
| technical_seo | 1/10 | No indexable pages exist. No canonical URLs. Nothing to assess beyond absence. |
| content_strategy | 4/10 | Substantial content volume (USAGE.md 2388 lines / 53 sections, 365 docs, Keep-a-Changelog changelog through 0.17.0) but zero keyword targeting, no blog, no tutorials beyond a stale R2Go2-era guide, and legacy-name fragments (`CosmoDev-R2Go2`) throughout. |
| discoverability | 2/10 | Repo private; no description, no topics (`[]`), no homepage, no GitHub Releases (21 local tags, `latestRelease: null`); pkg.go.dev returns 404; `cosmoflare` brand term returns zero relevant SERP results. |
| conversion_cta | 3/10 | README Quick Start is a working copy-paste CTA, but the install path is broken for outsiders and trust signals are thin. |

### Critical Findings

1. **Repository is private while the product claims to be open-source** — The README (line 10) says "Open-source CLI and Go library" and ships an MIT LICENSE, but `gh api repos/CosmoLabs-org/cosmoflare` returns `isPrivate: true`. The repo has no description, no topics, and no homepage set. Zero search visibility is structural: no engine can index any of the 365 docs.
   - **Severity**: high
   - **File**: repository metadata (verified via GitHub API, 2026-08-31)
   - **Fix**: Flip public, then set description ("Open-source CLI and Go library for the full Cloudflare platform — agent-first, --json everywhere"), 15-20 topics (`cloudflare`, `cli`, `go`, `r2`, `wrangler-alternative`, `mcp`, `devops`, `dns`, `workers`, `d1`, `tauri`), and homepage URL.

2. **`go install` instruction in README is a dead path** — `README.md:50` instructs `go install github.com/CosmoLabs-org/cosmoflare@latest`. This fails for every user outside the org while the repo is private, and pkg.go.dev confirms 404. The primary conversion funnel is broken at step one.
   - **Severity**: high
   - **File**: `README.md:50`
   - **Fix**: Publish first, then verify the one-liner from a clean machine. Add `install.sh` curl one-liner and a Homebrew tap as alternates. Note `install.sh:3` still brands itself "R2Go2 Universal Installation Script" — rename before pointing anyone at it.

3. **README services table marks 13 shipped services as "Planned"** — `README.md:35-43` lists D1, Pages, Email Routing, Images, Stream, WAF/Firewall, Page Rules, Workers AI as "Planned". All are implemented: `cmd/d1.go`, `cmd/pages.go`, `cmd/email.go`, `cmd/images.go`, `cmd/stream.go`, `cmd/waf.go`, `cmd/pagerules.go`, `cmd/ai.go` exist and `docs/USAGE.md` documents all of them (53 sections, updated 2026-07-01). The roadmap section `README.md:206-212` repeats the stale claims. A visitor reading the README concludes the tool covers 7 services; it covers 20+. The front door undersells the product by two-thirds.
   - **Severity**: high
   - **File**: `README.md:35-43`, `README.md:206-212` (last commit to README: 2026-06-06)
   - **Fix**: Regenerate the table from the CLAUDE.md service matrix (already accurate) or from `cmd/*.go` command registration. Same fix for `cmd/root.go:33-40`, whose `--help` long text also lists only 7 services.

4. **No GitHub Releases despite 21 tags** — CHANGELOG documents 19 versions through 0.17.0 (2026-06-21) and 21 git tags exist locally, but `latestRelease` is null. No release notes page, no prebuilt binaries, no install artifacts. This kills the trust funnel (changelog, stars trajectory, downloadable assets) that GitHub surfaces in search.
   - **Severity**: medium
   - **File**: `.github/workflows/release.yml` (exists but produced no releases)
   - **Fix**: Run the release workflow for the next tag with CHANGELOG-derived notes and cross-compiled binaries.

5. **Legacy brand fragments dilute the product identity** — `docs/README.md:1` is titled "CosmoDev-R2Go2 Documentation". `docs/FEATURES.md:1-5` says "CosmoDev-R2Go2 Features" and still contains the template placeholder "One-line description of the project" with "Version: X.Y.Z". `GETTING_STARTED.md` (last touched 2025-11-24) markets a two-mode "R2Go2" tool that no longer describes the product. Once public, these pages will rank for the wrong name and confuse the funnel.
   - **Severity**: medium
   - **File**: `docs/README.md:1`, `docs/FEATURES.md:1-5`, `GETTING_STARTED.md:1-19`, `install.sh:3`, `BUILD-SUMMARY.md`, `INTERACTIVE-SETUP-DEMO.md`
   - **Fix**: Rename or archive legacy-named docs. Rewrite FEATURES.md with real content. Fold GETTING_STARTED.md into the README or regenerate it as a Cosmoflare tutorial.

6. **USAGE.md has no installation section** — Grep for install instructions in the 2388-line agent-reference guide matches only plugin installs and a health-check curl. An agent or developer landing on USAGE.md cannot learn how to obtain the binary.
   - **Severity**: medium
   - **File**: `docs/USAGE.md` (no install section; structure verified via heading grep)
   - **Fix**: Add an Installation section at the top with `go install`, curl script, and build-from-source, duplicated from README.

7. **No social preview image** — GitHub falls back to the org avatar (`openGraphImageUrl` is the avatar URL). Link shares on X/Reddit/Slack render with no product visual. No OG image asset exists anywhere in the repo (searched for png/social/og assets at root).
   - **Severity**: low
   - **File**: repository root (asset absent)
   - **Fix**: Create a 1280x640 OG image (terminal aesthetic, service logos, "CLI for the full Cloudflare platform") and upload as repo social preview.

8. **Go library ships without package documentation** — `pkg/cosmoflare/client.go:1` starts `package cosmoflare` with no package-level doc comment; no `doc.go` exists (searched). When the module goes public, pkg.go.dev will render the package with no synopsis — the single most-viewed registry page for a Go library will have no marketing text.
   - **Severity**: low
   - **File**: `pkg/cosmoflare/client.go:1`
   - **Fix**: Add `pkg/cosmoflare/doc.go` with a package comment covering scope, the 20+ service clients, and a runnable example.

### Keyword Analysis

WebSearch (2026-08-31) shows two things. First, the term "cosmoflare" returns nothing about this project — the brand SERP is empty, which means the name is winnable but also that no one can find the product by name today. Second, Cloudflare itself just launched `cf`, a unified CLI for the whole platform (see Sources). The "one CLI for everything Cloudflare" positioning is now contested by the platform owner. The defensible angles are: open-source (vs official), importable Go library, agent-first `--json` design, MCP server mode, and the desktop/mobile tiers.

**Top 10 keywords to target** (estimated monthly volume, US, dev-tool intent; estimates from SERP composition, not a paid tool):

1. `wrangler alternative` — ~300/mo, low difficulty, highest intent. Comparison page is the single best content play.
2. `cloudflare r2 command line` / `r2 cli` — ~500/mo, low. R2 is the strongest implemented surface (upload, presign, multipart).
3. `cloudflare cli` — ~6K/mo, high difficulty (official docs + `cf` launch). Target via long-tail pages, not head term.
4. `cloudflare mcp server` — ~1K/mo and rising, low-medium. `cosmoflare mcp` exists (`cmd/mcp.go`); a dedicated guide can own this.
5. `cloudflare dns command line` / `manage dns from terminal` — ~400/mo, low.
6. `cloudflare-go` / `cloudflare go library` — ~800/mo, medium. Wraps the official SDK; position as the higher-level alternative.
7. `cloudflare terraform alternative` — ~400/mo, low-medium. `cosmoflare terraform` export exists (`cmd/terraform.go`) — direct content hook.
8. `cloudflare cost calculator` — ~2K/mo, medium. `cosmoflare cost` exists (`cmd/cost.go`); free tool page is linkable.
9. `cloudflare presigned url cli` — ~150/mo, very low, high intent.
10. `cosmoflare` — 0/mo today, branded moat. Own it with the repo, docs site, and pkg.go.dev listing before anything else claims the term.

### Quick Wins for SEO

- Make the repo public and fill description/topics/homepage (one session, largest single visibility gain).
- Regenerate the README services table and roadmap from CLAUDE.md's accurate matrix (content already exists).
- Publish a GitHub Release with CHANGELOG notes and binaries; the release workflow file already exists.
- Upload a social preview image.
- Add `pkg/cosmoflare/doc.go` before the module proxy indexes the package.

### Recommendations

- [ ] Make repository public; set description, 15-20 topics, homepage (effort: small)
- [ ] Rewrite README services table + roadmap section from CLAUDE.md matrix; fix `cmd/root.go:33-40` help text (effort: small)
- [ ] Publish GitHub Release for current version with binaries; verify `go install ...@latest` from clean machine (effort: small)
- [ ] Archive/rename R2Go2- and CosmoDev-branded docs; delete FEATURES.md placeholders (effort: small)
- [ ] Add Installation section to top of `docs/USAGE.md` (effort: small)
- [ ] Add `pkg/cosmoflare/doc.go` package documentation with example (effort: small)
- [ ] Build a docs site on the stack-standard Astro 5 + Tailwind 4, splitting USAGE.md's 53 sections into 53 indexable pages with per-page titles/descriptions/sitemap (effort: large)
- [ ] Publish a "Cosmoflare vs Wrangler vs cf" comparison page targeting `wrangler alternative` intent (effort: medium)
- [ ] Link SECURITY.md from README; add a terminal GIF/screenshot above the fold (effort: small)

### Roadmap Suggestions

- **Public launch readiness** — Flip repo public, complete GitHub metadata, first Release, verified install paths, social preview (priority: high, effort: small)
- **Astro documentation site** — 53 USAGE sections become 53 SEO pages with sitemap, JSON-LD SoftwareApplication schema, and Algolia-free client search (priority: high, effort: large)
- **Comparison content hub** — Pages for wrangler-alternative, terraform-alternative, cloudflare-cost-calculator intent, each wired to an existing CLI command (priority: medium, effort: medium)
- **pkg.go.dev presence program** — doc.go, runnable examples, doc-tested snippets so the registry page markets the library (priority: medium, effort: medium)

Sources: [Building a CLI for all of Cloudflare](https://blog.cloudflare.com/cf-cli-local-explorer/), [Cloudflare: One CLI tool for everything — heise online](https://www.heise.de/en/news/Cloudflare-One-CLI-tool-for-everything-11256616.html), [cloudflare-cli on GitHub](https://github.com/danielpigott/cloudflare-cli), [pkg.go.dev module page (404)](https://pkg.go.dev/github.com/CosmoLabs-org/cosmoflare), [cosmolabs.org](https://cosmolabs.org)
