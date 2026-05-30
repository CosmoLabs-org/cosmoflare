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

func TestVectorizeInsertVectorsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewVectorizeService(cf, "account123")

	t.Run("empty index name", func(t *testing.T) {
		vectors := []VectorizeVector{{ID: "v1", Values: []float64{0.1, 0.2}}}
		err := svc.InsertVectors(context.Background(), "", vectors)
		if err == nil {
			t.Error("expected error when index name is empty")
		}
	})

	t.Run("empty vectors slice", func(t *testing.T) {
		err := svc.InsertVectors(context.Background(), "test-index", []VectorizeVector{})
		if err == nil {
			t.Error("expected error when vectors slice is empty")
		}
	})

	t.Run("nil vectors slice", func(t *testing.T) {
		err := svc.InsertVectors(context.Background(), "test-index", nil)
		if err == nil {
			t.Error("expected error when vectors slice is nil")
		}
	})
}

func TestVectorizeQueryVectorsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewVectorizeService(cf, "account123")

	t.Run("empty index name", func(t *testing.T) {
		_, err := svc.QueryVectors(context.Background(), "", []float64{0.1, 0.2}, 10)
		if err == nil {
			t.Error("expected error when index name is empty")
		}
	})

	t.Run("empty values slice", func(t *testing.T) {
		_, err := svc.QueryVectors(context.Background(), "test-index", []float64{}, 10)
		if err == nil {
			t.Error("expected error when values slice is empty")
		}
	})

	t.Run("nil values slice", func(t *testing.T) {
		_, err := svc.QueryVectors(context.Background(), "test-index", nil, 10)
		if err == nil {
			t.Error("expected error when values slice is nil")
		}
	})

	t.Run("zero topK defaults to 10 (no validation error)", func(t *testing.T) {
		// topK <= 0 defaults to 10, so this should not be a validation error
		// It will fail at the API level since we have no mock, but should pass validation
		_, err := svc.QueryVectors(context.Background(), "test-index", []float64{0.1}, 0)
		if err != nil {
			errStr := err.Error()
			if errStr == "" {
				t.Error("unexpected nil error string")
			}
			// Should NOT fail with a validation error about topK
			if contains := func(s, sub string) bool {
				for i := 0; i <= len(s)-len(sub); i++ {
					if s[i:i+len(sub)] == sub {
						return true
					}
				}
				return false
			}; contains(errStr, "topK") {
				t.Errorf("expected no topK validation error, got: %s", errStr)
			}
		}
	})
}

func TestVectorizeVectorJSONRoundtrip(t *testing.T) {
	t.Run("basic vector", func(t *testing.T) {
		v := VectorizeVector{
			ID:     "vec-001",
			Values: []float64{0.1, 0.2, 0.3, 0.4},
		}
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		var decoded VectorizeVector
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if decoded.ID != "vec-001" {
			t.Errorf("expected ID vec-001, got %s", decoded.ID)
		}
		if len(decoded.Values) != 4 {
			t.Fatalf("expected 4 values, got %d", len(decoded.Values))
		}
		if decoded.Values[0] != 0.1 {
			t.Errorf("expected first value 0.1, got %f", decoded.Values[0])
		}
	})

	t.Run("vector with metadata", func(t *testing.T) {
		v := VectorizeVector{
			ID:       "vec-002",
			Values:   []float64{1.0, 2.0},
			Metadata: map[string]interface{}{"category": "test", "score": float64(42)},
		}
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		var decoded VectorizeVector
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if decoded.Metadata == nil {
			t.Fatal("expected non-nil metadata after round-trip")
		}
		if decoded.Metadata["category"] != "test" {
			t.Errorf("expected metadata category=test, got %v", decoded.Metadata["category"])
		}
		score, ok := decoded.Metadata["score"].(float64)
		if !ok || score != 42 {
			t.Errorf("expected metadata score=42, got %v", decoded.Metadata["score"])
		}
	})

	t.Run("vector without metadata omits field", func(t *testing.T) {
		v := VectorizeVector{
			ID:     "vec-003",
			Values: []float64{0.5},
		}
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		jsonStr := string(data)
		if jsonContains(jsonStr, "metadata") {
			t.Errorf("expected metadata to be omitted from JSON, got: %s", jsonStr)
		}
	})
}

func TestVectorizeQueryResultJSONRoundtrip(t *testing.T) {
	t.Run("basic query result", func(t *testing.T) {
		qr := VectorizeQueryResult{
			ID:    "match-001",
			Score: 0.95,
		}
		data, err := json.Marshal(qr)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		var decoded VectorizeQueryResult
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if decoded.ID != "match-001" {
			t.Errorf("expected ID match-001, got %s", decoded.ID)
		}
		if decoded.Score != 0.95 {
			t.Errorf("expected score 0.95, got %f", decoded.Score)
		}
	})

	t.Run("query result with values and metadata", func(t *testing.T) {
		qr := VectorizeQueryResult{
			ID:       "match-002",
			Score:    0.87,
			Values:   []float64{0.1, 0.2, 0.3},
			Metadata: map[string]interface{}{"label": "positive"},
		}
		data, err := json.Marshal(qr)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		var decoded VectorizeQueryResult
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if len(decoded.Values) != 3 {
			t.Fatalf("expected 3 values, got %d", len(decoded.Values))
		}
		if decoded.Metadata["label"] != "positive" {
			t.Errorf("expected metadata label=positive, got %v", decoded.Metadata["label"])
		}
	})
}

func TestVectorizeConfigJSONRoundtrip(t *testing.T) {
	cfg := VectorizeConfig{Dimensions: 1536, Metric: "euclidean"}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded VectorizeConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Dimensions != 1536 {
		t.Errorf("expected dimensions 1536, got %d", decoded.Dimensions)
	}
	if decoded.Metric != "euclidean" {
		t.Errorf("expected metric euclidean, got %s", decoded.Metric)
	}
}

func TestVectorizeGetIndexValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewVectorizeService(cf, "account123")

	_, err := svc.GetIndex(context.Background(), "")
	if err == nil {
		t.Error("expected error when index name is empty")
	}
}

func jsonContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
