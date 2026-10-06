# FEAT-044 — Wire knowledge.Transport into the shared CF client factory

Spec (approved design — follow EXACTLY): docs/brainstorming/2026-10-06-feat044-knowledge-transport-wiring.md

## Context

`pkg/cosmoflare/knowledge` embeds an endpoint registry enforced by
`knowledge.Transport` (an http.RoundTripper): requests to routes inside a
knowledge-pack scope that match no registered endpoint fail fast in-process
with an actionable error (Cloudflare would return a misleading 10405).
Today the Transport is wired only in `NewRateLimitServiceFromCreds`
(pkg/cosmoflare/ratelimit.go). FEAT-039 unified all Cloudflare API traffic
behind the `controlPlaneClient()` / `newCloudflareAPI()` chokepoint in
pkg/cosmoflare/transport.go. This task installs the Transport at that
chokepoint and folds in the two remaining direct construction sites.

Read first (context files): pkg/cosmoflare/transport.go,
pkg/cosmoflare/knowledge/transport.go, pkg/cosmoflare/knowledge/knowledge.go,
pkg/cosmoflare/client.go (NewClient + WithHTTPClient option),
pkg/cosmoflare/ratelimit.go (NewRateLimitServiceFromCreds),
pkg/cosmoflare/rest_client.go, and the brainstorm spec above.

## Exact changes

### 1. pkg/cosmoflare/transport.go

`controlPlaneClient()` becomes:

```go
func controlPlaneClient() *http.Client {
	return &http.Client{Timeout: DefaultControlPlaneTimeout, Transport: &knowledge.Transport{}}
}
```

Update the file-header policy comment: the control-plane tier now carries
the knowledge endpoint registry (knowledge.Transport) in addition to the
whole-request timeout. Add one sentence documenting the WithHTTPClient
wrap-always semantics (see change 3). Import
`github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge` (child
package; no import cycle exists — verify with `go build`).

### 2. pkg/cosmoflare/ratelimit.go

`NewRateLimitServiceFromCreds` currently calls
`cloudflare.NewWithAPIToken(apiToken, cloudflare.HTTPClient(&http.Client{Transport: &knowledge.Transport{}}))`.
Replace with the chokepoint:

```go
cf, err := newCloudflareAPI(apiToken)
```

Remove the now-unused `net/http` and knowledge imports if nothing else in
the file uses them. Error wrapping stays as-is (`newError("NewRateLimitService", ...)`).

### 3. pkg/cosmoflare/client.go — NewClient

Two cases:
- Caller passed NO custom http client (cfg.httpClient == nil): the default
  becomes `&http.Client{Timeout: cfg.timeout, Transport: &knowledge.Transport{}}`.
- Caller DID pass one via WithHTTPClient: wrap its transport, preserving the
  chain: `httpClient.Transport = &knowledge.Transport{Base: httpClient.Transport}`
  (mutate the copy semantics carefully — do NOT mutate a caller-owned client
  struct in place if it is shared; if in doubt build a shallow copy of the
  http.Client with the wrapped Transport).

Rationale (from the approved design): wrap-always — the caller keeps their
tuning; the registry still applies; a bare RoundTripper remains the
documented bypass. Note this in a short comment.

### 4. pkg/cosmoflare/rest_client.go

Wherever restClient constructs its `http.Client` (find its zero-value or
default construction), wrap the same way:
`Transport: &knowledge.Transport{}` on the default construction. Do not
alter its retry/backoff logic — the Transport composes underneath it.

### 5. Tests (write FIRST, red, then implement)

Create `pkg/cosmoflare/knowledge_wiring_test.go` (package cosmoflare,
build-tag-free) with:

- `TestKnowledgeTransportFullPathBlock`: build a `NewClient` with test
  creds (env or WithAPIToken/WithAccountID options) and a Base pointing at
  an `httptest.Server` that fails the test if reached; issue a request to
  an in-scope UNREGISTERED route (e.g.
  `DELETE /zones/00000000000000000000000000000000/rulesets/phases/http_ratelimit/entrypoint`)
  through any service method that reaches the transport, or directly via
  the constructed http.Client if NewClient's shape requires it — assert the
  error contains "not a registered endpoint" and that the httptest server
  saw zero requests. Follow existing test patterns in
  pkg/cosmoflare/*_test.go for how clients are constructed in tests.
- `TestKnowledgeTransportWrapAlways`: `NewClient` with a custom
  `WithHTTPClient(&http.Client{Transport: marker})` where `marker` is a
  counting RoundTripper; an out-of-scope route (e.g. `GET /zones`) passes
  through and increments the counter (marker stays in the chain), while the
  in-scope unregistered route still errors (registry applied).
- Extend or add a chokepoint guard test (pattern:
  `TestTransportChokepoint_Guard` in pkg/cosmoflare — find and follow it):
  after your change, `grep -c "cloudflare.NewWithAPIToken"` across
  pkg/cosmoflare non-test files must equal exactly 1 (transport.go only);
  ratelimit.go and client.go no longer construct clients directly. Pin this
  count.

Do not weaken or delete any existing test. The knowledge package's own
tests (pkg/cosmoflare/knowledge/) must stay green untouched.

## Verification (mandatory, in order)

```bash
go test ./pkg/cosmoflare/ -run 'TestKnowledgeTransport|TestTransportChokepoint' -v   # red before changes, green after
go test ./pkg/cosmoflare/ ./pkg/cosmoflare/knowledge/                                # packages green
go test ./cmd/ -timeout 120s                                                          # consumers green
go vet ./pkg/cosmoflare/...
gofmt -l pkg/cosmoflare/ | grep -v _test.go || true                                   # no output = clean
```

## Commit

Conventional, body required:

```
feat(transport): install knowledge.Transport at the control-plane chokepoint (FEAT-044)

Root cause: the endpoint registry was wired only into the rate-limit
service; every other control-plane path lost the fast-fail protection
against in-scope unregistered routes. FEAT-039's chokepoint made a single
install point possible.

- controlPlaneClient() carries knowledge.Transport; newCloudflareAPI callers inherit it
- NewRateLimitServiceFromCreds goes through newCloudflareAPI; manual wiring removed
- NewClient wraps default AND WithHTTPClient-override transports (wrap-always, Base chain preserved)
- restClient default construction wrapped (same CF v4 surface)

Spec: docs/brainstorming/2026-10-06-feat044-knowledge-transport-wiring.md
```

## Out of scope

- Any change to knowledge pack data (permissions.json / ratelimit.json).
- The R2 S3 data-plane client (BUG-042 policy — stays unwired by design).
- Desktop/mobile tiers.
- Any file not listed above.
