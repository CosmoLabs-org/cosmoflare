# Cloudflare OAuth 2.0 Device Authorization Grant — research note (2026-09-17)

Scope of research: exact endpoints, parameters, polling contract, whether the device-flow
access token works directly against `api.cloudflare.com/client/v4` (cloudflare-go), and
token lifetime/refresh semantics. Every claim carries a citation. Sources are either
current developers.cloudflare.com pages, the Cloudflare blog, or the wrangler/workers-sdk
source on GitHub (which is the de-facto specification of the device flow, since the
first-party endpoints are not fully published in docs).

Key context up front: the device flow as shipped today is used by Cloudflare's own
first-party CLIs (`wrangler login --device`, the `cf` CLI). Third-party OAuth clients
(you register these yourself at dash.cloudflare.com > Manage Account > OAuth clients)
are documented to support **only** the Authorization Code flow — "Cloudflare does not
support Client Credentials, Implicit, Resource Owner Password Credentials, Device
Authorization, or other OAuth grant types for third-party clients"
([Create your OAuth client](https://developers.cloudflare.com/fundamentals/oauth/create-an-oauth-client/)).
All endpoint/parameter detail below is therefore sourced from wrangler's implementation
plus the wrangler docs/changelog.

---

## 1. Device-authorization endpoint

**Exact URL (API): `https://dash.cloudflare.com/oauth2/device/auth`**

- The path is `DEVICE_AUTH_PATH = "/oauth2/device/auth"` on the auth domain
  `dash.cloudflare.com` (staging: `dash.staging.cloudflare.com`), defined in wrangler's
  `env-vars.ts`, which cites "the OAuth 2.0 Device Authorization endpoint (RFC 8628 §3.1)".
  It is deliberately not environment-overridable.
  Source: [workers-auth/src/env-vars.ts](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/env-vars.ts)
- This endpoint is NOT listed on the public endpoint reference page — that page lists
  only `oauth2/auth`, `oauth2/token`, `oauth2/revoke`, `oauth2/logout`, `oauth2/userinfo`
  and the `.well-known` URLs
  ([Integrate your OAuth client with Cloudflare](https://developers.cloudflare.com/fundamentals/oauth/integrate-with-cloudflare/)).
  The docs do not publish the device endpoint because third-party clients are not
  allowed to use the device grant
  ([Create your OAuth client](https://developers.cloudflare.com/fundamentals/oauth/create-an-oauth-client/)).

**Request** — `POST`, `Content-Type: application/x-www-form-urlencoded`, body params:

| Parameter   | Value |
|-------------|-------|
| `client_id` | The OAuth client ID (for wrangler this is Cloudflare's first-party client ID) |
| `scope`     | Space-delimited scope list; wrangler appends `offline_access` unconditionally so the token response includes a refresh token |

Source: `requestDeviceAuthorization()` in
[workers-auth/src/device-flow.ts](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/device-flow.ts)
(`const params = new URLSearchParams({ client_id: clientId, scope: [...scopes, "offline_access"].join(" ") })`).
No client_secret is sent (public client).

**Response** (JSON): `device_code`, `user_code`, `verification_uri`,
`verification_uri_complete` (optional), `expires_in` (seconds), `interval` (optional,
seconds) — RFC 8628 §3.2 shape, per the `DeviceAuthorizationResponse` type in
[device-flow.ts](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/device-flow.ts).

**User-facing verification URL (what the human visits): `https://dash.cloudflare.com/oauth2/device`**

- Documented in the wrangler docs: "`wrangler login --device` ... uses the OAuth 2.0
  Device Authorization Grant (RFC 8628). This flow does not start a local callback
  server. You will visit `https://dash.cloudflare.com/oauth2/device` and enter the code"
  ([Wrangler general commands](https://developers.cloudflare.com/workers/wrangler/commands/general/));
  the browser link carries the code pre-filled, e.g.
  `https://dash.cloudflare.com/oauth2/device?user_code=jPqK6Qvs` (same page).
- Confirmed by the changelog entry for the feature (Wrangler 4.119.0+)
  ([changelog 2026-08-04](https://developers.cloudflare.com/changelog/post/2026-08-04-wrangler-login-device-flow/)).

**Valid scope values for account/user read**

- First-party format (colon-delimited — what the device flow actually uses, since the
  device grant is first-party only): `account:read` = "See your account info such as
  account details, analytics, and memberships." and `user:read` = "See your user info
  such as name, email address, and account memberships." — wrangler's scope catalog in
  [wrangler@2.20.1 user.ts](https://github.com/cloudflare/workers-sdk/blob/wrangler@2.20.1/packages/wrangler/src/user/user.ts)
  and the same catalog carried forward in the current
  [workers-auth cf/scopes.ts](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/cf/scopes.ts)
  (`account:read`, `user:read`, `zone:read`, `workers:write`, `dns_records:edit`, ...).
  The docs example is literally `npx wrangler login --scopes account:read user:read`
  ([Wrangler general commands](https://developers.cloudflare.com/workers/wrangler/commands/general/)).
- Third-party client format (dot-delimited): if you build your own OAuth client
  (authorization-code flow only), scope names "correspond to Cloudflare API token
  permission names" and are listed via `GET https://api.cloudflare.com/client/v4/oauth/scopes`;
  docs example scope IDs are `workers-platform.read` / `workers-platform.write`
  ([Create your OAuth client](https://developers.cloudflare.com/fundamentals/oauth/create-an-oauth-client/)).
  A community integration guide reports the valid shape there is dot-delimited
  (`account.read`) and the colon form is rejected for third-party clients
  ([ubitools guide](https://www.ubitools.com/cloudflare-oauth-client) — third-party source).
  `offline_access` is the protocol scope that gets you a refresh token (same guide;
  also wrangler appends it automatically, see device-flow.ts above).

---

## 2. Token endpoint

**Exact URL: `https://dash.cloudflare.com/oauth2/token`**

- Documented on the endpoint reference page:
  "Token: `https://dash.cloudflare.com/oauth2/token`"
  ([Integrate your OAuth client with Cloudflare](https://developers.cloudflare.com/fundamentals/oauth/integrate-with-cloudflare/)).
- Same URL is wrangler's token endpoint default (`getTokenUrlFromEnv` →
  `https://dash.cloudflare.com/oauth2/token`)
  [workers-auth/src/env-vars.ts](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/env-vars.ts).
  Related: revoke endpoint `https://dash.cloudflare.com/oauth2/revoke` (same file and
  the docs endpoint page above).

**Grant type string: `urn:ietf:params:oauth:grant-type:device_code`** — sent as
`grant_type` in the form body together with `device_code` and `client_id` (no secret).
Per `pollDeviceToken()` in
[workers-auth/src/device-flow.ts](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/device-flow.ts),
which cites RFC 8628 §3.4.

**Polling error contract** (RFC 8628 §3.5; errors arrive with HTTP 400 and an `error`
JSON member — the `error` member is authoritative over the HTTP status):

| `error` value        | Meaning / client behavior |
|----------------------|---------------------------|
| `authorization_pending` | User has not approved yet — keep polling at the current interval |
| `slow_down`           | Increase the polling interval by 5 seconds "for this and all subsequent requests" (wrangler: `intervalSeconds += 5`) |
| `access_denied`       | Terminal — user denied consent; abort login |
| `expired_token`       | Terminal — device code expired before approval; restart the flow to get a new code |

Source: the `switch (result.error)` block in
[workers-auth/src/device-flow.ts](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/device-flow.ts),
with the comment "RFC 8628 §3.5 delivers `authorization_pending` / `slow_down` as an
`error` with HTTP 400 on every poll while the user is still approving".

**Recommended poll-interval handling** (wrangler's implementation, same file):

- Start from the server-provided `interval`; RFC 8628 §3.5 default is 5 s when omitted.
  Wrangler floors it at 1 s for CLI responsiveness ("a 5 second baseline feels
  unacceptably slow") and honors the server value whenever it is larger;
  `slow_down` adds +5 s on top, permanently.
- Poll immediately (first poll is not delayed behind `interval`).
- Hard cap on total polling: min(server `expires_in`, 300 s) — "Wrangler stops polling
  after 5 minutes, or sooner if Cloudflare sets a shorter expiry on the user code"
  ([Wrangler general commands](https://developers.cloudflare.com/workers/wrangler/commands/general/));
  the changelog example prints "You have 5 minutes to approve this request."
  ([changelog](https://developers.cloudflare.com/changelog/post/2026-08-04-wrangler-login-device-flow/)).
- Transient failures (network errors, HTML/bot-challenge bodies instead of JSON, 5xx,
  429) are retried within the deadline; any other unusable non-2xx aborts.

**Success response** (JSON): `access_token`, `expires_in` (seconds), `refresh_token`
(optional; present when `offline_access` was granted), `scope` (space-delimited) —
per the `DeviceTokenGrant` type in
[device-flow.ts](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/device-flow.ts).

---

## 3. Direct use vs. exchange — the critical question

**Verdict: NO exchange step exists. The device-flow OAuth access token is used
DIRECTLY as `Authorization: Bearer <token>` against `api.cloudflare.com/client/v4`.**
The premise that wrangler exchanges its OAuth token for a scoped API token is not what
current (or even 2.x-era) wrangler does. Evidence:

1. **Wrangler source (current, `workers-auth`):** the credential resolver returns the
   stored OAuth access token as the API credential — `return { apiToken: stored.accessToken.value };`
   — in
   [workers-auth/src/credentials.ts](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/credentials.ts).
   That `apiToken` is sent by wrangler's API fetcher as
   `headers.set("Authorization", `Bearer ${auth.apiToken}`)` for every call to
   `api.cloudflare.com`
   ([workers-utils/src/cfetch/index.ts](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-utils/src/cfetch/index.ts)).
2. **Wrangler source (2.x, 2022-2023):** identical pattern —
   `headers["Authorization"] = `Bearer ${auth.apiToken}`` in
   [wrangler@2.20.1 cfetch/internal.ts](https://github.com/cloudflare/workers-sdk/blob/wrangler@2.20.1/packages/wrangler/src/cfetch/internal.ts),
   where the token comes from the OAuth `oauth_token` field of the stored config
   ([wrangler@2.20.1 user.ts](https://github.com/cloudflare/workers-sdk/blob/wrangler@2.20.1/packages/wrangler/src/user/user.ts)).
3. **Repo-wide search** for an OAuth→API-token exchange endpoint
   (`user/tokens/oauth`, similar) in cloudflare/workers-sdk returns no hits — there is
   no exchange call anywhere in the current tree.
4. **Wrangler docs:** `wrangler auth token` returns the stored credential as
   `{"type": "oauth", "token": "..."}` — i.e. the OAuth token itself is the API
   credential wrangler operates with; API tokens (env vars) are merely an alternative,
   higher-priority auth method
   ([Wrangler general commands](https://developers.cloudflare.com/workers/workers/wrangler/commands/general/) —
   see "wrangler auth token" section on that page).
5. **Cloudflare blog:** after login wrangler "uses the access token for API calls";
   refresh tokens exist to replace "expired short-lived access tokens"
   ([Bringing OAuth 2.0 to Wrangler](https://blog.cloudflare.com/wrangler-oauth/)).
6. **Scopes are the permission mechanism**, not token minting: "OAuth scope names
   correspond to Cloudflare API token permission names" — the consented scopes
   directly define what the bearer OAuth token may call
   ([Create your OAuth client](https://developers.cloudflare.com/fundamentals/oauth/create-an-oauth-client/)).

**If an exchange were required (it is not):** there is no documented endpoint to
exchange an OAuth access token for an API token. The normal way to create a scoped API
token is `POST https://api.cloudflare.com/client/v4/user/tokens` with an API-token
bearer — documented at
[Create tokens via API](https://developers.cloudflare.com/fundamentals/api/how-to/create-via-api/) —
but that endpoint expects a (pre-existing) API token, not an OAuth token, and wrangler
does not call it during login. A stale code comment in wrangler's env-vars.ts still
calls `/oauth2/token` "the path that is used to exchange an OAuth token for an API
token" — it is leftover wording; the URL is the plain OAuth token endpoint and no API
token is produced.

**cloudflare-go side (what cosmoflare pins, v0.116.0):** the SDK builds
`https://api.cloudflare.com/client/v4` as its default base
([consts.go: defaultHostname = "api.cloudflare.com", defaultBasePath = "/client/v4"](https://github.com/cloudflare/cloudflare-go/blob/v0.116.0/consts.go))
and authenticates with `req.Header.Set("Authorization", "Bearer "+api.APIToken)`
([cloudflare.go](https://github.com/cloudflare/cloudflare-go/blob/v0.116.0/cloudflare.go),
the `AuthToken` path used by `NewAPITokenClient`). That is byte-for-byte the same
header wrangler sends with its OAuth token, against the same base URL — so the
device-flow access token slots straight into `cloudflare.NewAPITokenClient(token)`.
cloudflare-go has no OAuth/device/refresh support of its own; it only knows "a bearer
string" (verified: no OAuth-flow code exists in the repo beyond IAM endpoints that
*manage* OAuth clients).

---

## 4. Token lifetime / refresh semantics

- **Access token: short-lived, server-set TTL.** The token response carries `expires_in`
  (seconds); wrangler stores `expiry = now + expires_in` and refreshes before use
  ([device-flow.ts success path](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/device-flow.ts)).
  The Cloudflare blog describes them as "short-lived access tokens" with three
  mitigations for theft ([wrangler-oauth blog](https://blog.cloudflare.com/wrangler-oauth/)).
  The exact numeric TTL is not published in Cloudflare docs; it is server-controlled,
  so a client must honor `expires_in` rather than assume a fixed value.
- **Refresh token: issued because wrangler requests `offline_access`** (appended to
  every device-flow scope request,
  [device-flow.ts](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/device-flow.ts)).
  Refresh uses the same token endpoint with `grant_type=refresh_token`,
  `refresh_token=<token>`, `client_id=<id>` (public client — no secret)
  ([workers-auth/src/token-exchange.ts](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/token-exchange.ts)).
- **Refresh response semantics:** returns a new `access_token` + `expires_in` +
  `scope`; `refresh_token` may be omitted, in which case "the previously issued
  refresh token remains valid (RFC 6749 §6)" — wrangler preserves the stored one
  (same file). The docs behavior: "`wrangler auth token` ... the OAuth token is
  automatically refreshed if expired"
  ([Wrangler general commands](https://developers.cloudflare.com/workers/wrangler/commands/general/)).
- **Revocation:** `wrangler logout` invalidates the refresh token, which invalidates
  the associated access token ([wrangler-oauth blog](https://blog.cloudflare.com/wrangler-oauth/));
  the endpoint is `POST https://dash.cloudflare.com/oauth2/revoke`
  ([endpoint reference](https://developers.cloudflare.com/fundamentals/oauth/integrate-with-cloudflare/)).
  Users can also revoke authorizations in the dashboard at "Manage OAuth authorizations"
  ([Authorizing an application](https://developers.cloudflare.com/fundamentals/oauth/authorizing-an-application/)).
- **Device/user code lifetime:** the `expires_in` on the device-authorization response
  governs the user code; wrangler caps polling at 300 s regardless
  ([device-flow.ts](https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/device-flow.ts)
  and [wrangler docs](https://developers.cloudflare.com/workers/wrangler/commands/general/)).

---

## VERDICT

**YES — cloudflare-go can consume the device-flow access token directly, with two
operational caveats.** The device-flow token from `dash.cloudflare.com/oauth2/token`
is a bearer credential that wrangler itself sends verbatim as
`Authorization: Bearer <token>` to `https://api.cloudflare.com/client/v4`
(source-proven in both current workers-auth and wrangler 2.x: the stored OAuth
`accessToken.value` is passed as `apiToken` and wrapped only in a Bearer header), and
cloudflare-go v0.116.0 — the version cosmoflare pins — sends exactly that header from
that exact base URL via `NewAPITokenClient`, so `cloudflare.NewAPITokenClient(oauthAccessToken)`
works out of the box for any endpoint covered by the consented scopes (`account:read`
/ `user:read` for the read paths this research was scoped to). No exchange step exists
or is needed — the "wrangler creates a scoped API token" model is a misconception;
scopes on the OAuth token themselves map to API-token permission names. Caveat one:
the token is short-lived (`expires_in`) and cloudflare-go has zero refresh support, so
the caller must run the `grant_type=refresh_token` exchange (same token endpoint, plus
`offline_access` in the original scope request) out-of-band and rebuild the SDK client
on refresh, or accept 401s after expiry. Caveat two: the device grant itself is
restricted to Cloudflare first-party clients — current docs state third-party OAuth
clients support authorization-code only — so a custom CLI like cosmoflare cannot
register its own `client_id` for the device flow; adopting this flow means borrowing
first-party treatment (an arrangement with Cloudflare) or falling back to
authorization-code + PKCE for a self-registered client, where the same direct-bearer
usage applies but scopes use dot-delimited names (`account.read`).

Sources:
- https://developers.cloudflare.com/fundamentals/oauth/integrate-with-cloudflare/
- https://developers.cloudflare.com/fundamentals/oauth/create-an-oauth-client/
- https://developers.cloudflare.com/fundamentals/oauth/authorizing-an-application/
- https://developers.cloudflare.com/workers/wrangler/commands/general/
- https://developers.cloudflare.com/changelog/post/2026-08-04-wrangler-login-device-flow/
- https://developers.cloudflare.com/fundamentals/api/how-to/create-via-api/
- https://blog.cloudflare.com/wrangler-oauth/
- https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/device-flow.ts
- https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/token-exchange.ts
- https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/env-vars.ts
- https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/credentials.ts
- https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-utils/src/cfetch/index.ts
- https://github.com/cloudflare/workers-sdk/blob/main/packages/workers-auth/src/cf/scopes.ts
- https://github.com/cloudflare/workers-sdk/blob/wrangler@2.20.1/packages/wrangler/src/user/user.ts
- https://github.com/cloudflare/workers-sdk/blob/wrangler@2.20.1/packages/wrangler/src/cfetch/internal.ts
- https://github.com/cloudflare/cloudflare-go/blob/v0.116.0/cloudflare.go
- https://github.com/cloudflare/cloudflare-go/blob/v0.116.0/consts.go
- https://www.ubitools.com/cloudflare-oauth-client (community, third-party)
