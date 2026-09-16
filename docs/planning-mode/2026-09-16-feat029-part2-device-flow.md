---
title: "FEAT-029 part 2 plan — auth login --device"
created: 2026-09-16T00:00:00+04:00
status: PENDING
issue: FEAT-029
brainstorm: docs/brainstorming/2026-09-16-feat029-part2-device-flow.md
deliverables:
    - id: P-01
      title: "internal/auth/deviceflow.go + deviceflow_test.go — device-grant client with poll state machine (BR-01)"
    - id: P-02
      title: "cmd/auth.go --device wiring + cmd/auth_device_test.go — login flow, Secrets persistence, redaction (BR-02)"
    - id: P-03
      title: "OQ-1 research note appended to the brainstorm — verified exchange step (BR-03)"
    - id: P-04
      title: "USAGE.md section + help text (BR-04)"
---

# Goal

Ship `cosmoflare auth login --device`: OAuth 2.0 Device Authorization Grant
against Cloudflare, storing the resulting credential in the existing Secrets
store. Preconditions: operator registers the CosmoLabs OAuth client (D1) and
sets the client_id.

# Steps (ordered)

1. **OQ-1 research (first, blocks nothing else)** — verify against current
   developers.cloudflare.com: the device-authorization endpoint
   (`POST https://dash.cloudflare.com/oauth2/device`), token endpoint
   (`POST /oauth2/token`, `grant_type=urn:ietf:params:oauth:grant-type:device_code`),
   poll-interval/slow_down semantics, and whether an exchange step
   (device access token → scoped API token) is required for `cloudflare-go`
   use. Append findings with citations to the brainstorm (BR-03).
2. **internal/auth/deviceflow.go** — `DeviceFlowClient` (endpoint + client_id
   injectable for tests; defaults per D2: const + `COSMOFLARE_OAUTH_CLIENT_ID`
   override). `Start(ctx) (*DeviceAuth, error)`; `Poll(ctx, da)` returning
   typed states: `StatePending`, `StateSlowDown` (double interval),
   `StateExpired`, `StateDeclined`, `TokenResult{AccessToken, ...}`.
   Timeouts respect `expires_in`; no secrets in error strings.
3. **deviceflow_test.go** — httptest doubles for both endpoints; table tests
   for every poll state, interval doubling on slow_down, expiry, network
   error.
4. **cmd/auth.go** — `--device` flag on `login`: replaces the token-prompt
   path when set. Prints verification URL + user_code (never device_code),
   tries `open`/`xdg-open` unless `--no-open`, polls to completion, stores
   credential via `Secrets` into the active profile (mirror how the existing
   login stores the pasted token). `--json` emits one final envelope.
5. **cmd/auth_device_test.go** — full happy path against httptest doubles;
   decline/expiry error messages; `--no-open`; store-write assertion with a
   fake Secrets backend (pattern exists from FEAT-029 part 1 tests).
6. **USAGE.md** — `auth login --device` section, examples, security note.
7. **Gates** — `go test ./internal/auth/ ./cmd/ -count=1` green; funlen=80;
   conventional commit `feat(auth): device-flow login (FEAT-029 part 2)`.

# Files

- `internal/auth/deviceflow.go` (new), `internal/auth/deviceflow_test.go` (new)
- `cmd/auth.go` (edit), `cmd/auth_device_test.go` (new)
- `docs/USAGE.md` (edit), brainstorm doc (OQ-1 note)

# Dispatch note

Steps 2-6 are bounded GLM work AFTER step 1 lands the verified endpoints.
Step 1 is a web-research task: `ccs glm-agent exec --web-research` (GLM
reads web directly in its isolated context) or one Agent() URL read.
