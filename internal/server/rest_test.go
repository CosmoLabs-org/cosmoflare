package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// fakeSource is a test ServeSource. It records the profile each CF method was
// called with so tests can assert the read-only ?profile= selection contract.
type fakeSource struct {
	zones     []map[string]any
	accounts  []map[string]any
	r2        []map[string]any
	workers   []map[string]any
	kv        []map[string]any
	zoneErr   error
	seenProf  string // last profile passed to a CF method
}

func (f *fakeSource) Accounts(_ context.Context) (any, error) { return f.accounts, nil }
func (f *fakeSource) Zones(_ context.Context, profile string) (any, error) {
	f.seenProf = profile
	return f.zones, f.zoneErr
}
func (f *fakeSource) R2Buckets(_ context.Context, profile string) (any, error) {
	f.seenProf = profile
	return f.r2, nil
}
func (f *fakeSource) Workers(_ context.Context, profile string) (any, error) {
	f.seenProf = profile
	return f.workers, nil
}
func (f *fakeSource) KV(_ context.Context, profile string) (any, error) {
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
	s, url := newSourcedServer(t, &fakeSource{zones: []map[string]any{{"name": "z"}}})
	defer func() { _ = s }()

	resp := get(t, url+"/zones", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token /zones: got %d, want 401", resp.StatusCode)
	}
}

func TestREST_ZonesReturnsData(t *testing.T) {
	src := &fakeSource{zones: []map[string]any{{"name": "example.com"}, {"name": "cosmolabs.org"}}}
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
	src := &fakeSource{accounts: []map[string]any{{"name": "work"}, {"name": "personal"}}}
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
	src := &fakeSource{zones: []map[string]any{{"name": "z"}}}
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
