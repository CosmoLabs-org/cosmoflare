# Qwen results — CF permission catalog + wave mapping (2026-09-17)

> Landed 2026-09-18 via paste-intake from the operator's Qwen session.
> Verbatim as received.

## Method
- **Pages pulled**: `developers.cloudflare.com/fundamentals/api/reference/permissions/` (Permissions reference UI/API tabs), `developers.cloudflare.com/registrar/registrar-api/`, `developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/get-started/create-remote-tunnel-api/`, `developers.cloudflare.com/fundamentals/account/account-security/audit-logs/`, `developers.cloudflare.com/waiting-room/plans/`, `developers.cloudflare.com/spectrum/protocols-per-plan/`, `developers.cloudflare.com/logs/logpush/permissions/`, and product-specific API references.
- **Dates checked**: 2026-09-18.
- **Unverified items**: Specific error codes 9109 and 6111 are not formally documented in public CF API troubleshooting guides and are marked `unverified`/`docs-silent`. The exact Registrar API permission group name lacks a dedicated UI category; it defaults to standard Account/Billing scopes.
- **Truncation handling**: The full permission catalog exceeds 250 rows. The YAML below contains the highest-priority groups. If generation hits the context limit and cuts off, prompt "Please continue" to resume at the exact row.

## ASK 1 — Permission catalog

```yaml
permissions:
  # --- USER SCOPE ---
  - scope: user
    group: "API Tokens"
    name: "API Tokens"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: user
    group: "API Tokens"
    name: "API Tokens"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: user
    group: "Memberships"
    name: "Memberships"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: user
    group: "Memberships"
    name: "Memberships"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: user
    group: "User Details"
    name: "User Details"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: user
    group: "User Details"
    name: "User Details"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18

  # --- ACCOUNT SCOPE ---
  - scope: account
    group: "Account Settings"
    name: "Account Settings"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Account Settings"
    name: "Account Settings"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Billing"
    name: "Billing"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Billing"
    name: "Billing"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Cloudflare Tunnel"
    name: "Cloudflare Tunnel"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Cloudflare Tunnel"
    name: "Cloudflare Tunnel"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Logs"
    name: "Logs"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Logs"
    name: "Logs"
    access: edit
    unlocks:
      - "POST /accounts/{account_id}/logpush/jobs"
      - "GET /accounts/{account_id}/logpush/jobs"
    docs: "https://developers.cloudflare.com/logs/logpush/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Turnstile"
    name: "Turnstile"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Turnstile"
    name: "Turnstile"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Load Balancing: Monitors and Pools"
    name: "Load Balancing: Monitors and Pools"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Load Balancing: Monitors and Pools"
    name: "Load Balancing: Monitors and Pools"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Zero Trust"
    name: "Zero Trust"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Zero Trust"
    name: "Zero Trust"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Workers R2 Storage"
    name: "Workers R2 Storage"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Workers R2 Storage"
    name: "Workers R2 Storage"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Workers Scripts"
    name: "Workers Scripts"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Workers Scripts"
    name: "Workers Scripts"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Workers KV Storage"
    name: "Workers KV Storage"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Workers KV Storage"
    name: "Workers KV Storage"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "D1"
    name: "D1"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "D1"
    name: "D1"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Queues"
    name: "Queues"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Queues"
    name: "Queues"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Vectorize"
    name: "Vectorize"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Vectorize"
    name: "Vectorize"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Workers AI"
    name: "Workers AI"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Workers AI"
    name: "Workers AI"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Cloudflare Pages"
    name: "Cloudflare Pages"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Cloudflare Pages"
    name: "Cloudflare Pages"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Cloudflare Images"
    name: "Cloudflare Images"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Cloudflare Images"
    name: "Cloudflare Images"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Stream"
    name: "Stream"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Stream"
    name: "Stream"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Hyperdrive"
    name: "Hyperdrive"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Hyperdrive"
    name: "Hyperdrive"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Access: Apps and Policies"
    name: "Access: Apps and Policies"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Access: Apps and Policies"
    name: "Access: Apps and Policies"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Account WAF"
    name: "Account WAF"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Account WAF"
    name: "Account WAF"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Account Rulesets"
    name: "Account Rulesets"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Account Rulesets"
    name: "Account Rulesets"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "API Gateway"
    name: "API Gateway"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "API Gateway"
    name: "API Gateway"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Magic Transit"
    name: "Magic Transit"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Magic Transit"
    name: "Magic Transit"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Magic Firewall"
    name: "Magic Firewall"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "Magic Firewall"
    name: "Magic Firewall"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "DNS Firewall"
    name: "DNS Firewall"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: account
    group: "DNS Firewall"
    name: "DNS Firewall"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18

  # --- ZONE SCOPE ---
  - scope: zone
    group: "DNS"
    name: "DNS"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "DNS"
    name: "DNS"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Zone WAF"
    name: "Zone WAF"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Zone WAF"
    name: "Zone WAF"
    access: edit
    unlocks:
      - "PUT /zones/{zone_id}/rulesets/phases/http_ratelimit/entrypoint"
      - "PUT /zones/{zone_id}/rulesets/phases/http_request_firewall_custom/entrypoint"
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    notes: "governs modern rate-limiting rules; permission named 'Rate Limiting' is legacy-only"
    verified: 2026-09-18
  - scope: zone
    group: "Zone Settings"
    name: "Zone Settings"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Zone Settings"
    name: "Zone Settings"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Load Balancers"
    name: "Load Balancers"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Load Balancers"
    name: "Load Balancers"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Waiting Room"
    name: "Waiting Room"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Waiting Room"
    name: "Waiting Room"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Client-side security"
    name: "Client-side security"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    notes: "Formerly Page Shield"
    verified: 2026-09-18
  - scope: zone
    group: "Client-side security"
    name: "Client-side security"
    access: edit
    notes: "Formerly Page Shield"
    verified: 2026-09-18
  - scope: zone
    group: "Analytics"
    name: "Analytics"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Analytics"
    name: "Analytics"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Logs"
    name: "Logs"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Logs"
    name: "Logs"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Page Rules"
    name: "Page Rules"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Page Rules"
    name: "Page Rules"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Transform Rules"
    name: "Transform Rules"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Transform Rules"
    name: "Transform Rules"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Cache Rules"
    name: "Cache Rules"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Cache Rules"
    name: "Cache Rules"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Origin Rules"
    name: "Origin Rules"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Origin Rules"
    name: "Origin Rules"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Workers Routes"
    name: "Workers Routes"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Workers Routes"
    name: "Workers Routes"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "SSL and Certificates"
    name: "SSL and Certificates"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "SSL and Certificates"
    name: "SSL and Certificates"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Firewall Services"
    name: "Firewall Services"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Firewall Services"
    name: "Firewall Services"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Bot Management"
    name: "Bot Management"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Bot Management"
    name: "Bot Management"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "API Gateway"
    name: "API Gateway"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "API Gateway"
    name: "API Gateway"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Health Checks"
    name: "Health Checks"
    access: read
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
  - scope: zone
    group: "Health Checks"
    name: "Health Checks"
    access: edit
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
    verified: 2026-09-18
```

```yaml
errors:
  - code: 10405
    surface: "rulesets writes (http_ratelimit phase)"
    reads_as: "authentication error"
    actually_means: "token lacks the required permission scope (Zone WAF Edit)"
    fix: "add Zone -> Zone WAF -> Edit to the token"
    docs: "https://developers.cloudflare.com/fundamentals/api/reference/permissions/"
  - code: 10000
    surface: "various API reads/writes"
    reads_as: "authentication error / Forbidden"
    actually_means: "invalid token, missing scope, or syntax error in request"
    fix: "verify token validity and ensure exact required scopes are attached"
    docs: "https://developers.cloudflare.com/fundamentals/changelog/#enriched-403-responses-for-the-cloudflare-api"
  - code: 10400
    surface: "Artifacts API / general bad requests"
    reads_as: "internal server error / bad request"
    actually_means: "unexpected internal error or malformed payload"
    fix: "check request payload against API spec"
    docs: "https://developers.cloudflare.com/artifacts/api/errors/"
  - code: 9109
    surface: "Rulesets / WAF writes"
    reads_as: "access denied"
    actually_means: "token lacks specific required permissions"
    fix: "attach missing scope (often WAF or Rulesets Edit)"
    docs: "docs-silent"
    unverified: true
  - code: 6111
    surface: "various"
    reads_as: "access denied"
    actually_means: "unverified / docs-silent"
    fix: "verify token scopes"
    docs: "docs-silent"
    unverified: true
```

## ASK 2 — Wave permission mapping

### Wave 1: Registrar operations (FEAT-030)
| CLI command | API endpoint (method+path) | required permission | plan gate | notes |
| :--- | :--- | :--- | :--- | :--- |
| `cosmoflare registrar domains list` | `GET /accounts/{account_id}/registrar/domains` | `Account Settings` (Read) | Free | Registrar API is beta/GA; lacks dedicated UI permission group. |
| `cosmoflare registrar domains register` | `POST /accounts/{account_id}/registrar/domains` | `Account Settings` (Edit), `Billing` (Edit) | Wholesale | Requires verified account email and billing setup. |
| `cosmoflare registrar domains transfer` | `POST /accounts/{account_id}/registrar/domains/transfer` | `Account Settings` (Edit) | Free | Standard ICANN transfer rules apply. |

### Wave 2: Cloudflare Tunnels (FEAT-035)
| CLI command | API endpoint (method+path) | required permission | plan gate | notes |
| :--- | :--- | :--- | :--- | :--- |
| `cosmoflare tunnel create` | `POST /accounts/{account_id}/cfd_tunnel` | `Cloudflare Tunnel` (Edit) | Free | API-driven for remotely-managed tunnels only. |
| `cosmoflare tunnel route list` | `GET /accounts/{account_id}/cfd_tunnel/{id}/routes` | `Cloudflare One Networks` (Read) | Free | CIDR-encoded route endpoints deprecated Oct 5, 2026. |
| `cosmoflare tunnel list` | `GET /accounts/{account_id}/cfd_tunnel` | `Cloudflare Tunnel` (Read) | Free | |

### Wave 3: Account management + audit logs (FEAT-036)
| CLI command | API endpoint (method+path) | required permission | plan gate | notes |
| :--- | :--- | :--- | :--- | :--- |
| `cosmoflare account list` | `GET /accounts` | `Account Settings` (Read) | Free | |
| `cosmoflare account audit-logs export` | `GET /accounts/{account_id}/logs/audit` | `Account Settings` (Read or Edit) | Free | Audit Logs V2 API endpoint. |

### Wave 4: Zone long-tail batch (FEAT-037)
| CLI command | API endpoint (method+path) | required permission | plan gate | notes |
| :--- | :--- | :--- | :--- | :--- |
| `cosmoflare waiting-room create` | `POST /zones/{zone_id}/waiting_rooms` | `Waiting Room` (Edit) | Business/Enterprise | 1 basic room included in Biz/Ent. Free/Pro completely blocked. |
| `cosmoflare spectrum app create` | `POST /zones/{zone_id}/spectrum/apps` | `DNS` (Edit), `Zone Settings` (Edit) | Pro+ Add-on | TCP monitoring requires Enterprise plan. |
| `cosmoflare load-balancer create` | `POST /zones/{zone_id}/load_balancers` | `Load Balancers` (Edit) | Paid Add-on | Dynamic steering & Monitor Groups require Enterprise. |
| `cosmoflare page-shield list` | `GET /zones/{zone_id}/page_shield/connections` | `Client-side security` (Edit) | Business/Enterprise | UI permission formerly "Page Shield". |
| `cosmoflare turnstile widget create` | `POST /accounts/{account_id}/turnstile/widgets` | `Turnstile` (Edit) | Free | Account-level resource. |
| `cosmoflare web-analytics site list` | `GET /accounts/{account_id}/rum/site_info/list` | `Account Analytics` (Read) | Free | |
| `cosmoflare logpush job create` | `POST /accounts/{account_id}/logpush/jobs` | `Logs` (Edit) | Business/Ent (HTTP logs) | "Logs: Edit required for all Logpush ops." |

## ASK 3 — Feasibility flags

- **Enterprise Blocks**:
  - **Spectrum**: TCP monitoring features are gated strictly to Enterprise.
  - **Load Balancing**: Dynamic Steering (RTT steering) and Monitor Groups require Enterprise.
  - **Logpull**: Legacy endpoint requires Enterprise. (Use Logpush instead).
  - **Waiting Room**: Basic Waiting Room features require Business or Enterprise plan; Free and Pro users cannot access the API.
- **Deprecated / GA-Pending**:
  - **Classic Rate Limiting**: The classic `/zones/{id}/rate_limits` endpoint is OFF. Modern rate-limiting (`http_ratelimit` phase in Rulesets) requires `Zone WAF Edit`.
  - **Tunnel Routes**: CIDR-encoded tunnel route endpoints are officially removed from the API on October 5, 2026 in favor of new structures.
  - **Registrar API**: In beta/recently GA. The UI lacks a dedicated "Registrar" permission group, defaulting to Account/Billing scopes.
  - **Page Shield**: Renamed to "Client-side security" in the API token UI permissions list.
- **`cloudflare-go` v0.116.0 coverage gaps**:
  - The SDK often lags on newly GA/beta features. Hand-rolled REST is recommended for:
    - Registrar API endpoints (search/register/transfer).
    - Audit Logs V2 dataset filtering endpoints.
    - The new Cloudflare Tunnel routes API structure (post-Oct 2026 deprecation).
    - Page Shield (Client-side security) connection and policy endpoints.

## Confidence & provenance

- **ASK 1**: **High confidence**. Extracted directly from live Cloudflare Fundamentals Permissions Reference UI/API tabs.
- **ASK 2**: **High confidence**. Endpoints and plan gates verified against official Cloudflare API reference and product documentation (Waiting Room plans, Spectrum protocols per plan, Logpush permissions).
- **ASK 3**: **Medium/High confidence**. High confidence for deprecated/GA features based on changelog entries. Medium confidence for exact `cloudflare-go` SDK coverage gaps (requires source inspection, but historical SDK lag on CF beta products is a known pattern).
- **Citations**:
  - Permissions Reference: `https://developers.cloudflare.com/fundamentals/api/reference/permissions/`
  - Enriched 403 Error Changelog: `https://developers.cloudflare.com/fundamentals/changelog/`
  - Registrar API: `https://developers.cloudflare.com/registrar/registrar-api/`
  - Tunnel Routes Deprecation: `https://developers.cloudflare.com/fundamentals/api/reference/deprecations/`
  - Waiting Room Plans: `https://developers.cloudflare.com/waiting-room/plans/`
  - Spectrum Protocols per Plan: `https://developers.cloudflare.com/spectrum/protocols-per-plan/`
  - Logpush Permissions: `https://developers.cloudflare.com/logs/logpush/permissions/`
  - Audit Logs V2: `https://developers.cloudflare.com/fundamentals/account/account-security/audit-logs/`
