# Cosmoflare Ops (FEAT-052)

Private, phone-installable dashboard and push-alert pager for one Cloudflare
account. A small Worker serves the `pager/` PWA as static assets and exposes:

- `GET /api/summary` — month-to-date pacing (Workers requests, D1 rows read),
  D1 rows read per database (24h), zone cache health (uncached volume, miss %)
  — live from the GraphQL Analytics API, cached 5 minutes.
- `GET /api/vapid-public-key` — the key the pager needs to subscribe to push.

## Lock-down (do this BEFORE the first deploy)

1. Create a Cloudflare Access application (self-hosted) for the Worker's
   hostname, `cosmoflare-ops.<your-subdomain>.workers.dev`, with an Allow
   policy that includes only your email. Note its **audience (AUD) tag** and
   your team domain (`<team>.cloudflareaccess.com`).
2. Keep `preview_urls` off — the Access app covers the main hostname only.

The Worker verifies the Access JWT on every `/api/*` request (RS256 against
the team keys; audience, issuer and expiry checked) and refuses everything
when `ACCESS_TEAM_DOMAIN` or `ACCESS_AUD` is unset, so the API stays closed
even if the Access app is later removed.

## Deploy

```bash
bun install
bun run deploy                           # builds pager/ then wrangler deploy
wrangler secret put ACCESS_TEAM_DOMAIN   # <team>.cloudflareaccess.com
wrangler secret put ACCESS_AUD           # the Access app audience tag
wrangler secret put CF_ACCOUNT_ID
wrangler secret put CF_API_TOKEN         # read-only scope is enough: Account Analytics, Zone, D1 read
wrangler secret put VAPID_PUBLIC_KEY     # from: cosmoflare alerts push list --json
```

Pair the phone: open the site, Add to Home Screen (iOS 16.4+), open it from the
Home Screen, Pairing → Enable notifications → Copy subscription, then on the
machine running `cosmoflare alerts watch`: `cosmoflare alerts push add '<json>'`.
