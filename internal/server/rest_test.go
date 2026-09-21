package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// fakeSource is a test ServeSource. It records the profile each CF method was
// called with so tests can assert the read-only ?profile= selection contract.
type fakeSource struct {
	zones     []*cosmoflare.Zone
	accounts  []AccountProfile
	r2        []*cosmoflare.Bucket
	workers   []*cosmoflare.Worker
	kv        []*cosmoflare.KVNamespace
	zoneErr   error
	seenProf  string // last profile passed to a CF method
}

func (f *fakeSource) Accounts(_ context.Context) ([]AccountProfile, error) {
	return f.accounts, nil
}
func (f *fakeSource) CurrentProfileName() string { return "" }
func (f *fakeSource) Zones(_ context.Context, profile string) ([]*cosmoflare.Zone, error) {
	f.seenProf = profile
	return f.zones, f.zoneErr
}
func (f *fakeSource) R2Buckets(_ context.Context, profile string) ([]*cosmoflare.Bucket, error) {
	f.seenProf = profile
	return f.r2, nil
}
func (f *fakeSource) Workers(_ context.Context, profile string) ([]*cosmoflare.Worker, error) {
	f.seenProf = profile
	return f.workers, nil
}
func (f *fakeSource) KV(_ context.Context, profile string) ([]*cosmoflare.KVNamespace, error) {
	f.seenProf = profile
	return f.kv, nil
}

func newSourcedServer(t *testing.T, src ServeSource) (*Server, string) {
	t.Helper()
	s := New(Config{Token: "t", Version: "test"})
	s.SetData(src)
	return s, s.testServer(t)
}

// get is a small helper for token-authed GETs.
func get(t *testing.T, url, token string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp
}

func TestREST_RequiresToken(t *testing.T) {
	s, url := newSourcedServer(t, &fakeSource{zones: []*cosmoflare.Zone{{Name: "z"}}})
	defer func() { _ = s }()

	resp := get(t, url+"/zones", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token /zones: got %d, want 401", resp.StatusCode)
	}
}

func TestREST_ZonesReturnsData(t *testing.T) {
	src := &fakeSource{zones: []*cosmoflare.Zone{{Name: "example.com"}, {Name: "cosmolabs.org"}}}
	_, url := newSourcedServer(t, src)

	resp := get(t, url+"/zones", "t")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
	var out []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 2 || out[0]["name"] != "example.com" {
		t.Fatalf("unexpected body: %v", out)
	}
}

func TestREST_AccountsIsLocal(t *testing.T) {
	src := &fakeSource{accounts: []AccountProfile{{Name: "work"}, {Name: "personal"}}}
	s, url := newSourcedServer(t, src)

	resp := get(t, url+"/accounts", "t")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}

	// /accounts is a local read, NOT a CF call — it must not flip cloudflare_online.
	s.SetCloudflareOnline(true) // pretend a prior CF call succeeded
	get2 := get(t, url+"/accounts", "t")
	get2.Body.Close()
	if !s.CloudflareOnline() {
		t.Fatalf("/accounts flipped cloudflare_online to false; it must stay local-only")
	}
}

func TestREST_ProfileQueryParam(t *testing.T) {
	src := &fakeSource{zones: []*cosmoflare.Zone{{Name: "z"}}}
	_, url := newSourcedServer(t, src)

	resp := get(t, url+"/zones?profile=work", "t")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if src.seenProf != "work" {
		t.Fatalf("source saw profile %q, want work", src.seenProf)
	}
}

func TestREST_CFErrorSetsOfflineAnd502(t *testing.T) {
	src := &fakeSource{
		zones:   nil,
		zoneErr: fmt.Errorf("invalid api token"),
	}
	s, url := newSourcedServer(t, src)
	s.SetCloudflareOnline(true) // pretend it was online

	resp := get(t, url+"/zones", "t")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status %d, want 502", resp.StatusCode)
	}
	if s.CloudflareOnline() {
		t.Fatalf("CF error did not flip cloudflare_online to false")
	}
	// Error body should mention the failure for agent-friendly diagnostics.
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if msg, _ := body["error"].(string); !strings.Contains(msg, "invalid api token") {
		t.Fatalf("error body = %v, want message containing the cause", body)
	}
}

func TestREST_NoSourceIs503(t *testing.T) {
	s := New(Config{Token: "t", Version: "test"})
	url := s.testServer(t)

	resp := get(t, url+"/zones", "t")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("no source /zones: got %d, want 503", resp.StatusCode)
	}
}

// TestMapError_Classes pins the mapError classification contract for every
// error class TASK-012 touches: typed constructors, and plain R2Errors with
// and without an HTTP status. These expectations define the contract that
// the unified HTTP-status seam must preserve (or deliberately improve).
func TestMapError_Classes(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "auth",
			err:        &cosmoflare.R2AuthError{R2Error: cosmoflare.R2Error{Op: "op", Message: "invalid api token"}},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:       "access denied",
			err:        &cosmoflare.R2AccessDeniedError{R2Error: cosmoflare.R2Error{Op: "op", Bucket: "b", Message: "no permission"}},
			wantStatus: http.StatusForbidden,
			wantCode:   "forbidden",
		},
		{
			name:       "quota",
			err:        &cosmoflare.R2QuotaError{R2Error: cosmoflare.R2Error{Op: "op", Message: "rate limit exceeded"}},
			wantStatus: http.StatusTooManyRequests,
			wantCode:   "rate_limited",
		},
		{
			name:       "not found",
			err:        &cosmoflare.R2NotFoundError{R2Error: cosmoflare.R2Error{Op: "op", Message: "resource not found"}},
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
		// TASK-012: plain R2Errors carrying a classifiable HTTP status now
		// map through the shared ErrorStatus seam instead of a blanket 502.
		{
			name:       "plain R2Error with 401 status",
			err:        &cosmoflare.R2Error{Op: "op", Message: "m", Status: http.StatusUnauthorized},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:       "plain R2Error with 403 status",
			err:        &cosmoflare.R2Error{Op: "op", Message: "m", Status: http.StatusForbidden},
			wantStatus: http.StatusForbidden,
			wantCode:   "forbidden",
		},
		{
			name:       "plain R2Error with 429 status",
			err:        &cosmoflare.R2Error{Op: "op", Message: "m", Status: http.StatusTooManyRequests},
			wantStatus: http.StatusTooManyRequests,
			wantCode:   "rate_limited",
		},
		// A 5xx status stays a 502 upstream_error — the daemon's transient
		// failure semantics.
		{
			name:       "plain R2Error with 500 status",
			err:        &cosmoflare.R2Error{Op: "op", Message: "m", Status: http.StatusInternalServerError},
			wantStatus: http.StatusBadGateway,
			wantCode:   "upstream_error",
		},
		{
			name:       "plain R2Error without status",
			err:        &cosmoflare.R2Error{Op: "op", Message: "m"},
			wantStatus: http.StatusBadGateway,
			wantCode:   "upstream_error",
		},
		{
			name:       "plain error",
			err:        errors.New("connection reset"),
			wantStatus: http.StatusBadGateway,
			wantCode:   "upstream_error",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, code := mapError(tc.err)
			if status != tc.wantStatus || code != tc.wantCode {
				t.Fatalf("mapError(%v) = %d/%q, want %d/%q", tc.err, status, code, tc.wantStatus, tc.wantCode)
			}
		})
	}
}
