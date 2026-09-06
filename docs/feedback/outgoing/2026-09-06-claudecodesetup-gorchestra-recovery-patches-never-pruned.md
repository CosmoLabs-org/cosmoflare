---
ulid: 01M1W0DTGERVVWR2TD5FTBCHWG
title: GOrchestra recovery patches never pruned + tracked in git — ~1.5GB machine-wide, scanner-noise class BUG-761
type: bug
status: pending
priority: high
complexity: medium
from_project: cosmoflare
from_path: /Users/gabstudio/PROJECTS/cosmoflare
to_project: ClaudeCodeSetup
to_target: project
created: "2026-09-06T22:41:54.190106+04:00"
updated: "2026-09-06T22:41:54.190106+04:00"
suggested_conversion: bug
converted_to: null
related_issues: []
brainstorm_ref: null
session: 38
suggested_workflow:
  - brainstorming
  - implementation
response:
  acknowledged: null
  acknowledged_by: null
  started: null
  implemented: null
  rejected: null
  rejection_reason: null
  notes: ""
---

# FB-pTBCHWG: GOrchestra recovery patches never pruned + tracked in git — ~1.5GB machine-wide, scanner-noise class BUG-761

# SUMMARY (what happened)

`ccs kill` writes `GOrchestra/sessions/{name}/recovery.patch` (tools/ccsession/cmd/kill.go:741, `recoverBranchContent`) as the only surviving copy of a killed worktree's unmerged commits — then nothing ever prunes it, and nothing gitignores the directory. Across 53 projects on this machine the accumulation is ~1.5GB, and in 10+ repos the files are TRACKED in git, committing 229MB-class blobs into every clone and checkout forever.

Machine-wide census (2026-09-06, `du` + `git ls-files` per repo):

| Project | GOrchestra/sessions size | Files tracked in git |
|---|---|---|
| WebFetcher | 299MB | 296 |
| cosmoflare | 233MB | 555 (untracked same day — reference fix below) |
| SmokeSig | 202MB | 283 |
| Cerebro | 121MB | 351 |
| LFGoMail | 111MB | 0 |
| CountCountries | 101MB | 252 |
| CosmoQuickView | 75MB | 177 |
| Churches-app | 70MB | 39 |
| ClaudeCodeSetup (self) | 62MB | 3,986 |
| CosmoKit | 15MB | 1,204 |
| + 43 more projects | remainder | mixed |

Composition detail from cosmoflare: 84 recovery.patch files = 229MB of the 233MB dir (98%); the rest is small session metadata (.ccsession.json, HISTORY.md, session.json, cleanup.json). Also note 78 of ~106 session dirs report status "working" — the known BUG-076 zombie-status pattern (agents die without flipping status), which makes the attic look alive and deters cleanup.

# MOTIVATION (why it matters)

1. **Scanner noise masks real findings.** The patches embed AWS example-key patterns (`AKIAIOSFODNN7EXAMPLE` from test fixtures in the diffed trees). Tracked patches fire recurring CRITICAL secrets-scanner findings — cosmoflare's 2026-08-31 audit traced recurring criticals to exactly this (ROAD-85/IDEA-046 class). A false critical per scan trains operators to ignore criticals.
2. **Clone/checkout bloat, forever.** Tracked blobs stay in history even after untracking (cosmoflare now faces a filter-repo decision before going public because 234MB of blobs remain in history). Same class of recurring-tree-dirt as BUG-761 (.version-registry.json).
3. **The retained data is effectively dead.** A recovery.patch is written ONLY when a worktree is killed with unmerged commits (merged-first kills write none — verified 2026-09-06: 4 merged agents left zero patches). The patches are therefore abandoned pre-merge work: failed runs and discarded experiments. No salvage flow reads the archive — `ccs glm-agent salvage` operates on live sessions. In practice these patches are write-only.
4. **CCS dogfoods the bug.** ClaudeCodeSetup itself carries 3,986 tracked session files — the tool's own repo is among the worst offenders, so every contributor clones the noise.

# SOLUTION (how we fix it — all repos, three layers)

**Layer 1 — stop the bleeding (ccsession code):**
- `recoverBranchContent` (kill.go:665): after writing recovery.patch, record the write timestamp (file mtime suffices; or a `created` field in the session's cleanup.json).
- New prune pass in `ccs sync` and at the start of `ccs kill`: delete every `GOrchestra/sessions/*/recovery.patch` older than N days (default 30, configurable via ccs config). Keep the small metadata files — they are kilobytes and carry the audit trail.
- `ccs init`/`ccs spawn`/project scaffolding: add `GOrchestra/sessions/` to the default .gitignore block it writes (next to the existing `go.work` block).

**Layer 2 — untrack what's already committed (one-time sweep):**
- New command or `ccs sync` guidance step (mirror the BUG-761 .version-registry.json untrack): detect `git ls-files GOrchestra/sessions/` non-empty → `git rm -r --cached GOrchestra/sessions/` + append gitignore entry + conventional commit. Per-repo this is exactly what cosmoflare shipped 2026-09-06: `chore(repo): untrack GOrchestra session attic and stray binaries (ROAD-085)` — 558 files, 3,596,566 lines removed from tracking, files kept on disk.
- For repos that must stay lean in history too (public-facing: cosmoflare pre-launch, CosmoKit as a published package), document the `git filter-repo --path GOrchestra/sessions --invert-paths` step as a deliberate, user-approved operation (rewrites SHAs; coordination required).

**Layer 3 — one-time disk prune (operator sweep):**
- `find */GOrchestra/sessions -name recovery.patch -mtime +30 -delete` across ~/PROJECTS — recovers ~1.4GB. I already ran the equivalent manually for cosmoflare (84 files, 229MB → 4.1MB dir) after verifying zero live sessions and that patches are the only copy of work nobody re-applies. A `ccs` command (`ccs cleanup sessions --prune-patches --older-than 30d`?) should replace the manual find.

# REPRO / EVIDENCE

- Writer: `tools/ccsession/cmd/kill.go:741` (`os.WriteFile(patchPath, patchData, 0644)`); guard at :725 skips patches for merged branches — confirming patches == unmerged-at-kill work only.
- Census command: `for d in ~/PROJECTS/*/GOrchestra/sessions; do du -sh; git -C <repo> ls-files GOrchestra/sessions/ | wc -l; done`
- cosmoflare reference fix: untrack commit (ROAD-085) + patch deletion 2026-09-06 (233MB → 4.1MB, chat transcripts in docs/conversation-transcripts/ and docs/sessions/transcripts/ untouched — those are NOT part of this cleanup scope).

# PRIORITY

high — not data loss, but it (a) actively degrades secrets-scanner signal in 10+ repos, (b) compounds daily with every agent kill, (c) blocks clean public launch for cosmoflare until the history decision is made, and (d) is dirt-cheap to fix relative to the recurring cost every repo pays per clone.

## Suggested Workflow

1. brainstorming
2. implementation

