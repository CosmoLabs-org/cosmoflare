package knowledge

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

// newOKServer spins up a test upstream that always answers 200 OK and
// registers its own cleanup with the test. Both transport tests need it as
// the only observable signal that a request either left the process or was
// blocked locally.
func newOKServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestTransportBlocksUnregisteredInScopeRoute verifies the Transport's core
// guarantee: a route inside a pack scope that matches no registered endpoint
// fails locally without ever reaching the upstream, and the error cites
// Cloudflare's 10405 wording so it is not misread as an auth problem.
func TestTransportBlocksUnregisteredInScopeRoute(t *testing.T) {
	t.Parallel()
	var sawRequest bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRequest = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := &Transport{Base: http.DefaultTransport}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
		srv.URL+"/zones/z1/rulesets/phases/http_ratelimit/entrypoint/rules", nil)
	_, err := tr.RoundTrip(req)

	if err == nil {
		t.Fatal("unregistered in-scope route must be blocked")
	}
	if sawRequest {
		t.Fatal("blocked route must not reach the server")
	}
	if !strings.Contains(err.Error(), "10405") {
		t.Fatalf("block message must cite CF's 10405 wording: %v", err)
	}
}

// TestTransportPassesRegisteredRouteAndOutOfScope verifies the Transport
// stays transparent for every route it has no authority over: registered
// in-scope endpoints (with and without the /client/v4 prefix) and paths
// outside all pack scopes.
func TestTransportPassesRegisteredRouteAndOutOfScope(t *testing.T) {
	t.Parallel()
	srv := newOKServer(t)
	tr := &Transport{Base: http.DefaultTransport}
	tests := []struct {
		name string
		path string
	}{
		{"in scope and registered", "/zones/z1/rulesets"},
		{"in scope after prefix strip", "/client/v4/zones/z1/rulesets"},
		{"out of scope entirely", "/zones/z1/dns_records"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+tt.path, nil)
			resp, err := tr.RoundTrip(req)
			if err != nil {
				t.Fatalf("GET %s must pass: %v", tt.path, err)
			}
			resp.Body.Close()
		})
	}
}

// TestDecodeCFError verifies the error decoder against realistic error
// chains: a service-wrapped typed SDK error must surface as a *KnowledgeError
// carrying the cause and fix for its code, while unrecognized errors must
// pass through unchanged.
func TestDecodeCFError(t *testing.T) {
	t.Parallel()
	t.Run("typed SDK error decodes to KnowledgeError", func(t *testing.T) {
		t.Parallel()
		// Build the error the way the SDK really returns 4xx: a typed wrapper
		// (RequestError) around *cloudflare.Error, then service-layer wrapping.
		// A bare value-type cloudflare.Error would not match errors.As against
		// real chains (see the header facts).
		wrapped := wrapLikeServiceLayer(cloudflare.NewRequestError(&cloudflare.Error{
			StatusCode: 400, ErrorCodes: []int{20155},
		}))

		out := DecodeCFError(wrapped, "")
		var ke *KnowledgeError
		if !errors.As(out, &ke) {
			t.Fatalf("DecodeCFError must return *KnowledgeError, got %T", out)
		}
		if ke.Code != 20155 || !strings.Contains(ke.Fix, "cf.colo.id") {
			t.Fatalf("decode wrong: %+v", ke)
		}
		if !strings.Contains(out.Error(), "colocation") {
			t.Fatalf("error string must carry the cause: %v", out)
		}
	})

	t.Run("unknown error passes through unchanged", func(t *testing.T) {
		t.Parallel()
		unknown := DecodeCFError(errors.New("plain"), "")
		if unknown.Error() != "plain" {
			t.Fatalf("unknown error must pass through unchanged, got %v", unknown)
		}
	})
}

// wrapLikeServiceLayer mimics a wrapped cloudflare error the way service
// layers do.
func wrapLikeServiceLayer(err error) error {
	return fmt.Errorf("wrapped: %w", err)
}
