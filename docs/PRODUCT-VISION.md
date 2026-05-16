# Cosmoflare — Product Vision & Constitution

**Company**: CosmoLabs (https://cosmolabs.org)
**License**: MIT (open-source)
**Repo**: github.com/CosmoLabs-org/CosmoDev-R2Go2

## Mission

**Control Cloudflare 100% from the terminal — securely, efficiently, and agent-first.**

Cosmoflare is the MCP-powered Cloudflare control plane. It exists so that Claude Code, agentic workflows, and human developers can manage every Cloudflare service from a single tool with zero browser dependency.

## Constitution (Non-Negotiable Principles)

1. **Agent-first UX** — Every command must be usable by an AI agent without human guidance. `--json` on every command, predictable exit codes, rich `--help` for agent discovery, actionable error messages.
2. **Full platform coverage** — If the Cloudflare API exposes it, Cosmoflare manages it. No service left behind.
3. **Security is paramount** — API keys handled securely (never logged, never in output unless explicit). The CLI is the trust boundary between agents and Cloudflare infrastructure.
4. **Library-first** — The Go library (`pkg/r2go2/`) is the stable API surface. The CLI wraps it. The future mobile app wraps it. No separate implementations.
5. **MCP-native** — Cosmoflare will serve as an MCP server, letting Claude Code and other AI tools control Cloudflare natively through tool-use protocol.
6. **Open-source core, paid mobile** — The CLI is free (MIT). The React Native mobile app with push notifications and one-tap actions is the paid product.
7. **Backward compatible** — `r2go2` binary always works. `cosmoflare` is the primary name. Module path changes are planned, not forced.

## What is Cosmoflare?

Cosmoflare is the ultimate open-source CLI for managing the entire Cloudflare developer platform. It builds on top of wrangler and makes it better — more services, better UX, and a public Go library that wrangler doesn't provide.

**The chain**: Go library → CLI → MCP server → Mobile app (same API logic, native interface)

## 3-Tier Product

| Tier | Product | Model | Status |
|------|---------|-------|--------|
| **CLI** | `cosmoflare` (alias: `r2go2`) | Free, MIT, open-source | Active |
| **Mobile** | React Native (iOS/Android) | Paid subscription | Planned |
| **Desktop** | Tauri app (macOS/Windows/Linux) | Paid | Future |

**Mobile is the priority over desktop.** People already have dash.cloudflare.com for a browser-based dashboard. What they lack is a native mobile app with push notifications and one-tap actions. That's the gap worth filling.

## Why It Works

- **Wrangler is limited** — focused on Workers, clunky for R2, nothing for DNS/SSL/WAF
- **Cosmoflare covers more services** with better UX and a public Go library
- **The Go library is the leverage** — same API logic powers both CLI and mobile app
- **Agent-friendly** — AI tools (Claude, GLM, Codex) use the CLI from terminal

## Platform Coverage

Everything the Cloudflare API allows us to interact with:

### Implemented
- **R2** (storage) — buckets, objects, uploads, downloads, presigned URLs
- **Workers** (compute) — deploy, list, get, delete, logs, settings, bindings
- **KV** (key-value) — namespaces, get/put/delete, list keys

### Planned (see roadmap)
- **DNS Records** (ROAD-035) — CRUD for A, AAAA, CNAME, MX, TXT, SRV
- **Zones** (ROAD-036) — list zones, settings, diagnostics
- **SSL/TLS** (ROAD-037) — certificates, TLS settings, always-https
- **Cache** (ROAD-038) — purge by all/URL/tag/host, cache rules
- **Page/Redirect Rules** (ROAD-039) — forwarding URLs, cache levels
- **WAF/Firewall** (ROAD-040) — rules, managed rulesets, rate limiting
- **Email Routing** (ROAD-041) — rules, destinations, catch-all
- **D1** (ROAD-042) — SQLite at the edge, query, migrate
- **Pages** (ROAD-043) — static hosting, deploy, custom domains
- **Images** (ROAD-044) — upload, transform, variants
- **Stream** (ROAD-045) — video upload, live inputs
- **Hyperdrive** (ROAD-046) — database connection pooling
- **Vectorize** (ROAD-048) — vector storage for AI
- **Workers AI** (ROAD-049) — run models at edge, AI Gateway

### CLI Infrastructure
- `cosmoflare status` (ROAD-050) — dashboard-at-a-glance
- `cosmoflare logs --follow` (ROAD-051) — real-time log tailing
- `cosmoflare diff` (ROAD-052) — show changes before applying
- `cosmoflare import/export` (ROAD-053) — full account config as YAML
- `cosmoflare doctor` (ROAD-054) — diagnose common issues
- Shell completion (ROAD-055) — Bash, Zsh, Fish
- Config as code (ROAD-056) — `.cosmoflare.yaml` desired state
- `cosmoflare init` (ROAD-057) — project scaffolding
- `cosmoflare dev` (ROAD-058) — local dev server with hot-reload
- Template library (ROAD-059) — starter project gallery

### Dashboard & Mobile
- Real-time metrics (ROAD-060)
- Cost estimator (ROAD-061)
- Alert rules (ROAD-062)
- Tauri desktop app (ROAD-063)
- React Native mobile app (ROAD-064)
- Plugin system (ROAD-065)

## Repo Strategy

- **CosmoDev-R2Go2** (this repo) — CLI + Go library. Open-source, community contributions. Pure Go.
- **Cosmoflare-apps** (future repo) — React Native mobile (+ eventually Tauri desktop). Can be private/paid.
- Keeps contribution simple: Go devs don't need Node. Mobile devs import the Go library's logic via API layer.

## Design Principles

1. **Library-first** — importable by any Go project. This IS the mobile app's backend.
2. **Agent-friendly** — AI tools use the CLI from terminal. Rich `--help`, `--json` on every command.
3. **Everything Cloudflare** — if the Cloudflare API exposes it, Cosmoflare manages it.
4. **Backward compatible** — `r2go2` binary remains as alias for `cosmoflare`.
5. **Mobile > Desktop** — native mobile app with push notifications is the paid product. Desktop can come later.
