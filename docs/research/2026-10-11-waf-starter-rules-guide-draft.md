# Cloudflare WAF Starter Rules — Guide Skeleton for a Small Team

Scope: a small SaaS/website team running sites and APIs on Cloudflare (zone-level features; plan differences noted inline). No marketing prose; facts sourced from Cloudflare developer docs (URLs cited per section, accessed 2026-10-11).

---

## 1. Highest-value starter custom rules

Background facts that constrain every rule below:

- Custom rules = a [rules-language expression](https://developers.cloudflare.com/ruleset-engine/rules-language/expressions/) + one action. They are **evaluated in order**; a terminal action (Block, Managed Challenge) stops evaluation of later rules (https://developers.cloudflare.com/waf/custom-rules/).
- Plan limits: Free = 5 custom rules, Pro = 20, Business = 100; the **Log action is Enterprise-only** for custom rules, and regex in expressions is Business+ (availability table, https://developers.cloudflare.com/waf/custom-rules/). On Free–Business, "log-first" rollout means observing matches in Security Events before switching actions (see §4), not a Log action.
- Execution order: custom rules run in `http_request_firewall_custom`, **before** Rate Limiting (`http_ratelimit`) and Managed Rules (`http_request_firewall_managed`) (https://developers.cloudflare.com/waf/managed-rules/). So keep Skip/allow rules narrow — a broad early allow exempts traffic from everything downstream.

Rules in suggested deployment order (exact expressions are sketches; adapt hostnames/paths):

- **Rule 1 — Allow verified good bots (Skip)**
  - Condition: `cf.client.bot` (Cloudflare-verified bots: Googlebot, Bingbot, etc.) — optionally narrowed to specific verified bots per https://developers.cloudflare.com/waf/custom-rules/use-cases/allow-traffic-from-verified-bots/
  - Action: Skip (skip later custom rules / managed rulesets as needed)
  - Why Skip, not Block: verified search/monitoring bots must never be caught by your later geo or challenge rules; this is an exemption, not a punishment. Place it first so it wins ordering.
  - Rollout: deploy day 1; verify in Security Events that only real verified bots match.

- **Rule 2 — Admin panel protection**
  - Condition: `starts_with(http.request.uri.path, "/admin")` (or your actual admin path) `and not ip.src in $office_ips` (IP list per https://developers.cloudflare.com/waf/tools/lists/custom-lists/)
  - Action: **Managed Challenge**
  - Why Managed Challenge, not Block: admins travel, switch networks, and use VPNs; Block would lock you out of your own panel. A challenge silently passes humans and stops scripted attacks. Pattern reference: https://developers.cloudflare.com/waf/custom-rules/use-cases/site-admin-only-known-ips/
  - Rollout: log/observe 3–7 days (Enterprise: Log action; otherwise watch the expression's matches via Security Analytics) before enabling.

- **Rule 3 — Known-bad path noise (Block)**
  - Condition: `http.request.uri.path contains "/wp-login" or http.request.uri.path contains "/.env" or http.request.uri.path contains "autodiscover"` (prune to paths your stack doesn't serve; Autodiscover example: https://developers.cloudflare.com/waf/custom-rules/use-cases/block-ms-exchange-autodiscover/)
  - Action: **Block**
  - Why Block: no human user of your site ever requests these; there is no false-positive cost to blocking, and these probes are pure scanner noise.
  - Rollout: safe to deploy day 1 after confirming none of the paths are legitimately used (check your origin access logs once).

- **Rule 4 — API requires credentials header**
  - Condition: `starts_with(http.request.uri.path, "/api/") and not any(http.request.headers["x-api-key"][*] != "")` — i.e., API paths without your API-key header (header-requirement pattern: https://developers.cloudflare.com/waf/custom-rules/use-cases/require-specific-headers/)
  - Action: **Block**
  - Why Block: API consumers are programs; a challenge is unsolvable for them, so a soft action buys nothing. If some public API endpoints exist, scope the expression to the authenticated prefix instead.
  - Rollout: observe 1 week — confirms your own clients all send the header (including cron jobs and partners) before enabling.

- **Rule 5 — Challenge bad bots**
  - Condition: Enterprise: `cf.bot_management.score lt 10` (definitely automated; use `< 30` for "likely automated" if you want wider net) — https://developers.cloudflare.com/bots/. Non-Enterprise: instead enable **Bot Fight Mode / Super Bot Fight Mode** toggles (no per-request bot score available) — https://developers.cloudflare.com/bots/. A plan-independent fallback: `cf.threat_score`-based rules.
  - Action: **Managed Challenge** (not Block: bot scores are probabilistic; a challenge is recoverable if a rare human scores low; the bot won't solve it)
  - Rollout: Enterprise: run with the `< 10` threshold first for 1–2 weeks before widening to `< 30`.

- **Rule 6 — Geo restriction (only if your customers are regional)**
  - Condition: `not ip.src.country in {"US" "CA" "GB" "DE" ...}` (your actual customer countries) — pattern: https://developers.cloudflare.com/waf/custom-rules/use-cases/block-traffic-from-specific-countries/ or the allowlist-only inverse: https://developers.cloudflare.com/waf/custom-rules/use-cases/allow-traffic-from-specific-countries/
  - Action: start **Managed Challenge**, move to Block only if the false-positive rate is zero.
  - Why Challenge first: geo rules have the highest false-positive risk of any starter rule (travelers, VPNs, cloud egress of your own integrations). Do NOT skip the observation period on this one.
  - Rollout: 2 weeks observed before enforcement; longest log period in this list.

- **Rule 7 (optional) — Staging/internal hostnames lockdown**
  - Condition: `http.host eq "staging.example.com" and not ip.src in $team_ips` (allowlist pattern: https://developers.cloudflare.com/waf/custom-rules/use-cases/allow-traffic-from-ips-in-allowlist/)
  - Action: **Block**
  - Why Block: staging should be unreachable by the public; a challenge leaks that the host exists and adds friction for no benefit.
  - Rollout: day 1 — you know exactly who should have access.

Cited for this section: https://developers.cloudflare.com/waf/custom-rules/ , https://developers.cloudflare.com/waf/custom-rules/use-cases/

---

## 2. Managed rules / OWASP Core Ruleset: ON vs custom rules

Source: https://developers.cloudflare.com/waf/managed-rules/ (execution order + availability), https://developers.cloudflare.com/waf/tools/

- **Division of labor.** Managed rulesets catch *generic exploitation* (known CVEs, injection payloads, zero-day techniques) and are updated continuously by Cloudflare — you cannot replicate this with custom rules. Custom rules encode *your* business logic (admin paths, geo policy, API contract, bot handling) that no signature knows. Run both; they are complementary, not alternatives.
- **Which managed rulesets to enable by plan** (availability table, https://developers.cloudflare.com/waf/managed-rules/):
  - Free: **Cloudflare Free Managed Ruleset** — "safe to deploy on most applications," covers high-impact widely exploited vulns. Enable it.
  - Pro/Business/Enterprise: **Cloudflare Managed Ruleset** (superset of the Free one — don't run both) + optionally **Cloudflare OWASP Core Ruleset** (Pro+).
- **OWASP Core Ruleset: when ON.** Turn it on for classic web apps (WordPress, PHP frameworks, traditional server-rendered apps) where injection/RFI payloads in query/body/headers are the realistic threat. It uses a cumulative **threat-score model** — each matched rule adds score, action fires above your threshold (https://developers.cloudflare.com/waf/managed-rules/reference/owasp-core-ruleset/). Start with a **high (permissive) threshold or log-mode**, then tighten; note the documented false-positive mechanism: larger request bodies accumulate more score, so file-upload endpoints can trip it (https://developers.cloudflare.com/waf/managed-rules/).
- **When custom rules carry the load instead:** pure JSON APIs behind an API key (OWASP signature noise on JSON bodies is common; your Rule 4 header requirement + rate limits in §3 do the real work), and any endpoint with non-HTTP-normal semantics.
- **Ordering caveat that bites small teams:** custom rules run BEFORE managed rules. A terminal custom action (your broad Block/Skip) prevents managed rules from evaluating the request at all (execution-order section, https://developers.cloudflare.com/waf/managed-rules/). Keep custom allow rules (Rule 1) narrow.
- **Tuning tools:** use WAF **exceptions** (skip rules) to exempt specific traffic from managed rulesets, and **overrides** to change/disable individual rules or tags; exceptions take priority over overrides. Rule/tags overrides also apply to *future* rules in the ruleset — prefer rule-level overrides if you only want to change existing rules (https://developers.cloudflare.com/waf/managed-rules/).
- **Not substitutes:** the legacy WAF tools page (https://developers.cloudflare.com/waf/tools/) — IP Access rules, User Agent Blocking, Zone Lockdown, Security Level — are coarse controls; fine-grained policy belongs in custom rules + managed rules.

Cited for this section: https://developers.cloudflare.com/waf/managed-rules/ , https://developers.cloudflare.com/waf/tools/

---

## 3. Rate limiting: three starter patterns

Source: https://developers.cloudflare.com/waf/rate-limiting-rules/best-practices/ (all thresholds below are Cloudflare's own documented examples), https://developers.cloudflare.com/waf/rate-limiting-rules/use-cases/

### 3.1 Login path (credential stuffing / brute force)

Cloudflare's documented escalation ladder — three rules, increasing penalty (https://developers.cloudflare.com/waf/rate-limiting-rules/best-practices/):

- **Login tier 1**: `http.request.uri.path eq "/login" and http.request.method eq "POST"` — count by IP, **counting expression** `... and http.response.code in {401 403}` — **4 requests / 1 minute → Managed Challenge**
- **Login tier 2**: same match/count — **10 requests / 10 minutes → Managed Challenge**
- **Login tier 3**: rule matches the whole host (`http.host eq "example.com"`) but counts only failed `/login` POSTs — **20 failures / 1 hour → Block for 1 day**

- Why this shape: humans mistype passwords ~3–4 times; tier 1's Managed Challenge passes them silently while automation stalls. The counting expression (count only `401/403` responses) means successful logins never increment — this is the core anti-lockout mechanism. Requires Business+ for response-code counting expressions (per best-practices page); on Free/Pro use plain request counting at a gentler threshold (e.g., 10 POSTs / 1 min per IP → Managed Challenge).
- Avoiding lockouts: Managed Challenge (not Block) on the first two tiers; shared-NAT offices share one IP — the challenge, not a block, is what a whole office can still pass.

### 3.2 API abuse

- **Condition**: `starts_with(http.request.uri.path, "/api/") and http.request.method eq "POST"` — **count by `x-api-key` header** (falls back to IP for unauthenticated endpoints). Threshold: derive from your own traffic (see §4); Cloudflare's documented example class is ~100 requests / 10 minutes per client for app-like traffic (https://developers.cloudflare.com/waf/rate-limiting-rules/best-practices/). **Action: Block.**
- Why Block, not challenge: API clients are programs and can never solve a Managed Challenge; Cloudflare returns a rate-limit response the client can implement backoff against. Counting by API key rather than IP is the documented approach so a team behind one NAT isn't throttled collectively (https://developers.cloudflare.com/waf/rate-limiting-rules/use-cases/).
- Avoiding lockouts: exclude your highest-volume first-party integrations via a Skip-style carve-out or an `and not ip.src in $partner_ips` clause (documented example: https://developers.cloudflare.com/waf/rate-limiting-rules/best-practices/).

### 3.3 Scrapers

Cloudflare's documented two-rule softening pattern plus a failure-code tripwire (https://developers.cloudflare.com/waf/rate-limiting-rules/best-practices/):

- **Scraper tier 1**: content/list endpoints (`http.request.uri.path eq "/catalog"`-style) — count by IP — **10 requests / 2 minutes → Managed Challenge**
- **Scraper tier 2**: same expression — **20 requests / 5 minutes → Block**
- **404/403 tripwire**: whole host, counting expression `http.response.code in {403 404}` — **5 responses / 3 minutes → Managed Challenge** (Business+ for the counting expression). Scrapers probing your URL space generate error codes far above human rates; humans occasionally hit one 404 and are unaffected.

- Avoiding lockouts: add `and not cf.client.bot` so verified crawlers are exempt (documented pattern: https://developers.cloudflare.com/waf/rate-limiting-rules/use-cases/); use the Challenge tier first; on Advanced Rate Limiting, count by session cookie or JA4 fingerprint instead of IP so distributed botnets are grouped and NAT'd humans aren't (https://developers.cloudflare.com/waf/rate-limiting-rules/best-practices/).
- Path caveat from the docs: verify your origin and Cloudflare normalize paths identically before trusting path-equality expressions (URL normalization note, https://developers.cloudflare.com/waf/rate-limiting-rules/best-practices/).

Cited for this section: https://developers.cloudflare.com/waf/rate-limiting-rules/best-practices/ , https://developers.cloudflare.com/waf/rate-limiting-rules/use-cases/

---

## 4. Safe rollout sequence

Sources: https://developers.cloudflare.com/waf/rate-limiting-rules/find-rate-limit/ , https://developers.cloudflare.com/waf/custom-rules/ (Log action = Enterprise only), https://developers.cloudflare.com/waf/managed-rules/

1. **Baseline first (before deploying anything).** In Security Analytics, filter to the target path/host and look at a normal (non-attack) week: top IPs, response codes, bot scores. For rate limits, Enterprise accounts can use the **Request rate analysis** tab — distribution of per-client rates for the top 50 clients, with a slider to preview how many clients a given threshold would catch; validate the threshold against a period when you know an attack happened ("should block outlier traffic during the attack and allow traffic during regular periods") (https://developers.cloudflare.com/waf/rate-limiting-rules/find-rate-limit/).
2. **Deploy in log/observe mode.** Enterprise: set custom rules and rate limiting rules to the **Log** action. Free–Business: the custom-rule Log action is unavailable (availability table, https://developers.cloudflare.com/waf/custom-rules/) — instead keep rules disabled, or deploy with the mildest action and watch Security Events for what *would* have matched; for rate limiting specifically, docs recommend deploying with Log and switching when confident (https://developers.cloudflare.com/waf/rate-limiting-rules/find-rate-limit/).
3. **Log-first durations (suggested minimums):**
   - Zero-risk rules (§1 Rules 3, 7 — scanner noise, staging lockdown): 1–3 days, mostly to catch typos in the expression.
   - Moderate risk (§1 Rules 2, 4, 5; all §3 rate limits): 3–7 days — must span at least one weekend/traffic cycle.
   - Highest false-positive risk (§1 Rule 6 geo; OWASP Core Ruleset): 2 weeks. For OWASP, additionally start at a high/permissive score threshold and tighten only after clean weeks (https://developers.cloudflare.com/waf/managed-rules/).
4. **Reading Security Events before flipping to Block/Challenge.** For each rule: filter Security Events by the rule/service, check (a) **volume and trend** — a flatline means your expression never matches (fix before it matters), (b) **sampled logs** — inspect actual matched requests for anything a real user would send (https://developers.cloudflare.com/waf/rate-limiting-rules/find-rate-limit/ — sampled logs step), (c) **top matching IPs/ASNs** — if your own office IP or monitoring vendor dominates, add a carve-out before enforcing, (d) for rate limits, whether the clients that would be throttled are the obvious long-tail outliers, not the bulk of users ("Can I visually identify abusers with above-average rate vs. the long tail of average users?" — https://developers.cloudflare.com/waf/rate-limiting-rules/find-rate-limit/).
5. **Switch actions one rule at a time**, mildest-first (Log → Managed Challenge → Block), with at least a day between changes on the risky rules. Terminal actions stop evaluation of later rules and of Managed Rules (§2 ordering), so every action change can shift what your downstream rules see — re-check Security Events after each flip. Keep one rollback path: set the action back; expressions never need to change.
6. **Ongoing:** review Security Events weekly for the first month; revisit rate thresholds after any product launch that changes traffic shape.

Cited for this section: https://developers.cloudflare.com/waf/rate-limiting-rules/find-rate-limit/ , https://developers.cloudflare.com/waf/custom-rules/ , https://developers.cloudflare.com/waf/managed-rules/

---

### Doc corrections found during research
- The task URL `https://developers.cloudflare.com/waf/rate-limiting/best-practices/` is a 404; the page moved to **https://developers.cloudflare.com/waf/rate-limiting-rules/best-practices/** (per https://developers.cloudflare.com/waf/llms.txt).
