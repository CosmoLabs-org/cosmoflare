---
ulid: 01M1WERP1JZHNT458FKVF4BF59
id: IDEA-050
title: Makefile release target codifying the local goreleaser + gh publish recipe
created: "2026-09-07T02:52:30.130093+04:00"
status: harvested
source: human
origin:
    session: 40
promoted_to: FEAT-016
---

# Makefile release target codifying the local goreleaser + gh publish recipe

# Makefile release target codifying the local goreleaser + gh publish recipe

Two releases in one session (v0.20.0, v0.21.0) hit identical friction: gh rejects --notes-from-tag with --repo; gh release create with file#label 404s on multi-asset upload; asset names need renamed copies because per-target dirs all contain 'cosmoflare'. Working recipe (proven twice 2026-09-07): shallow clone tag to mktemp dir, goreleaser release --clean, assert --version, extract CHANGELOG section to notes file, gh release create empty with --repo + --notes-file + --latest, cp binaries to final names, shasum -c verify, gh release upload, assert asset count 6. Codify as 'make release TAG=vX.Y.Z' so the next maintainer (or agent) cannot get it wrong. No-CI rule unaffected — still fully local.
