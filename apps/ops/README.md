# Cosmoflare Ops (FEAT-052)

Private, phone-installable dashboard and push-alert pager for one Cloudflare
account. A small Worker serves the `pager/` PWA as static assets and exposes:

- `GET /api/summary` — month-to-date pacing (Workers requests, D1 rows read),
  D1 rows read per database (24h), zone cache health (uncached volume, miss %)
  — live from the GraphQL Analytics API, cached 5 minutes.
- `GET /api/vapid-public-key` — the key the pager needs to subscribe to push.

## Lock-down

Put a Cloudflare Access application in front of the Worker's hostname with a
policy that allows only your email. Keep `preview_urls` off (the Access app
covers the main hostname only). The Worker also refuses `/api/*` without the
`Cf-Access-Jwt-Assertion` header.

## Deploy

```bash
bun install
bun run deploy                         # builds pager/ then wrangler deploy
wrangler secret put CF_ACCOUNT_ID
wrangler secret put CF_API_TOKEN       # read-only scope is enough: Account Analytics, Zone, D1 read
wrangler secret put VAPID_PUBLIC_KEY   # from: cosmoflare alerts push list --json
```

Pair the phone: open the site, Add to Home Screen (iOS 16.4+), open it from the
Home Screen, Pairing → Enable notifications → Copy subscription, then on the
machine running `cosmoflare alerts watch`: `cosmoflare alerts push add '<json>'`.
