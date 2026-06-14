---
created: "2026-06-14T08:12:48-03:00"
last_reviewed: "2026-06-14T08:12:48-03:00"
review: "independent-review (inline) — 5 findings applied: registrar-name overclaim,
  redirect-source aggregation, CF API phase verification flag, attention-criteria
  definition, DI-note tightening + duplicate-bullet dedup"
status: DRAFT
priority: high
title: "Domain Management Center — TUI Dashboard, Redirect Visibility, Registrar Overlay"
origin: "/brainplan"
tags: [domains, tui, redirects, registrar, design]
deliverables:
  - id: BR-01
    title: "RedirectService library — modern CF Redirect Rules CRUD (Rulesets API)"
  - id: BR-02
    title: "RegistrarService library — registration overlay (expiry, auto-renew, transfer-lock)"
  - id: BR-03
    title: "DomainService enrichment — DomainDetail carries Redirects + Registrar"
  - id: BR-04
    title: "domains CLI command tree — get / stats / redirects / ns (backward-compatible)"
  - id: BR-05
    title: "redirects CLI command group — modern Redirect Rules CRUD (mirrors pagerules)"
  - id: BR-06
    title: "DomainBrowserModel TUI — split-pane browser with per-domain detail"
  - id: BR-07
    title: "Quick-add redirect prompt + domainDataSource dashboard wiring"
related:
  - docs/brainstorming/2026-05-16-domain-operations.md   # prior (completed) — library + domains/doctor CLI
  - docs/brainstorming/2026-06-07-road020-dashboard-tui.md # existing dashboard pattern
  - docs/brainstorming/2026-06-07-road002-tui-object-browser.md # split-pane BrowserModel pattern
---

# Domain Management Center — Design

## Context

All of the operator's domains are already Cloudflare zones (managed via CF DNS and
features). The completed "Domain Operations Center" (2026-05-16) shipped the
`DomainService` library and the `cosmoflare domains` / `cosmoflare doctor` CLI
commands, plus `cosmoflare pagerules` for legacy Page Rules. The CLI can already
list every domain with count, NS status, SSL/health, and create redirects.

Three things are **missing** and are the subject of this design:

1. **A TUI domain view** — the dashboard and object browser monitor R2/Workers/KV;
   there is no domain/zone pane anywhere in the TUI. This is the operator's top ask.
2. **Aggregated redirect visibility** — `pagerules list` is per-zone; there is no
   "show me, across every domain, which are redirecting and to where" view, and no
   modern **Redirect Rules** support (CF is deprecating Page Rules).
3. **A richer `domains` command tree** — today it is a single command with flags.

A thin **registrar overlay** rounds out the picture: registration expiry / auto-renew
for CF-registered domains, with an `external` badge for domains whose TLD isn't
supported by Cloudflare Registrar (registered elsewhere, e.g. GoDaddy, but managed on
CF). Note: CF's API exposes registration data only for domains it registers — it
cannot *name* an external registrar, so the badge reads `external`, not `GoDaddy`
(determining the registrar name requires a WHOIS lookup — a future enhancement).

## Goals

- See **all** domains in one place (TUI + CLI) with count and attention summary.
- Per domain: NS status **and** the actual nameservers, SSL/expiry, health, record
  breakdown, **active redirects (pattern → destination)**, registration expiry /
  auto-renew.
- Quick-add a redirect from the TUI; full redirect CRUD from the CLI.
- Modern **Redirect Rules** support alongside legacy Page Rules.
- Aggregate stats (totals by NS status, SSL, registrar; "needs attention" list).

## Non-Goals (explicitly deferred)

- Domain **transfers** / registration lifecycle (register, transfer-in, renew-by-payment).
  The operator has already transferred everything feasible to CF; the few GoDaddy-held
  TLDs stay until CF supports them. Captured as a future phase.
- **Manual expiry tracking** for externally-registered domains (GoDaddy). Future.
- **Buying** new domains.

## "Needs attention" criteria

The `domains stats` attention list and the TUI footer count flag a domain when **any** of:
- SSL expiring (<30 days), expired, or `none`
- NS `mismatch` or `external` (not on Cloudflare nameservers)
- Registration expiry <30 days (CF-registered domains only; external shows as unknown)
- A redirect target that resolves to ≥400, or a redirect loop — **Phase 2, best-effort**
  (may defer if it proves noisy/slow)

## Key Decisions (from brainstorm Q&A)

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | Scope = **Zones + Registrar**, registrar depth = **renewals/expiry only** | All described needs (NS, redirects) are zone concepts. Registrar adds expiry visibility; transfers deferred. |
| D2 | All domains are **CF zones**; registrar data is an **overlay**, not a separate inventory | Operator confirmed every domain is already on CF DNS/features. |
| D3 | TUI shape = **split-pane browser** (mirrors `BrowserModel`) | Best density for "see everything per domain"; consistent with the object browser. |
| D4 | Redirects = **TUI read + quick-add; full CRUD via CLI**; add modern **Redirect Rules** | Speed in the TUI, power in the CLI. CF is deprecating Page Rules. |
| D5 | TUI built on the existing **Charm stack** (lipgloss v1.1.0, bubbletea v1.3.10, bubbles) | Matches dashboard + object browser for visual consistency. |
| D6 | Plan = **one design, two waves** (Wave 1 library+CLI, Wave 2 TUI) | Ship the CLI foundation first; TUI builds on it. |

## Architecture — four independently-testable layers

### Layer 1 — Library (`pkg/cosmoflare/`)

Extend the existing `DomainService`; add two focused services.

- **`RegistrarService` (new, thin)** — `ListRegistrarDomains(ctx)` returns registration
  info (registrar, expiry, auto-renew, transfer-lock) via CF Registrar API
  (`/accounts/{id}/registrar/domains`). Domains not registered through CF are simply
  absent from this list; the enrichment layer records them as `external`. CF's API
  exposes registration data **only** for domains registered through Cloudflare — it
  cannot name an external registrar, so `external` domains carry no registrar name
  until a WHOIS lookup is added (out of scope).
- **`RedirectService` (new)** — CRUD for **Redirect Rules** via the CF Rulesets API
  (`/zones/{id}/rulesets/phases/http_request_dynamic_redirect` and
  `/zones/{id}/rulesets/phases/http_request_redirect`). Methods: `List(ctx, zoneID)`,
  `Get`, `Create`, `Update`, `Delete`. Legacy Page Rules continue to flow through the
  existing `pagerules` path; `RedirectService` is the modern surface. Display-level
  visibility (`domains redirects`, TUI right pane) **merges** Redirect Rules with
  legacy Page-Rule `forwarding_url` entries so "which domains redirect where" is
  complete regardless of which system the operator used. *(Exact phase names and the
  request/response shape must be confirmed against the current CF Rulesets API docs
  during implementation — flagged as an implementation-time verification step.)*
- **`DomainService` extension** — `GetDetail` enriches its `DomainDetail` with
  `Redirects []RedirectRule` (from `RedirectService`) and `Registrar *RegistrarInfo`
  (from `RegistrarService`, or an `external` sentinel). Pure classification/formatting
  helpers are extracted (e.g. `classifyRegistrarStatus`, `FormatRedirectRule`,
  `SummarizeDomains`) so they are unit-testable without the network.

### Layer 2 — CLI (`cmd/`)

Refactor `cosmoflare domains` into a **command tree**, backward-compatible (cobra lets
the parent keep its bare-invocation list behavior *and* host subcommands):

```
cosmoflare domains                    # list — unchanged current behavior
cosmoflare domains get <name>         # single-domain detail card
cosmoflare domains stats              # aggregate: total, by NS/SSL/registrar, attention list
cosmoflare domains redirects          # cross-domain: which domains redirect where
cosmoflare domains ns                 # NS-focused: who's on CF NS vs external/mismatch
cosmoflare domains tui                # launch the split-pane browser (Wave 2)
cosmoflare redirects list|create|...  # modern Redirect Rules CRUD (mirrors pagerules)
```

Every command supports `--json`. New handlers resolve their services via a
package-level factory variable (`var newDomainService = liveFactory`) that defaults to
the live `NewClient()` path — consistent with existing commands when not injected, and
unit-testable by swapping the var for a fake in tests. This is a minimal, local DI
pattern; it does **not** require the broader cmd/ DI refactoring (next-session Goal #2).

### Layer 3 — TUI (`internal/tui/domain.go`)

New `DomainBrowserModel`, structured exactly like `BrowserModel` (lipgloss rounded
borders, `JoinHorizontal`/`JoinVertical`, the shared primary/muted palette).

- **Left pane** — all-domains list: `cf✓ / cf⚠ / ext✗` status icons, NS badge,
  footer with total + "needs attention" count.
- **Right pane** — selected domain detail: nameservers, SSL mode + expiry, registrar
  (auto-renew/expiry, or an `external` badge when not registered via CF), **active
  redirects (pattern → destination)**, DNS record-type breakdown.
- **Quick-add redirect** — a `bubbles/textinput`-driven prompt creates a Redirect Rule
  in seconds (URL pattern → destination + status code).
- A `domainDataSource` adapts the domain data to the existing dashboard `DataSource`
  abstraction so the browser plugs into the app shell and can be reached from the main
  dashboard's section navigation.

### Layer 4 — Data flow

`CLI/TUI → DomainService.List / GetDetail → enrich with RedirectService (per-zone
rules) + RegistrarService (registration)`. Domain state is slow-changing, so the model
fetches on launch and refreshes manually (matches the dashboard's tiered-refresh
pattern — auto-poll for monitoring, manual elsewhere).

## Data model (new types)

```go
// RedirectRule is a modern CF Redirect Rule (Rulesets API).
type RedirectRule struct {
    ID             string `json:"id"`
    ZoneID         string `json:"zone_id"`
    When           string `json:"when"`           // URL pattern / expression
    Destination    string `json:"destination"`     // target URL (may use $1..$n captures)
    StatusCode     int    `json:"status_code"`     // 301, 302, 307, 308
    PreserveQuery  bool   `json:"preserve_query"`
    Enabled        bool   `json:"enabled"`
}

// RegistrarInfo is the registration overlay for a domain.
type RegistrarInfo struct {
    Registrar     string     `json:"registrar"`      // "cloudflare" | "external"
    RegistrarName string     `json:"registrar_name"` // empty for external — CF API does
                                                   // not name external registrars (WHOIS = future)
    ExpiresAt     *time.Time `json:"expires_at,omitempty"` // nil when external/unknown
    AutoRenew     bool       `json:"auto_renew"`
    TransferLock  bool       `json:"transfer_lock"`
}
```

`DomainDetail` gains `Redirects []RedirectRule` and `Registrar *RegistrarInfo`.

## Error handling

- **No credentials** → graceful launch with a placeholder state (existing dashboard
  pattern); never a hard crash.
- **Registrar API miss / unsupported** → `Registrar{Registrar:"external"}`, never
  blocks the zone display.
- **Redirect Rules unavailable** (plan/API limits) → fall back to displaying legacy
  Page Rules; surface a muted note.
- Every failure path produces an actionable, agent-readable error (what failed, why,
  suggested fix), consistent with the rest of cosmoflare.

## Testing strategy (TDD)

- **Pure helpers** (`classifyRegistrarStatus`, `FormatRedirectRule`,
  `SummarizeDomains`, NS/SSL classification already present) → direct unit tests, no
  network.
- **Services** (`RedirectService`, `RegistrarService`, `DomainService` enrichment) →
  tested against an `httptest` mock of the CF API, mirroring the webhook test style.
- **CLI handlers** → testable via the package-level factory-var injection described in
  Layer 2 (swap in a fake service — no live credentials). Independent of the broader
  cmd/ DI refactoring (next-session Goal #2).
- **TUI** (`DomainBrowserModel`) → view + navigation tests mirroring
  `internal/tui/browser_test.go`.

## Implementation phasing (two waves, one plan)

- **Wave 1 — CLI foundation.** `RedirectService` + `RegistrarService`; `DomainService`
  enrichment; the `domains` command tree (`get`, `stats`, `redirects`, `ns`); the
  `redirects` command group. Ships a complete, scriptable, agent-friendly CLI surface.
- **Wave 2 — TUI.** `DomainBrowserModel` + `domainDataSource`; right-pane detail;
  quick-add redirect prompt; wiring into `cosmoflare domains tui` and dashboard
  navigation.

Each wave is independently shippable. Wave 2 depends on Wave 1's library surface but
not on its CLI commands.

## Open questions

None blocking. The two-wave split, the registrar-overlay depth, the redirect
experience, and the TUI shape are all resolved (D1–D6).
