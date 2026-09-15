package cosmoflare

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// workerVersionsMockSetup builds a credentials-based WorkerService whose raw
// REST base URL points at a throwaway test server. Every versions test uses
// it so no call can reach the real Cloudflare API.
func workerVersionsMockSetup(t *testing.T, handler http.HandlerFunc) (*WorkerService, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	svc, err := NewWorkerServiceFromCreds("acct-ver-123", "test-token")
	if err != nil {
		t.Fatalf("NewWorkerServiceFromCreds: %v", err)
	}
	svc.apiBaseURL = server.URL
	return svc, server
}

// workerVersionsNoTokenService returns a service built without credentials —
// its apiToken is zero, so every raw call must fail validation.
func workerVersionsNoTokenService(t *testing.T) *WorkerService {
	t.Helper()
	cf, err := cloudflare.NewWithAPIToken("test-token", cloudflare.HTTPClient(controlPlaneClient()))
	if err != nil {
		t.Fatalf("cloudflare.NewWithAPIToken: %v", err)
	}
	svc, err := NewWorkerService(cf, "acct-ver-123")
	if err != nil {
		t.Fatalf("NewWorkerService: %v", err)
	}
	return svc
}

// ---------------------------------------------------------------------------
// Happy paths
// ---------------------------------------------------------------------------

func TestVersionUpload(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/accounts/acct-ver-123/workers/scripts/my-worker/versions" {
			t.Errorf("unexpected path %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("expected bearer auth, got %q", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		if body["script"] != "export default {}" {
			t.Errorf("expected script body, got %v", body["script"])
		}
		if body["compatibility_date"] != "2024-09-01" {
			t.Errorf("expected compatibility_date, got %v", body["compatibility_date"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result": map[string]any{
				"id":         "ver-upload-1",
				"number":     3,
				"created_at": "2026-09-15T10:00:00Z",
			},
		})
	})

	v, err := svc.VersionUpload(context.Background(), "my-worker",
		strings.NewReader("export default {}"),
		WithWorkerCompatibilityDate("2024-09-01"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID != "ver-upload-1" || v.Number != 3 {
		t.Errorf("got %+v", v)
	}
	if v.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be decoded")
	}
}

func TestVersionList(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result": []map[string]any{
				{"id": "ver-1", "number": 1, "created_at": "2026-09-14T10:00:00Z"},
				{"id": "ver-2", "number": 2, "created_at": "2026-09-15T10:00:00Z"},
			},
		})
	})

	versions, err := svc.VersionList(context.Background(), "my-worker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if versions[1].ID != "ver-2" || versions[1].Number != 2 {
		t.Errorf("got %+v", versions[1])
	}
}

func TestVersionListEmpty(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": []any{}})
	})

	versions, err := svc.VersionList(context.Background(), "my-worker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if versions == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(versions) != 0 {
		t.Errorf("expected 0 versions, got %d", len(versions))
	}
}

func TestVersionGet(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/accounts/acct-ver-123/workers/scripts/my-worker/versions/ver-42" {
			t.Errorf("unexpected path %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result":  map[string]any{"id": "ver-42", "number": 42, "source": "api"},
		})
	})

	v, err := svc.VersionGet(context.Background(), "my-worker", "ver-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID != "ver-42" || v.Source != "api" {
		t.Errorf("got %+v", v)
	}
}

func TestVersionDeploy(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/accounts/acct-ver-123/workers/scripts/my-worker/deployments" {
			t.Errorf("unexpected path %q", got)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body["version_id"] != "ver-7" {
			t.Errorf("expected version_id=ver-7, got %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result": map[string]any{
				"id":           "dep-1",
				"version_id":   "ver-7",
				"created_at":   "2026-09-15T11:00:00Z",
				"author_email": "dev@example.com",
			},
		})
	})

	d, err := svc.VersionDeploy(context.Background(), "my-worker", "ver-7")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.ID != "dep-1" || d.VersionID != "ver-7" || d.AuthorEmail != "dev@example.com" {
		t.Errorf("got %+v", d)
	}
}

func TestVersionDelete(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/accounts/acct-ver-123/workers/scripts/my-worker/versions/ver-9" {
			t.Errorf("unexpected path %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": nil})
	})

	if err := svc.VersionDelete(context.Background(), "my-worker", "ver-9"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVersionRollback(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT (rollback redeploys), got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result":  map[string]any{"id": "dep-2", "version_id": "ver-3", "source": "rollback"},
		})
	})

	d, err := svc.VersionRollback(context.Background(), "my-worker", "ver-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.VersionID != "ver-3" || d.Source != "rollback" {
		t.Errorf("got %+v", d)
	}
}

// ---------------------------------------------------------------------------
// No-token guard
// ---------------------------------------------------------------------------

func TestRawRequestRequiresToken(t *testing.T) {
	svc := workerVersionsNoTokenService(t)

	_, err := svc.VersionList(context.Background(), "my-worker")
	if err == nil {
		t.Fatal("expected validation error without API token")
	}
	var verr *R2ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *R2ValidationError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "NewWorkerServiceFromCreds") {
		t.Errorf("error should point at credentials constructor: %v", err)
	}
}

func TestRawRequestRequiresTokenAllMethods(t *testing.T) {
	svc := workerVersionsNoTokenService(t)
	ctx := context.Background()

	if _, err := svc.VersionUpload(ctx, "w", strings.NewReader("x")); err == nil {
		t.Error("VersionUpload: expected error without token")
	}
	if _, err := svc.VersionGet(ctx, "w", "v"); err == nil {
		t.Error("VersionGet: expected error without token")
	}
	if _, err := svc.VersionDeploy(ctx, "w", "v"); err == nil {
		t.Error("VersionDeploy: expected error without token")
	}
	if err := svc.VersionDelete(ctx, "w", "v"); err == nil {
		t.Error("VersionDelete: expected error without token")
	}
	if _, err := svc.VersionRollback(ctx, "w", "v"); err == nil {
		t.Error("VersionRollback: expected error without token")
	}
}

// ---------------------------------------------------------------------------
// Error envelopes
// ---------------------------------------------------------------------------

func TestRawRequestNon2xxTypedError(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"errors":  []map[string]any{{"code": 1000, "message": "workers version not found"}},
		})
	})

	_, err := svc.VersionGet(context.Background(), "my-worker", "ver-404")
	if err == nil {
		t.Fatal("expected error on 404")
	}
	var r2err *R2Error
	if !errors.As(err, &r2err) {
		t.Fatalf("expected *R2Error, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "workers version not found") {
		t.Errorf("error should carry envelope message: %v", err)
	}
}

func TestRawRequestNon2xxUnparseableBody(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<html>gateway exploded</html>"))
	})

	if _, err := svc.VersionList(context.Background(), "my-worker"); err == nil {
		t.Fatal("expected error on 500")
	} else if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status code: %v", err)
	}
}

func TestRawRequestEnvelopeSuccessFalse(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		// 200 OK but the CF envelope reports failure.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"errors":  []map[string]any{{"code": 10035, "message": "version is currently deployed"}},
		})
	})

	err := svc.VersionDelete(context.Background(), "my-worker", "ver-live")
	if err == nil {
		t.Fatal("expected error on success:false envelope")
	}
	var r2err *R2Error
	if !errors.As(err, &r2err) {
		t.Fatalf("expected *R2Error, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "version is currently deployed") {
		t.Errorf("error should carry envelope message: %v", err)
	}
}

func TestRawRequestResultDecodeFailure(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result":  map[string]any{"number": "not-an-int"},
		})
	})

	if _, err := svc.VersionGet(context.Background(), "my-worker", "ver-1"); err == nil {
		t.Fatal("expected decode error for malformed result")
	}
}

// ---------------------------------------------------------------------------
// Argument validation
// ---------------------------------------------------------------------------

func TestVersionMethodsValidateArguments(t *testing.T) {
	svc, _ := workerVersionsMockSetup(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be reached during validation failures")
	})
	ctx := context.Background()

	if _, err := svc.VersionUpload(ctx, "", strings.NewReader("x")); err == nil {
		t.Error("VersionUpload: expected error for empty worker")
	}
	if _, err := svc.VersionUpload(ctx, "w", nil); err == nil {
		t.Error("VersionUpload: expected error for nil script")
	}
	if _, err := svc.VersionList(ctx, ""); err == nil {
		t.Error("VersionList: expected error for empty worker")
	}
	if _, err := svc.VersionGet(ctx, "w", ""); err == nil {
		t.Error("VersionGet: expected error for empty version ID")
	}
	if _, err := svc.VersionDeploy(ctx, "", "v"); err == nil {
		t.Error("VersionDeploy: expected error for empty worker")
	}
	if err := svc.VersionDelete(ctx, "w", ""); err == nil {
		t.Error("VersionDelete: expected error for empty version ID")
	}
}

// ---------------------------------------------------------------------------
// Constructor wiring
// ---------------------------------------------------------------------------

func TestNewWorkerServiceFromCredsSetsRawAPIFields(t *testing.T) {
	svc, err := NewWorkerServiceFromCreds("acct-1", "tok-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.apiToken != "tok-1" {
		t.Errorf("apiToken = %q, want tok-1", svc.apiToken)
	}
	if svc.apiBaseURL != "https://api.cloudflare.com/client/v4" {
		t.Errorf("apiBaseURL = %q", svc.apiBaseURL)
	}
}

func TestNewWorkerServiceLeavesRawAPIFieldsZero(t *testing.T) {
	svc := workerVersionsNoTokenService(t)
	if svc.apiToken != "" || svc.apiBaseURL != "" {
		t.Errorf("expected zero raw-API fields, got token=%q base=%q", svc.apiToken, svc.apiBaseURL)
	}
}

func TestWorkerVersionTypesJSON(t *testing.T) {
	v := WorkerVersion{ID: "v1", Number: 2, CreatedAt: timeNowUTC(), Source: "api"}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"id":"v1"`, `"number":2`, `"created_at"`, `"source":"api"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("expected %s in %s", want, b)
		}
	}

	d := WorkerDeployment{ID: "d1", CreatedAt: timeNowUTC(), VersionID: "v1", AuthorEmail: "a@b.c"}
	b, err = json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"version_id":"v1"`, `"author_email":"a@b.c"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("expected %s in %s", want, b)
		}
	}
}

// timeNowUTC is a stable timestamp source for JSON round-trip assertions.
func timeNowUTC() time.Time { return time.Now().UTC() }
