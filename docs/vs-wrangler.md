# Cosmoflare vs Wrangler

Wrangler is the official Cloudflare CLI and the right choice if your work is
Workers-first and you already run Node.js. Cosmoflare exists for a different
job: **the full Cloudflare platform — R2, DNS, Zones, SSL, Cache, WAF, D1, KV,
Queues, Images, Stream and 10+ more services — in one static Go binary, with
no Node runtime, and an interface built for scripts and AI agents as much as
for humans.**

| | Cosmoflare | Wrangler |
|---|---|---|
| Runtime | Single static binary (Go), zero deps | Node.js 18+ |
| Platform coverage | 22 services: R2, DNS, Zones, SSL/TLS, Cache, WAF, Email Routing, D1, KV, Queues, Pages, Images, Hyperdrive, Vectorize, Workers AI, Stream, Healthchecks, Domains… | Workers-centric; R2 get/put/delete; no DNS/Zones/SSL/WAF/Email |
| R2 depth | `object list`, presigned URLs, multipart (incl. resumable), directory `sync`, `watch` | get/put/delete only |
| JSON output | `--json` on **every** command, standard success/error envelope, deterministic exit codes | Human-first output |
| AI agents | Built-in MCP server (140+ tools), read-only by default, mutations behind an explicit allow switch | None |
| Upload guardrails | Bucket allowlist, blocked-key patterns, max-size caps | — |
| Error messages | Actionable: what failed, why, how to fix; embedded knowledge packs decode Cloudflare API errors | Varies |
| Cost awareness | `cosmoflare cost` estimates monthly spend for R2/Workers/KV | — |
| Config | One `.cosmoflare.yaml`; declarative `diff`/`apply` for workers, DNS, KV, R2 | `wrangler.toml` (per-project deploy) |
| Migration bridges | **Import** wrangler.toml; **export** Terraform `.tf` + import blocks (5 services) | — |
| Local Workers dev | Proxy only (no workerd runtime) | Full workerd runtime, hot-reload |
| Workers lifecycle | deploy/list/get/delete/logs/settings | + secrets, versions, rollback, gradual deploys |
| Library | Public Go API (`pkg/cosmoflare`) reused by CLI, MCP, and a desktop app | n/a |

## When to pick which

**Use Wrangler when** you want `wrangler dev`'s real workerd runtime for
local iteration — Cosmoflare's `dev` is a proxy, not a runtime. The rest of
the Workers daily loop — secrets, versions, deployments with rollback,
custom domains — is in Cosmoflare as of v0.28.2.

**Use Cosmoflare when** you:

- operate the **whole platform** — DNS records, zone settings, SSL mode,
  cache purges, WAF rules, email routing — not just Workers;
- want a **single 37 MB binary** with no Node install, on servers, CI, or
  rescue USB sticks;
- **script or automate**: every command emits the same JSON envelope and
  exit codes, so `jq` pipelines and shell scripts are first-class;
- run **AI agents** against your infrastructure: the built-in MCP server is
  read-only until you explicitly opt into mutations, and upload guardrails
  cap what an agent can touch;
- are **leaving a tool**: `cosmoflare wrangler` imports your existing
  `wrangler.toml`; `cosmoflare terraform` exports live state to Terraform.

Many workflows combine both: `wrangler dev` while building, Cosmoflare for
everything around the code.

## The niche

`flarectl` — Cloudflare's earlier Go CLI — is deprecated and unmaintained,
and no maintained CLI currently covers DNS + Zones + SSL + WAF + R2 + the
rest in one binary. That is the slot Cosmoflare fills.

## Try it

```bash
go install github.com/CosmoLabs-org/cosmoflare@latest
cosmoflare zone list --json
```

Full command reference: [USAGE.md](USAGE.md)
