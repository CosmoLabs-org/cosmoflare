# r/Cloudflare draft — Cosmoflare

Status: DRAFT — awaiting operator voice/edits. Do not publish as-is.
Subreddit rules re self-promotion: frame as tool + question, disclose
authorship in the post, engage in comments.

## Title

Cosmoflare — an open-source Go CLI for the whole Cloudflare platform (not
just Workers). Feedback wanted on the agent-safety design

## Body

Full disclosure: I'm one of the developers. Cosmoflare [1] is an MIT-licensed
CLI + Go library covering R2, DNS, Zones, SSL, Cache, WAF, D1, KV, Queues,
Images, Stream, Vectorize, and the rest — as a single static binary, no
Node.js. We started it because flarectl is deprecated and wrangler didn't
fit platform-wide ops work (DNS/SSL/WAF/email routing aren't its scope).

Things r/Cloudflare folks might care about:

- `cosmoflare object presign`, `sync`, and `watch` for R2 directories
  (wrangler's r2 object surface is get/put/delete)
- `cosmoflare cache purge` / `ssl settings` / `dns create` as plain commands
- JSON output on everything with one envelope shape — good for scripts
- Built-in MCP server for AI agents, read-only until you flip an explicit
  allow switch; uploads can be capped by bucket allowlist / key patterns /
  max size. This is the part I'd most like feedback on: is fail-closed the
  right default for agent-driven infra changes?
- `cosmoflare wrangler` imports your existing wrangler.toml;
  `cosmoflare terraform` exports live state if you're heading to IaC

What it does NOT do: local workerd dev runtime (`cosmoflare dev` is a
proxy). The rest of the Workers lifecycle — secrets, versions, deployments
with rollback, custom domains — landed in v0.28.2. Comparison here [2].

Install: `go install github.com/CosmoLabs-org/cosmoflare@latest` or
`brew install CosmoLabs-org/cosmoflare/cosmoflare`.

If you manage Cloudflare for a project: what's missing from your current
tooling? That's the roadmap input I care about most.

[1] https://github.com/CosmoLabs-org/cosmoflare
[2] https://github.com/CosmoLabs-org/cosmoflare/blob/master/docs/vs-wrangler.md

## Posting notes (operator)

- r/Cloudflare allows dev-tool posts with disclosure; keep the feedback ask
  genuine, answer technical questions with specifics.
- Consider cross-posting to r/golang with the library-first angle (separate
  draft, link pkg.go.dev once re-indexed).
