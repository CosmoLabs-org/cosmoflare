---
title: "R2Go2 Product Vision and Integration Design"
created: "2026-03-28T10:00:00-03:00"
status: COMPLETE
deliverables:
  - BR-01: Product vision defining 3-tier Cosmoflare architecture and library-first design principle
tags:
  - product-vision
  - architecture
  - library-design
  - cloudflare-platform
schema_version: 1
---

# R2Go2 Product Vision and Integration Design

**Date**: 2026-03-28
**Status**: COMPLETE
**Participants**: GΛB + Claude

## Deliverables

When this brainstorming session is complete, the following artifacts will be produced:

### Documents
- [x] This brainstorming doc (`docs/brainstorming/2026-03-28-r2go2-product-vision-and-integration.md`)
- [ ] Design doc (`docs/plans/2026-03-28-r2go2-library-cli-design.md`) — formal architecture spec
- [ ] Implementation plan (`docs/plans/2026-03-28-r2go2-implementation-plan.md`) — task-by-task build sequence
- [ ] Updated `CLAUDE.md` with product vision (already done)
- [ ] Rewritten `README.md` reflecting reality + new vision (Phase 2)
- [ ] Rewritten `USAGE.md` as agent reference doc (Phase 2)

### Code Artifacts (Phase 1+)
- [x] `pkg/r2go2/` — public Go library (client, storage, upload, download, config, types, options, errors)
- [x] `.r2go2.yaml` schema + parser
- [x] Real S3 API integration (re-enable from `internal/api_disabled/`)
- [x] Cache policy engine (rule-based Cache-Control header management)
- [x] Workers management module (Phase 3)

### Roadmap Items to File
- [ ] ROAD: Library extraction into `pkg/r2go2/`
- [ ] ROAD: Cloudflare Workers support
- [ ] ROAD: Advanced edge caching strategy engine
- [ ] ROAD: Project config (`.r2go2.yaml`) implementation
- [ ] ROAD: CCS `r2` subcommand integration (deferred)

### Bugs to Fix First (from audit)
- BUG-001 (speed calc), BUG-002 (PersistentPreRun), BUG-004 (keyboard), BUG-007 (placeholder API), BUG-010 (LICENSE), and 1 more (printErrorAndExit)

## Context

Following a comprehensive 10-agent audit (score: 46.5/100), the project needs to evolve from a polished UX shell with stubbed APIs into a fully functional, open-source product. This brainstorming session defines the product vision, architecture, and integration strategy.

## Decisions Made

### Decision 1: Three-Tier Product Architecture
**Decided**: R2Go2 is three things:
1. **Go library** — clean public API, importable by any Go project (OpenCode, Codex, anyone)
2. **Standalone CLI** — open-source, installable, agent-friendly (LLMs can call it from terminal)
3. **CCS subcommand** (future) — `ccs r2` wrapping the CLI, like GoRalph

**Rationale**: Library-first ensures ecosystem independence. CLI on top ensures human + agent usability. CCS integration comes last as a wrapper, not a dependency.

### Decision 2: Agent-First UX Design
**Decided**: The CLI must be equally usable by AI agents (Claude, GLM 5.1, etc.) as by humans.

**Requirements**:
- Rich `--help` on every command (agents read help to discover usage)
- Comprehensive `USAGE.md` as the agent reference document
- `--json` output on all commands (agents parse JSON, not tables)
- Clear error messages with actionable fix suggestions
- Predictable, consistent command structure
- Deterministic exit codes for scripting
- No interactive prompts in non-TTY mode (agents pipe commands)

**Rationale**: AI agents are a primary user class. They learn tools by reading `--help` and parsing JSON output. R2Go2 in a user's PATH means any LLM session can manage R2 storage.

### Decision 3: Open Source, Ecosystem Independent
**Decided**: R2Go2 will be open-sourced (MIT). No CCS, GoRalph, or CosmoLabs-internal dependencies in the core library or CLI. Any developer, any tool, any AI agent can use it.

**Rationale**: Maximizes adoption. CCS-specific features go in the CCS wrapper layer, never in R2Go2 core.

### Decision 4: CCS Integration Deferred
**Decided**: CCS subcommand (`ccs r2`) comes later. No feedback to CCS from this project for now. Build R2Go2 standalone first, then integrate.

**TODO for later**:
- [ ] Add R2Go2 as CCS core requirement (like GoRalph)
- [ ] Build `ccs r2` subcommand wrapping the CLI
- [ ] Send cross-project feedback to CCS when ready

---

## Open Questions

### Q1: Primary Use Cases — DECIDED
**Answer**: All of the above, with **asset management as the primary focus**.

**Core value proposition**: R2Go2 is the intelligent layer between projects/apps and Cloudflare R2. It enables:

1. **Direct bucket interaction** — Projects upload/retrieve assets, serve users via their apps
2. **Security and resource governance** — R2Go2 enforces proper use of storage resources
3. **Per-project bucket configuration** — Cache-Control headers, lifecycle policies, access rules
4. **CDN/edge optimization** — Smart cache duration management (1 month for changing content, years for immutable assets), leveraging Cloudflare's global edge network
5. **Cross-session persistence** — Agents save/retrieve data across sessions
6. **Inter-project data sharing** — R2 as a shared data bus between projects

**The key insight**: R2Go2 isn't just a storage CLI — it's an **intelligent storage management layer** that understands caching strategy, content lifecycle, and resource governance. It makes the right decisions about how content should be stored and served.

### Q2: Library API Surface — DECIDED
**Answer**: Three-layer API (high-level convenience, mid-level client, low-level S3). See Decision Q3b.

**Status**: RESOLVED by Decision Q3b

### Q3: Configuration Architecture — DECIDED
**Answer**: Both global profiles AND per-project config.

**Two-layer config model**:

**Layer 1: Machine-level (`~/.r2go2/config.yaml`)**
- Authentication: API tokens, access keys, account IDs
- Profiles: production, staging, dev (per Cloudflare account)
- Global defaults: default region, timeout, retry settings
- Installed once per machine, used by all projects
- **Never checked into git** (contains secrets)

**Layer 2: Project-level (`.r2go2.yaml` in project root)**
- Which bucket(s) this project uses
- Cache-control policies per content type (immutable = years, dynamic = months)
- Upload rules: max size, allowed content types, naming conventions
- CDN/custom domain mapping
- Lifecycle policies (auto-delete after N days, transition to cheaper storage)
- **Checked into git** — every developer and AI agent inherits the same rules

**Resolution order**: Project config > Environment variables > Machine profile > Defaults

**The key benefit**: Install R2Go2 once, configure credentials once. Every project just needs a `.r2go2.yaml` and it's connected to R2 with smart defaults, proper caching, and governance rules. An AI agent in any project reads the project config and knows exactly how to use storage for that project.

### Decision 5: Per-Project Config Location
**Decided**: `.r2go2.yaml` in the project root.

- Follows dotfile convention (`.gitignore`, `.editorconfig`, etc.)
- Auto-discoverable by CLI walking up directory tree
- **Checked into git** — this IS the contract for how the project uses R2
- Contains **no secrets** — only bucket names, cache policies, content rules, lifecycle settings
- Auth always comes from machine-level `~/.r2go2/config.yaml` (gitignored)

**Example `.r2go2.yaml`**:
```yaml
# R2Go2 project configuration
profile: production          # which ~/.r2go2 profile to use
bucket: my-app-assets        # default bucket for this project

cache:
  immutable: "max-age=31536000, immutable"   # fonts, hashed JS/CSS
  static: "max-age=2592000"                   # images, documents (30 days)
  dynamic: "max-age=3600"                     # API responses, generated content
  rules:
    - pattern: "*.woff2"
      policy: immutable
    - pattern: "images/*"
      policy: static
    - pattern: "uploads/*"
      policy: dynamic

upload:
  max_size: "100MB"
  allowed_types: ["image/*", "font/*", "application/pdf", "text/css", "application/javascript"]

lifecycle:
  - prefix: "tmp/"
    expire_days: 7
  - prefix: "logs/"
    expire_days: 90
```

### Q3b: Library API Abstraction Level — DECIDED
**Answer**: Yes, three-layer API. Both high-level and low-level, layered.

**Design**:

```go
// High-level (reads .r2go2.yaml, applies smart defaults)
r2go2.Upload("logo.png")                    // bucket + cache policy from project config
r2go2.Download("logo.png", "./local.png")
r2go2.List()

// Mid-level (explicit, but still ergonomic)
client := r2go2.NewClient(r2go2.WithProfile("production"))
client.Upload("my-bucket", "logo.png", r2go2.WithCacheControl("immutable"))

// Low-level (raw S3 SDK access for power users)
s3client := client.S3()
s3client.PutObject(ctx, &s3.PutObjectInput{...})
```

- **High-level**: For agents and quick scripts. Reads `.r2go2.yaml`, applies cache policies automatically.
- **Mid-level**: For app developers. Explicit control but ergonomic Go API.
- **Low-level**: For tool builders. Direct S3 SDK escape hatch.

**Rationale**: Different users need different levels of control. An AI agent wants `r2go2.Upload("file.png")` and done. A backend developer building a media pipeline wants explicit bucket/cache/metadata control. A tool builder extending R2Go2 wants raw S3 access.

---

### Decision 6: Library Structure — Single Repo with pkg/
**Decided**: `pkg/r2go2/` in this repo, not a separate module.

**Rationale**: One repo = one PR = simpler development. CLI and library evolve together. Standard Go pattern (Kubernetes, Docker, Terraform all use `pkg/`). Can split later if adoption warrants it.

**New project structure**:
```
pkg/r2go2/              ← PUBLIC library (importable by any Go project)
  client.go             ← R2Client interface + NewClient factory
  bucket.go             ← Bucket operations
  object.go             ← Object operations
  upload.go             ← Upload with progress + smart cache headers
  download.go           ← Download operations
  config.go             ← Project config (.r2go2.yaml) + machine config
  types.go              ← Shared types (Bucket, Object, UploadResult, etc.)
  options.go            ← Functional options (WithProfile, WithCacheControl, etc.)
  errors.go             ← Custom error types (R2NotFoundError, R2AuthError, etc.)

internal/               ← PRIVATE (CLI internals only)
  tui/                  ← TUI dashboard (CLI-only feature)
  interactive/          ← Setup wizards, themes, animations
  cli/                  ← Batch manager, progress bars, visual effects

cmd/                    ← CLI commands (import from pkg/r2go2/)
```

**Import path for external projects**:
```go
import "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"
```

---

### Decision 7: Build Order — APPROVED
**Decided**: Phased build plan approved.

**Phase 0 — Critical bug fixes (1 day)**
- 6 one-liner fixes from audit (speed calc, PersistentPreRun, keyboard off-by-one, printErrorAndExit, LICENSE, install.sh binary name)

**Phase 1 — Real API + Library extraction (1-2 weeks)**
- Extract `R2Client` interface into `pkg/r2go2/`
- Re-enable S3 implementation from `api_disabled/`
- Wire up real Cloudflare R2 calls (list, create, delete, upload, download)
- Three-layer API (high-level / mid-level / low-level S3)
- `.r2go2.yaml` project config with cache policy engine
- Machine-level `~/.r2go2/config.yaml` profiles

**Phase 2 — Agent-friendly CLI polish (1 week)**
- Rewrite `--help` on every command for agent consumption
- Ensure `--json` works on every command
- Rewrite `USAGE.md` as definitive agent reference
- Strip README to reality
- Deterministic exit codes
- No interactive prompts in non-TTY mode

**Phase 3 — Advanced features (ongoing)**
- Multipart uploads with resume
- Sync/mirror command
- Pre-signed URL generation
- Lifecycle policy management via CLI
- TUI dashboard wired to real data
- **NEW: Cloudflare Workers support** (see Decision 8)
- **NEW: Advanced edge caching strategies** (see Decision 9)

**Phase 4 — CCS integration (deferred)**
- `ccs r2` subcommand wrapping the CLI
- R2Go2 as CCS core requirement (like GoRalph)
- Cross-project feedback to CCS

### Q5: New Features Beyond Current Roadmap — IN PROGRESS

**Confirmed scope additions**:
- **Full Cloudflare Workers support** — manage Workers that serve content from R2
- **Advanced edge caching strategies** — year+ cache for immutable static content
- **R2 + Workers as integrated pair** — the standard Cloudflare app pattern

### Decision 8: Workers Lives Inside R2Go2 — DECIDED
**Decided**: Option A — keep the R2Go2 name, expand scope to include Cloudflare Workers.

**Rationale**:
- Workers + R2 is the standard Cloudflare app pattern (Workers serve content from R2)
- AI agents working on a project shouldn't context-switch between tools
- Single install, single config (`.r2go2.yaml` covers both buckets and workers)
- The catchy "R2Go2" branding is valuable and stays
- Name becomes a slight misnomer but is still recognizable

**Scope**:
- `r2go2 worker list` — list deployed workers
- `r2go2 worker deploy` — deploy a worker (from local file or template)
- `r2go2 worker logs` — tail worker logs
- `r2go2 worker delete` — remove a worker
- `r2go2 worker bind` — bind R2 buckets, KV, D1 to a worker
- `r2go2 worker secret` — manage worker secrets
- TUI dashboard adds Workers section alongside Buckets

**`.r2go2.yaml` extension**:
```yaml
workers:
  - name: asset-server
    script: ./workers/asset-server.js
    routes:
      - "cdn.example.com/*"
    bindings:
      r2:
        - binding: ASSETS
          bucket: my-app-assets
    cache:
      browser_ttl: 31536000      # 1 year
      edge_ttl: 31536000          # 1 year at Cloudflare edge
      cache_everything: true
```

### Decision 11: Full Cloudflare Platform Coverage — DECIDED
**Decided**: R2Go2 expands to cover ALL Cloudflare developer platform services: R2, Workers, KV, D1, Pages, Queues, and future services.

**Rationale**: Same logic as Decision 8 — one install, one config, one learning curve. Developers using Cloudflare use multiple services together (R2 + Workers + KV is the standard trio). The `cloudflare-go` SDK already has clients for all services. Each new service is an additive subcommand.

**Service coverage by phase**:

| Phase | Service | CLI Commands | Library Package |
|-------|---------|-------------|-----------------|
| Phase 1 | **R2** (storage) | `r2go2 bucket`, `r2go2 object`, `r2go2 upload`, `r2go2 download` | `pkg/r2go2/storage.go` |
| Phase 3 | **Workers** (compute) | `r2go2 worker deploy/list/logs/delete/bind/secret` | `pkg/r2go2/worker.go` |
| Phase 3 | **KV** (key-value) | `r2go2 kv list/get/put/delete/namespaces` | `pkg/r2go2/kv.go` |
| Phase 4 | **D1** (SQL database) | `r2go2 d1 create/query/migrate/export/list` | `pkg/r2go2/d1.go` |
| Phase 4 | **Pages** (static hosting) | `r2go2 pages deploy/list/logs/delete` | `pkg/r2go2/pages.go` |
| Phase 5 | **Queues** (message queues) | `r2go2 queue create/list/delete/send/consume` | `pkg/r2go2/queue.go` |
| Future | **R2 Public Buckets** | `r2go2 public enable/disable/url` | — |
| Future | **Custom Domains** | `r2go2 domain add/remove/list` | — |
| Future | **Stream** | `r2go2 stream upload/list/capture` | — |

**Updated `.r2go2.yaml` full config**:
```yaml
profile: production

storage:
  bucket: my-app-assets
  cache:
    default: static
    rules:
      - pattern: "*.{woff2,ttf}"
        tier: immutable

workers:
  - name: asset-server
    script: ./workers/asset-server.js
    bindings:
      r2: [{binding: ASSETS, bucket: my-app-assets}]
      kv: [{binding: SESSIONS, namespace: user-sessions}]
      d1: [{binding: DB, database: my-app-db}]

kv:
  - name: user-sessions
    ttl: 86400
  - name: feature-flags

d1:
  - name: my-app-db
    migrations: ./migrations/

pages:
  - name: my-app-docs
    source: ./docs/dist
    custom_domain: docs.example.com

queues:
  - name: email-notifications
    max_batch_size: 10
  - name: image-processing
    max_batch_size: 5
```

**Updated library API**:
```go
// High-level convenience
r2go2.Upload("logo.png")
r2go2.KVGet("session-abc123")
r2go2.D1Query("SELECT * FROM users LIMIT 10")
r2go2.PagesDeploy("./dist")

// Mid-level client
client := r2go2.NewClient(r2go2.WithProfile("production"))
client.Storage().Upload(ctx, "bucket", "key", reader)
client.Workers().Deploy(ctx, "asset-server", script)
client.KV().Put(ctx, "namespace", "key", value)
client.D1().Exec(ctx, "my-db", "INSERT INTO users ...")

// Low-level SDK access
cfClient := client.Cloudflare()  // *cloudflare.API
s3Client := client.S3()          // *s3.Client
```

**Updated `pkg/r2go2/` structure**:
```
pkg/r2go2/
  client.go         — R2Client interface + factory
  storage.go        — R2 bucket + object operations
  upload.go         — Upload with progress + smart cache headers
  download.go       — Download operations
  worker.go         — Workers management (Phase 3)
  kv.go             — KV namespace operations (Phase 3)
  d1.go             — D1 database operations (Phase 4)
  pages.go          — Pages deployment (Phase 4)
  queue.go          — Queue management (Phase 5)
  config.go         — Project config + machine config
  cache.go          — 5-tier cache policy engine
  guardrails.go     — Validation, access scoping, quotas
  audit.go          — Audit logging
  types.go          — Shared types
  options.go        — Functional options
  errors.go         — Custom error types
  convenience.go    — Package-level convenience functions
```


### Decision 9: Five-Tier Rule-Based Cache Engine — DECIDED
**Decided**: R2Go2 ships an opinionated 5-tier caching engine driven by `.r2go2.yaml` rules. Defaults applied automatically on every upload. Per-upload override available via CLI flag.

**Cache tiers**:
| Tier | Cache-Control | When to use |
|------|--------------|-------------|
| **immutable** | `max-age=31536000, immutable` (1 year) | Hashed bundles, fonts, versioned assets — the "save massive request costs" tier |
| **long-static** | `max-age=2592000` (30 days) | Logos, marketing images, rarely-changing |
| **static** | `max-age=86400` (1 day) | Typical static content (default fallback) |
| **dynamic** | `max-age=3600` (1 hour) | Frequently-changing API responses |
| **no-cache** | `no-store` | Sensitive/per-user content |

**Rule engine in `.r2go2.yaml`**:
```yaml
cache:
  default: static                              # fallback for anything unmatched
  rules:
    - pattern: "*.{woff2,ttf,otf}"             # fonts → 1 year
      tier: immutable
    - pattern: "assets/[a-f0-9]{8,}*"          # hashed bundles → 1 year
      tier: immutable
    - pattern: "uploads/users/*"               # user uploads → dynamic
      tier: dynamic
    - pattern: "api/*"                         # API responses → no-cache
      tier: no-cache
```

**On upload**: R2Go2 walks rules in order, applies first match, sets `Cache-Control` header automatically. Zero config per upload.

**CLI override**: `r2go2 upload logo.png --cache=immutable` for one-off cases.

**Why this matters**: Cloudflare R2 charges per request. Edge cache for 1 year = ~1 R2 hit per edge location per year per asset. At scale this is the difference between $5/month and $5000/month in request costs. R2Go2 makes this the default behavior, not a setting users have to remember.

### Decision 10: Four-Pillar Security & Governance — DECIDED
**Decided**: All four guardrail categories, with access scoping as the most critical.

**Pillar 1: Upload Validation**
- Reject uploads violating `.r2go2.yaml` rules before they hit R2
- Enforced checks: file size limits, allowed content types, legal path patterns, naming conventions
- Agents and developers can't bypass — validation happens at the library layer
- Clear error messages: `"Upload rejected: file size 250MB exceeds project limit of 100MB (see .r2go2.yaml → upload.max_size)"`

**Pillar 2: Quota / Cost Guardrails**
- Track per-project storage usage (bucket size, object count, request volume)
- Configurable budgets in `.r2go2.yaml`:
  ```yaml
  quotas:
    storage: "10GB"              # warn at 80%, block at 100%
    requests_monthly: 100000     # R2 request budget
    upload_max_single: "500MB"
  ```
- `r2go2 quota` command shows current usage vs limits
- Optional: `--dry-run` flag on upload to check if it would exceed quota without actually uploading

**Pillar 3: Access Scoping (most critical)**
- `.r2go2.yaml` declares which buckets the project can touch
- Machine credentials may have access to 50 buckets, but the project can only see/modify the ones declared in its config
- Prevents accidental cross-project damage by agents or developers
- Enforced at library layer: `r2go2.List()` only returns scoped buckets, `r2go2.Upload()` rejects if target bucket isn't in scope
- Example:
  ```yaml
  access:
    buckets:
      - name: my-app-assets        # read/write
        mode: rw
      - name: my-app-logs          # write-only
        mode: w
      - name: shared-templates     # read-only
        mode: r
  ```

**Pillar 4: Audit Logging**
- Every upload, delete, and significant operation logged locally
- Log format (JSON for agent parsing):
  ```json
  {
    "timestamp": "2026-03-28T15:30:00Z",
    "action": "upload",
    "bucket": "my-app-assets",
    "key": "images/logo.png",
    "size": 24576,
    "cache_control": "max-age=2592000",
    "source": "agent:claude-opus-4-7",
    "project": "CosmoDev-R2Go2"
  }
  ```
- Optional push to R2 itself (audit log bucket)
- `r2go2 audit log` / `r2go2 audit search --action=delete --since=24h`

**Rationale**: These four pillars make R2Go2 genuinely "intelligent" — not just a thin S3 wrapper but a governance layer that prevents the most common storage accidents (wrong bucket, wrong file, cost overruns, untraceable changes). Access scoping (#3) is the structural safety net since AI agents with broad credentials are the primary users.

---

## Bugs to Address (from audit)

### Critical (must fix before any new features)
1. All 9 API methods are stubs (BUG-007, ROAD-000)
2. Speed calc Inf/NaN (BUG-001) — enhanced_client.go:198,277
3. PersistentPreRun blocks setup (BUG-002) — root.go:63
4. printErrorAndExit doesn't exit — list.go:130
5. Keyboard off-by-one (BUG-004) — menu_selection.go:170
6. Missing LICENSE file (BUG-010)

### High (should fix for open-source readiness)
7. README documents non-existent features
8. CI uses non-existent Go 1.26 (BUG-003)
9. install.sh binary name mismatch
10. No HTTP client timeout (resource exhaustion)
11. Auth rotate always fails (hardcoded error)
12. YAML output uses JSON (bucket.go:427)
13. No Client interface (TASK-002)

---

## Architecture Notes

> **Note**: The canonical file listing is in Decision 6 / Decision 11 above. The library now includes 30+ files covering R2, Workers, KV, D1, DNS, Zones, SSL, Cache, Domains, Doctor, and more. See `pkg/r2go2/` for the current state.

### CLI Layer (cmd/)

The CLI now covers all implemented services. Run `cosmoflare --help` for the current command tree.

### Future CCS Layer
```
ccs r2 list          → shells to cosmoflare bucket list --json
ccs r2 upload file   → shells to cosmoflare object put --json
```
