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

func TestTransportBlocksUnregisteredInScopeRoute(t *testing.T) {
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

func TestTransportPassesRegisteredRouteAndOutOfScope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := &Transport{Base: http.DefaultTransport}
	for _, path := range []string{
		"/zones/z1/rulesets",           // in scope, registered
		"/client/v4/zones/z1/rulesets", // in scope after prefix strip, registered
		"/zones/z1/dns_records",        // out of scope entirely
	} {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+path, nil)
		resp, err := tr.RoundTrip(req)
		if err != nil {
			t.Fatalf("GET %s must pass: %v", path, err)
		}
		resp.Body.Close()
	}
}

func TestDecodeCFError(t *testing.T) {
	// Build the error the way the SDK really returns 4xx: a typed wrapper
	// (RequestError) around *cloudflare.Error, then service-layer wrapping.
	// A bare value-type cloudflare.Error would not match errors.As against
	// real chains (see the header facts).
	wrapped := fmt_wrap(cloudflare.NewRequestError(&cloudflare.Error{
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

	unknown := DecodeCFError(errors.New("plain"), "")
	if unknown.Error() != "plain" {
		t.Fatalf("unknown error must pass through unchanged, got %v", unknown)
	}
}

// fmt_wrap mimics a wrapped cloudflare error the way service layers do.
func fmt_wrap(err error) error {
	return fmt.Errorf("wrapped: %w", err)
}
