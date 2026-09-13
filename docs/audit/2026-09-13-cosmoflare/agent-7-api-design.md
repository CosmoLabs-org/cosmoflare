# Agent 7: API Design — cosmoflare

Audit date: 2026-09-13. Scope: public Go library (`pkg/cosmoflare/`), desktop daemon REST+SSE API (`internal/server/` + `cmd/serve.go`), MCP tool server (`pkg/cosmoflare/mcp.go`), dev proxy (`pkg/cosmoflare/dev.go`), desktop TS client (`desktop/src/api/`), docs (`docs/USAGE.md`).

**API surface audited** (4 distinct surfaces):
1. **Public Go library** — `R2Client` interface + per-service constructors (22+ services), typed errors, functional options, generic `ListResult[T]` pagination.
2. **Desktop daemon** (`cosmoflare serve`) — token-gated localhost REST (6 endpoints) + multiplexed SSE (3 channels), stdout JSON handshake.
3. **MCP tool server** — JSON-RPC 2.0, 8 default tools with JSON Schema inputs, protocol version negotiation.
4. **Dev proxy** (`cosmoflare dev`) — stub HTTP surface (see Finding 1).

## Agent 7: API Design: 7/10

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| api_design | 7/10 | Strong typed contracts (interface, error hierarchy, generics, tested error mapping) undermined by three divergent transports, `any`-typed wire contract to desktop, and any-method REST handlers. |
| versioning | 6/10 | Semver canonical via `.version-registry.json` (0.26.0, no conflicts); MCP negotiates 3 protocol versions with fallback; but no Go API compat policy for a public v0 library and no explicit version marker on the daemon REST surface. |
| auth_security | 7/10 | Bearer-token daemon (128-bit random, all endpoints), fail-closed MCP mutation gate, least-privilege permission manifest; deductions for non-constant-time token compare, `?token=` query fallback, 0644-then-chmod token file write. |
| documentation | 7/10 | USAGE.md (2923 lines) documents every command, daemon endpoints table verified accurate against code, retry semantics documented exactly; deductions for dev-server docs promising unimplemented behavior and stale package doc comment. |

### Critical Findings

1. **`cosmoflare dev` proxy is a stub but documented as functional** — Every service route returns a hard-coded 502; the advertised proxying, `--watch` hot-reload, and `--profile` credential routing do not exist.
   - **Severity**: high
   - **File**: `pkg/cosmoflare/dev.go:161-168` (all service handlers return `{"error":"upstream not configured"}`), `dev.go:17,54,64` (`watch`/`profile` stored, never used — no fsnotify import in file), `docs/USAGE.md:51-53` ("local development proxy that routes requests to Cloudflare services through your configured credentials") and `USAGE.md:70` ("Watch `.cosmoflare.yaml` for changes and hot-reload").
   - **Evidence**: `pkg/cosmoflare/dev.go:161-167`:
     ```go
     mux.HandleFunc("/"+svcName+"/", func(w http.ResponseWriter, r *http.Request) {
         w.WriteHeader(http.StatusBadGateway)
         fmt.Fprintf(w, `{"error":"upstream not configured","service":"%s"}`, svcName)
     })
     ```
     `grep -n "watch" pkg/cosmoflare/dev.go` shows `watch` only in the field, option setter, getter, and startup event — never consumed by `Start()`. An agent or user following USAGE.md gets 502 for every request; the public options `WithDevWatch`/`WithDevProfile` are API contract promises with no implementation.
   - **Fix**: Either implement the proxy (reverse-proxy per service to the Cloudflare API with profile-resolved credentials, fsnotify on config) or cut the command + options and correct USAGE.md to match. A documented API surface that 502s on everything is worse than no surface.

2. **No HTTP timeout on the cloudflare-go transport used by 27 `FromCreds` constructors** — All non-R2 services (KV, Workers, Zones, DNS, Cache, … — used by the daemon per request, the MCP handlers, and most CLI commands) build the API client via `cloudflare.NewWithAPIToken(apiToken)` with no `http.Client`, which falls back to `http.DefaultClient` — verified in `cloudflare-go@v0.116.0/cloudflare.go:81-84`:
   ```go
   // Fall back to http.DefaultClient if the package user does not provide
   if api.httpClient == nil {
       api.httpClient = http.DefaultClient
   }
   ```
   `http.DefaultClient` has no timeout. The library thus ships **three transports with three timeout policies**: R2 `NewClient` threads a shared client honoring `WithTimeout` (default 30s, `pkg/cosmoflare/client.go:111-114`); `restClient` hardcodes 30s (`pkg/cosmoflare/rest_client.go:73`); the `FromCreds` path has none. The daemon path is partially saved by `r.Context()` propagation (`internal/server/rest.go:97`), but a stalled connection on a CLI command without a deadline hangs indefinitely.
   - **Severity**: medium
   - **File**: `pkg/cosmoflare/kv.go:93` (representative of 27 `NewWithAPIToken` call sites, none pass `cloudflare.HTTPClient`)
   - **Fix**: Give service constructors a shared default: create one package-level `http.Client{Timeout: 30s}` (or accept an optional client) and pass `cloudflare.HTTPClient(...)` in every `NewXServiceFromCreds`. One policy, stated in the package doc.

3. **Daemon REST handlers accept any HTTP method** — All routes are registered via `mux.HandleFunc` with no `r.Method` check; `grep -rn "MethodGet|405|StatusMethodNotAllowed" internal/server/` returns nothing. `POST /zones`, `DELETE /kv`, `PUT /healthz` all return the read-only payload with 200.
   - **Severity**: medium
   - **File**: `internal/server/server.go:125` and `internal/server/rest.go:48-60`
   - **Fix**: One guard in `authMiddleware` or a `getOnly` wrapper: `if r.Method != http.MethodGet { writeJSONStatus(w, 405, ...) ; return }`. A read-only API should fail closed on unsafe verbs.

4. **Daemon↔desktop wire contract is untyped (`any` end-to-end)** — `ServeSource` methods return `(any, error)` (`internal/server/rest.go:24-31`), `MetricsSnapshot` embeds `Zones/R2Buckets/Workers/KVNamespaces any` (`internal/server/metrics.go:17-20`), and the TS client's `get<T>` is an unchecked cast (`desktop/src/api/client.ts:39` — `return resp.json() as Promise<T>`). The only typed shape is `interface Profile { name: string }` (`client.ts:44-46`). The product principle "GUI apps wrap the same core" has no compile-time enforcement at the wrapping boundary: rename a JSON field in Go and nothing fails until runtime in the desktop app.
   - **Severity**: medium
   - **File**: `internal/server/rest.go:24-31`, `internal/server/metrics.go:14-22`, `desktop/src/api/client.ts:26-46`
   - **Fix**: Define concrete response structs for each endpoint in `internal/server` (they already exist downstream — `ZoneService.List` returns a typed slice; return it instead of `any`), then generate TS types from Go (tygo/ts-goapi) in the desktop build. The error contract is already typed and tested; extend the same rigor to success payloads.

5. **Exported public retry API is dead code** — `pkg/cosmoflare/retry.go` exports `Do`, `RetryConfig`, and 5 `RetryOption`s from the public library, but nothing in the codebase calls `retry.Do`. The codebase admits it: `pkg/cosmoflare/rest_client.go:49-51` — "the exported WithMaxRetries / WithInitialDelay already used by the generic retry.Do helper, to avoid a naming collision with that unrelated (**currently unwired**) retry package."
   - **Severity**: medium
   - **File**: `pkg/cosmoflare/retry.go:68` (`func Do`), `pkg/cosmoflare/rest_client.go:49-51`
   - **Fix**: Either wire `retry.Do` into a call path (it duplicates `restClient`'s superior Retry-After-aware logic, so unlikely) or unexport/delete it. Exported-but-unwired API in a v0 public library becomes load-bearing the moment an external user depends on it.

6. **Split product identity across the public API: `r2go2` vs `cosmoflare`** — The package is `cosmoflare`, but its doc comment says "Package r2go2" (`pkg/cosmoflare/types.go:2`), every error string is prefixed `r2go2:` (`pkg/cosmoflare/errors.go:25-30`), machine config lives at `~/.r2go2/config.yaml` (`pkg/cosmoflare/config.go:108`) while project config migrated to `.cosmoflare.yaml` with legacy fallback (`config.go:76-84`), and `ProjectConfig`'s doc comment still says ".r2go2.yaml project-level configuration" (`config.go:11`) — contradicting the loader three lines of behavior below it. Consumers see two names for one product in errors, paths, and docs.
   - **Severity**: medium
   - **File**: `pkg/cosmoflare/types.go:2`, `pkg/cosmoflare/errors.go:25`, `pkg/cosmoflare/config.go:11,108`
   - **Fix**: Rename error prefix to `cosmoflare:` (major-version-safe at v0), fix the two stale doc comments now, and add a `~/.cosmoflare/config.yaml` path with `~/.r2go2` read-fallback mirroring the project-config migration.

7. **Inconsistent error-response content types and non-constant-time token check** — 503 "no data source" and 401 "unauthorized" are emitted via `http.Error` (Content-Type `text/plain; charset=utf-8` + `nosniff`) with JSON-shaped bodies (`internal/server/rest.go:70,93`, `internal/server/server.go:135`), while API errors use `writeJSONStatus` (`application/json`, `rest.go:109-113,145-148`). The bearer-token check compares with `==` rather than `subtle.ConstantTimeCompare`, and accepts `?token=` in the URL (documented as acceptable for localhost EventSource, `server.go:142-151`) — URL tokens can persist in client-side fetch logs/history.
   - **Severity**: low
   - **File**: `internal/server/rest.go:70,93`, `internal/server/server.go:134-152`
   - **Fix**: Route all error paths through `writeAPIError`/`writeJSONStatus`; swap the comparison to `subtle.ConstantTimeCompare` (one line); keep the query fallback but document token scrubbing on the client.

8. **Token file written world-readable then chmod'd; not-found classified by substring** — `MachineConfig.Save()` lets viper create `~/.r2go2/config.yaml` at default perms (0644) and only then `os.Chmod(configPath, 0600)` (`pkg/cosmoflare/config.go:164-173`) — a crash between write and chmod leaves the API token world-readable. `isNotFound` falls back to substring matching `"not found" | "could not find" | "404"` for non-cloudflare errors (`pkg/cosmoflare/errors.go:115-118`); any wrapped error whose message merely contains "404" (a size, a key name) is misclassified as not-found, which the daemon maps to HTTP 404 (`internal/server/rest.go:132-135`).
   - **Severity**: low
   - **File**: `pkg/cosmoflare/config.go:164-173`, `pkg/cosmoflare/errors.go:115-118`
   - **Fix**: Write to a temp file created with `os.OpenFile(..., 0600)` then rename; for classification, match on typed status codes first and restrict the string fallback to `*R2Error` messages the library itself produced.

### What is done well (evidence-backed)

- **Typed, tested error contract**: five error types with `Op/Bucket/Key` context and `Unwrap` (`pkg/cosmoflare/errors.go:15-58`), mapped in the daemon to stable HTTP statuses + `{"error","code"}` bodies, verified by a dedicated contract test (`internal/server/error_contract_test.go:15-78` covers all 5 mappings). This is better than most local daemons.
- **Rate-limit-aware retry transport** (`pkg/cosmoflare/rest_client.go`): honors `Retry-After` as integer or HTTP-date (`:106-124`), retries 429 always but 502/503/504 only for idempotent methods (`:169-184`), ±25% jitter capped at 30s, drains bodies for connection reuse — and USAGE.md documents these exact semantics (`docs/USAGE.md:27-49`). cloudflare-go's own path gets 3-retry backoff from the dependency (verified `cloudflare.go:68-70`).
- **Daemon auth model**: random 128-bit token, bearer gate on every endpoint including SSE, single-line JSON stdout handshake with stderr separation (`cmd/serve.go:119-127,170-177`), per-request lazy credential resolution with no write-side account switch (`cmd/serve.go:261-350`), keychain probing hard-disabled (`cmd/serve.go:80`).
- **MCP server protocol hygiene**: JSON-RPC 2.0 with standard error codes (`mcp.go:104-110`), protocol version negotiation across 2024-11-05/2025-03-26/2025-06-18 with canonical fallback (`mcp.go:78-101,288-300`), JSON Schema input contracts on all 8 tools, tool errors returned as `isError` content per spec (`mcp.go:350-355`), and mutations fail-closed behind `mcp.allow_mutations: true` (`pkg/cosmoflare/config.go:26-31`).
- **SSE wire discipline**: channel names are named constants documented as the shared wire contract (`internal/server/sse.go:10-18`), backpressure policy is explicit (drop frames, never block the publisher, `sse.go:27-29,53-63`), headers flushed early, clean drain on Close.
- **Generic pagination contract**: `ListResult[T]{Items, NextToken, IsTruncated, CommonPrefixes}` (`pkg/cosmoflare/types.go:61-66`) used by `ListObjects` with real continuation-token plumbing (`pkg/cosmoflare/storage.go:105-150`).
- **Authorization guidance**: least-privilege permission manifest per command group (`cmd/auth_permissions.go:57-70`, `permdata.LeastPrivilege`) — rare and valuable for a Cloudflare tool.
- **Partial-failure metrics model**: `MetricsSnapshot` publishes per-source errors instead of dropping the whole snapshot (`internal/server/metrics.go:11-29`), with change detection that zeroes the timestamp so polls don't re-publish (`metrics.go:93-107`).

### Recommendations

- [ ] Implement or remove the `cosmoflare dev` proxy; until then mark it experimental in USAGE.md and delete the unimplemented `--watch`/`--profile` claims (effort: medium)
- [ ] Pass a timeout-bearing `http.Client` (package default 30s) into `cloudflare.NewWithAPIToken` at all 27 `FromCreds` constructors; document the single timeout policy (effort: small)
- [ ] Reject non-GET methods with 405 on all daemon routes via one wrapper (effort: small)
- [ ] Replace `any` in `ServeSource`/`MetricsSnapshot` with concrete response structs and generate TS types for `desktop/src/api` (effort: medium)
- [ ] Unexport or delete the unwired `retry.go` public API before external users depend on it (effort: small)
- [ ] Fix stale identity: "Package r2go2" doc, `.r2go2.yaml` comment on `ProjectConfig`, and plan the `cosmoflare:` error prefix + `~/.cosmoflare` machine-config path with read fallback (effort: medium)
- [ ] Use `writeJSONStatus` for the 503/401 paths and `subtle.ConstantTimeCompare` for the token check (effort: small)
- [ ] Write machine config atomically at 0600 (temp file + rename) in `MachineConfig.Save()` (effort: small)
- [ ] Add a Go API compatibility policy (v0 semver caveats, what's stable: error types, `R2Client`, options) to the package doc before the library gains external importers (effort: small)

### Roadmap Suggestions

- **Dev server: real proxy or removal** — `cosmoflare dev` currently 502s every request while docs describe a working proxy; implement per-service reverse proxying or deprecate the command (priority: high, effort: medium)
- **Unified transport resilience policy** — one timeout/retry/backoff policy across the three transports (R2 client, restClient, cloudflare-go), configurable via `ClientOption`, documented once (priority: medium, effort: medium)
- **Typed daemon wire contract + TS codegen** — concrete Go response structs for all daemon endpoints with generated TypeScript types in `desktop/src/api`, CI-checked for drift (priority: medium, effort: medium)
- **Public library v1 readiness** — compat policy doc, `cosmoflare` identity cleanup, and removal of dead exported API (`retry.go`) ahead of a 1.0 of `pkg/cosmoflare` (priority: low, effort: small)

---

```json:audit-result
{
  "agent": "api-design",
  "overall_score": 7,
  "sub_scores": {
    "api_design": 7,
    "versioning": 6,
    "auth_security": 7,
    "documentation": 7
  },
  "critical_findings": [
    {
      "title": "cosmoflare dev proxy is a stub but documented as functional",
      "severity": "high",
      "file": "pkg/cosmoflare/dev.go:161",
      "fix": "Implement the per-service reverse proxy with profile-resolved credentials and fsnotify config watch, or remove the command/options and correct docs/USAGE.md to match current behavior",
      "effort": "medium"
    },
    {
      "title": "No HTTP timeout on cloudflare-go transport in 27 FromCreds constructors (http.DefaultClient fallback)",
      "severity": "medium",
      "file": "pkg/cosmoflare/kv.go:93",
      "fix": "Pass a package-default http.Client{Timeout: 30s} via cloudflare.HTTPClient in every NewXServiceFromCreds; document the single timeout policy",
      "effort": "small"
    },
    {
      "title": "Daemon REST handlers accept any HTTP method (no 405 enforcement)",
      "severity": "medium",
      "file": "internal/server/rest.go:48",
      "fix": "Add a GET-only guard (405 for other verbs) in authMiddleware or a wrapper applied to all routes",
      "effort": "small"
    },
    {
      "title": "Daemon-to-desktop wire contract untyped (any in ServeSource and MetricsSnapshot, unchecked cast in TS client)",
      "severity": "medium",
      "file": "internal/server/rest.go:24",
      "fix": "Return concrete typed structs from ServeSource methods and generate TypeScript types for desktop/src/api from Go definitions",
      "effort": "medium"
    },
    {
      "title": "Exported public retry API (retry.Do) is unwired dead code",
      "severity": "medium",
      "file": "pkg/cosmoflare/retry.go:68",
      "fix": "Unexport or delete the retry helper package; restClient already implements superior Retry-After-aware retry",
      "effort": "small"
    },
    {
      "title": "Split product identity: r2go2 vs cosmoflare across public API surface",
      "severity": "medium",
      "file": "pkg/cosmoflare/types.go:2",
      "fix": "Fix stale doc comments (Package r2go2, .r2go2.yaml on ProjectConfig); plan error-prefix rename to cosmoflare: and ~/.cosmoflare machine-config path with read fallback",
      "effort": "medium"
    },
    {
      "title": "Inconsistent error content types (text/plain 401/503 vs application/json) and non-constant-time token comparison with ?token= query fallback",
      "severity": "low",
      "file": "internal/server/server.go:134",
      "fix": "Route all error paths through writeJSONStatus; use subtle.ConstantTimeCompare for the bearer token check",
      "effort": "small"
    },
    {
      "title": "Token config file written at default perms then chmod'd to 0600; isNotFound classifies errors by substring matching",
      "severity": "low",
      "file": "pkg/cosmoflare/config.go:164",
      "fix": "Write machine config atomically at 0600 (temp file + rename); restrict isNotFound string fallback to library-produced R2Error messages",
      "effort": "small"
    }
  ],
  "recommendations": [
    {
      "action": "Implement or remove the cosmoflare dev proxy; correct USAGE.md until then",
      "effort": "medium",
      "priority": "high"
    },
    {
      "action": "Thread a timeout-bearing http.Client through all NewXServiceFromCreds constructors",
      "effort": "small",
      "priority": "high"
    },
    {
      "action": "Reject non-GET methods with 405 on all daemon REST routes",
      "effort": "small",
      "priority": "medium"
    },
    {
      "action": "Type the daemon wire contract (concrete Go structs) and generate TS types for the desktop client",
      "effort": "medium",
      "priority": "medium"
    },
    {
      "action": "Unexport/delete the unwired retry.go public API",
      "effort": "small",
      "priority": "medium"
    },
    {
      "action": "Unify naming: fix stale r2go2 doc comments, plan error-prefix and machine-config path migration",
      "effort": "medium",
      "priority": "medium"
    },
    {
      "action": "Use writeJSONStatus for 503/401 error paths and constant-time token comparison",
      "effort": "small",
      "priority": "low"
    },
    {
      "action": "Write machine config atomically at 0600 permissions",
      "effort": "small",
      "priority": "low"
    },
    {
      "action": "Document a Go API compatibility policy for the public pkg/cosmoflare library",
      "effort": "small",
      "priority": "low"
    }
  ],
  "roadmap_suggestions": [
    {
      "title": "Dev server: real proxy or removal",
      "description": "cosmoflare dev currently returns 502 for every request while USAGE.md documents a working proxy with config hot-reload; implement per-service proxying or deprecate the command",
      "priority": "high",
      "effort": "medium"
    },
    {
      "title": "Unified transport resilience policy",
      "description": "One timeout/retry/backoff policy across the three transports (R2 client, restClient, cloudflare-go path), configurable via ClientOption and documented once",
      "priority": "medium",
      "effort": "medium"
    },
    {
      "title": "Typed daemon wire contract with TS codegen",
      "description": "Concrete Go response structs for all daemon REST/SSE payloads with generated TypeScript types in desktop/src/api, drift-checked in CI",
      "priority": "medium",
      "effort": "medium"
    },
    {
      "title": "Public library v1 readiness",
      "description": "Compat policy doc, cosmoflare identity cleanup, dead exported API removal ahead of a 1.0 of pkg/cosmoflare",
      "priority": "low",
      "effort": "small"
    }
  ]
}
```
