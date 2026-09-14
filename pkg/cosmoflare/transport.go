package cosmoflare

// Unified transport resilience policy (FEAT-039).
//
// The package speaks to Cloudflare over three transports. Each has exactly
// one policy, defined here:
//
//   - CONTROL PLANE (Cloudflare REST API via cloudflare-go and raw calls):
//     a shared http.Client carrying DefaultControlPlaneTimeout. API calls
//     are small request/response pairs — a whole-request timeout is the
//     correct hang protection. All constructors and raw request paths MUST
//     go through newCloudflareAPI / controlPlaneClient; the
// TestTransportChokepoint_Guard test enforces this structurally.
//   - DATA PLANE (R2 S3 SDK): timeout-free client — transfers are bounded
//     by context deadlines and SDK retries, never a whole-request timeout
//     (BUG-042: the 30s control-plane timeout killed large transfers
//     mid-body when it was shared).
//   - REST CLIENT (restClient): its own Retry-After-aware retry policy
//     with exponential backoff — see rest_client.go.
//
// An explicit WithHTTPClient on NewClient overrides the control-plane
// client for that client instance (caller's deliberate choice).

import (
	"net/http"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// DefaultControlPlaneTimeout bounds every Cloudflare API call made through
// the shared control-plane client. Changing it is a documented behavioral
// change.
const DefaultControlPlaneTimeout = 30 * time.Second

// controlPlaneClient is the shared timeout-bearing client for Cloudflare
// API calls. One instance, one policy — never http.DefaultClient (which
// hangs forever).
func controlPlaneClient() *http.Client {
	return &http.Client{Timeout: DefaultControlPlaneTimeout}
}

// newCloudflareAPI is the ONLY sanctioned way to build a cloudflare-go
// client in this package: it wires the shared control-plane HTTP client so
// every service inherits the timeout policy.
func newCloudflareAPI(apiToken string) (*cloudflare.API, error) {
	return cloudflare.NewWithAPIToken(apiToken, cloudflare.HTTPClient(controlPlaneClient()))
}
