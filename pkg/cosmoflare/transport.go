package cosmoflare

// Unified transport resilience policy (FEAT-039).
//
// The package speaks to Cloudflare over three transports. Each has exactly
// one policy, defined here:
//
//   - CONTROL PLANE (Cloudflare REST API via cloudflare-go and raw calls):
//     a shared http.Client carrying DefaultControlPlaneTimeout AND the
//     knowledge endpoint registry (knowledge.Transport): in-scope
//     unregistered routes fail fast in-process instead of drawing a
//     misleading 10405 from Cloudflare. API calls are small
//     request/response pairs — a whole-request timeout is the correct hang
//     protection. All constructors and raw request paths MUST go through
//     newCloudflareAPI / controlPlaneClient; the
// TestTransportChokepoint_Guard test enforces this structurally.
//   - DATA PLANE (R2 S3 SDK): timeout-free client — transfers are bounded
//     by context deadlines and SDK retries, never a whole-request timeout
//     (BUG-042: the 30s control-plane timeout killed large transfers
//     mid-body when it was shared).
//   - REST CLIENT (restClient): its own Retry-After-aware retry policy
//     with exponential backoff — see rest_client.go.
//
// An explicit WithHTTPClient on NewClient overrides the control-plane
// client for that client instance (caller's deliberate choice), but the
// override is WRAPPED, not replaced: knowledge.Transport{Base: existing}
// keeps the caller's tuning while the registry still applies (FEAT-044
// wrap-always semantics). A caller who truly wants to bypass knowledge
// passes a client whose Transport is already a bare RoundTripper of their
// own.

import (
	"net/http"
	"time"

	"github.com/cloudflare/cloudflare-go"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"
)

// DefaultControlPlaneTimeout bounds every Cloudflare API call made through
// the shared control-plane client. Changing it is a documented behavioral
// change.
const DefaultControlPlaneTimeout = 30 * time.Second

// controlPlaneClient is the shared timeout-bearing client for Cloudflare
// API calls. One instance, one policy — never http.DefaultClient (which
// hangs forever). The knowledge.Transport registry rides under bounded
// retry (RetryTransport, FEAT-061: 2 retries, exponential backoff +
// jitter, 429 always / 5xx for idempotent requests) on top of the
// whole-request timeout (FEAT-044).
func controlPlaneClient() *http.Client {
	return &http.Client{Timeout: DefaultControlPlaneTimeout, Transport: &RetryTransport{Base: &knowledge.Transport{}}}
}

// newCloudflareAPI is the ONLY sanctioned way to build a cloudflare-go
// client in this package: it wires the shared control-plane HTTP client so
// every service inherits the timeout + registry policy.
func newCloudflareAPI(apiToken string) (*cloudflare.API, error) {
	return newCloudflareAPIWithClient(apiToken, controlPlaneClient())
}

// newCloudflareAPIWithClient is the chokepoint variant for the one caller
// that must supply its own control-plane http.Client (NewClient, whose
// WithHTTPClient override is wrapped per the wrap-always policy). The
// client must already carry knowledge.Transport in its chain. This keeps
// cloudflare.NewWithAPIToken a single construction site (transport.go).
func newCloudflareAPIWithClient(apiToken string, hc *http.Client) (*cloudflare.API, error) {
	return cloudflare.NewWithAPIToken(apiToken, cloudflare.HTTPClient(hc))
}
