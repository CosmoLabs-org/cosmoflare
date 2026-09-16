# TASK-010 handoff — /cosmoflare product page on cosmolabs.org

Run this in a session opened at `~/PROJECTS/cosmolabs.org`. The artifact to
integrate lives in the cosmoflare repo at `docs/launch/product-page.html`
(source of truth — do not fork the copy; adapt markup only).

## Goal

Serve the Cosmoflare product page at `https://cosmolabs.org/cosmoflare`
with OG tags and JSON-LD intact. TASK-010 acceptance: page live with
OG + JSON-LD.

## Integration steps

1. Copy the `<head>` metadata verbatim: title, description, OG block,
   Twitter card, canonical (`https://cosmolabs.org/cosmoflare`), and both
   JSON-LD scripts (SoftwareApplication + BreadcrumbList). These are final
   copy — only adapt to the site's templating mechanism.
2. Port the `<body>` sections (hero, features, quickstart, license,
   footer) into the site's design system. The artifact uses semantic
   classes (`hero`, `features`, `quickstart`, `cta`); restyle freely, keep
   copy and link targets unchanged.
3. Route: `/cosmoflare`. If the site is static, add the page to its
   generator; ensure trailing-slash behavior matches the canonical.
4. Social card: generate a 1200x630 image at `/img/cosmoflare-og.png`
   (marked `TODO(site)` in the artifact). The repo social preview can be
   reused as the base.
5. Post-deploy checks:
   - `curl -s https://cosmolabs.org/cosmoflare | grep 'og:title'`
   - Rich-results check on the JSON-LD (search Google's Rich Results Test
     for the URL).
   - Canonical resolves 200; `/cosmoflare/` does not double-redirect.

## Facts already true (do not re-verify in copy)

- Version at time of writing: v0.28.2 (JSON-LD `softwareVersion` is set
  to it). Bump this field on future releases — the cosmoflare repo's
  `docs/launch/product-page.html` is the SSOT; update there first, then
  mirror in the site.
- Both install paths work: `go install github.com/CosmoLabs-org/cosmoflare@latest`
  (proxy.golang.org serves v0.28.2+) and `brew install
  CosmoLabs-org/cosmoflare/cosmoflare` (tap live).
- Link targets: GitHub repo, USAGE.md, vs-wrangler.md (all `master` branch).
- "Not affiliated with Cloudflare" disclaimer stays in the footer.

## Closing TASK-010

When the page is live, close the task in the cosmoflare repo
(`ccs issues update TASK-010 --status closed`) with the live URL in the
note, and mention the closure in the next cosmoflare session so the
continuation prompt goal ticks.
