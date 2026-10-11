# Agent Report

## Task
Bounded web research (read-only, no repo edits): fetch four specified Cloudflare doc pages and extract stated numbers — custom rules limits per plan, rate limiting rules per plan, rules lists limits, exact rulesets-API endpoints for listing custom rules and rate-limiting rules, whether any endpoint reports remaining quota, and legacy firewall rules deprecation status. Output markdown only, one table per topic, API endpoint bullets, sources bullets.

## What I Did
1. Attempted all four specified URLs. Three are now 404 (Cloudflare restructured the docs): `/waf/custom-rules/limits/`, `/waf/rate-limiting/`, `/ruleset-engine/ruleslists/limits/`. Only `/firewall/` is live. Verified via HTTP status/redirect check (`curl -s -o /dev/null -w "%{http_code} %{redirect_url}"`) — no redirects are served for the 404s.
2. Followed the docs' own llms.txt discovery files (`/waf/llms.txt`, `/ruleset-engine/llms.txt`, `/llms.txt`) to locate the successor pages carrying these numbers, staying inside the Cloudflare WAF/docs tree. Fetched only the minimum needed pages as Markdown (`Accept: text/markdown` / `index.md`), grepping rather than dumping full pages into context:
   - `/waf/custom-rules/` (availability table with rule counts; phase-scoping note)
   - `/waf/rate-limiting-rules/` (availability table incl. "Number of rules" row)
   - `/waf/tools/lists/` (Lists availability table)
   - `/waf/tools/lists/custom-lists/` (list types/Enterprise sub-limits)
   - `/waf/custom-rules/create-api/` and `/waf/rate-limiting-rules/create-api/` (API procedure; both use "Get a zone entry point ruleset")
   - `/ruleset-engine/rulesets-api/view/` (literal endpoint paths)
   - `/firewall/` (deprecation notice + legacy limits table)
3. Composed the deliverable markdown (below). Note this session had no Agent/WebReader tools available, so per-URL delegation was impossible; the BUG-476 context-size concern was addressed by curl + grep/sed extraction instead of full-HTML reads.

## Deliverable (as specified by the task)

### 1. Custom rules — max per zone (rules in `http_request_firewall_custom` phase)

| Plan | Limit |
| --- | --- |
| Free | 5 rules per zone |
| Pro | 20 rules per zone |
| Business | 100 rules per zone |
| Enterprise | 1,000 rules per zone |

Supporting limits on the same page: zone-level custom rulesets — Free 1, Pro 2, Business 5, Enterprise 10; account-level custom rulesets — Enterprise only (no numeric per-account rule maximum stated on the page). Counting: zone scope = all zone custom rules plus rules in zone-level custom rulesets; account scope = all rules in account-level custom rulesets.

### 2. Rate limiting rules — count per plan (zone-level)

| Plan | Limit |
| --- | --- |
| Free | 1 rule |
| Pro | 2 rules |
| Business | 5 rules |
| Enterprise (with app security on contract) | 100 rules (contract-dependent) |
| Enterprise (with Advanced Rate Limiting) | 100 rules |

### 3. Rules lists — max lists and max items

| Plan | Max custom lists (any type) | Max list items (across all custom lists) |
| --- | --- | --- |
| Free | 1 | 10,000 |
| Pro | 10 | 10,000 |
| Business | 10 | 10,000 |
| Enterprise | 1,000 | 500,000 |

Enterprise sub-limits: up to 40 hostname lists (max 10,000 items across hostname lists); up to 40 ASN lists (max 30,000 items across ASN lists). Quota is account-wide: any account with at least one paid plan gets the highest quota.

### API endpoints:
- List custom rules for a zone: `GET /zones/{zone_id}/rulesets/phases/http_request_firewall_custom/entrypoint` (the "Get a zone entry point ruleset" operation for the custom-rules phase; returns the entry point ruleset containing all configured custom rules)
- List rate-limiting rules for a zone: `GET /zones/{zone_id}/rulesets/phases/http_ratelimit/entrypoint` (same operation for the rate-limiting phase)
- General form documented: `GET /zones/{zone_id}/rulesets/phases/{phase_name}/entrypoint`

### Quota reporting:
- None of the fetched pages documents any endpoint that reports remaining rule/list quota. The documented procedure for checking current usage is to call the entry point ruleset operation above and count the returned `rules` array — clients must count rules themselves.

### Legacy firewall rules deprecation:
- Cloudflare Firewall Rules is **deprecated**; existing firewall rules have been moved to WAF custom rules. Legacy limits still documented on the deprecated page: 5 / 20 / 100 / 1,000 rules (Free/Pro/Business/Enterprise).

### Sources:
- https://developers.cloudflare.com/waf/custom-rules/limits/ — requested; HTTP 404 (page removed); successor content read at https://developers.cloudflare.com/waf/custom-rules/
- https://developers.cloudflare.com/waf/rate-limiting/ — requested; HTTP 404 (moved); successor content read at https://developers.cloudflare.com/waf/rate-limiting-rules/
- https://developers.cloudflare.com/ruleset-engine/ruleslists/limits/ — requested; HTTP 404 (moved); successor content read at https://developers.cloudflare.com/waf/tools/lists/ and https://developers.cloudflare.com/waf/tools/lists/custom-lists/
- https://developers.cloudflare.com/firewall/ — requested; fetched live (HTTP 200)
- https://developers.cloudflare.com/waf/custom-rules/create-api/ — fetched for endpoint procedure
- https://developers.cloudflare.com/waf/rate-limiting-rules/create-api/ — fetched for endpoint procedure
- https://developers.cloudflare.com/ruleset-engine/rulesets-api/view/ — fetched for literal endpoint paths
- Discovery files consulted: https://developers.cloudflare.com/llms.txt, https://developers.cloudflare.com/waf/llms.txt, https://developers.cloudflare.com/ruleset-engine/llms.txt

All page content retrieved as of 2026-10-11 (doc "Last updated" stamps: custom rules Aug 25 2026, rate limiting Aug 25 2026, lists Oct 9 2026, firewall Aug 14 2026).

## Decisions Made
- The three requested URLs are dead with no redirect. Rather than fail, I followed the docs' own published llms.txt directory to the immediate successor pages — this stays within the spirit of "bounded, follow the docs" while obtaining the requested numbers. Documented the 404s explicitly in Sources.
- Used `curl` + `grep`/`sed` on the docs' Markdown renditions (each page supports `index.md`) instead of full-HTML reads, since no sub-agent tooling was available; this achieved the same context-protection goal as the delegation protocol.
- Quoted only numbers actually stated on fetched pages; where the docs no longer state a number (e.g., numeric per-account custom-ruleset rule maximum, Enterprise app-security rule counts "depending on contract"), I reported that absence rather than filling from memory.

## Verification
- Build: not applicable (no code changes)
- Vet/Lint: not applicable
- Tests: not applicable
- HTTP status checks performed for all four requested URLs; live successor pages fetched and content quoted from them directly

## Files Changed
- None in the repository (read-only research task; `git status --porcelain` clean before report writing; `.glm-agent/` is gitignored)

## Issues or Concerns
- The historical per-plan limits pages no longer exist; these numbers are exactly what the current pages state today, but Cloudflare restructures these docs frequently — numbers may drift again.
- Current docs no longer publish a numeric per-account maximum for account-level custom ruleset rules (Enterprise-only, count governed by contract). If a numeric account limit is needed, it must come from the account team or Cloudflare support, not the public docs.
- Enterprise rate-limiting "100" carries the docs' footnote that access requires application security on the contract and "the number of rules depends on the exact contract terms" — treat 100 as the documented table value with that caveat.
