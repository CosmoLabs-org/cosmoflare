package cosmoflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

// AIModel represents a Cloudflare Workers AI model.
type AIModel struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Task        AITask   `json:"task"`
	Properties  []AIProp `json:"properties,omitempty"`
}

// AITask describes the task type a model performs.
type AITask struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// AIProp holds a model property key-value pair.
type AIProp struct {
	PropertyID string `json:"property_id"`
	Value      string `json:"value"`
}

// AIInferenceInput holds input for a text-generation inference call.
type AIInferenceInput struct {
	Prompt   string          `json:"prompt,omitempty"`
	Messages []AIMessage     `json:"messages,omitempty"`
	Raw      json.RawMessage `json:"-"`
}

// AIMessage represents a chat message for multi-turn inference.
type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIInferenceResult holds the result of an inference call.
type AIInferenceResult struct {
	Response string          `json:"response,omitempty"`
	Result   json.RawMessage `json:"result,omitempty"`
}

// AIGateway represents a Cloudflare AI Gateway configuration.
type AIGateway struct {
	ID                string `json:"id"`
	Name              string `json:"name,omitempty"`
	Slug              string `json:"slug,omitempty"`
	CacheTTL          int    `json:"cache_ttl,omitempty"`
	CacheInvalidateOn string `json:"cache_invalidate_on,omitempty"`
	CollectLogs       *bool  `json:"collect_logs,omitempty"`
	RateLimitingRule  *AIGatewayRateLimit `json:"rate_limiting,omitempty"`
	RateLimitInterval int    `json:"rate_limiting_interval,omitempty"`
	RateLimitLimit    int    `json:"rate_limiting_limit,omitempty"`
	RateLimitTechnique string `json:"rate_limiting_technique,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
	ModifiedAt        string `json:"modified_at,omitempty"`
}

// AIGatewayRateLimit holds rate limiting configuration for a gateway.
type AIGatewayRateLimit struct {
	Interval  int    `json:"interval"`
	Limit     int    `json:"limit"`
	Technique string `json:"technique,omitempty"`
}

// AIGatewayLog represents a single log entry from an AI Gateway.
type AIGatewayLog struct {
	ID         string  `json:"id"`
	Model      string  `json:"model,omitempty"`
	Provider   string  `json:"provider,omitempty"`
	Path       string  `json:"path,omitempty"`
	Duration   int     `json:"duration,omitempty"`
	StatusCode int     `json:"status_code,omitempty"`
	Cost       float64 `json:"cost,omitempty"`
	Cached     bool    `json:"cached"`
	Tokens     int     `json:"tokens_used,omitempty"`
	CreatedAt  string  `json:"created_at,omitempty"`
	Success    bool    `json:"success"`
}

// AIGatewayCreateParams holds parameters for creating an AI Gateway.
type AIGatewayCreateParams struct {
	CacheTTL          int    `json:"cache_ttl,omitempty"`
	CollectLogs       *bool  `json:"collect_logs,omitempty"`
	RateLimitInterval int    `json:"rate_limiting_interval,omitempty"`
	RateLimitLimit    int    `json:"rate_limiting_limit,omitempty"`
	RateLimitTechnique string `json:"rate_limiting_technique,omitempty"`
}

// AIService implements Cloudflare Workers AI and AI Gateway operations.
type AIService struct {
	cf        *cloudflare.API
	accountID string
	apiToken  string
	baseURL   string
}

// NewAIService creates a new AI service client.
func NewAIService(api *cloudflare.API, accountID string) (*AIService, error) {
	if api == nil {
		return nil, validationError("NewAIService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewAIService", "account ID is required")
	}
	baseURL := api.BaseURL
	if baseURL == "" {
		baseURL = "https://api.cloudflare.com/client/v4"
	}
	return &AIService{
		cf:        api,
		accountID: accountID,
		baseURL:   baseURL,
	}, nil
}

// NewAIServiceFromCreds creates an AIService from raw credentials.
func NewAIServiceFromCreds(accountID, apiToken string) (*AIService, error) {
	if accountID == "" {
		return nil, validationError("NewAIServiceFromCreds", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewAIServiceFromCreds", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, newError("NewAIServiceFromCreds", "failed to create API client", err)
	}
	return &AIService{
		cf:        cf,
		accountID: accountID,
		apiToken:  apiToken,
		baseURL:   "https://api.cloudflare.com/client/v4",
	}, nil
}

// ---------------------------------------------------------------------------
// Workers AI — Models
// ---------------------------------------------------------------------------

// ListModels lists available Workers AI models, optionally filtered by task type.
func (s *AIService) ListModels(ctx context.Context, filter string) ([]AIModel, error) {
	url := fmt.Sprintf("%s/accounts/%s/ai/models/search", s.baseURL, s.accountID)
	if filter != "" {
		url += "?task=" + filter
	}

	var result []AIModel
	err := s.aiDoJSON(ctx, "GET", url, nil, &result)
	if err != nil {
		return nil, newError("AIService.ListModels", "failed to list AI models", err)
	}
	if result == nil {
		result = []AIModel{}
	}
	return result, nil
}

// GetModel retrieves details for a specific Workers AI model.
func (s *AIService) GetModel(ctx context.Context, modelName string) (*AIModel, error) {
	if modelName == "" {
		return nil, validationError("AIService.GetModel", "model name is required")
	}

	// The search endpoint returns all models; filter client-side for exact match
	models, err := s.ListModels(ctx, "")
	if err != nil {
		return nil, newError("AIService.GetModel", fmt.Sprintf("failed to get model %q", modelName), err)
	}

	for i := range models {
		if models[i].Name == modelName {
			return &models[i], nil
		}
	}

	return nil, newError("AIService.GetModel", fmt.Sprintf("model %q not found", modelName), nil)
}

// ---------------------------------------------------------------------------
// Workers AI — Inference
// ---------------------------------------------------------------------------

// RunInference runs inference on a Workers AI model with the given input.
func (s *AIService) RunInference(ctx context.Context, modelName string, input *AIInferenceInput) (*AIInferenceResult, error) {
	if modelName == "" {
		return nil, validationError("AIService.RunInference", "model name is required")
	}
	if input == nil {
		return nil, validationError("AIService.RunInference", "input is required")
	}

	url := fmt.Sprintf("%s/accounts/%s/ai/run/%s", s.baseURL, s.accountID, modelName)

	// Build the request body: use Raw if provided, otherwise serialize input fields
	var body interface{}
	if input.Raw != nil {
		body = input.Raw
	} else if len(input.Messages) > 0 {
		body = map[string]interface{}{
			"messages": input.Messages,
		}
	} else if input.Prompt != "" {
		body = map[string]interface{}{
			"prompt": input.Prompt,
		}
	} else {
		return nil, validationError("AIService.RunInference", "prompt, messages, or raw input is required")
	}

	var result AIInferenceResult
	err := s.aiDoJSON(ctx, "POST", url, body, &result)
	if err != nil {
		return nil, newError("AIService.RunInference", fmt.Sprintf("failed to run inference on model %q", modelName), err)
	}
	return &result, nil
}

// RunInferenceRaw runs inference and returns the raw response bytes.
// Useful for binary outputs like image generation.
func (s *AIService) RunInferenceRaw(ctx context.Context, modelName string, input *AIInferenceInput) ([]byte, error) {
	if modelName == "" {
		return nil, validationError("AIService.RunInferenceRaw", "model name is required")
	}
	if input == nil {
		return nil, validationError("AIService.RunInferenceRaw", "input is required")
	}

	url := fmt.Sprintf("%s/accounts/%s/ai/run/%s", s.baseURL, s.accountID, modelName)

	var body interface{}
	if input.Raw != nil {
		body = input.Raw
	} else if input.Prompt != "" {
		body = map[string]interface{}{
			"prompt": input.Prompt,
		}
	} else {
		return nil, validationError("AIService.RunInferenceRaw", "prompt or raw input is required")
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, newError("AIService.RunInferenceRaw", "failed to marshal request body", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, newError("AIService.RunInferenceRaw", "failed to create request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, newError("AIService.RunInferenceRaw", "request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newError("AIService.RunInferenceRaw", "failed to read response", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return respBody, nil
}

// ---------------------------------------------------------------------------
// AI Gateway — CRUD
// ---------------------------------------------------------------------------

// ListGateways lists all AI Gateway configurations.
func (s *AIService) ListGateways(ctx context.Context) ([]AIGateway, error) {
	url := fmt.Sprintf("%s/accounts/%s/ai-gateway/gateways", s.baseURL, s.accountID)

	var result []AIGateway
	err := s.aiDoJSON(ctx, "GET", url, nil, &result)
	if err != nil {
		return nil, newError("AIService.ListGateways", "failed to list AI gateways", err)
	}
	if result == nil {
		result = []AIGateway{}
	}
	return result, nil
}

// GetGateway retrieves a specific AI Gateway by ID.
func (s *AIService) GetGateway(ctx context.Context, gatewayID string) (*AIGateway, error) {
	if gatewayID == "" {
		return nil, validationError("AIService.GetGateway", "gateway ID is required")
	}

	url := fmt.Sprintf("%s/accounts/%s/ai-gateway/gateways/%s", s.baseURL, s.accountID, gatewayID)

	var result AIGateway
	err := s.aiDoJSON(ctx, "GET", url, nil, &result)
	if err != nil {
		return nil, newError("AIService.GetGateway", fmt.Sprintf("failed to get gateway %q", gatewayID), err)
	}
	return &result, nil
}

// CreateGateway creates a new AI Gateway.
func (s *AIService) CreateGateway(ctx context.Context, name string, params AIGatewayCreateParams) (*AIGateway, error) {
	if name == "" {
		return nil, validationError("AIService.CreateGateway", "gateway name is required")
	}

	url := fmt.Sprintf("%s/accounts/%s/ai-gateway/gateways", s.baseURL, s.accountID)

	body := map[string]interface{}{
		"id":   name,
		"name": name,
	}
	if params.CacheTTL > 0 {
		body["cache_ttl"] = params.CacheTTL
	}
	if params.CollectLogs != nil {
		body["collect_logs"] = *params.CollectLogs
	}
	if params.RateLimitInterval > 0 {
		body["rate_limiting_interval"] = params.RateLimitInterval
	}
	if params.RateLimitLimit > 0 {
		body["rate_limiting_limit"] = params.RateLimitLimit
	}
	if params.RateLimitTechnique != "" {
		body["rate_limiting_technique"] = params.RateLimitTechnique
	}

	var result AIGateway
	err := s.aiDoJSON(ctx, "POST", url, body, &result)
	if err != nil {
		return nil, newError("AIService.CreateGateway", fmt.Sprintf("failed to create gateway %q", name), err)
	}
	return &result, nil
}

// DeleteGateway deletes an AI Gateway by ID.
func (s *AIService) DeleteGateway(ctx context.Context, gatewayID string) error {
	if gatewayID == "" {
		return validationError("AIService.DeleteGateway", "gateway ID is required")
	}

	url := fmt.Sprintf("%s/accounts/%s/ai-gateway/gateways/%s", s.baseURL, s.accountID, gatewayID)

	err := s.aiDoJSON(ctx, "DELETE", url, nil, nil)
	if err != nil {
		return newError("AIService.DeleteGateway", fmt.Sprintf("failed to delete gateway %q", gatewayID), err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// AI Gateway — Logs
// ---------------------------------------------------------------------------

// GetGatewayLogs retrieves logs from an AI Gateway.
func (s *AIService) GetGatewayLogs(ctx context.Context, gatewayID string, limit int) ([]AIGatewayLog, error) {
	if gatewayID == "" {
		return nil, validationError("AIService.GetGatewayLogs", "gateway ID is required")
	}
	if limit <= 0 {
		limit = 25
	}

	url := fmt.Sprintf("%s/accounts/%s/ai-gateway/gateways/%s/logs?per_page=%d&order_by=created_at&order_by_direction=desc",
		s.baseURL, s.accountID, gatewayID, limit)

	var result []AIGatewayLog
	err := s.aiDoJSON(ctx, "GET", url, nil, &result)
	if err != nil {
		return nil, newError("AIService.GetGatewayLogs", fmt.Sprintf("failed to get logs for gateway %q", gatewayID), err)
	}
	if result == nil {
		result = []AIGatewayLog{}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// HTTP helpers (same pattern as VectorizeService)
// ---------------------------------------------------------------------------

// aiAPIResponse matches the Cloudflare API envelope format.
type aiAPIResponse struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (s *AIService) aiDoJSON(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		// Handle json.RawMessage directly
		if raw, ok := body.(json.RawMessage); ok {
			bodyReader = bytes.NewReader(raw)
		} else {
			data, err := json.Marshal(body)
			if err != nil {
				return err
			}
			bodyReader = bytes.NewReader(data)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if result == nil {
		return nil
	}

	var apiResp aiAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return json.Unmarshal(respBody, result)
	}

	if !apiResp.Success && len(apiResp.Errors) > 0 {
		return fmt.Errorf("API error: %s", apiResp.Errors[0].Message)
	}

	if apiResp.Result != nil {
		return json.Unmarshal(apiResp.Result, result)
	}
	return nil
}
