# Action Plan — what to do next

Copy-paste next-steps, prioritized DD-4: data-integrity > drift > quality > growth. Each action carries its source dimension.

## 🔴 Now (blocking / data-integrity)

1. **Fix 4 sync-engine data-loss bugs before any public user touches the CLI** (agent-2-core-logic.md)
   - Direction-aware deletes — `pkg/cosmoflare/sync.go:425` (down-sync must `os.Remove(op.LocalPath)`, never `DeleteRemoteObject`)
   - Exclude-protect delete loops — `pkg/cosmoflare/sync.go:275`
   - Split 30s API client from timeout-free S3 client — `pkg/cosmoflare/client.go:113`
   - `context.WithoutCancel` multipart aborts — `pkg/cosmoflare/upload.go:181`
   Red-first TDD each; regression test asserting down-sync emits zero remote deletes.

2. **Stop `make dist` from taring docs/ into release archives** (agent-5-distribution.md)
   - Edit `Makefile:97` — remove `../docs/` from the dist target.

3. **Fix `auth rotate --revoke-old` revoking the NEW token** (agent-9-infrastructure.md)
   - Capture old token before overwrite — `cmd/auth.go:227-239`; implement or remove the subcommand (`cmd/auth.go:479`).

4. **Close 2 merged-but-open issues after human scope verification** (work-completion.md)
   $ ccs issues update FEAT-025 --status closed   # merge 562a602 verified by agent-15
   $ ccs issues update FEAT-019 --status closed   # merge 4f512ef verified by agent-15

5. **Rebuild the empty issues index** (agent-12-roadmap-health.md)
   $ ccs issues index-rebuild    # docs/issues/index.yaml is `issues: []` despite 81 files

6. **Accessibility CRITICAL: app-level live region** (design-quality.md / agent-18-accessibility.md)
   - Always-mounted `aria-live=polite` region at `desktop/src/App.tsx` (regions currently unmount on tab switch, SC 4.1.3).

## 🟡 Soon (drift / staleness)

7. **Publish v0.26.0 (assets staged in dist/upload/), backfill v0.23–v0.25** (agent-5-distribution.md)
   $ gh release create v0.26.0 dist/upload/* --notes-from-tag
   Then add a post-release guard: `gh release view <tag> || exit 1` in the release SOP.

8. **Purge internal artifacts before any visibility flip** (agent-10-documentation.md)
   - git rm --cached docs/conversation-transcripts/ docs/sessions/transcripts/ GOrchestra/**/*.log; gitignore; filter-repo history purge or private mirror. The public flip (README's `go install` path) is blocked on this.

9. **Fix 4 broken three-tier chains** (doc-integrity.md)
   - Add `plan_ref:` to the 4 brainstorms: cf-api-knowledge-layer, domain-center-residuals, quota-limit-view, ratelimit-traffic-classes (2026-09-09/10 batch).

10. **Reconcile ROAD-084/085 — do NOT run the auto-fix** (work-completion.md, agent-12-roadmap-health.md)
    - ROAD-084: watchdog half genuinely unfinished → relink to IDEA-038, keep captured (auto-fix would erase real scope).
    - ROAD-085: purge landed → mark completed_in, re-scope to transcript/fixture residue.

11. **Brand unification sweep** (agent-6, agent-7, agent-5, agent-4)
    - `pkg/cosmoflare/types.go:2` package doc ("Package r2go2" — the pkg.go.dev landing text), `docs/README.md:1` ("CosmoDev-R2Go2"), `.version-registry.json:4`, Makefile binary names, SECURITY.md URLs, GETTING_STARTED.md install URL (dead repo).

12. **Review 15 unreviewed September design docs** (doc-integrity.md — detect-and-point, DD-6)
    $ /independent-review docs/brainstorming/2026-09-09-cf-api-knowledge-layer.md docs/brainstorming/2026-09-09-domain-center-residuals.md docs/brainstorming/2026-09-09-quota-limit-view.md docs/brainstorming/2026-09-10-cf-limits-awareness-layer.md docs/brainstorming/2026-09-10-ratelimit-traffic-classes.md docs/brainstorming/2026-09-11-d1-depth.md

13. **Fix documented-but-fake `cosmoflare dev`** (agent-7-api-design.md)
    - Implement the proxy or deprecate and correct USAGE.md — agent-first UX dies when help lies.

## 🟢 Later (quality / growth)

14. **Flip repo public + launch program** — AFTER #8 purge; then Show HN / r/Cloudflare / awesome-lists, vs-Wrangler comparison page (upgrade-plan.md Phase 3; agent-4, agent-6).
15. **Enrich the 23 flagged plans** — `ccs prompts enrich <plan> --apply` (doc-integrity.md).
16. **Quality wave** — output presenter interface (618 branches), cmd/ coverage 44.6%→60%, RunE migration, typed daemon wire contract + TS codegen (upgrade-plan.md Phase 2).
17. **Security polish pack** — constant-time token compare, Tauri CSP, silent token prompt, 0600-first config writes, url.PathEscape CopySource (upgrade-plan.md Phase 2.8).
18. **Desktop design pass** — light theme, notification severity encoding, motion micro-transitions (design-quality.md).
19. **Link 16 unlinked open FEATs during next /triage** (agent-12-roadmap-health.md).
