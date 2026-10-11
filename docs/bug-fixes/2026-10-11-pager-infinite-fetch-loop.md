---
title: Ops pager infinite fetch loop stuck iOS on skeleton loader
severity: critical
status: fixed
date_reported: "2026-10-11T19:20:00+04:00"
date_fixed: "2026-10-11T19:45:00+04:00"
session: 2037
id: BUG-pZKJ5VB
introduced_in: pager React rebuild P-04..P-06 (session 2036, 2026-10-10)
fixed_in: worker version 30fa2042
commits:
    - a43e03b
    - 1fd77f4
affected_files:
    - pager/src/app/views/AIView.tsx
    - pager/src/app/views/D1View.tsx
    - pager/src/app/views/DurableObjectsView.tsx
    - pager/src/app/views/DomainsView.tsx
    - pager/src/app/views/ProjectsView.tsx
    - pager/src/app/views/WorkerProfileView.tsx
    - pager/src/app/views/RulesView.tsx
    - pager/src/app/views/ZonesView.tsx
    - pager/src/app/views/WorkersView.tsx
related_issues:
    - FEAT-pT0AFDM
related_bugs:
    - BUG-057
---

# Ops pager infinite fetch loop stuck iOS on skeleton loader

## Executive Summary

Every React view in the Ops pager rebuilt its fetch helper inline, creating a new function object on every render while that object sat in the load effect's dependency array. The result was a self-feeding fetch loop (~50 requests/second per open page) running since the React rebuild. On iOS, WebKit's per-host connection cap let the loop starve the page's own data fetch, leaving the operator permanently stuck on the skeleton loader. Fixed by hoisting one stable module-level fetcher per view, with regression tests that exercise the previously untested production path.

## Impact Assessment

| Dimension | Assessment |
|---|---|
| Severity | Critical — production UI unusable on iOS |
| Scope | All 9 data views, every platform; fatal only on iOS |
| Duration | Introduced 2026-10-10 (React rebuild), fixed 2026-10-11 (~1 day live) |
| Data impact | None — read-only dashboard; no corruption, no writes |
| Cost impact | Worker invocations + phone CPU/battery. Success-path loop iterations were absorbed by the client's 60s cache (no network); failure paths (any HTTP error or expired session) hit the network every iteration — worst case ~180k requests/hour per open page against the Workers free allowance (100k/day) |
| Upstream Cloudflare API | Shielded throughout by the worker's `cached()` layer (5min/1h/24h TTLs + single-flight); upstream quota (1,200 req/min/token) was never the exposed surface |

## Symptom

Operator report (2026-10-11, iPhone PWA): "cant load the page its stuck in skeleton". The page never finished loading; the skeleton loader persisted indefinitely. Desktop browsers rendered normally. `wrangler tail` showed an unexplained steady stream of 200 responses.

## Root Cause Analysis

Each view built its fetcher inside the component body:

```tsx
const fetchJson: ViewFetcher = fetcher ?? ((endpoint, opts) => api.fetchJson(endpoint, opts));
```

and listed it in the load effect's dependency array:

```tsx
useEffect(() => { ... fetchJson(...) ... }, [refreshSeq, retryTick, repaint, fetchJson]);
```

A closure created in the component body is a **new object every render**. React's effect system compares dependency identity, so after every render the effect saw "fetchJson changed" and re-fired. Every fetch completion called `setData` with a **new result object**, forcing a re-render, which rebuilt the closure, which re-fired the effect:

```
render → new closure → effect fires → fetch resolves →
setData(new object) → render → new closure → effect fires → …
```

~20ms per turn (~50/second), per open page, on every route, from the moment the React rebuild shipped.

Why the two faces:

- **Success path**: the `ApiClient` 60-second memory cache answers iterations instantly with no network. Desktop Chrome absorbs 50 cheap cache hits + re-renders per second invisibly — the page renders from the first response while the loop burns CPU in the background.
- **Failure path**: `fromNetwork` throws before caching on any HTTP error or expired Access session, so **nothing caches and every iteration is a real network request**. iOS WebKit allows ~6 concurrent connections per host; the request storm saturates the pool and the page's primary load fetch queues forever — permanent skeleton.

Why it wasn't caught: every view test injects a stable `fetcher` prop (the seam exists for testing), so the dependency never churned under test. The production path — no prop — is the one that looped. Same lesson as the earlier "Illegal invocation" bug (fixed 2026-10-09): **the untested path is the shipped one**; fixtures and injected seams both hide the production code path.

## Reproduction Steps

1. Serve the built pager (worker version cc52e279 or any post-React-rebuild build) with any `/api/*` endpoint failing.
2. Open the app with the network tab visible: the same endpoint is re-requested ~50×/second indefinitely (harness measured 1,500+ requests in 30 seconds on the AI route).
3. On iOS (Safari or installed PWA): the page never leaves the skeleton.

## The Fix

Hoist a module-level default so the identity is stable for the app's lifetime:

```tsx
const defaultFetchJson: ViewFetcher = (endpoint, opts) => api.fetchJson(endpoint, opts);

// inside the component:
const fetchJson: ViewFetcher = fetcher ?? defaultFetchJson;
```

Applied identically in all 9 views (GLM flash agent 0401 through the S334 gate). No effect bodies, deps arrays, or other logic changed — with a stable default, the existing `fetchJson` dependency becomes correct.

- Fix commit: `a43e03b` (merge `1fd77f4`)
- Files: the 9 view files listed in frontmatter
- Deployed: worker version `30fa2042`

## Verification

- New regression tests — `does not loop fetches when no fetcher prop is injected (production path)` in `AIView.test.tsx` and `DomainsView.test.tsx` — render each view with **no** fetcher prop, spy on `api.fetchJson`, and assert the call count stays flat over time. Before the fix they measured **1,720 and 1,203 calls**; after, flat. These tests exercise the production path directly.
- Full suite: 236/236 green on master.
- Post-deploy harness verification (scratchpad server + Playwright against the shipped bundle): request counts per route in a 4-second window — overview 8 (5 nav-count endpoints + summary + revalidations), ai 1, domains 0 (cache). Previously 1,500+.
- Operator confirmed path: close-and-reopen loads the app normally on iOS.

## Prevention

1. **Regression tests on the production path** (landed): the two no-prop loop tests above now fail on any reintroduction of unstable fetcher identity.
2. **Client-side resilience — filed as FEAT-pT0AFDM**: in-flight single-flight dedup in `ApiClient` (concurrent identical fetches share one promise, mirroring the worker's `cached()`), exponential backoff for failed fetches with short-lived error caching so a failing endpoint can never be hammered, and a request-budget Playwright test asserting total `/api/*` calls over N seconds stay under a bound on every route.
3. **Review rule**: any function referenced in a `useEffect` dependency array must have stable identity across renders — module-level, `useMemo` with primitive deps, or `useCallback`. Flag inline `??`-fallback closures built in component bodies on sight.

## Lessons Learned

- An inline `fetcher ?? (() => …)` fallback is invisible test-seam sugar that becomes a production-only defect: the injected prop is stable, the fallback is not. Every seam added "for tests" must be tested from both sides.
- React effect loops are **load-dependent**: a bug can be invisible on the success path (cache absorbs it) and fatal on the failure path (nothing caches). Testing only happy paths hides exactly the case that pages an operator.
- Platform connection limits turn client-side request volume into an availability bug: iOS WebKit's ~6-connections-per-host cap means a client fetch storm is a self-DoS, even against a healthy backend.
- The scratchpad-harness + Playwright route-sweep technique (real handlers, real bundle, request counting per route) caught in 30 seconds what 234 unit tests could not; it should run as a standing pre-deploy check (folded into FEAT-pSGBF7D's request-budget test).

## Timeline

| Time (2026-10-11, +04:00) | Event |
|---|---|
| ~19:0x | Operator reports stuck skeleton on iPhone after morning UI deploy |
| 19:1x | Collectors verified healthy (0.6–4.8s); worker code unchanged; KV/cache ruled out |
| 19:2x | Playwright route sweep on the shipped bundle: 1,500+ requests in 30s on the AI route |
| 19:2x | Root cause isolated by code read: unstable fetcher identity in effect deps |
| 19:3x | BUG-pZKJ5VB filed (critical); fix agent 0401 dispatched with failing-test-first spec |
| 19:40 | Fix merged through S334 gate (236/236); deployed as 30fa2042 |
| 19:4x | Post-deploy harness verification: storm eliminated (8/1/0 requests per route) |
