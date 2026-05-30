package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

func TestNewVectorizeServiceValidation(t *testing.T) {
	_, err := NewVectorizeService(nil, "account123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewVectorizeService(cf, "")
	if err == nil {
		t.Error("expected error when account ID is empty")
	}
}

func TestNewVectorizeServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewVectorizeService(cf, "account123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewVectorizeServiceFromCredsValidation(t *testing.T) {
	_, err := NewVectorizeServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when account ID is empty")
	}
	_, err = NewVectorizeServiceFromCreds("acct", "")
	if err == nil {
		t.Error("expected error when token is empty")
	}
}

func TestNewVectorizeServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewVectorizeServiceFromCreds("acct123", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
}

func vectorizeMockSetup(handler http.HandlerFunc) (*VectorizeService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewVectorizeService(cf, "acct-vectorize-123")
	return svc, server
}

func TestVectorizeCreateIndexValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewVectorizeService(cf, "account123")

	_, err := svc.CreateIndex(context.Background(), "", 768, "cosine")
	if err == nil {
		t.Error("expected error when name is empty")
	}

	_, err = svc.CreateIndex(context.Background(), "test", 0, "cosine")
	if err == nil {
		t.Error("expected error when dimensions is 0")
	}

	_, err = svc.CreateIndex(context.Background(), "test", 768, "invalid")
	if err == nil {
		t.Error("expected error when metric is invalid")
	}
}

func TestVectorizeListIndexes(t *testing.T) {
	svc, server := vectorizeMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  []map[string]interface{}{},
		})
	})
	defer server.Close()

	indexes, err := svc.ListIndexes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if indexes == nil {
		t.Error("expected non-nil result")
	}
}

func TestVectorizeDeleteValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewVectorizeService(cf, "account123")

	err := svc.DeleteIndex(context.Background(), "")
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestVectorizeMetricConstants(t *testing.T) {
	validMetrics := []string{"cosine", "euclidean", "dot-product"}
	for _, m := range validMetrics {
		if !isValidMetric(m) {
			t.Errorf("expected %q to be a valid metric", m)
		}
	}
	if isValidMetric("manhattan") {
		t.Error("expected 'manhattan' to be invalid")
	}
}

func TestVectorizeIndexFields(t *testing.T) {
	idx := VectorizeIndex{
		Name:       "test-index",
		Dimensions: 768,
		Metric:     "cosine",
	}
	if idx.Name != "test-index" {
		t.Errorf("expected Name=test-index, got %s", idx.Name)
	}
	if idx.Dimensions != 768 {
		t.Errorf("expected Dimensions=768, got %d", idx.Dimensions)
	}

	data, err := json.Marshal(idx)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded VectorizeIndex
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Name != "test-index" {
		t.Errorf("decoded Name mismatch: %s", decoded.Name)
	}
}
