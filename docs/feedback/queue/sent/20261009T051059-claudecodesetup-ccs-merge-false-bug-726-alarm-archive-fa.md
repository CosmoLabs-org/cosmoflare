---
ulid: 01M4F3D87JVCCJQWC3AEM2MW89
from: cosmoflare
type: bug
severity: medium
title: 'ccs merge: false BUG-726 alarm, archive fails on gitignored GOrchestra, ''not ready'' with no reason after PASSED review'
created: "2026-10-09T05:10:59.058082+04:00"
---

# ccs merge: false BUG-726 alarm, archive fails on gitignored GOrchestra, 'not ready' with no reason after PASSED review

**What happened (cosmoflare, 2026-10-09, 7 merges):** three `ccs merge` defects, each reproduced more than once.

1. **False BUG-726 ancestry alarm.** After an auto-rebased GLM merge, `ccs merge` printed `❌ BUG-726: merge reported success but branch tip 4b176c5 is not an ancestor of HEAD`. The checked tip was the post-merge bookkeeping commit (`chore: record post-merge session bookkeeping (BUG-1219)`), which is created in the worktree *after* the merge. The merged content (rebased 333ffb2) was on master. Seen on agents 0326 and 0327.
2. **Archive commit always fails when `GOrchestra/` is gitignored.** `❌ Archive NOT committed ... exit status 1` on every merge. cosmoflare's `.gitignore:225` ignores `GOrchestra/`, so `git add GOrchestra/sessions` can never succeed. The printed repair command cannot work either.
3. **"not ready to merge" right after a PASSED review.** `ccs verify-worktree NAME --approve --score 9 --issues 0` printed PASSED at SHA cf8d0b7, then `ccs merge NAME` immediately printed `❌ worktree NAME not ready to merge` with no reason. Earlier the same day the reason shown was `Review: ⚠ stale (was 40b805e, HEAD is d020d47)`. A re-run of verify then merge worked once, failed once.

**Why it matters:** false ❌ lines train operators and agents to ignore real failures. Defect 3 combined with a `;`-chained `ccs kill` removed a worktree whose merge had not landed (the commit was recovered from the object store and merged as a plain branch, so the gate was skipped).

**Proposed fix:** (1) run the ancestry check against the pre-bookkeeping merged tip, or create the bookkeeping commit before merging. (2) Detect an ignored archive path, skip with an info line, and do not print a ❌. (3) Always print the refusal reason, and make verify-worktree stamp the SHA that merge will check (or have merge re-validate an equivalent tree). Also make `ccs kill` refuse a worktree whose branch tip is not an ancestor of master unless `--force`.

**Priority:** medium.
