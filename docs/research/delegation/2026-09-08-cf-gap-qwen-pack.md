# Research Pack — Cloudflare Coverage Gap Analysis (for Qwen)

> **How to use:** paste everything below the cut line into Qwen (or attach this file).
> Save Qwen's full answer to `docs/research/delegation/2026-09-08-cf-gap-qwen-results.md` in this repo.
> Then run `/run-continuation` and pick `2026-09-08-ingest-gap-research`.

---

## Project context

**cosmoflare** is an open-source Go library + CLI managing the Cloudflare developer platform (module `github.com/CosmoLabs-org/cosmoflare`, v0.21.0). Positioning: agent-first — as usable by an AI agent as by a human (`--json` on every command, rich help, MCP server mode, deterministic exit codes, guardrails + audit trail). Its origin and deepest service is R2 storage. A paid Desktop (Tauri) and Mobile (React Native) tier wrap the same core.

**Currently implemented (Go library `pkg/cosmoflare/` + CLI):**

- R2 storage: buckets, objects, multipart, resumable uploads, sync (bidirectional, pagination, mtime+checksum compare), watch, copy, bucket custom domains (attach/list/get/verify/update/detach, REST)
- Workers (compute), KV, D1, Queues, Hyperdrive, Vectorize, Workers AI + AI Gateway
- DNS records, Zones, SSL/TLS, Cache, Page/Redirect Rules, WAF/Firewall
- Email Routing, CORS, Pages, Images, Stream, Healthchecks, Domains + Registrar + Redirects (Domain Center Wave A)
- Workflow commands: dev, init, diff, apply (declarative for workers/dns/kv/r2), sync, watch, cost, export/import, templates, validate, terraform export, mcp server, wrangler.toml import, audit log, alerts (rules), account switching, plugins, doctor

**Known already (from our 2026-08-31 internal audit — do not re-derive, extend):**

- Cloudflare's official `cf` CLI + MCP server (April 2026) exposes ~2,500 endpoints; our MCP parity is a roadmap item.
- Our known in-repo gaps: Domain Center Waves B–D, metric producers → alert evaluator, declarative apply/diff breadth, docs site.
- Services we already know we lack: Zero Trust suite (Access/Gateway/Tunnel/WARP), Load Balancing, Waiting Rooms, Logpush, Bot Management, Rate Limiting rules, Durable Objects, Cloudflare Workflows, Containers, Browser Rendering, Calls, Spectrum, R2 event notifications + lifecycle policies, GraphQL Analytics, account/user management, billing API, secondary DNS, custom nameservers.

## Research question

Produce a coverage gap analysis of cosmoflare against Cloudflare's full product and API surface as of September 2026:

1. **Taxonomy:** enumerate Cloudflare's product families and API areas (public api.cloudflare.com v4 + product-specific APIs incl. R2 S3-compatible, GraphQL analytics). Group them (Network, Security, Compute, Storage, Observability, Account/Platform, AI, etc.). Include endpoints-per-area counts where the docs state them.
2. **Coverage verdict per area:** `covered` (we implement it), `thin` (basic CRUD only — name what depth is missing), or `missing`. Judge "thin" honestly against the API surface, not against our marketing table.
3. **Ranked gaps:** the top 15 gaps by value for our target user (a developer or AI agent operating R2-centric infrastructure from a terminal). For each: what it is, why it matters for this positioning, the Cloudflare API involved, rough implementation shape in our Go library (service file + commands), and effort (S/M/L).
4. **Competitive angle:** what `wrangler`, the official `cf` CLI, and `cf` MCP cover that our list above does not mention at all — especially anything R2/Workers-adjacent an agent would reach for.
5. **Deprecation watch:** Cloudflare APIs/products being sunset (e.g. Page Rules → Rulesets) that we implement naively and should migrate.

## Output format (required)

Markdown with these exact headings:

```
## 1. Taxonomy
<table or list: family → areas → endpoint counts>
## 2. Coverage verdicts
<table: area → verdict (covered/thin/missing) → notes>
## 3. Ranked gaps
### Gap N: <name> (effort S/M/L)
what/why/API/shape
## 4. Competitive delta
## 5. Deprecation watch
## 6. Sources
<URLs consulted>
```

Cite Cloudflare doc URLs for every claimed fact in sections 1, 4, and 5. Where docs are ambiguous, say so instead of guessing. Do not modify any files; this is research only.
