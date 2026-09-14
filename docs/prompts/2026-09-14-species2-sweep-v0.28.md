---
status: PENDING
type: continuation
priority: high
created: 2026-09-14T17:55:00+04:00
---

# Continuation: FEAT-040 Species-2 Presenter Sweep → v0.28.0

## State (2026-09-14, session ~20h)

Cosmoflare is **PUBLIC** (flipped 2026-09-13 after history purge — every
pre-2026-09-13 SHA is dead; pre-scrub mirror in
~/PROJECTS/cosmoflare-backups/cosmoflare-pre-scrub-2026-09-13.git).
v0.27.0 is the live release. All 12 audit bugs fixed (BUG-040..051 closed).
FEAT-038 (parallel sync, --concurrency) and FEAT-039 (transport policy,
transport.go chokepoint + guard test) are DONE. JSON-mode errors now exit 1
with envelope on stdout (738623e) — behavior change, changelogged.

## Goal 1: FEAT-040 species-2 — collapse the remaining ~410 payload branches

`if JSONOutput { return printJSON(x) }` + human rendering → one
`outPayload(...)` / `outResult(...)` call. Pattern is PROVEN; two exemplars
in history:

- **a6c7df1** — cmd/account.go full hand conversion (both species, closures,
  early-return handling)
- **e06d513** — species-1 mechanical sweep + `out*` helpers in cmd/output.go

Method, per file (branch density): email.go ~23, stream.go ~15, images.go
~13, kv.go ~11, alerts/waf/cache ~11-17 each, then the long tail (~39 files
total: `grep -c "if JSONOutput" cmd/*.go | grep -v :0 | sort -t: -k2 -rn`).

CAUTION (why species-2 is hand-work, not regex): human blocks contain
`return` statements that must become closure `return`s — but returns that
carry handler semantics (early error exits) must STAY at handler level.
Watch for species-3 stragglers while there: plain-only errors with no JSON
branch (e.g. bucket.go:300 was one) — route through `outErr`.

GLM-batchable per file IF each agent gets: the file path, both exemplar
commit SHAs, and "behavior identical, closures per account.go" — one prompt
file per agent, never a shared plan (S313).

Verification per tranche: `go build ./cmd/ && go test ./cmd/ -timeout 300s`
+ branch count `grep -rn "if JSONOutput" cmd/*.go | wc -l`.

## Goal 2: v0.28.0 release

After the sweep (or a meaningful tranche): `ccs version --release --bump
minor --no-push --highlights "..."` → `make release-prepare` (govulncheck
gate now built in; govulncheck binary was rebuilt with go1.27 — if stale
again: `(cd ~ && GOTOOLCHAIN=local go install golang.org/x/vuln/cmd/govulncheck@latest)`)
→ stage assets: Makefile now builds `cosmoflare-*` directly into dist/;
copy to dist/upload + `shasum -a 256 * > checksums-sha256.txt` → push →
`gh release create v0.28.0 dist/upload/*`. NOTE: this release finally
fixes pkg.go.dev's landing text (still "Package r2go2" from v0.27.0).

## Traps and rules (hard-won this session)

- Never raw `git commit`/`git push --force` — hooks block; use
  `ccs commit --direct` and `--force-with-lease` (+ delete-then-push for tags)
- File removals: safe Trash-move (`mv → ~/.Trash`), never raw rm (user rule)
- filter-repo: delete stale `.git/filter-repo/already_ran` first
- ROAD-084 auto-fix "mark completed" is WRONG (watchdog unfinished) — do not run
- Scanner "critical" AWS keys = AKIAIOSFODNN7EXAMPLE placeholders (false positives)
- Test invocation quirk (cmd handlers): `rotateCmd.ParseFlags([...])` then
  call the run function directly — cobra child `.Execute()` runs root help instead
- Desktop tests: `(cd desktop && bun run test ...)` subshell only; `tsc -b`
  for types (never `-b --noEmit` — TS6310)

## Also open (lower priority)

- ROAD-096 launch program: Show HN / r/Cloudflare / awesome-lists /
  vs-Wrangler page (needs the user's voice for posts)
- FEAT-017 (first-run wizard), FEAT-011/014/018 reliability loop (ROAD-099)
- 15 unreviewed September design docs → /independent-review (doc-integrity.md)
- cosmolabs.org/cosmoflare product page + Homebrew tap (ROAD-096 seeds)

## Files to read first

- docs/audit/latest/brief.md (project context, ~2 min)
- cmd/output.go (the presenter + out* helpers)
- cmd/account.go (the conversion exemplar)
