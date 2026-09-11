package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newBucketPolicyTestServer(t *testing.T, handler http.HandlerFunc) *BucketPolicyService {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewBucketPolicyService("ACC", "tok-secret", WithBucketPolicyBaseURL(srv.URL))
}

func TestBucketPolicyGetValidation(t *testing.T) {
	svc := NewBucketPolicyService("ACC", "tok-secret")
	_, err := svc.GetBucketPolicy(context.Background(), "")
	if err == nil {
		t.Error("expected error when bucket name is empty")
	}
}

func TestBucketPolicyGetSuccess(t *testing.T) {
	policy := `{"Version":"2012-10-17","Statement":[]}`
	svc := newBucketPolicyTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/accounts/ACC/r2/buckets/my-bucket/policy") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tok-secret" {
			t.Errorf("expected bearer auth header, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  json.RawMessage(policy),
		})
	})

	result, err := svc.GetBucketPolicy(context.Background(), "my-bucket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(result), `"Version":"2012-10-17"`) {
		t.Errorf("expected policy document in result, got: %s", result)
	}
}

func TestBucketPolicySetValidation(t *testing.T) {
	svc := NewBucketPolicyService("ACC", "tok-secret")

	t.Run("empty bucket", func(t *testing.T) {
		err := svc.SetBucketPolicy(context.Background(), "", json.RawMessage(`{}`))
		if err == nil {
			t.Error("expected error when bucket name is empty")
		}
	})

	t.Run("empty policy", func(t *testing.T) {
		err := svc.SetBucketPolicy(context.Background(), "my-bucket", nil)
		if err == nil {
			t.Error("expected error when policy is empty")
		}
	})

	t.Run("malformed JSON never reaches the API", func(t *testing.T) {
		called := false
		svc := newBucketPolicyTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
		err := svc.SetBucketPolicy(context.Background(), "my-bucket", json.RawMessage(`{not valid json`))
		if err == nil {
			t.Fatal("expected validation error for malformed JSON")
		}
		if called {
			t.Error("expected no API call for malformed JSON")
		}
		var verr *R2ValidationError
		if !isValidationError(err, &verr) {
			t.Errorf("expected R2ValidationError, got %T: %v", err, err)
		}
	})
}

func TestBucketPolicySetSuccess(t *testing.T) {
	var gotMethod, gotPath string
	svc := newBucketPolicyTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": nil})
	})

	err := svc.SetBucketPolicy(context.Background(), "my-bucket", json.RawMessage(`{"Version":"2012-10-17","Statement":[]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("expected PUT, got %s", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/accounts/ACC/r2/buckets/my-bucket/policy") {
		t.Errorf("unexpected path: %s", gotPath)
	}
}

// isValidationError is a small helper so the malformed-JSON test can assert
// the concrete error type without importing errors.As boilerplate inline.
func isValidationError(err error, target **R2ValidationError) bool {
	verr, ok := err.(*R2ValidationError)
	if ok {
		*target = verr
	}
	return ok
}
