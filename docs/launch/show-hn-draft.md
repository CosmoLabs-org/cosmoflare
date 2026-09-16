# Show HN draft — Cosmoflare

Status: DRAFT — awaiting operator voice/edits. Do not publish as-is.

## Title (pick one)

- Show HN: Cosmoflare – The whole Cloudflare platform in one Go binary
- Show HN: Cosmoflare – Cloudflare CLI without Node.js, built for agents too

## Body

Hi — we built Cosmoflare [1], an open-source (MIT) CLI + Go library for the
entire Cloudflare developer platform: R2 storage, DNS, Zones, SSL/TLS, Cache,
WAF, D1, KV, Queues, Pages, Images, Hyperdrive, Vectorize, Workers AI,
Stream, Email Routing and more — 22 services, one static ~37 MB binary, zero
runtime dependencies.

Why we built it: Cloudflare's own `flarectl` (Go) is deprecated, and Wrangler
requires Node 18+. If you operate the whole platform — DNS records, zone
settings, cache purges, WAF rules — rather than only Workers, there was no
maintained single-binary tool. So:

- `--json` on every command, standard success/error envelope, deterministic
  exit codes. Scripts and shells are first-class users.
- A built-in MCP server: 140+ auto-generated tools exposing the same CLI
  surface to AI agents. It starts read-only; mutations require an explicit
  allow switch, and R2 uploads can be constrained by bucket allowlists,
  blocked-key patterns and size caps. We think "agent-safe by default" is
  the interesting part — happy to dig into the design in comments.
- Upload guardrails, an embedded knowledge layer that decodes Cloudflare API
  errors into causes + fixes, a monthly cost estimator, wrangler.toml import,
  and Terraform export for migration in the other direction.

Honest limits: `cosmoflare dev` is a proxy, not a workerd runtime — if your
daily loop is `wrangler dev`, keep Wrangler for that. The rest of the
Workers lifecycle (secrets, versions, deployments with rollback, custom
domains) shipped in v0.28.2. The comparison table [2] says exactly where
each tool wins.

Install: `go install github.com/CosmoLabs-org/cosmoflare@latest` or
`brew install CosmoLabs-org/cosmoflare/cosmoflare`. Docs: USAGE.md is a
2900-line agent-friendly reference [3]. The CLI is the free tier; a desktop
dashboard and mobile app wrap the same Go library.

What we'd love feedback on: the MCP mutation-gating model, and what else
belongs in a single-binary Cloudflare tool.

[1] https://github.com/CosmoLabs-org/cosmoflare
[2] https://github.com/CosmoLabs-org/cosmoflare/blob/master/docs/vs-wrangler.md
[3] https://github.com/CosmoLabs-org/cosmoflare/blob/master/docs/USAGE.md

## Posting notes (operator)

- Best time: Tue–Thu, 8–10am ET per HN convention; avoid US holidays.
- Reply to every top-level question in the first 2 hours.
- Lead comments with the agent-safety angle if the thread skews AI.
