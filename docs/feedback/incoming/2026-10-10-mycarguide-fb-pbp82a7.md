---
ulid: 01M4JE1QH8JZX9M7BD2KBP82A7
title: Dead functions/ detector — Pages ignores functions/ beside _worker.js; endpoints silently never run
type: feature
status: duplicate
priority: medium
complexity: ""
from_project: MyCarGuide
from_path: /Users/gabstudio/PROJECTS/MyCarGuide
to_project: cosmoflare
to_target: project
created: "2026-10-10T12:14:39.144347+04:00"
updated: "2026-10-10T12:14:39.144347+04:00"
suggested_conversion: feature
converted_to: null
related_issues: []
brainstorm_ref: null
session: 19
suggested_workflow: []
response:
  acknowledged: null
  acknowledged_by: null
  started: null
  implemented: null
  rejected: null
  rejection_reason: null
  notes: ""
duplicate_of: /Users/gabstudio/PROJECTS/cosmoflare/docs/feedback/incoming/2026-10-10-mycarguide-dead-functions-detector--pages-ignor.md
---

# FB-pBP82A7: Dead functions/ detector — Pages ignores functions/ beside _worker.js; endpoints silently never run

**What happened:** MyCarGuide's `/api/subscribe` (email capture for the coming-soon list) lived in `functions/api/subscribe.ts`. The Astro Cloudflare adapter emits `dist/_worker.js`, and Cloudflare Pages IGNORES the functions/ directory entirely when _worker.js exists. Result: the endpoint NEVER executed in production — every POST fell through to the homepage HTML (HTTP 200, full page). This shipped unnoticed through multiple deploys; every submission since launch was silently lost. Found only by a manual live smoke test (curl -X POST returning HTML instead of JSON) on 2026-10-10.

**Why it matters:** this is a silent-failure class, not a one-off. Any Astro-adapter (or _worker.js) project that keeps a functions/ dir — or accidentally creates one — has handlers that look deployed and are dead. Nothing errors at build or deploy time; Cloudflare documents the behavior but nothing surfaces it per-project. A fleet audit tool should catch exactly this: code that LOOKS live and isn't.

**Proposed:** an audit check (and optionally a deploy-time hook) for Pages projects: if `functions/` contains handler files AND the build output contains `_worker.js` (or `dist/_worker.js`), flag every function route as SHADOWED, listing the dead endpoints and the fix (port to framework routes — Astro src/pages/api, or merge into the worker). Bonus generalization: diff declared routes (functions/ + framework routes) against the deployed project's actual behavior on a sample of endpoints.

**Repro/evidence:** pre-fix: `curl -X POST https://mycar.guide/api/subscribe -d '{}'` → 200 + homepage HTML. Post-fix (ported to `src/pages/api/subscribe.ts`): 400 JSON validation error, OPTIONS 204. Commit MyCarGuide 2026-10-10 "fix(web): subscribe endpoint never executed — functions/ ignored beside _worker.js".

**Priority:** high — data-loss-class silent failure, likely present in other CosmoLabs Pages projects.

