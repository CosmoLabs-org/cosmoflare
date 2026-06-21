package server

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// TestHealthz_RequiresToken verifies the token-auth middleware: a request
// without the Bearer token gets 401, a request with the right token gets 200
// and the healthz JSON body.
func TestHealthz_RequiresToken(t *testing.T) {
	s := New(Config{Token: "secret", Version: "test"})
	url := s.testServer(t)

	// No token -> 401.
	resp, err := http.Get(url + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: got %d, want 401", resp.StatusCode)
	}

	// Wrong token -> 401.
	req, _ := http.NewRequest(http.MethodGet, url+"/healthz", nil)
	req.Header.Set("Authorization", "Bearer nope")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /healthz (wrong token): %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong token: got %d, want 401", resp.StatusCode)
	}

	// Correct token -> 200 + systems_online true.
	req, _ = http.NewRequest(http.MethodGet, url+"/healthz", nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /healthz (token): %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("with token: status %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode body: %v (body=%s)", err, body)
	}
	if out["systems_online"] != true {
		t.Fatalf("systems_online: got %v, want true", out["systems_online"])
	}
	if out["version"] != "test" {
		t.Fatalf("version: got %v, want test", out["version"])
	}
}

// TestHealthz_QueryTokenFallback verifies the ?token= auth fallback used by the
// browser EventSource API (which cannot set the Authorization header).
func TestHealthz_QueryTokenFallback(t *testing.T) {
	s := New(Config{Token: "secret", Version: "test"})
	url := s.testServer(t)

	// Correct token in query string -> 200.
	resp, err := http.Get(url + "/healthz?token=secret")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("query token: got %d, want 200", resp.StatusCode)
	}

	// Wrong query token -> 401.
	resp, err = http.Get(url + "/healthz?token=nope")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong query token: got %d, want 401", resp.StatusCode)
	}
}
func TestHealthz_ReflectsCloudflareOnline(t *testing.T) {
	s := New(Config{Token: "t", Version: "test"})
	url := s.testServer(t)

	if got := healthzCloudflare(t, url, "t"); got != false {
		t.Fatalf("initial cloudflare_online = %v, want false", got)
	}
	s.SetCloudflareOnline(true)
	if got := healthzCloudflare(t, url, "t"); got != true {
		t.Fatalf("after online cloudflare_online = %v, want true", got)
	}
	s.SetCloudflareOnline(false)
	if got := healthzCloudflare(t, url, "t"); got != false {
		t.Fatalf("after offline cloudflare_online = %v, want false", got)
	}
}
