# Cosmoflare v0.28.1

Patch release: fixed release packaging, a 21-agent code-quality wave, and a
typed daemon wire contract.

## Fixes

- **Release archives are now installable** — v0.28.0 tarballs/zip archives
  contained `../README.md` and `../LICENSE` entries (paths escaping the
  archive root), which tar and Homebrew refuse to extract. Archives now
  carry root-level `cosmoflare`, `README.md`, `LICENSE`, and the binary
  inside is named plain `cosmoflare` instead of the platform-suffixed name.

## Changes

- **TASK-009: god-function wave complete.** All 85 functions over the
  funlen limit (lines > 80 or statements > 50, worst 410 lines) are split
  into behavior-preserving helpers; `golangci-lint` now runs clean with
  `funlen=80` enforced via the new `.golangci.yml`.
- **FEAT-042: typed daemon wire contract.** `internal/server`'s
  `ServeSource` and `MetricsSnapshot` return concrete types instead of
  `any`; TypeScript mirrors are generated (`make wire-types`) and a drift
  check gates releases (`make wire-types-check`). The desktop client gets
  typed endpoint methods.
- New docs: **Cosmoflare vs Wrangler** comparison (`docs/vs-wrangler.md`)
  and a README section linking it.

## Internal

- 21 GLM-agent refactors merged through the S334 verification gate
  (diff review + package tests re-run per worktree).
- `tygo.yaml` pins Go→TS generation to the wire-definition files.

---

Homebrew tap: `brew install CosmoLabs-org/cosmoflare/cosmoflare`
(available with this release).
