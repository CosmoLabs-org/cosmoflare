# Cosmoflare Ops (FEAT-052)

Private, phone-installable dashboard and push-alert pager for one Cloudflare
account. A small Worker serves the `pager/` PWA as static assets and exposes a
small private API. Every `/api/*` request is verified against the Cloudflare
Access JWT (RS256; audience, issuer and expiry checked); the Worker refuses
everything when `ACCESS_TEAM_DOMAIN` or `ACCESS_AUD` is unset, so the API
stays closed even if the Access app is later removed.

## API

- `GET /api/summary` — 24h Workers requests/errors/CPU, D1 rows per database,
  zone cache health — live from the GraphQL Analytics API.
- `GET /api/billing` — month-to-date billing: period resolution
  (subscription → anchor day → calendar), per-product allowance / usage /
  projection / projected overage, top consumers, project attribution, verified
  prices. `?refresh` forces a reload behind the 60 s refresh-storm guard.
- `GET /api/subscribe` — count of registered push devices.
- `POST /api/subscribe` — register a Web Push subscription
  (`{ endpoint, keys: { p256dh, auth } }`, max 10; same endpoint replaces).
- `DELETE /api/subscribe` — remove a subscription (`{ endpoint }`).
- `POST /api/test-fire` — fire a real push at every subscribed device
  (`{ sent, pruned, issues }`). Other methods: 405.
- `GET /api/vapid-public-key` — the key the pager needs to subscribe to push.

Every API response carries `Cache-Control: private, no-store`.

## Caching (cache.ts)

`cached()` is the single upstream-cache layer: L1 module memo, L2 KV when
`OPS_KV` is bound, else the Cache API. Stale-while-revalidate with a
single-flight dedup; `?refresh` forces a reload at most once per 60 s per key.
The response envelope carries `cache: { ageSec, stale }`.

| Dataset | Fresh TTL | Stale window |
|---------|-----------|--------------|
| 24h analytics (summary) | 5 min | 1 h |
| Zone / D1 lists | 1 h | 24 h |
| MTD billing usage | 15 min | 6 h |
| Billing subscription (period) | 12 h | 7 d |

**The Cache API is unavailable behind Cloudflare Access** (Workers docs,
checked 2026-10-09). This hostname is Access-fronted, so the Cache-API layer
is a best-effort no-op and **KV is the shared cache** — bind `OPS_KV` for the
cache, the cron rules and the subscriptions to work.

## Cron (every 5 minutes)

The `*/5 * * * *` trigger evaluates the FEAT-049 alert rules against live
telemetry and pages every subscribed device via Web Push (aes128gcm). Rules,
cooldown state and subscriptions live in `OPS_KV` (`rules`, `fire-state`,
`subs`); dead endpoints (404/410) are pruned after each run. Each run logs one
`{ cron: "alerts", ... }` result line.

## Lock-down (do this BEFORE the first deploy)

1. Create a Cloudflare Access application (self-hosted) for the Worker's
   hostname, `cosmoflare-ops.<your-subdomain>.workers.dev`, with an Allow
   policy that includes only your email. Note its **audience (AUD) tag** and
   your team domain (`<team>.cloudflareaccess.com`).
2. Keep `preview_urls` off — the Access app covers the main hostname only.
3. Optional custom domain: attach it as a Workers Custom Domain (dashboard or
   `PUT /accounts/{id}/workers/domains`), add the hostname to the same Access
   app (same audience tag), and keep `workers_dev: false`.

## Deploy

```bash
bun install
bun run deploy                           # builds pager/ then wrangler deploy
wrangler secret put ACCESS_TEAM_DOMAIN   # <team>.cloudflareaccess.com
wrangler secret put ACCESS_AUD           # the Access app audience tag
wrangler secret put CF_ACCOUNT_ID
wrangler secret put CF_API_TOKEN         # read-only scope: Analytics, Zone, D1, Billing read
wrangler secret put VAPID_PUBLIC_KEY     # from: cosmoflare alerts push list --json
wrangler secret put VAPID_PRIVATE_KEY    # the matching VAPID private key
wrangler secret put VAPID_SUBJECT        # mailto:alerts@cosmolabs.org
# KV namespace for cache + rules + subscriptions (fill the id into
# wrangler.jsonc kv_namespaces — the orchestrator owns this step):
#   wrangler kv namespace create OPS_KV
```

Optional var `BILLING_ANCHOR_DAY` ("1"-"31", e.g. `wrangler secret put` or a
`[vars]` entry): day of month the billing period restarts, used when the
subscriptions endpoint is not readable with the account token (live-verified
2026-10-09: it returns 403 with the ops token's scopes) — without it the
period falls back to the calendar month.

## Pairing the phone

Open the site, Add to Home Screen (iOS 16.4+), open it from the Home Screen,
Pairing → **Enable notifications** — the VAPID key is fetched before the tap
(Apple requires `subscribe()` to follow the user gesture directly) and the
subscription is registered with the Worker automatically; the device then
receives cron alerts. **Copy subscription** stays as the secondary, CLI-driven
path (`cosmoflare alerts push add '<json>'`), and **Send test alert** fires a
real push so the whole pipeline can be checked from the phone.
