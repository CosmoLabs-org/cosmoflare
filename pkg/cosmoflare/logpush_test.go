package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// logpushStubServer spins up an httptest server standing in for the
// Cloudflare API and returns a client pointed at it (pattern:
// registrar_test.go registrarStubServer).
func logpushStubServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *cloudflare.API) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	cf, err := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to build stub cloudflare client: %v", err)
	}
	return server, cf
}

// logpushEnvelope wraps a result in the standard Cloudflare response shape.
func logpushEnvelope(result any) map[string]any {
	return map[string]any{
		"success": true,
		"errors":  []any{},
		"result":  result,
	}
}

// logpushJobJSON builds a raw API job body with the fields the mapping
// layer reads back.
func logpushJobJSON(id int, name, dataset, destination, frequency string, enabled bool) map[string]any {
	return map[string]any{
		"id":              id,
		"name":            name,
		"dataset":         dataset,
		"destination_conf": destination,
		"enabled":         enabled,
		"frequency":       frequency,
	}
}

// TestLogpushService_Create verifies Create posts the job fields to the
// account jobs endpoint and maps the API's job back into the public view.
func TestLogpushService_Create(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := logpushStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/logpush/jobs"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["dataset"] != "http_requests" {
			t.Errorf("expected dataset http_requests, got %v", body["dataset"])
		}
		if body["destination_conf"] != "r2://bucket/logs?account=x" {
			t.Errorf("expected destination_conf, got %v", body["destination_conf"])
		}
		if body["name"] != "edge-logs" {
			t.Errorf("expected name edge-logs, got %v", body["name"])
		}
		if body["frequency"] != "high" {
			t.Errorf("expected frequency high, got %v", body["frequency"])
		}
		if body["enabled"] != true {
			t.Errorf("expected enabled true (new jobs start enabled), got %v", body["enabled"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(logpushEnvelope(logpushJobJSON(100237, "edge-logs", "http_requests", "r2://bucket/logs?account=x", "high", true)))
	})

	svc, err := NewLogpushService(cf, accountID)
	if err != nil {
		t.Fatalf("unexpected error building service: %v", err)
	}
	job, err := svc.Create(context.Background(), LogpushJobCreate{
		Name:            "edge-logs",
		Dataset:         "http_requests",
		DestinationConf: "r2://bucket/logs?account=x",
		Frequency:       "high",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.ID != 100237 {
		t.Errorf("expected ID 100237, got %d", job.ID)
	}
	if job.Dataset != "http_requests" || !job.Enabled || job.Frequency != "high" {
		t.Errorf("job fields not mapped: %+v", job)
	}
}

// TestLogpushService_CreateValidation verifies Create rejects an empty
// dataset or destination_conf before any request is issued.
func TestLogpushService_CreateValidation(t *testing.T) {
	t.Parallel()
	_, cf := logpushStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request expected on validation failure, got %s %s", r.Method, r.URL.Path)
	})
	svc, _ := NewLogpushService(cf, "account-test-123")

	if _, err := svc.Create(context.Background(), LogpushJobCreate{DestinationConf: "r2://b/p"}); err == nil || !strings.Contains(err.Error(), "dataset is required") {
		t.Fatalf("expected dataset error, got %v", err)
	}
	if _, err := svc.Create(context.Background(), LogpushJobCreate{Dataset: "http_requests"}); err == nil || !strings.Contains(err.Error(), "destination_conf is required") {
		t.Fatalf("expected destination_conf error, got %v", err)
	}
}

// TestLogpushService_List verifies List issues a GET to the account jobs
// endpoint and returns every mapped job.
func TestLogpushService_List(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := logpushStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/logpush/jobs"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(logpushEnvelope([]map[string]any{
			logpushJobJSON(100237, "edge-logs", "http_requests", "r2://bucket/a", "high", true),
			logpushJobJSON(100238, "fw", "firewalls", "r2://bucket/b", "", false),
		}))
	})

	svc, _ := NewLogpushService(cf, accountID)
	jobs, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
	if jobs[0].ID != 100237 || jobs[0].Dataset != "http_requests" {
		t.Errorf("first job not mapped: %+v", jobs[0])
	}
	if jobs[1].Enabled {
		t.Errorf("expected second job disabled, got %+v", jobs[1])
	}
}

// TestLogpushService_Get verifies Get fetches a single job by ID and maps
// timestamps into RFC 3339 strings.
func TestLogpushService_Get(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := logpushStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/accounts/"+accountID+"/logpush/jobs/100237"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		job := logpushJobJSON(100237, "edge-logs", "http_requests", "r2://bucket/a", "high", true)
		job["last_complete"] = "2026-09-22T10:00:00Z"
		job["error_message"] = ""
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(logpushEnvelope(job))
	})

	svc, _ := NewLogpushService(cf, accountID)
	job, err := svc.Get(context.Background(), 100237)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.ID != 100237 {
		t.Errorf("expected ID 100237, got %d", job.ID)
	}
	want := time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC).Format(time.RFC3339)
	if job.LastComplete != want {
		t.Errorf("expected last_complete %q, got %q", want, job.LastComplete)
	}
}

// TestLogpushService_GetBadID verifies Get rejects a non-positive ID
// without issuing a request.
func TestLogpushService_GetBadID(t *testing.T) {
	t.Parallel()
	_, cf := logpushStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request expected for bad ID, got %s %s", r.Method, r.URL.Path)
	})
	svc, _ := NewLogpushService(cf, "account-test-123")
	if _, err := svc.Get(context.Background(), 0); err == nil || !strings.Contains(err.Error(), "positive integer") {
		t.Fatalf("expected ID validation error, got %v", err)
	}
}

// TestLogpushService_Update verifies Update fetches the current job, merges
// only the supplied changes onto it, PUTs the merged job, and returns the
// post-update state.
func TestLogpushService_Update(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := logpushStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/"+accountID+"/logpush/jobs/100237":
			_ = json.NewEncoder(w).Encode(logpushEnvelope(logpushJobJSON(100237, "edge-logs", "http_requests", "r2://bucket/a", "high", true)))
		case r.Method == http.MethodPut && r.URL.Path == "/accounts/"+accountID+"/logpush/jobs/100237":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode PUT body: %v", err)
			}
			// merged: new name + disable, everything else kept from GET
			if body["name"] != "edge-logs-v2" {
				t.Errorf("expected merged name edge-logs-v2, got %v", body["name"])
			}
			if body["enabled"] != false {
				t.Errorf("expected merged enabled=false, got %v", body["enabled"])
			}
			if body["dataset"] != "http_requests" {
				t.Errorf("expected dataset kept from current job, got %v", body["dataset"])
			}
			if body["destination_conf"] != "r2://bucket/a" {
				t.Errorf("expected destination kept from current job, got %v", body["destination_conf"])
			}
			_ = json.NewEncoder(w).Encode(logpushEnvelope(logpushJobJSON(100237, "edge-logs-v2", "http_requests", "r2://bucket/a", "high", false)))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	svc, _ := NewLogpushService(cf, accountID)
	disabled := false
	job, err := svc.Update(context.Background(), 100237, LogpushJobUpdate{
		Name:    "edge-logs-v2",
		Enabled: &disabled,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.Name != "edge-logs-v2" || job.Enabled {
		t.Errorf("expected renamed disabled job, got %+v", job)
	}
}

// TestLogpushService_Delete verifies Delete issues a DELETE against the
// job endpoint.
func TestLogpushService_Delete(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := logpushStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/logpush/jobs/100237"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(logpushEnvelope(map[string]any{"id": 100237}))
	})

	svc, _ := NewLogpushService(cf, accountID)
	if err := svc.Delete(context.Background(), 100237); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLogpushService_OwnershipChallenge verifies the ownership-challenge
// request posts the destination_conf and surfaces the challenge filename.
func TestLogpushService_OwnershipChallenge(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := logpushStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/logpush/ownership"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		if body["destination_conf"] != "r2://bucket/logs?account=x" {
			t.Errorf("expected destination_conf, got %v", body["destination_conf"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(logpushEnvelope(map[string]any{
			"filename": "challenge-abc123.txt",
			"valid":    true,
			"message":  "",
		}))
	})

	svc, _ := NewLogpushService(cf, accountID)
	challenge, err := svc.OwnershipChallenge(context.Background(), "r2://bucket/logs?account=x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if challenge.Filename != "challenge-abc123.txt" || !challenge.Valid {
		t.Errorf("challenge not mapped: %+v", challenge)
	}
}

// TestLogpushService_OwnershipValidate verifies the file-based check posts
// both the destination and the uploaded challenge contents, and returns the
// API's verdict.
func TestLogpushService_OwnershipValidate(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := logpushStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/accounts/"+accountID+"/logpush/ownership/validate"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		if body["destination_conf"] != "r2://bucket/logs?account=x" {
			t.Errorf("expected destination_conf, got %v", body["destination_conf"])
		}
		if body["ownership_challenge"] != "challenge-file-contents" {
			t.Errorf("expected ownership_challenge, got %v", body["ownership_challenge"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(logpushEnvelope(map[string]any{"valid": true}))
	})

	svc, _ := NewLogpushService(cf, accountID)
	valid, err := svc.OwnershipValidate(context.Background(), "r2://bucket/logs?account=x", "challenge-file-contents")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("expected valid=true")
	}
}

// TestLogpushService_MalformedEnvelope verifies a non-JSON response body
// surfaces as an error rather than a zero-value job, on both the job list
// and the ownership challenge paths.
func TestLogpushService_MalformedEnvelope(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"

	_, cf := logpushStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "this is not json")
	})

	svc, _ := NewLogpushService(cf, accountID)

	if _, err := svc.List(context.Background()); err == nil {
		t.Error("expected error listing against malformed envelope")
	}
	if _, err := svc.Get(context.Background(), 100237); err == nil {
		t.Error("expected error getting against malformed envelope")
	}
	if _, err := svc.OwnershipChallenge(context.Background(), "r2://bucket/logs?account=x"); err == nil {
		t.Error("expected error on ownership challenge against malformed envelope")
	}
}

// TestLogpushService_EmptyAccountID verifies every method guards against a
// missing account ID before issuing a request.
func TestLogpushService_EmptyAccountID(t *testing.T) {
	t.Parallel()
	_, cf := logpushStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request expected with empty account ID, got %s %s", r.Method, r.URL.Path)
	})
	// The constructor rejects an empty account ID, so exercise the method
	// guards by building the service struct directly (in-package).
	svc := &LogpushService{cf: cf, accountID: ""}

	if _, err := svc.Create(context.Background(), LogpushJobCreate{Dataset: "http_requests", DestinationConf: "r2://b/p"}); err == nil {
		t.Error("expected error on Create with empty account ID")
	}
	if _, err := svc.List(context.Background()); err == nil {
		t.Error("expected error on List with empty account ID")
	}
	if _, err := svc.Get(context.Background(), 1); err == nil {
		t.Error("expected error on Get with empty account ID")
	}
	if _, err := svc.Update(context.Background(), 1, LogpushJobUpdate{Name: "x"}); err == nil {
		t.Error("expected error on Update with empty account ID")
	}
	if err := svc.Delete(context.Background(), 1); err == nil {
		t.Error("expected error on Delete with empty account ID")
	}
	if _, err := svc.OwnershipChallenge(context.Background(), "r2://b/p"); err == nil {
		t.Error("expected error on OwnershipChallenge with empty account ID")
	}
	if _, err := svc.OwnershipValidate(context.Background(), "r2://b/p", "c"); err == nil {
		t.Error("expected error on OwnershipValidate with empty account ID")
	}
}

// TestLogpushService_FromCredsValidation verifies the credentials
// constructor rejects missing account ID or token offline.
func TestLogpushService_FromCredsValidation(t *testing.T) {
	t.Parallel()
	if _, err := NewLogpushServiceFromCreds("", "token"); err == nil {
		t.Error("expected error for missing account ID")
	}
	if _, err := NewLogpushServiceFromCreds("acct", ""); err == nil {
		t.Error("expected error for missing API token")
	}
}
