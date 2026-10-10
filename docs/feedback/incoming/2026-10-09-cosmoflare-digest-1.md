---
kind: digest
ulid: 01M4H219JFGM1P7N5A1DZ2G4GP
id: FB-33
from: cosmoflare
to_project: handleshop
created: "2026-10-09T23:25:27.502773+04:00"
---

# Feedback digest — handleshop (2026-10-09)

| title | type | severity | one-line |
| --- | --- | --- | --- |
| handle.shop: scanner probes (secrets.json/.env/.aws/debugbar) reach the Worker — WAF block + cache / | feature | medium | # handle.shop: scanner probes (secrets.json/.env/.aws/debugbar) reach the Worker… |

## 01M4F65J2HTC2Q66747ZD7VF83 — handle.shop: scanner probes (secrets.json/.env/.aws/debugbar) reach the Worker — WAF block + cache /

# handle.shop: scanner probes (secrets.json/.env/.aws/debugbar) reach the Worker — WAF block + cache /

Evidence from cosmoflare live zone analytics, 2026-10-09 (read-only, last 24h, eyeball traffic).

## What is happening

Most of this zone's uncached traffic is not real users — it is automated vulnerability scanners probing for secrets and admin panels, and every probe reaches the Worker as a billed invocation:

| Path | Requests/24h |
|---|---|
| `/userfiles` | 111 |
| `/` | 97 |
| `/secrets.json` | 59 |
| `/_debugbar/open` | 57 |
| `/api/system/fileView` | 52 |
| `/.aws/config` | 51 |
| `/userfiles/x` | 51 |
| `/api/config` | 51 |
| `/config/.env` | 49 |
| `/api/console/api_server` | 48 |
| `/settings.json` | 48 |
| `/console` | 47 |

These requests return `cacheStatus=dynamic`, so the Worker runs for each one. Beyond cost, they are attack reconnaissance: make sure none of these paths can ever return real data.

## Fix

1. **Block scanner paths at the edge with a WAF custom rule** before the Worker runs, for example: `(http.request.uri.path contains ".php") or (http.request.uri.path contains "/.env") or (http.request.uri.path contains "/.git") or (http.request.uri.path contains "wp-") or (http.request.uri.path contains "/.aws") or (http.request.uri.path contains "secrets.json")` → action Block. Free plan: 5 custom rules per zone, all actions except Log, no regex — the `contains` expression above works on Free (https://developers.cloudflare.com/waf/custom-rules/, updated Aug 25, 2026). Do not block `/admin` if your app uses it.
2. **Cache the public pages** (`/` and other identical-for-everyone pages) with `Cache-Control: public, s-maxage=…`. Reference implementation of the CosmoLabs CF efficiency standard: Churches-app commit 155d1d3e (IMP-017).
3. Confirm the Worker returns a plain 404 (never a stack trace, config or env dump) for unknown paths.

## Verify

`cosmoflare` can manage WAF rules (`cosmoflare waf --help`). After the rule is live, the zone's dynamic request count for these paths should drop to zero in the Cloudflare dashboard (Security → Events shows the blocks).

**Priority:** medium — cost plus security hygiene.

