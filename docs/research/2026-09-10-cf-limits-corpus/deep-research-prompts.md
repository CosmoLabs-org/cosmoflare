# Deep-research prompt queue — prepared prompts, paste-ready

Provenance: session 2026-09-12. Prompts are stored here verbatim; the
clipboard is only transport. Results land in this directory as
`gemini-results-<topic>.md` (or `grok-results-<topic>.md`) with the prompt
quoted at top.

---

## Prompt 1 — MCP ecosystem (ON CLIPBOARD 2026-09-12)

DEEP RESEARCH — Cloudflare MCP ecosystem and agentic-tooling surface

Map the ENTIRE current MCP (Model Context Protocol) ecosystem for managing
Cloudflare, to position an open-source Go CLI's MCP server ("cosmoflare
mcp") that wraps the full account.

PART 1 — Official Cloudflare MCP servers: every server Cloudflare
publishes/maintains (local stdio packages AND remote hosted MCP servers —
docs.cloudflare.com/agents/model-context-protocol,
github.com/cloudflare/mcp-server-cloudflare). For EACH: hosting model,
auth model (OAuth 2.1? API token? none), COMPLETE tool inventory, covered
services, rate limits, known issues. Plus Cloudflare's remote-MCP auth
guidance (OAuth, PKCE, token binding) — mechanisms, not just existence.

PART 2 — Third-party/community Cloudflare MCP servers (GitHub/npm/PyPI):
notable by adoption, coverage, quality, maintenance. Plus adjacent
non-MCP tools (dashboards, GUIs, CLIs beyond wrangler).

PART 3 — MCP protocol state of the art (2025-2026 spec): tool
naming/annotation conventions, structured output, resource vs tool vs
prompt, error semantics agents act on, destructive-op protections,
discovery surfaces (registries, .mcp.json, marketplaces + requirements).

PART 4 — GAP ANALYSIS: operations with NO MCP coverage by service (R2,
Workers, KV, D1, Queues, Pages, DNS, zones, SSL, WAF, email, registrar,
Zero Trust, billing, audit logs, account); where official servers are
weak; what an agentic-first account-wide MCP must do to be default choice.

OUTPUT (binding — parsed by a tool): sections 1-4 with tables; then
"## MCP JSON" one fenced ```json block, array of
{"name","vendor","hosting","auth","tools_count","covers","quality","gaps","source_url","as_of"};
then "## GAP JSON": array of
{"service","mcp_coverage","best_server","opportunity"}.
Rules: source URL + as-of per claim; null when unverifiable.

---

## Prompt 2 — Pillar A: coverage completion (queued)

DEEP RESEARCH — Cloudflare platform coverage completion for full-account
control: Durable Objects, Workflows, Tunnels, Gateway/WARP, long-tail
zone services, account/org management, audit logs, billing API.

For EACH service below: REST API management surface (endpoints, methods,
key objects), wrangler/CLI coverage if any, permission scopes required
(account vs zone), rate limits, and the 5 most common real-world
management operations:
1. Durable Objects (scripting API boundaries vs REST management)
2. Workflows (create/list/update/delete, instance operations)
3. Cloudflare Tunnels (cloudflared config vs API: tunnel CRUD, public
   hostnames, private network routes, WARP connector)
4. Zero Trust Gateway + WARP (policies, device enrollment, device posture)
5. Waiting Room, Spectrum, Load Balancing (pools/monitors), Page Shield,
   Bot Management, Turnstile, Web Analytics, Logpush jobs
6. Account/org management: members, roles, RBAC, invitations, API
   audit-log endpoints (retention, export, filtering)
7. Billing API: subscriptions, invoices, billing profile — what is
   automatable vs dashboard-only

OUTPUT (binding): one section per service with a management-operations
table (operation | method+endpoint | scope | wrangler? | notes) and a
final "## COVERAGE JSON": array of {"service","management_ops_count",
"wrangler_coverage","rest_endpoints_documented","permission_scopes",
"automatable_pct_estimate","source_url","as_of"}. Null when unverifiable.

---

## Prompt 3 — Registrar mechanics + domain monitoring doctrine (queued)

DEEP RESEARCH — domain lifecycle mechanics and monitoring doctrine for a
domain-management pillar.

PART 1 — Registrar mechanics: transfer rules (auth codes, 60-day ICANN
lock, losing-registrar behavior), expiry → grace → redemption windows per
TLD class, DNSSEC DS record sequencing (enable/disable without breaking
resolution), WHOIS privacy interactions, Cloudflare Registrar constraints
(at-cost model, transfer-in only?), bulk operations.
PART 2 — Monitoring doctrine: what expert teams monitor per domain
(expiry lead times, NS drift, DS mismatch, SOA serial lag, cert chain
expiry, proxied-record integrity), alert thresholds considered standard,
open-source monitoring approaches worth mirroring.
OUTPUT: tables + "## DOCTRINE JSON": array of {"check","why","threshold",
"severity","source_url","as_of"}.
