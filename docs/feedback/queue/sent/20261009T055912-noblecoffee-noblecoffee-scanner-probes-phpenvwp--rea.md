---
ulid: 01M4F65J1F0RQHPGZKYVBB5WW0
from: cosmoflare
type: feature
severity: medium
title: 'noble.coffee: scanner probes (.php/.env/wp-) reach the Worker as billed invocations — WAF block + cache /'
created: "2026-10-09T05:59:12.687076+04:00"
---

# noble.coffee: scanner probes (.php/.env/wp-) reach the Worker as billed invocations — WAF block + cache /

Evidence from cosmoflare live zone analytics, 2026-10-09 (read-only, last 24h, eyeball traffic).

## What is happening

Most of this zone's uncached traffic is not real users — it is automated vulnerability scanners probing for secrets and admin panels, and every probe reaches the Worker as a billed invocation:

| Path | Requests/24h |
|---|---|
| `/` | 102 |
| `/appsettings.Development.json` | 50 |
| `/wp-content/admin.php` | 37 |
| `/this_is_a_new_hello_world.php` | 29 |
| `/wp-admin/js/index.php` | 29 |
| `/about.php` | 23 |
| `/admin` | 23 |
| `/ioxi-o.php` | 22 |
| `/wk/index.php` | 22 |
| `/info.php` | 22 |

These requests return `cacheStatus=dynamic`, so the Worker runs for each one. Beyond cost, they are attack reconnaissance: make sure none of these paths can ever return real data.

## Fix

1. **Block scanner paths at the edge with a WAF custom rule** before the Worker runs, for example: `(http.request.uri.path contains ".php") or (http.request.uri.path contains "/.env") or (http.request.uri.path contains "/.git") or (http.request.uri.path contains "wp-") or (http.request.uri.path contains "/.aws") or (http.request.uri.path contains "secrets.json")` → action Block. Free plan: 5 custom rules per zone, all actions except Log, no regex — the `contains` expression above works on Free (https://developers.cloudflare.com/waf/custom-rules/, updated Aug 25, 2026). Do not block `/admin` if your app uses it.
2. **Cache the public pages** (`/` and other identical-for-everyone pages) with `Cache-Control: public, s-maxage=…`. Reference implementation of the CosmoLabs CF efficiency standard: Churches-app commit 155d1d3e (IMP-017).
3. Confirm the Worker returns a plain 404 (never a stack trace, config or env dump) for unknown paths.

## Verify

`cosmoflare` can manage WAF rules (`cosmoflare waf --help`). After the rule is live, the zone's dynamic request count for these paths should drop to zero in the Cloudflare dashboard (Security → Events shows the blocks).

**Priority:** medium — cost plus security hygiene.
