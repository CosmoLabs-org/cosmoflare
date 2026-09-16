---
title: "FEAT-029 part 2 — auth login --device (OAuth Device Authorization Grant)"
created: 2026-09-16T00:00:00+04:00
status: DEFERRED
issue: FEAT-029
deliverables:
    - id: BR-01
      title: "Device-flow client (internal/auth/deviceflow.go): Start + Poll state machine (pending/slow_down/expired/declined/success), fully tested"
    - id: BR-02
      title: "auth login --device wired: URL+code display, best-effort browser open (--no-open), Secrets-store persistence, --json, secrets never logged"
    - id: BR-03
      title: "OQ-1 resolution note: verified device-token → API-token exchange step with doc citations"
    - id: BR-04
      title: "USAGE.md auth login --device section + rich --help"
---

# Why

`cosmoflare auth login` today only accepts a pasted API token. The OAuth 2.0
Device Authorization Grant removes the paste: the CLI shows a URL + code,
the user approves in a browser anywhere, and the CLI receives credentials —
no local callback server. Wrangler shipped the same flow 2026-08-04; parity
matters for first-run UX (FEAT-017 wizard) and for headless/SSH setups.

# Decisions

| # | Decision | Why |
|---|----------|-----|
| D1 | Register CosmoLabs' OWN OAuth client; do NOT reuse wrangler's public client_id | Operator decision 2026-09-16. Own identity = correct branding, scope control, no breakage when Cloudflare rotates/restricts wrangler's client. |
| D2 | client_id resolution: compiled-in default + `COSMOFLARE_OAUTH_CLIENT_ID` env override | Lets tests and future rotations run without a release; SSOT is the const, env is the escape hatch. |
| D3 | Token storage: the existing `Secrets` store (keychain/file fallback), same path as profile credentials | One credential pipeline; honors `COSMOFLARE_NO_KEYCHAIN`. |
| D4 | Terminal UX: print verification URL + user_code; attempt `open`/`xdg-open` unless `--no-open` | Headless-safe by construction; browser launch is best-effort convenience. |
| D5 | Never log device_code / access_token; user_code is safe to display (single-use, short-lived) | Security posture mirrors FEAT-029 part 1's redaction rules. |

# Open questions

- **OQ-1 — RESOLVED 2026-09-17** (research note:
  `docs/research/2026-09-17-feat029-device-flow-oq1.md`, agent-verified
  against wrangler source + current docs):
  - Device endpoint `POST https://dash.cloudflare.com/oauth2/device/auth`
    (form-encoded `client_id` + `scope`, colon-format scopes); user page
    `https://dash.cloudflare.com/oauth2/device`.
  - Token endpoint `POST https://dash.cloudflare.com/oauth2/token`, grant
    `urn:ietf:params:oauth:grant-type:device_code`; poll errors
    `authorization_pending` / `slow_down` (+5s) / `access_denied` /
    `expired_token`; interval honored, 1s floor, 300s cap.
  - **No exchange step exists** — wrangler passes the OAuth access token
    straight through as the bearer for `api.cloudflare.com`; cloudflare-go
    consumes it directly. The original premise ("create a scoped API token
    the way wrangler does") was a misconception.
  - Refresh: `grant_type=refresh_token` needs `offline_access` at request
    time; cloudflare-go has NO refresh support — caller refreshes
    out-of-band and rebuilds the client.
  - **D1 IS INFEASIBLE**: current docs state third-party OAuth clients
    support authorization-code ONLY; the device grant is first-party
    (wrangler/cf CLI). Cosmoflare cannot register its own device-flow
    client_id. → strategy decision reopened; see D6.

# Decisions (post-research)

| # | Decision | Why |
|---|----------|-----|
| D6 | **DEFER part 2** (operator, 2026-09-17) | D1 infeasible (first-party-only grant); D-reuse of wrangler's client_id declined; auth-code pivot violates the issue's no-local-server constraint. Revisit when Cloudflare opens the device grant to third-party clients — the verified contract (endpoints, poll errors, refresh semantics, direct cloudflare-go use) is preserved in the research note, so implementation is spec-ready the day policy changes. |

# Alternatives rejected

- **Reuse wrangler's public client_id** — zero setup, but borrows
  first-party identity; breaks on their rotation; scope mismatch risk.
- **Browser callback (authorization code) flow** — needs a local HTTP
  server; explicitly out of scope per the FEAT-029 issue (wrangler parity
  argument is device-flow, not callback).

# Relation

Part 1 (`auth token`, redact-safe retrieval) merged 2026-09-16 (f2d3c7e).
This part adds `auth login --device`. Feeds FEAT-017 first-run wizard.

## Deliverables

- **BR-1** Device-flow client (`internal/auth/deviceflow.go`): Start →
  {user_code, verification_uri, device_code, interval, expires}; Poll state
  machine (pending / slow_down / expired / declined / success); fully tested
  against httptest doubles.
- **BR-2** `cosmoflare auth login --device`: prints URL + user code,
  best-effort browser open (`--no-open` to skip), completes the flow, stores
  the resulting credential via the `Secrets` store into the active profile;
  `--json` envelope; secrets never logged.
- **BR-3** OQ-1 resolution note appended to this doc: the verified
  device-token → API-token exchange step with doc citations.
- **BR-4** USAGE.md `auth login --device` section + rich `--help`.
