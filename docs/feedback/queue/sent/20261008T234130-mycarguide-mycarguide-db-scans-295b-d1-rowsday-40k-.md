---
ulid: 01M4EGHYQTWCFE12BYQFS5173B
from: cosmoflare
type: bug
severity: high
title: mycarguide-db scans ~2.95B D1 rows/day (~40k rows per query) — ~63B/month over the Paid allowance
created: "2026-10-08T23:41:30.234395+04:00"
---

# mycarguide-db scans ~2.95B D1 rows/day (~40k rows per query) — ~63B/month over the Paid allowance

**What happened:** A live Cloudflare GraphQL read from cosmoflare on 2026-10-08 (d1AnalyticsAdaptiveGroups, last 24h, grouped by databaseId) shows `mycarguide-db` (id prefix e561653d, 1.27 GB) read **2,945,546,702 rows from 73,151 read queries** — about 40,000 rows scanned per query. The next-largest database on the account read 52M.

**Why it matters:** D1 bills rows *scanned*, not rows returned (https://developers.cloudflare.com/d1/platform/pricing/, "Last updated Apr 21, 2026": Workers Paid includes "First 25 billion / month" then "$0.001 / million rows"). At this rate the database reads ~88B rows/month, ~63B over the account-wide allowance (≈ $63/month), and it consumes the allowance shared by every CosmoLabs project. ~40k rows per query is the signature of full-table scans on unindexed filter columns.

**Proposed fix:** Run `wrangler d1 insights mycarguide-db --sort-by reads --sort-type sum` to find the top-scanning query strings. Add indexes on the filtered columns (D1 pricing def. 5: indexes cut rows read on the indexed filter), or cache/materialize hot aggregate endpoints in KV (the Churches-app IMP-017 pattern, commit 155d1d3e). Re-check rows read per query afterwards; the target is hundreds, not tens of thousands.

**Priority:** high — ongoing cost and shared-allowance burn, found live. Cosmoflare wave 3a adds a `d1-rows-read` alert that would page on this.
