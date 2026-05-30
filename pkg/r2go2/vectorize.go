package r2go2

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

// VectorizeIndex represents a Cloudflare Vectorize index.
type VectorizeIndex struct {
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Dimensions   int               `json:"dimensions"`
	Metric       string            `json:"metric"`
	VectorCount  int64             `json:"vectors_count,omitempty"`
	Config       *VectorizeConfig  `json:"config,omitempty"`
}

// VectorizeConfig holds index configuration details.
type VectorizeConfig struct {
	Dimensions int    `json:"dimensions"`
	Metric     string `json:"metric"`
}

// VectorizeVector represents a single vector for insert/query.
type VectorizeVector struct {
	ID       string                 `json:"id"`
	Values   []float64              `json:"values"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// VectorizeQueryResult holds a single query match.
type VectorizeQueryResult struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Values   []float64              `json:"values,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// VectorizeService implements Cloudflare Vectorize operations.
type VectorizeService struct {
	cf        *cloudflare.API
	accountID string
	apiToken  string
	baseURL   string
}

// NewVectorizeService creates a new Vectorize service client.
func NewVectorizeService(api *cloudflare.API, accountID string) (*VectorizeService, error) {
	if api == nil {
		return nil, validationError("NewVectorizeService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewVectorizeService", "account ID is required")
	}
	baseURL := api.BaseURL
	if baseURL == "" {
		baseURL = "https://api.cloudflare.com/client/v4"
	}
	return &VectorizeService{
		cf:        api,
		accountID: accountID,
		baseURL:   baseURL,
	}, nil
}

// NewVectorizeServiceFromCreds creates a VectorizeService from raw credentials.
func NewVectorizeServiceFromCreds(accountID, apiToken string) (*VectorizeService, error) {
	if accountID == "" {
		return nil, validationError("NewVectorizeServiceFromCreds", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewVectorizeServiceFromCreds", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, newError("NewVectorizeServiceFromCreds", "failed to create API client", err)
	}
	return &VectorizeService{
		cf:        cf,
		accountID: accountID,
		apiToken:  apiToken,
		baseURL:   "https://api.cloudflare.com/client/v4",
	}, nil
}

var validMetrics = map[string]bool{
	"cosine":      true,
	"euclidean":   true,
	"dot-product": true,
}

func isValidMetric(m string) bool {
	return validMetrics[m]
}

// CreateIndex creates a new Vectorize index.
func (s *VectorizeService) CreateIndex(ctx context.Context, name string, dimensions int, metric string) (*VectorizeIndex, error) {
	if name == "" {
		return nil, validationError("VectorizeService.CreateIndex", "index name is required")
	}
	if dimensions <= 0 {
		return nil, validationError("VectorizeService.CreateIndex", "dimensions must be positive")
	}
	if !isValidMetric(metric) {
		return nil, validationError("VectorizeService.CreateIndex", fmt.Sprintf("invalid metric %q, must be one of: cosine, euclidean, dot-product", metric))
	}

	body := map[string]interface{}{
		"name": name,
		"config": map[string]interface{}{
			"dimensions": dimensions,
			"metric":     metric,
		},
	}

	var result VectorizeIndex
	err := s.doJSON(ctx, "POST", s.indexesURL(), body, &result)
	if err != nil {
		return nil, newError("VectorizeService.CreateIndex", "failed to create index", err)
	}
	return &result, nil
}

// ListIndexes lists all Vectorize indexes.
func (s *VectorizeService) ListIndexes(ctx context.Context) ([]VectorizeIndex, error) {
	var result []VectorizeIndex
	err := s.doJSON(ctx, "GET", s.indexesURL(), nil, &result)
	if err != nil {
		return nil, newError("VectorizeService.ListIndexes", "failed to list indexes", err)
	}
	if result == nil {
		result = []VectorizeIndex{}
	}
	return result, nil
}

// GetIndex gets details of a specific index.
func (s *VectorizeService) GetIndex(ctx context.Context, name string) (*VectorizeIndex, error) {
	if name == "" {
		return nil, validationError("VectorizeService.GetIndex", "index name is required")
	}

	var result VectorizeIndex
	err := s.doJSON(ctx, "GET", s.indexURL(name), nil, &result)
	if err != nil {
		return nil, newError("VectorizeService.GetIndex", fmt.Sprintf("failed to get index %q", name), err)
	}
	return &result, nil
}

// DeleteIndex deletes a Vectorize index.
func (s *VectorizeService) DeleteIndex(ctx context.Context, name string) error {
	if name == "" {
		return validationError("VectorizeService.DeleteIndex", "index name is required")
	}

	err := s.doJSON(ctx, "DELETE", s.indexURL(name), nil, nil)
	if err != nil {
		return newError("VectorizeService.DeleteIndex", fmt.Sprintf("failed to delete index %q", name), err)
	}
	return nil
}

// InsertVectors inserts vectors into an index.
func (s *VectorizeService) InsertVectors(ctx context.Context, indexName string, vectors []VectorizeVector) error {
	if indexName == "" {
		return validationError("VectorizeService.InsertVectors", "index name is required")
	}
	if len(vectors) == 0 {
		return validationError("VectorizeService.InsertVectors", "at least one vector is required")
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, v := range vectors {
		if err := enc.Encode(v); err != nil {
			return newError("VectorizeService.InsertVectors", "failed to encode vector", err)
		}
	}

	url := fmt.Sprintf("%s/insert", s.indexURL(indexName))
	err := s.doRaw(ctx, "POST", url, "application/x-ndjson", &buf, nil)
	if err != nil {
		return newError("VectorizeService.InsertVectors", "failed to insert vectors", err)
	}
	return nil
}

// QueryVectors queries an index for nearest neighbors.
func (s *VectorizeService) QueryVectors(ctx context.Context, indexName string, values []float64, topK int) ([]VectorizeQueryResult, error) {
	if indexName == "" {
		return nil, validationError("VectorizeService.QueryVectors", "index name is required")
	}
	if len(values) == 0 {
		return nil, validationError("VectorizeService.QueryVectors", "query values are required")
	}
	if topK <= 0 {
		topK = 10
	}

	body := map[string]interface{}{
		"vector": values,
		"topK":   topK,
	}

	url := fmt.Sprintf("%s/query", s.indexURL(indexName))
	var result struct {
		Matches []VectorizeQueryResult `json:"matches"`
	}
	err := s.doJSON(ctx, "POST", url, body, &result)
	if err != nil {
		return nil, newError("VectorizeService.QueryVectors", "failed to query index", err)
	}
	return result.Matches, nil
}

func (s *VectorizeService) indexesURL() string {
	return fmt.Sprintf("%s/accounts/%s/vectorize/v2/indexes", s.baseURL, s.accountID)
}

func (s *VectorizeService) indexURL(name string) string {
	return fmt.Sprintf("%s/%s", s.indexesURL(), name)
}

type cfAPIResponse struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (s *VectorizeService) doJSON(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}
	return s.doRaw(ctx, method, url, "application/json", bodyReader, result)
}

func (s *VectorizeService) doRaw(ctx context.Context, method, url, contentType string, body io.Reader, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
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

	var apiResp cfAPIResponse
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
