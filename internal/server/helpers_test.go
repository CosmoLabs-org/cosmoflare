package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testServer starts an httptest server backed by s.Handler() and registers a
// cleanup. Returns the base URL for clients to use.
func (s *Server) testServer(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return srv.URL
}

// healthzCloudflare fetches /healthz with the given token and returns the
// reported cloudflare_online value.
func healthzCloudflare(t *testing.T, url, token string) bool {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url+"/healthz", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode healthz: %v", err)
	}
	b, _ := out["cloudflare_online"].(bool)
	return b
}
