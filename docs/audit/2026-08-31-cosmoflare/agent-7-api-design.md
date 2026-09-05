# Agent 7: API Design

> Audit: 2026-08-31, cosmoflare v0.17.0, fresh mode. Report saved verbatim from agent output.

I have completed the investigation across 16 source files. Here is the audit report.

## API Design: 6/10

Cosmoflare exposes four API surfaces: the `serve` desktop daemon (REST+SSE), the MCP tool server (JSON-RPC 2.0 over stdio), the CLI `--json` envelope, and the public Go library (`pkg/cosmoflare`). The library layer shows strong contract discipline. The transport layers flatten that discipline at the boundary.

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| api_design | 6/10 | Typed contracts, functional options, generic `ListResult[T]`, and a tested daemon. But all backend errors collapse to 502, error bodies mix `text/plain` and JSON, and no HTTP method is enforced. |
| versioning | 4/10 | No versioning anywhere. Daemon paths are unversioned (`/zones`, not `/v1/zones`). The stdout handshake carries `addr`+`token` only. MCP pins `2024-11-05` with no negotiation and reports a hardcoded `1.0.0`. |
| auth_security | 7/10 | Bearer token on every daemon endpoint including SSE. Random 128-bit token, ephemeral port, localhost bind, documented no-keychain gate. Weaknesses: non-constant-time compare, empty-token edge, `?token=` query fallback, no rate limit. |
| documentation | 7/10 | `docs/USAGE.md` (2388 lines) documents every daemon endpoint, flag, and the handshake format accurately. Every command has rich `--help` with examples. No OpenAPI spec, no machine-readable schema for SSE channels. |

### Critical Findings

1. **All backend failures collapse to HTTP 502; the typed error taxonomy is discarded** — `withCloudflare` maps every error from the service layer to `502` with `{"error": err.Error()}`. An invalid `?profile=` name (client fault) returns 502. An expired API token (auth fault) returns 502. A missing resource returns 502. The library defines five error classes (`R2AuthError`, `R2ValidationError`, `R2NotFoundError`, `R2QuotaError`, `R2AccessDeniedError` in `pkg/cosmoflare/errors.go:27-49`), and `pkg/cosmoflare/retry.go:123-156` already consumes them. The REST layer ignores them. The desktop client cannot distinguish "fix your profile" from "Cloudflare is down".
   - **Severity**: medium
   - **File**: `internal/server/rest.go:94`
   - **Fix**: In `withCloudflare`, switch on `errors.As`: `*R2ValidationError` → 400, `*R2AuthError` → 401, `*R2NotFoundError` → 404, `*R2QuotaError` → 429, default → 502. Return `{"error": msg, "code": class}`.

2. **`--json` mode fails silently on configuration errors** — `PersistentPreRun` validates the token, then calls `printError(...)` and `os.Exit(1)`. `printError` returns immediately when `JSONOutput` is true. So `cosmoflare zone list --json` with a missing token prints nothing at all and exits 1. This breaks the documented agent contract (`OutputResponse` envelope, `cmd/root.go:215-221`) and the project's own "agent-readable error messages" rule. An agent cannot tell what failed.
   - **Severity**: medium
   - **File**: `cmd/root.go:99-110` and `cmd/root.go:197-202`
   - **Fix**: In `PersistentPreRun`, branch on `JSONOutput` and emit `printErrorJSON(...)` before `os.Exit(1)`. Add a regression test that runs a command with `--json` and no credentials, and asserts a JSON error body on stdout.

3. **Error responses violate the JSON content contract** — The auth middleware writes plain text (`http.Error(w, "unauthorized", 401)` → `Content-Type: text/plain`). The no-source path writes a JSON body through `http.Error`, which still stamps `text/plain` (`internal/server/rest.go:64` and `rest.go:87`). Success paths set `application/json` (`server.go:169`). A client that parses error bodies as JSON fails on 401 and 503. The same defect exists across the dev server: `pkg/cosmoflare/dev.go:154-191` writes JSON bodies with `fmt.Fprintf` and never sets `Content-Type`.
   - **Severity**: medium
   - **File**: `internal/server/server.go:135`, `internal/server/rest.go:64`, `pkg/cosmoflare/dev.go:161-167`
   - **Fix**: Use `writeJSONStatus` for every error path in the daemon. Set `Content-Type: application/json` in all `DevServer` handlers.

4. **No HTTP method enforcement on any daemon route** — All routes register via `mux.HandleFunc` with no method check. `POST /zones`, `DELETE /events`, and `PUT /healthz` all execute the read handler and return 200. `docs/USAGE.md:93-105` documents these as `GET` only. The documented contract and the enforced contract differ. Any future write-side endpoint inherits this hole.
   - **Severity**: medium
   - **File**: `internal/server/rest.go:41-55`, `internal/server/server.go:124-128`
   - **Fix**: Add a `requireGet` wrapper that returns 405 with `Allow: GET` for other methods. Apply it in `registerRoutes`.

5. **MCP server hardcodes its version and does not negotiate the protocol version** — `handleInitialize` returns `protocolVersion: "2024-11-05"` regardless of the client's requested version, and `serverInfo.version: "1.0.0"` while the actual application version is 0.17.0 (`pkg/cosmoflare/mcp.go:196-207`). The version is not injectable, so every consumer reports the same wrong number. This also violates the project's single-source-of-truth rule for version values.
   - **Severity**: medium
   - **File**: `pkg/cosmoflare/mcp.go:198-205`
   - **Fix**: Echo the client's requested protocol version when supported, else fall back to the latest supported. Accept the version through a `NewMCPServer` option so `cmd/mcp.go` passes `AppVersion`.

6. **Token comparison is not constant-time and an empty configured token authenticates** — `authorized()` compares the bearer header with `==` (`internal/server/server.go:147-152`) instead of `crypto/subtle.ConstantTimeCompare`. Also, `server.New(Config{Token: ""})` (programmatic use) accepts `Authorization: Bearer ` (empty secret) because the header check has no non-empty guard, unlike the query check. The daemon binds localhost and `cmd/serve.go:76-83` always generates a token, which limits exposure. The library boundary still permits the weak state.
   - **Severity**: medium
   - **File**: `internal/server/server.go:147-152`
   - **Fix**: Use `subtle.ConstantTimeCompare` on both forms. Return an error (or panic in `New`) when `Config.Token` is empty. The `http.Server` at `cmd/serve.go:104` also sets no `ReadHeaderTimeout`; add one.

7. **`ZoneService.Get` classifies every error as not-found** — Any failure from `ZoneDetails` — including 403, 429, and 5xx — becomes `R2NotFoundError` (`pkg/cosmoflare/zone.go:110-113`). Callers that branch on `errors.As(err, &*R2NotFoundError)` get a false positive on permission and outage errors. The `R2Error.Bucket`/`Key` fields are also repurposed for zones, so the message reads `bucket= key=<zoneID>`.
   - **Severity**: low
   - **File**: `pkg/cosmoflare/zone.go:112`
   - **Fix**: Inspect the wrapped error status. Map 404 to `notFound`, everything else to `newError`. Add a dedicated `Resource` field instead of reusing `Bucket`.

8. **R2 bucket list truncates at the first page** — `client.ListBuckets` calls `cloudflare-go` v0.116 `ListR2Buckets`, which performs one request and discards the response's pagination cursor (verified in the module cache: `r2_bucket.go:57-74` of cloudflare-go). Accounts beyond the first page see a silent truncation. The library already models pagination correctly elsewhere (`ListResult[T]` in `pkg/cosmoflare/types.go:61-66`; KV exposes the cursor at `pkg/cosmoflare/kv.go:293-296`). Zones and KV auto-paginate inside cloudflare-go, so only R2 is affected.
   - **Severity**: low
   - **File**: `pkg/cosmoflare/storage.go:36-53`
   - **Fix**: Loop with the returned cursor, or call the raw endpoint, and return `ListResult[*Bucket]`.

9. **Legacy `r2go2` identity leaks into the public contract** — Every library error renders as `r2go2: <op>: <msg>` (`pkg/cosmoflare/errors.go:14-22`), `client.go:112` prefixes `r2go2:` in an error, and the package doc says "Package r2go2" while the package is `cosmoflare` (`pkg/cosmoflare/types.go:2-9`). A public Go API under `github.com/CosmoLabs-org/cosmoflare` reports a different product name in every error string.
   - **Severity**: low
   - **File**: `pkg/cosmoflare/errors.go:16-21`, `pkg/cosmoflare/types.go:2`
   - **Fix**: Change the format prefix to `cosmoflare:` and fix the package doc. Note the string change in the changelog as a breaking output change.

10. **SSE channel is lossy with no replay** — The hub drops frames for slow clients (`internal/server/sse.go:43-53`, documented deliberately) and `handleEvents` ignores `Last-Event-ID` (`sse.go:90-124`). The metrics producer mitigates cost with subscriber gating and delta detection (`internal/server/metrics.go:41-44`, `metrics.go:81-84`) — good hygiene. But a desktop client that reconnects misses the transition notifications that `SetCloudflareOnline` publishes only on state flips (`server.go:77-90`), so the health indicator can show stale state until the next poll.
    - **Severity**: low
    - **File**: `internal/server/sse.go:116`, `internal/server/server.go:77-90`
    - **Fix**: Keep the last frame per channel and send it as the first frame on subscribe. That is cheaper than full `Last-Event-ID` replay.

### Strengths observed (evidence)

- `pkg/cosmoflare/retry.go:68-118`: exponential backoff with jitter, context-aware waits, and status-code-aware retryability for both cloudflare-go and AWS SDK error shapes. Genuinely good resilience code.
- `internal/server/rest_test.go:66-96`: contract tests assert 401 without token and the `?profile=` scoping contract against a fake source.
- `cmd/serve.go:104-114`: clean stdout/stderr separation for the handshake line; `cmd/serve.go:137-148` drains SSE before `Shutdown`.
- `pkg/cosmoflare/mcp.go:68-74`: correct JSON-RPC 2.0 error codes; tool failures returned as `result.isError` per MCP spec; `mcp.go:366-376` re-validates required fields even though the schema declares them.
- `docs/USAGE.md:71-118`: the daemon endpoint table matches the implementation exactly (verified against `rest.go` and `sse.go`).

### Recommendations

- [ ] Map library error classes to HTTP statuses and structured bodies in `withCloudflare` (effort: small)
- [ ] Emit the `OutputResponse` JSON error envelope in `PersistentPreRun` before `os.Exit(1)` (effort: small)
- [ ] Add a GET-only method guard with 405 + `Allow` header to all daemon routes (effort: small)
- [ ] Use `subtle.ConstantTimeCompare` and reject empty `Config.Token` in `server.New`; add `ReadHeaderTimeout` (effort: small)
- [ ] Make the MCP `serverInfo.version` injectable and echo the client's protocol version (effort: small)
- [ ] Serve all daemon and dev-server error paths as `application/json` (effort: small)
- [ ] Fix `ZoneService.Get` error classification and stop reusing `Bucket`/`Key` for zones (effort: small)
- [ ] Cursor-paginate `ListBuckets` and return `ListResult[*Bucket]` (effort: medium)
- [ ] Publish an OpenAPI 3.1 spec for the daemon REST + SSE channels, generated or hand-written, under `docs/` (effort: medium)
- [ ] Add a `protocol_version` field to the serve handshake JSON and document the shell/daemon compatibility policy (effort: medium)

### Roadmap Suggestions

- **Daemon error contract v2** — Structured error bodies (`{error, code}`) with status mapping from the existing error taxonomy; desktop app can then branch on failure class (priority: high, effort: small)
- **Versioned daemon API** — `/v1/` prefix or handshake `protocol_version` field plus a documented skew policy between desktop shell and daemon (priority: medium, effort: medium)
- **OpenAPI spec for `serve`** — Machine-readable contract for the 6 REST endpoints and 3 SSE channels; enables generated TS clients for the desktop app (priority: medium, effort: medium)
- **MCP version SSOT + negotiation** — Wire `AppVersion` into `serverInfo` and support client-requested protocol versions (priority: medium, effort: small)
- **SSE last-frame replay** — Retain the newest frame per channel and replay on subscribe so reconnecting clients converge without waiting for the next poll (priority: low, effort: medium)
