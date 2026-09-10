# Qwen follow-up Q&A — Workers free-tier daily request cap semantics

Provenance: same Qwen conversation as `qwen-results.md`, questions asked by
the author after the main envelope return. Captured 2026-09-10T20:05:00+04:00.

## Q: How does the cap enforcement handle time zone boundaries, and what
happens if a user exceeds the limit across a UTC day boundary?

### 1. Time Zone Boundaries
The 100,000 daily request limit is strictly tied to **UTC**, not the user's
local time zone or the origin server's time zone.
* **Calendar Day, Not Rolling:** The counter resets exactly at **midnight
  UTC (00:00:00 UTC)** every day. It is a strict calendar-day boundary,
  *not* a rolling 24-hour window.
* **Edge Case for Global Apps:** A traffic spike at 11:55 PM UTC hits the
  cap and triggers enforcement; 5 minutes later at midnight UTC the quota
  instantly resets to 0 and normal processing resumes.

### 2. Exceeding the Limit (Pre-Midnight UTC)
Once the 100,000th request is processed within the UTC calendar day,
subsequent requests are handled by the **Route mode** configured per Worker
in the dashboard:
* **Fail Open (Default/Graceful Degradation):** The Worker is bypassed.
  Cloudflare proxies the request directly to the origin as if the Worker did
  not exist. Fits non-critical edge logic (A/B testing, analytics).
* **Fail Closed (Security-Critical):** Cloudflare returns an **Error 1027**
  ("Worker exceeded daily request limit") block page. Mandatory for
  security-critical Workers (auth, WAF-like logic, access controls) where
  bypassing would expose the origin.

### 3. Crossing the UTC Boundary
At `00:00:00 UTC` the internal counter drops to zero.
* The Worker **instantly resumes normal execution** on the next request.
* Fail-Closed 1027 errors stop immediately; Fail-Open traffic routes back
  through the Worker immediately.

*Enforceability note:* account-level aggregate tied to a UTC calendar day →
**opaque** to enforce locally. The CLI cannot know how many requests other
Workers consumed today. Rely on the `1027` error response (or origin
fallback) and handle it gracefully rather than preemptively blocking on a
local guess.

## Q: How does the system track and reset the daily request cap for users in
different time zones?

It doesn't. The system completely ignores the time zones of both the
account owner and the end-users. The cap is governed by a monolithic,
platform-wide **UTC boundary**. There is no account-level setting to shift
the quota day to a local time zone.

### 1. Tracking Mechanism: Account-Level Atomic Counter
Not tracked per-Worker — an **account-level aggregate**.
* Every HTTP request hitting *any* Worker on the Free plan increments a
  single shared counter tied to the Account ID.
* At `00:00:00.000` UTC the counter drops to `0` and the threshold re-arms.

### 2. The "Dashboard Illusion" (Ultimate Edge Case)
The enforcement engine operates strictly in UTC, but the **Cloudflare
Dashboard UI renders analytics in the browser's local time zone**.
* Scenario: developer in New York (EST) sees a "new day" start at midnight
  EST on the graph; the engine actually reset 5 hours earlier (7:00 PM EST
  = midnight UTC). A spike at 11:00 PM EST is 23 hours into the UTC day →
  Error 1027.
* **Rule for cosmoflare:** never rely on dashboard cues or local clocks for
  quota estimation. Query the GraphQL Analytics API and filter
  `datetime`/`date` with UTC bounds.

### 3. The UTC Midnight Spike
* `23:59:59 UTC`: request #100,001 is rejected (429 or 1027 page if Fail
  Closed, or Worker bypassed if Fail Open).
* `00:00:00 UTC`: request #1 processes normally — counter reset.
* **No grace period, no rollover.** Unused quota does not roll over; no
  token-bucket smoothing; strict integer threshold.

### 4. Enforceability Implications for the CLI
Classified `enforceability: opaque` for a local CLI tool.
* The CLI cannot know requests consumed by other scripts, cron triggers, or
  user traffic on the same account today.
* Handle by:
  1. Do not preemptively block deployments/requests on a local counter.
  2. Catch `1027` / `429` responses and emit a clear warning: "Account
     free-tier daily quota (100k) exhausted. Resets at 00:00 UTC."
  3. Offer a quota-check command: GraphQL query against the `account`
     viewer summing `requests` for today (UTC) compared against 100,000.

## Q: What happens to the daily request cap if a user switches time zones
while using the service?

Nothing happens to the system. The daily request cap is completely blind to
the physical location, IP address, or local time zone of both the developer
and the end-users.

### 1. The Cap is Account-Scoped, Not Session-Scoped
The 100,000 daily request limit is an **account-level aggregate quota**,
not a per-user or per-IP rate limit.
* When a request hits the edge, the enforcement engine looks at the
  `Account ID` associated with the Worker.
* It increments a single, centralized atomic counter for that Account ID.
* It compares that counter against the global **UTC clock**.

A developer flying New York → Tokyo, or an end-user roaming across Europe,
has zero impact on the quota mechanics. They add `+1` to the same global
account counter.

### 2. The "Jetlag" Edge Case (Human vs. Machine Time)
The *system* doesn't change; the *developer's perception* of the boundary
shifts — a classic source of confusion and support tickets.
* **Scenario:** Developer in San Francisco (UTC-8) expects the quota reset
  at 4:00 PM local (= midnight UTC). Flies to London (UTC+0). Wakes at
  8:00 AM local, assumes a "new day", pushes a load test.
* **Reality:** It is 8:00 AM UTC — the system is 8 hours into the quota
  day. Only 16 hours remain before reset; the test blows the remaining
  quota → Error 1027.

### 3. End-User Roaming (Client IP vs. Account Quota)
Distinguish the **daily request cap** from **WAF/Rate Limiting rules**:
* WAF rate limiting (e.g. 100 req/min) is tracked per client IP or session
  — a roaming user gets a fresh bucket when their IP changes.
* The daily cap is strictly per account — traffic from 50 time zones drinks
  from the same 100,000-request counter. Geographic distribution is
  irrelevant.

### 4. Enforceability Implications for cosmoflare
The limit is decoupled from the local environment → `opaque` for local
pre-flight checks. A quota-status command must implement strict UTC math:

```go
// WRONG: local machine time
now := time.Now()
midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.Local)

// RIGHT: UTC for Cloudflare boundaries
nowUTC := time.Now().UTC()
nextResetUTC := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day()+1, 0, 0, 0, 0, time.UTC)
hoursUntilReset := nextResetUTC.Sub(nowUTC).Hours()
```

Always anchor Cloudflare daily quotas to `00:00:00 UTC`.

---

**Session note (not from Qwen):** `pkg/cosmoflare/limits.go` already sums
today's UTC window via the analytics API for `workers.daily_requests` — the
follow-up validates that approach. New design idea for BR-03/BR-04 surfaced
by this exchange: show a "resets in Xh at 00:00 UTC" countdown on
daily-window rows.
