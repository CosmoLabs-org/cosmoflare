package cosmoflare

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

// newTurnstileTestService wires a TurnstileService at accountID to an
// httptest server whose handler is closed over by the caller's test.
func newTurnstileTestService(t *testing.T, accountID string, handler http.HandlerFunc) *TurnstileService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	cf, err := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to build client: %v", err)
	}
	svc, err := NewTurnstileService(cf, accountID)
	if err != nil {
		t.Fatalf("failed to build service: %v", err)
	}
	return svc
}

// TestTurnstileService_Create verifies Create issues a POST to the
// account's challenges/widgets endpoint with the name and hostnames in the
// body, and that the secret key from the response is surfaced exactly once.
func TestTurnstileService_Create(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	svc := newTurnstileTestService(t, accountID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/challenges/widgets"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if !strings.Contains(string(body), `"name":"signup-widget"`) {
			t.Errorf("expected name in body, got %s", body)
		}
		if !strings.Contains(string(body), `"example.com"`) || !strings.Contains(string(body), `"staging.example.com"`) {
			t.Errorf("expected both hostnames in body, got %s", body)
		}
		writeCloudflareResult(t, w, map[string]any{
			"sitekey": "0x4AAA-sitekey",
			"secret":  "0x4AAA-secret",
			"name":    "signup-widget",
			"domains": []string{"example.com", "staging.example.com"},
			"mode":    "managed",
		})
	})

	widget, err := svc.Create(context.Background(),
		WithTurnstileName("signup-widget"),
		WithTurnstileHostnames([]string{"example.com", "staging.example.com"}),
		WithTurnstileMode("managed"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if widget.SiteKey != "0x4AAA-sitekey" {
		t.Errorf("expected sitekey mapped, got %+v", widget)
	}
	if widget.Secret != "0x4AAA-secret" {
		t.Errorf("expected secret surfaced on create (shown exactly once), got %q", widget.Secret)
	}
	if len(widget.Domains) != 2 {
		t.Errorf("expected 2 domains, got %+v", widget.Domains)
	}
}

// TestTurnstileService_ListStripsSecret verifies List never carries the
// secret key even if the API echoes one.
func TestTurnstileService_ListStripsSecret(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	svc := newTurnstileTestService(t, accountID, func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/accounts/"+accountID+"/challenges/widgets"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		writeCloudflareResult(t, w, []map[string]any{
			{"sitekey": "0x4AAA-sitekey", "secret": "0x4AAA-secret", "name": "signup-widget"},
		})
	})

	widgets, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(widgets) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(widgets))
	}
	if widgets[0].Secret != "" {
		t.Errorf("list must never repeat the secret key, got %q", widgets[0].Secret)
	}
}

// TestTurnstileService_GetStripsSecret verifies Get never carries the
// secret key even if the API echoes one.
func TestTurnstileService_GetStripsSecret(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"
	const siteKey = "0x4AAA-sitekey"

	svc := newTurnstileTestService(t, accountID, func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/accounts/"+accountID+"/challenges/widgets/"+siteKey; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		writeCloudflareResult(t, w, map[string]any{
			"sitekey": siteKey,
			"secret":  "0x4AAA-secret",
			"name":    "signup-widget",
		})
	})

	widget, err := svc.Get(context.Background(), siteKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if widget.Secret != "" {
		t.Errorf("get must never repeat the secret key, got %q", widget.Secret)
	}
	if widget.Name != "signup-widget" {
		t.Errorf("expected name mapped, got %+v", widget)
	}
}

// TestTurnstileService_Update verifies Update PUTs the merged config
// (cloudflare-go transport) carrying the changed fields.
func TestTurnstileService_Update(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"
	const siteKey = "0x4AAA-sitekey"

	svc := newTurnstileTestService(t, accountID, func(w http.ResponseWriter, r *http.Request) {
		// cloudflare-go v0.116.0 UpdateTurnstileWidget issues PUT with the
		// merged config; assert the transport the SDK actually ships.
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/challenges/widgets/"+siteKey; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if !strings.Contains(string(body), `"name":"renamed-widget"`) {
			t.Errorf("expected renamed name in patch body, got %s", body)
		}
		if strings.Contains(string(body), `"domains"`) {
			t.Errorf("unchanged domains must not be sent, got %s", body)
		}
		// Re-fetch semantics: the response after a write reflects the
		// WRITTEN state.
		writeCloudflareResult(t, w, map[string]any{
			"sitekey": siteKey,
			"name":    "renamed-widget",
			"domains": []string{"example.com"},
		})
	})

	widget, err := svc.Update(context.Background(), siteKey, WithTurnstileName("renamed-widget"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if widget.Name != "renamed-widget" {
		t.Errorf("expected written state back, got %+v", widget)
	}
	if widget.Secret != "" {
		t.Errorf("update must never repeat the secret key, got %q", widget.Secret)
	}
}

// TestTurnstileService_Delete verifies Delete issues a DELETE to the
// widget's endpoint.
func TestTurnstileService_Delete(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"
	const siteKey = "0x4AAA-sitekey"

	svc := newTurnstileTestService(t, accountID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/challenges/widgets/"+siteKey; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		// The SDK decodes the response envelope; an empty 200 body errors.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "errors": []any{}, "result": nil})
	})

	if err := svc.Delete(context.Background(), siteKey); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestTurnstileService_Validation verifies constructor and method argument
// guards fail before any request is issued.
func TestTurnstileService_Validation(t *testing.T) {
	t.Parallel()

	if _, err := NewTurnstileService(nil, "acct-1"); err == nil {
		t.Error("expected error for nil API client")
	}
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewTurnstileService(cf, "acct-1")
	if err != nil {
		t.Fatalf("NewTurnstileService: %v", err)
	}
	ctx := context.Background()

	if _, err := svc.Create(ctx); err == nil {
		t.Error("expected error for missing name and hostnames on Create")
	}
	if _, err := svc.Create(ctx, WithTurnstileName("w")); err == nil {
		t.Error("expected error for missing hostnames on Create")
	}
	if _, err := svc.Create(ctx, WithTurnstileHostnames([]string{"example.com"})); err == nil {
		t.Error("expected error for missing name on Create")
	}
	if _, err := svc.Get(ctx, ""); err == nil {
		t.Error("expected error for empty site key on Get")
	}
	if _, err := svc.Update(ctx, "0x4AAA-sitekey"); err == nil {
		t.Error("expected error for no update options on Update")
	}
	if _, err := svc.Update(ctx, "", WithTurnstileName("w")); err == nil {
		t.Error("expected error for empty site key on Update")
	}
	if err := svc.Delete(ctx, ""); err == nil {
		t.Error("expected error for empty site key on Delete")
	}
	if _, err := NewTurnstileServiceFromCreds("acct-1", ""); err == nil {
		t.Error("expected error for empty API token")
	}
}
