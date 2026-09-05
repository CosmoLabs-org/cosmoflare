# Action Plan — what to do next

> Ranked copy-paste next-steps from the 2026-08-31 audit (DD-4 priority: data-integrity > drift > quality > growth). Every action names its source dimension.

## 🔴 Now (blocking / data-integrity)

1. **Fix the multipart silent-corruption path** — uploads can store truncated objects with no error (work: `cmd/sync.go` + `upload.go`; source: agent-2 via risk-map.md)
   - `pkg/cosmoflare/upload.go:207` and `multipart.go:493`: abort on `io.EOF/ErrUnexpectedEOF` before `uploadedBytes == size`; add a short-reader regression test
2. **Paginate sync's remote listing** — `--delete` on buckets >1000 objects destroys local files that exist remotely (source: agent-2 via risk-map.md)
   - `cmd/sync.go:428`: loop on `result.NextToken`
3. **Fix the data races in `internal/interactive`** — they fail CI's `-race` gate and block every release (source: agent-5 via risk-map.md)
   - Reproduce: `go test -race ./internal/interactive/`; isolate the package-level shared state; guard or per-test it
4. **Sweep the dead `CosmoDev-R2Go2` module path** — every built binary reports `version dev` (source: agents 5+9 via risk-map.md)
   - `Makefile:19`, `Dockerfile:30-32`, `.github/workflows/release.yml:62-64,87`, `install.sh:16-20`, `install.ps1:11-12`, `install-menu.sh:159,178`
5. **Decide FEAT-008 (event bus) and clean the abandoned chain** — 3 docs promise a feature with zero code, 71 days stalled (source: work-completion.md + doc-integrity.md)
   - Re-scope to "bridge `TriggerAlert` → existing sseHub" (the standalone bus is obsolete now the hub shipped), then:
   - `$ ccs prompts stale --cleanup`

## 🟡 Soon (drift / staleness)

6. **Repair the doc chains** — 7 orphaned back-links cost 25 points of the 47 doc-integrity score (source: doc-integrity.md)
   - Add `plan_ref:` to the 7 brainstorms listed in doc-integrity.md
   - `$ ccs timestamps backfill` (3 date-only stamps)
7. **Deduplicate the roadmap** — 4 double-imported pairs from the 2026-06-20 batch (source: work-completion.md)
   - Close ROAD-074/081/082/083 as duplicates (keep ROAD-080 — FEAT-008 links it)
   - Regenerate `docs/roadmap/index.yaml` (ROAD-035 title mismatch, dropped priorities)
8. **Review the unreviewed design docs** (source: doc-integrity.md; DD-6 — the audit points, it never stamps)
   - `$ /independent-review docs/brainstorming/2026-06-21-inprocess-event-bus.md docs/brainstorming/2026-07-01-live-metrics-producer.md` — newest chains first, then the remaining 22 listed in agent-14-doc-integrity.json
9. **Untrack the repo attic** — 234MB recovery patches + tracked binaries; scanner noise masks real findings (source: agent-1/9 via risk-map.md)
   - `git rm -r --cached GOrchestra/sessions/` + gitignore; untrack `simple-setup`, `test-setup`, `build/r2go2`
10. **Regenerate the README from reality** — 13 shipped services marked "Planned"; broken quick-start syntax (source: agents 4/6/10/12 — found by 4 agents independently)

## 🟢 Later (quality / growth)

11. **Ship v0.18.0 with a release smoke gate** — first GitHub Release ever; verify checksums + `--version` after publish (source: upgrade-plan.md Phase 1)
12. **Write the desktop `cf-*` stylesheet + screen-reader pass** — the app ships unstyled; health state invisible to everyone (source: design-quality.md; 3 a11y HIGHs)
13. **Wire the guardrails** — `.cosmoflare.yaml` safety config is parsed but never enforced (source: agent-2; the defensible-competitive-niche gap per agent-4)
14. **Daemon error contract v2** — map typed errors to 400/401/404/429/502 `{error, code}` (source: agent-7)
15. **Positioning pivot ADR** — respond to Cloudflare's `cf` CLI + official MCP servers; re-anchor on neutrality, agent trust, Go embeddability (source: agent-4)
16. Full phased roadmap: upgrade-plan.md (Phases 0-3)

## Dimensions not covered here

Work Completion scored N/A (zero possibly-done issues — the healthy outcome). Its roadmap reconciliation and stalled-plan cleanup are items 5 and 7 above. CosmoKit Synergy skipped (cosmokit not installed).
