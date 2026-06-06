package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

// ---------------------------------------------------------------------------
// Constructor validation
// ---------------------------------------------------------------------------

func TestNewAIServiceValidation(t *testing.T) {
	_, err := NewAIService(nil, "account123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewAIService(cf, "")
	if err == nil {
		t.Error("expected error when account ID is empty")
	}
}

func TestNewAIServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewAIService(cf, "account123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewAIServiceFromCredsValidation(t *testing.T) {
	_, err := NewAIServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when account ID is empty")
	}
	_, err = NewAIServiceFromCreds("acct", "")
	if err == nil {
		t.Error("expected error when token is empty")
	}
}

func TestNewAIServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewAIServiceFromCreds("acct123", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
	if svc.apiToken != "test-token" {
		t.Errorf("expected apiToken=test-token, got %s", svc.apiToken)
	}
}

// ---------------------------------------------------------------------------
// Mock setup helper
// ---------------------------------------------------------------------------

func aiMockSetup(handler http.HandlerFunc) (*AIService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewAIService(cf, "acct-ai-123")
	return svc, server
}

// ---------------------------------------------------------------------------
// Workers AI — Models
// ---------------------------------------------------------------------------

func TestAIListModels(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{
					"name":        "@cf/meta/llama-3-8b-instruct",
					"description": "Meta Llama 3 8B Instruct",
					"task":        map[string]string{"name": "text-generation"},
				},
				{
					"name":        "@cf/openai/whisper",
					"description": "OpenAI Whisper",
					"task":        map[string]string{"name": "speech-recognition"},
				},
			},
		})
	})
	defer server.Close()

	models, err := svc.ListModels(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].Name != "@cf/meta/llama-3-8b-instruct" {
		t.Errorf("expected model name @cf/meta/llama-3-8b-instruct, got %s", models[0].Name)
	}
}

func TestAIListModelsWithFilter(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		task := r.URL.Query().Get("task")
		if task != "text-generation" {
			t.Errorf("expected task=text-generation query param, got %q", task)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{
					"name": "@cf/meta/llama-3-8b-instruct",
					"task": map[string]string{"name": "text-generation"},
				},
			},
		})
	})
	defer server.Close()

	models, err := svc.ListModels(context.Background(), "text-generation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
}

func TestAIListModelsEmpty(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  nil,
		})
	})
	defer server.Close()

	models, err := svc.ListModels(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if models == nil {
		t.Error("expected non-nil slice for empty result")
	}
	if len(models) != 0 {
		t.Errorf("expected 0 models, got %d", len(models))
	}
}

func TestAIGetModelValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewAIService(cf, "account123")

	_, err := svc.GetModel(context.Background(), "")
	if err == nil {
		t.Error("expected error when model name is empty")
	}
}

func TestAIGetModelNotFound(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  []map[string]interface{}{},
		})
	})
	defer server.Close()

	_, err := svc.GetModel(context.Background(), "@cf/nonexistent")
	if err == nil {
		t.Error("expected error for non-existent model")
	}
}

// ---------------------------------------------------------------------------
// Workers AI — Inference
// ---------------------------------------------------------------------------

func TestAIRunInferenceValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewAIService(cf, "account123")

	t.Run("empty model name", func(t *testing.T) {
		_, err := svc.RunInference(context.Background(), "", &AIInferenceInput{Prompt: "hello"})
		if err == nil {
			t.Error("expected error when model name is empty")
		}
	})

	t.Run("nil input", func(t *testing.T) {
		_, err := svc.RunInference(context.Background(), "@cf/test", nil)
		if err == nil {
			t.Error("expected error when input is nil")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		_, err := svc.RunInference(context.Background(), "@cf/test", &AIInferenceInput{})
		if err == nil {
			t.Error("expected error when input has no prompt/messages/raw")
		}
	})
}

func TestAIRunInferenceWithPrompt(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["prompt"] != "hello" {
			t.Errorf("expected prompt=hello, got %v", body["prompt"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"response": "Hello! How can I help you?",
			},
		})
	})
	defer server.Close()

	result, err := svc.RunInference(context.Background(), "@cf/meta/llama-3-8b-instruct", &AIInferenceInput{
		Prompt: "hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Response != "Hello! How can I help you?" {
		t.Errorf("unexpected response: %s", result.Response)
	}
}

func TestAIRunInferenceWithMessages(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		messages, ok := body["messages"].([]interface{})
		if !ok || len(messages) != 2 {
			t.Errorf("expected 2 messages, got %v", body["messages"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"response": "I am doing well, thank you!",
			},
		})
	})
	defer server.Close()

	result, err := svc.RunInference(context.Background(), "@cf/meta/llama-3-8b-instruct", &AIInferenceInput{
		Messages: []AIMessage{
			{Role: "system", Content: "You are a friendly assistant."},
			{Role: "user", Content: "How are you?"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Response != "I am doing well, thank you!" {
		t.Errorf("unexpected response: %s", result.Response)
	}
}

func TestAIRunInferenceRawValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewAIService(cf, "account123")

	t.Run("empty model name", func(t *testing.T) {
		_, err := svc.RunInferenceRaw(context.Background(), "", &AIInferenceInput{Prompt: "hello"})
		if err == nil {
			t.Error("expected error when model name is empty")
		}
	})

	t.Run("nil input", func(t *testing.T) {
		_, err := svc.RunInferenceRaw(context.Background(), "@cf/test", nil)
		if err == nil {
			t.Error("expected error when input is nil")
		}
	})
}

// ---------------------------------------------------------------------------
// AI Gateway — CRUD
// ---------------------------------------------------------------------------

func TestAIListGateways(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{"id": "gw-1", "name": "my-gateway"},
				{"id": "gw-2", "name": "prod-gateway"},
			},
		})
	})
	defer server.Close()

	gateways, err := svc.ListGateways(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gateways) != 2 {
		t.Fatalf("expected 2 gateways, got %d", len(gateways))
	}
	if gateways[0].ID != "gw-1" {
		t.Errorf("expected gateway ID gw-1, got %s", gateways[0].ID)
	}
}

func TestAIListGatewaysEmpty(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  nil,
		})
	})
	defer server.Close()

	gateways, err := svc.ListGateways(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gateways == nil {
		t.Error("expected non-nil slice for empty result")
	}
}

func TestAIGetGatewayValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewAIService(cf, "account123")

	_, err := svc.GetGateway(context.Background(), "")
	if err == nil {
		t.Error("expected error when gateway ID is empty")
	}
}

func TestAICreateGatewayValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewAIService(cf, "account123")

	_, err := svc.CreateGateway(context.Background(), "", AIGatewayCreateParams{})
	if err == nil {
		t.Error("expected error when gateway name is empty")
	}
}

func TestAICreateGateway(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "test-gw" {
			t.Errorf("expected name=test-gw, got %v", body["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  map[string]interface{}{"id": "test-gw", "name": "test-gw", "cache_ttl": 300},
		})
	})
	defer server.Close()

	collectLogs := true
	gw, err := svc.CreateGateway(context.Background(), "test-gw", AIGatewayCreateParams{
		CacheTTL:    300,
		CollectLogs: &collectLogs,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw.ID != "test-gw" {
		t.Errorf("expected gateway ID test-gw, got %s", gw.ID)
	}
}

func TestAIDeleteGatewayValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewAIService(cf, "account123")

	err := svc.DeleteGateway(context.Background(), "")
	if err == nil {
		t.Error("expected error when gateway ID is empty")
	}
}

func TestAIDeleteGateway(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  nil,
		})
	})
	defer server.Close()

	err := svc.DeleteGateway(context.Background(), "test-gw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// AI Gateway — Logs
// ---------------------------------------------------------------------------

func TestAIGetGatewayLogsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewAIService(cf, "account123")

	_, err := svc.GetGatewayLogs(context.Background(), "", 10)
	if err == nil {
		t.Error("expected error when gateway ID is empty")
	}
}

func TestAIGetGatewayLogs(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		perPage := r.URL.Query().Get("per_page")
		if perPage != "5" {
			t.Errorf("expected per_page=5, got %s", perPage)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{"id": "log-1", "model": "@cf/meta/llama-3-8b-instruct", "status_code": 200, "success": true, "cached": false},
				{"id": "log-2", "model": "@cf/openai/whisper", "status_code": 200, "success": true, "cached": true},
			},
		})
	})
	defer server.Close()

	logs, err := svc.GetGatewayLogs(context.Background(), "test-gw", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(logs))
	}
	if logs[0].ID != "log-1" {
		t.Errorf("expected log ID log-1, got %s", logs[0].ID)
	}
}

func TestAIGetGatewayLogsDefaultLimit(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		perPage := r.URL.Query().Get("per_page")
		if perPage != "25" {
			t.Errorf("expected default per_page=25, got %s", perPage)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  []map[string]interface{}{},
		})
	})
	defer server.Close()

	_, err := svc.GetGatewayLogs(context.Background(), "test-gw", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Type JSON roundtrips
// ---------------------------------------------------------------------------

func TestAIModelJSONRoundtrip(t *testing.T) {
	model := AIModel{
		Name:        "@cf/meta/llama-3-8b-instruct",
		Description: "Meta Llama 3 8B Instruct model",
		Task:        AITask{Name: "text-generation", Description: "Generate text"},
		Properties:  []AIProp{{PropertyID: "beta", Value: "true"}},
	}
	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded AIModel
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Name != "@cf/meta/llama-3-8b-instruct" {
		t.Errorf("expected Name @cf/meta/llama-3-8b-instruct, got %s", decoded.Name)
	}
	if decoded.Task.Name != "text-generation" {
		t.Errorf("expected task name text-generation, got %s", decoded.Task.Name)
	}
	if len(decoded.Properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(decoded.Properties))
	}
}

func TestAIGatewayJSONRoundtrip(t *testing.T) {
	collectLogs := true
	gw := AIGateway{
		ID:        "gw-123",
		Name:      "prod-gateway",
		CacheTTL:  300,
		CollectLogs: &collectLogs,
	}
	data, err := json.Marshal(gw)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded AIGateway
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.ID != "gw-123" {
		t.Errorf("expected ID gw-123, got %s", decoded.ID)
	}
	if decoded.CacheTTL != 300 {
		t.Errorf("expected CacheTTL 300, got %d", decoded.CacheTTL)
	}
	if decoded.CollectLogs == nil || !*decoded.CollectLogs {
		t.Error("expected CollectLogs=true")
	}
}

func TestAIGatewayLogJSONRoundtrip(t *testing.T) {
	log := AIGatewayLog{
		ID:         "log-abc",
		Model:      "@cf/meta/llama-3-8b-instruct",
		Provider:   "workers-ai",
		StatusCode: 200,
		Cost:       0.0001,
		Cached:     true,
		Tokens:     150,
		Success:    true,
	}
	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded AIGatewayLog
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.ID != "log-abc" {
		t.Errorf("expected ID log-abc, got %s", decoded.ID)
	}
	if !decoded.Cached {
		t.Error("expected Cached=true")
	}
	if decoded.Tokens != 150 {
		t.Errorf("expected 150 tokens, got %d", decoded.Tokens)
	}
}

func TestAIInferenceInputJSONRoundtrip(t *testing.T) {
	input := AIInferenceInput{
		Messages: []AIMessage{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hi"},
		},
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded AIInferenceInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(decoded.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(decoded.Messages))
	}
	if decoded.Messages[0].Role != "system" {
		t.Errorf("expected first message role=system, got %s", decoded.Messages[0].Role)
	}
}

// ---------------------------------------------------------------------------
// API error handling
// ---------------------------------------------------------------------------

func TestAIServiceAPIError(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"success":false,"errors":[{"message":"forbidden"}]}`))
	})
	defer server.Close()

	_, err := svc.ListModels(context.Background(), "")
	if err == nil {
		t.Error("expected error on 403 response")
	}
}

func TestAIServiceAPIErrorInResult(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"errors":  []map[string]string{{"message": "rate limited"}},
			"result":  nil,
		})
	})
	defer server.Close()

	_, err := svc.ListGateways(context.Background())
	if err == nil {
		t.Error("expected error on success=false response")
	}
}
