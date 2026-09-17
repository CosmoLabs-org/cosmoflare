# Qwen Research Pack — Cloudflare API-Token Permission Catalog (FEAT-011) + Next-Wave Permission Mapping

transmission-cleared: yes (2026-09-17, public-repo content only)

You are doing deep technical research for **Cosmoflare**, an open-source Go
CLI + library by CosmoLabs. This pack is self-contained: everything you need
is below. You have web access — verify against live Cloudflare docs and cite
a URL for every claim. Where docs are silent, say `docs-silent` and mark the
row `unverified` rather than guessing.

## Project identity

- **Product**: `cosmoflare` — Go 1.26 CLI + importable library (`pkg/cosmoflare`) covering the full Cloudflare developer platform: R2, Workers, KV, DNS, D1, Pages, Queues, Images, Stream, Hyperdrive, Vectorize, AI, SSL, WAF, Email Routing, Cache, and 20+ more. MIT-licensed CLI; paid Desktop (Tauri) and Mobile (React Native) tiers wrap the same core.
- **Agent-first UX**: every command has `--json`, rich `--help`, deterministic exit codes. AI agents and humans are both first-class users.
- **Live at**: github.com/CosmoLabs-org/cosmoflare (v0.29.0), `brew install CosmoLabs-org/cosmoflare/cosmoflare`.
- **Stack**: Go 1.26, cobra, cloudflare-go v0.116.0 (pinned) for some paths, hand-rolled REST for others. Pure-Go only (release cross-compiles with CGO_ENABLED=0).

## Why this research exists (background you must know)

Cloudflare's API-token permission names have **drifted from product names**.
Verified real-world example (2026-09-09, API v4): modern rate-limiting rules
(Rulesets phase `http_ratelimit` entrypoint writes at
`PUT /zones/{zone_id}/rulesets/phases/http_ratelimit/entrypoint`) are governed
by the token permission **"Zone WAF → Edit"** — while the permission literally
NAMED "Rate Limiting" only covers the deprecated classic API
(`/zones/{id}/rate_limits`), which Cloudflare has turned OFF. Error code
**10405** on a rulesets write looks like an authentication-scheme failure but
actually means "missing Zone WAF Edit scope". Error **10000** on classic
endpoints means wrong/missing scope. This folklore is undocumented; we are
building the canonical catalog.

## ASK 1 (primary) — enumerate ALL current API-token permissions

Source of truth: `developers.cloudflare.com` — start at
`/fundamentals/api/reference/permissions/` (the permissions reference that
mirrors the token-creation UI), then per-product pages for endpoint
unlock-mappings. Enumerate **every** permission available when creating an API
token today, in all three scopes (Account-level groups, Zone-level groups,
User-level). Expect roughly 150–300 rows; do not truncate. If your output is
cut off, the user will say "Please continue" — resume at the exact row you
stopped, without repeating rows.

Return the catalog as YAML, one row per permission:

```yaml
permissions:
  - scope: zone                 # zone | account | user
    group: "Zone WAF"           # exact UI group name
    name: "Zone WAF"            # exact token-permission name as shown in dash/docs
    access: edit                # read | edit (emit two rows when both exist)
    unlocks:                    # concrete endpoints or phases this permission gates (best-effort, cite docs)
      - "PUT /zones/{zone_id}/rulesets/phases/http_ratelimit/entrypoint"
      - "PUT /zones/{zone_id}/rulesets/phases/http_request_firewall_custom/entrypoint"
    docs: "https://developers.cloudflare.com/<exact page>"
    notes: "governs modern rate-limiting rules; permission named 'Rate Limiting' is legacy-only"
    verified: 2026-09-17        # date you checked the live page
```

Then a second YAML block for the **error-signature table** — every
authentication/authorization error code you can document
(10000, 10400, 10405, 9109, 6111, and any others you find), shape:

```yaml
errors:
  - code: 10405
    surface: "rulesets writes"
    reads_as: "authentication error"
    actually_means: "token lacks the required permission scope"
    fix: "add Zone → Zone WAF → Edit to the token"
    docs: "<url or docs-silent>"
```

Include our three verified seed findings (zone-WAF/rate-limit drift, classic
API off, 10405/10000 meanings) — corrected if your research shows drift on
OUR side too. We want to be wrong about anything stale.

## ASK 2 — permission mapping for our next four feature waves

We are about to build these CLI command groups. For each, list the exact
token permissions required (from ASK 1's catalog), the core API endpoints
(method + path), plan-tier gating (free vs paid), and any
first-party-only/entitlement restrictions:

1. **Registrar operations** (FEAT-030): domain register, transfer-in, renew,
   lock/unlock, contact management, DNSSEC. Is the Registrar API
   (`/accounts/{id}/registrar/...`) available to all accounts or
   enterprise/wholesale-registrar only? This gate decides the wave's shape.
2. **Cloudflare Tunnels** (FEAT-035): cfd_tunnel CRUD, tokens, connections,
   configurations. Note where management is API-driven vs cloudflared-owned
   (config vs remotely-managed tunnels).
3. **Account management + audit logs** (FEAT-036): account list/details,
   members, roles, `/accounts/{id}/audit/logs` export. Permission for audit
   log reads?
4. **Zone long-tail batch** (FEAT-037): Waiting Room, Spectrum, Load
   Balancers, Page Shield, Turnstile, Web Analytics, Logpush. For Spectrum
   and Waiting Room note plan gating ( Spectrum = pro+? Waiting Room =
   free-with-limits? verify current state 2026).

Return format for ASK 2: one markdown table per wave:
`| CLI command | API endpoint (method+path) | required permission | plan gate | notes |`

## ASK 3 — feasibility notes (short)

- Any of the four waves blocked or degraded without Enterprise? Flag loudly.
- Endpoints that are deprecated/GA-pending (e.g., is classic Rate Limiting
  fully removed now? Pages build API vs Workers Builds convergence state?).
- cloudflare-go v0.116.0 coverage gaps for these surfaces (we can hand-roll
  REST; just note what the SDK lacks).

## Return envelope (how to format your whole response for re-ingestion)

```markdown
# Qwen results — CF permission catalog + wave mapping (2026-09-17)

## Method
<which pages you pulled, dates checked, anything you could not verify>

## ASK 1 — Permission catalog
<yaml block: permissions + errors>

## ASK 2 — Wave permission mapping
<4 tables>

## ASK 3 — Feasibility flags
<bullets>

## Confidence & provenance
<per-section: high/medium/low + why; list every URL cited>
```

Rules: cite a URL per row where possible; `verified:` dates must be real
check dates; never invent a permission name — if unsure, mark `unverified`.
Prefer completeness over prose. No preamble — start at "## Method".
