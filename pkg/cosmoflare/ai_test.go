package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// ---------------------------------------------------------------------------
// GetModel — found case
// ---------------------------------------------------------------------------

func TestAIGetModelFound(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{
					"name":        "@cf/meta/llama-3-8b-instruct",
					"description": "Meta Llama 3",
					"task":        map[string]string{"name": "text-generation"},
				},
				{
					"name": "@cf/openai/whisper",
					"task": map[string]string{"name": "speech-recognition"},
				},
			},
		})
	})
	defer server.Close()

	model, err := svc.GetModel(context.Background(), "@cf/meta/llama-3-8b-instruct")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.Name != "@cf/meta/llama-3-8b-instruct" {
		t.Errorf("expected model name @cf/meta/llama-3-8b-instruct, got %s", model.Name)
	}
	if model.Description != "Meta Llama 3" {
		t.Errorf("expected description 'Meta Llama 3', got %s", model.Description)
	}
	if model.Task.Name != "text-generation" {
		t.Errorf("expected task text-generation, got %s", model.Task.Name)
	}
}

// ---------------------------------------------------------------------------
// RunInference with Raw input
// ---------------------------------------------------------------------------

func TestAIRunInferenceWithRaw(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  map[string]interface{}{"response": "raw result"},
		})
	})
	defer server.Close()

	rawInput := json.RawMessage(`{"custom_field":"custom_value","steps":10}`)
	result, err := svc.RunInference(context.Background(), "@cf/stability/stable-diffusion", &AIInferenceInput{
		Raw: rawInput,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Response != "raw result" {
		t.Errorf("expected response 'raw result', got %s", result.Response)
	}
}

// ---------------------------------------------------------------------------
// RunInferenceRaw validation and success
// ---------------------------------------------------------------------------

func TestAIRunInferenceRawEmptyInputValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewAIService(cf, "account123")

	// Input with no prompt and no raw should fail
	_, err := svc.RunInferenceRaw(context.Background(), "@cf/test", &AIInferenceInput{})
	if err == nil {
		t.Error("expected error when input has no prompt or raw")
	}
}

func TestAIRunInferenceRawWithPrompt(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["prompt"] != "describe this image" {
			t.Errorf("unexpected prompt: %v", body["prompt"])
		}
		// Return binary-like raw bytes (plain JSON here to simplify)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`raw-binary-data`))
	})
	defer server.Close()

	data, err := svc.RunInferenceRaw(context.Background(), "@cf/unum/realesr-gan", &AIInferenceInput{
		Prompt: "describe this image",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "raw-binary-data" {
		t.Errorf("expected 'raw-binary-data', got %s", string(data))
	}
}

func TestAIRunInferenceRawWithRawInput(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`binary-image-bytes`))
	})
	defer server.Close()

	rawInput := json.RawMessage(`{"width":512,"height":512,"prompt":"a cat"}`)
	data, err := svc.RunInferenceRaw(context.Background(), "@cf/stability/stable-diffusion-xl-base-1.0", &AIInferenceInput{
		Raw: rawInput,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty response bytes")
	}
}

func TestAIRunInferenceRawHTTPError(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	})
	defer server.Close()

	_, err := svc.RunInferenceRaw(context.Background(), "@cf/meta/llama-3-8b-instruct", &AIInferenceInput{
		Prompt: "hello",
	})
	if err == nil {
		t.Error("expected error on 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to contain '401', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetGateway success
// ---------------------------------------------------------------------------

func TestAIGetGateway(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "gw-prod") {
			t.Errorf("expected path to contain 'gw-prod', got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"id":         "gw-prod",
				"name":       "Production Gateway",
				"slug":       "prod",
				"cache_ttl":  600,
				"created_at": "2026-01-01T00:00:00Z",
			},
		})
	})
	defer server.Close()

	gw, err := svc.GetGateway(context.Background(), "gw-prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw.ID != "gw-prod" {
		t.Errorf("expected ID gw-prod, got %s", gw.ID)
	}
	if gw.Name != "Production Gateway" {
		t.Errorf("expected name 'Production Gateway', got %s", gw.Name)
	}
	if gw.Slug != "prod" {
		t.Errorf("expected slug 'prod', got %s", gw.Slug)
	}
	if gw.CacheTTL != 600 {
		t.Errorf("expected CacheTTL 600, got %d", gw.CacheTTL)
	}
	if gw.CreatedAt != "2026-01-01T00:00:00Z" {
		t.Errorf("expected CreatedAt '2026-01-01T00:00:00Z', got %s", gw.CreatedAt)
	}
}

// ---------------------------------------------------------------------------
// GetGatewayLogs — empty result
// ---------------------------------------------------------------------------

func TestAIGetGatewayLogsEmpty(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  nil,
		})
	})
	defer server.Close()

	logs, err := svc.GetGatewayLogs(context.Background(), "gw-empty", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logs == nil {
		t.Error("expected non-nil slice for empty result")
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}

func TestAIGetGatewayLogFields(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{
					"id":          "log-xyz",
					"model":       "@cf/meta/llama-3-8b-instruct",
					"provider":    "workers-ai",
					"path":        "/ai/run/@cf/meta/llama-3-8b-instruct",
					"duration":    342,
					"status_code": 200,
					"cost":        0.0005,
					"cached":      false,
					"tokens_used": 512,
					"created_at":  "2026-06-01T12:00:00Z",
					"success":     true,
				},
			},
		})
	})
	defer server.Close()

	logs, err := svc.GetGatewayLogs(context.Background(), "gw-test", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	log := logs[0]
	if log.ID != "log-xyz" {
		t.Errorf("expected ID log-xyz, got %s", log.ID)
	}
	if log.Model != "@cf/meta/llama-3-8b-instruct" {
		t.Errorf("unexpected model: %s", log.Model)
	}
	if log.Provider != "workers-ai" {
		t.Errorf("unexpected provider: %s", log.Provider)
	}
	if log.Duration != 342 {
		t.Errorf("expected duration 342, got %d", log.Duration)
	}
	if log.StatusCode != 200 {
		t.Errorf("expected status_code 200, got %d", log.StatusCode)
	}
	if log.Cost != 0.0005 {
		t.Errorf("expected cost 0.0005, got %f", log.Cost)
	}
	if log.Cached {
		t.Error("expected Cached=false")
	}
	if log.Tokens != 512 {
		t.Errorf("expected 512 tokens, got %d", log.Tokens)
	}
	if !log.Success {
		t.Error("expected Success=true")
	}
}

// ---------------------------------------------------------------------------
// AIGatewayRateLimit struct
// ---------------------------------------------------------------------------

func TestAIGatewayRateLimitJSONRoundtrip(t *testing.T) {
	rl := AIGatewayRateLimit{
		Interval:  60,
		Limit:     100,
		Technique: "sliding",
	}
	data, err := json.Marshal(rl)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded AIGatewayRateLimit
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Interval != 60 {
		t.Errorf("expected Interval 60, got %d", decoded.Interval)
	}
	if decoded.Limit != 100 {
		t.Errorf("expected Limit 100, got %d", decoded.Limit)
	}
	if decoded.Technique != "sliding" {
		t.Errorf("expected Technique 'sliding', got %s", decoded.Technique)
	}
}

func TestAIGatewayWithRateLimit(t *testing.T) {
	collectLogs := false
	rl := &AIGatewayRateLimit{Interval: 30, Limit: 50, Technique: "fixed"}
	gw := AIGateway{
		ID:               "gw-rl",
		CacheTTL:         120,
		CollectLogs:      &collectLogs,
		RateLimitingRule: rl,
		RateLimitInterval: 30,
		RateLimitLimit:    50,
		RateLimitTechnique: "fixed",
	}
	data, err := json.Marshal(gw)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded AIGateway
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.RateLimitInterval != 30 {
		t.Errorf("expected RateLimitInterval 30, got %d", decoded.RateLimitInterval)
	}
	if decoded.RateLimitLimit != 50 {
		t.Errorf("expected RateLimitLimit 50, got %d", decoded.RateLimitLimit)
	}
	if decoded.RateLimitTechnique != "fixed" {
		t.Errorf("expected RateLimitTechnique 'fixed', got %s", decoded.RateLimitTechnique)
	}
	if decoded.CollectLogs == nil || *decoded.CollectLogs {
		t.Error("expected CollectLogs=false")
	}
}

// ---------------------------------------------------------------------------
// CreateGateway with rate limiting params
// ---------------------------------------------------------------------------

func TestAICreateGatewayWithRateLimiting(t *testing.T) {
	svc, server := aiMockSetup(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// Verify rate limiting fields forwarded
		if body["rate_limiting_interval"] != float64(60) {
			t.Errorf("expected rate_limiting_interval=60, got %v", body["rate_limiting_interval"])
		}
		if body["rate_limiting_limit"] != float64(200) {
			t.Errorf("expected rate_limiting_limit=200, got %v", body["rate_limiting_limit"])
		}
		if body["rate_limiting_technique"] != "sliding" {
			t.Errorf("expected rate_limiting_technique='sliding', got %v", body["rate_limiting_technique"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"id":   "rl-gw",
				"name": "rl-gateway",
			},
		})
	})
	defer server.Close()

	gw, err := svc.CreateGateway(context.Background(), "rl-gateway", AIGatewayCreateParams{
		RateLimitInterval:  60,
		RateLimitLimit:     200,
		RateLimitTechnique: "sliding",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw.ID != "rl-gw" {
		t.Errorf("expected gateway ID rl-gw, got %s", gw.ID)
	}
}

// ---------------------------------------------------------------------------
// AIGatewayCreateParams JSON roundtrip
// ---------------------------------------------------------------------------

func TestAIGatewayCreateParamsJSONRoundtrip(t *testing.T) {
	collectLogs := true
	params := AIGatewayCreateParams{
		CacheTTL:           300,
		CollectLogs:        &collectLogs,
		RateLimitInterval:  60,
		RateLimitLimit:     100,
		RateLimitTechnique: "sliding",
	}
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded AIGatewayCreateParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.CacheTTL != 300 {
		t.Errorf("expected CacheTTL 300, got %d", decoded.CacheTTL)
	}
	if decoded.RateLimitInterval != 60 {
		t.Errorf("expected RateLimitInterval 60, got %d", decoded.RateLimitInterval)
	}
	if decoded.RateLimitLimit != 100 {
		t.Errorf("expected RateLimitLimit 100, got %d", decoded.RateLimitLimit)
	}
	if decoded.RateLimitTechnique != "sliding" {
		t.Errorf("expected RateLimitTechnique 'sliding', got %s", decoded.RateLimitTechnique)
	}
	if decoded.CollectLogs == nil || !*decoded.CollectLogs {
		t.Error("expected CollectLogs=true")
	}
}

// ---------------------------------------------------------------------------
// NewAIService base URL fallback
// ---------------------------------------------------------------------------

func TestNewAIServiceBaseURLDefault(t *testing.T) {
	// When api.BaseURL is empty the service should use the default CF base URL
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewAIService(cf, "acct-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.baseURL == "" {
		t.Error("expected non-empty baseURL")
	}
	if !strings.Contains(svc.baseURL, "cloudflare.com") {
		t.Errorf("expected baseURL to contain 'cloudflare.com', got %s", svc.baseURL)
	}
}

func TestNewAIServiceFromCredsBaseURL(t *testing.T) {
	svc, err := NewAIServiceFromCreds("acct-789", "some-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.baseURL != "https://api.cloudflare.com/client/v4" {
		t.Errorf("expected default base URL, got %s", svc.baseURL)
	}
}

// ---------------------------------------------------------------------------
// AIInferenceResult JSON roundtrip
// ---------------------------------------------------------------------------

func TestAIInferenceResultJSONRoundtrip(t *testing.T) {
	raw := json.RawMessage(`{"choices":[{"text":"hello world"}]}`)
	result := AIInferenceResult{
		Response: "hello world",
		Result:   raw,
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded AIInferenceResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Response != "hello world" {
		t.Errorf("expected Response 'hello world', got %s", decoded.Response)
	}
	if decoded.Result == nil {
		t.Error("expected non-nil Result")
	}
}

// ---------------------------------------------------------------------------
// Error message content
// ---------------------------------------------------------------------------

func TestAIValidationErrorMessages(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewAIService(cf, "acct")

	_, err := svc.GetModel(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "model name is required") {
		t.Errorf("expected 'model name is required' in error, got: %v", err)
	}

	_, err = svc.RunInference(context.Background(), "", &AIInferenceInput{Prompt: "hi"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "model name is required") {
		t.Errorf("expected 'model name is required' in error, got: %v", err)
	}

	err = svc.DeleteGateway(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "gateway ID is required") {
		t.Errorf("expected 'gateway ID is required' in error, got: %v", err)
	}
}

func TestAIRunInferenceEmptyInputError(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewAIService(cf, "acct")

	_, err := svc.RunInference(context.Background(), "@cf/test", &AIInferenceInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "prompt, messages, or raw input is required") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// AIModel with properties roundtrip
// ---------------------------------------------------------------------------

func TestAIModelPropertiesRoundtrip(t *testing.T) {
	model := AIModel{
		Name: "@cf/meta/llama-3-8b-instruct",
		Task: AITask{Name: "text-generation"},
		Properties: []AIProp{
			{PropertyID: "beta", Value: "true"},
			{PropertyID: "max_tokens", Value: "4096"},
		},
	}
	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded AIModel
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(decoded.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(decoded.Properties))
	}
	if decoded.Properties[0].PropertyID != "beta" {
		t.Errorf("expected PropertyID 'beta', got %s", decoded.Properties[0].PropertyID)
	}
	if decoded.Properties[1].Value != "4096" {
		t.Errorf("expected Value '4096', got %s", decoded.Properties[1].Value)
	}
}
