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
	cf, err := newCloudflareAPI(apiToken)
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

// UpsertResult is the response from an UpsertVectors call.
type UpsertResult struct {
	MutationID string `json:"mutationId"`
	Count      int    `json:"count,omitempty"`
}

// DeleteResult is the response from a DeleteVectors call.
type DeleteResult struct {
	MutationID string `json:"mutationId"`
	Count      int    `json:"count,omitempty"`
}

// maxUpsertVectorsBatch caps vectors per UpsertVectors call. Cloudflare does
// not document an explicit upsert batch limit; 50 mirrors the NDJSON insert
// convention used elsewhere in this service until confirmed otherwise.
const maxUpsertVectorsBatch = 50

// UpsertVectors inserts or updates vectors in an index, overwriting any
// vector that already exists with a matching ID.
func (s *VectorizeService) UpsertVectors(ctx context.Context, indexName string, vectors []VectorizeVector) (*UpsertResult, error) {
	if indexName == "" {
		return nil, validationError("VectorizeService.UpsertVectors", "index name is required")
	}
	if len(vectors) == 0 {
		return nil, validationError("VectorizeService.UpsertVectors", "at least one vector is required")
	}
	if len(vectors) > maxUpsertVectorsBatch {
		return nil, validationError("VectorizeService.UpsertVectors", fmt.Sprintf("too many vectors: %d (maximum %d per call)", len(vectors), maxUpsertVectorsBatch))
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, v := range vectors {
		if err := enc.Encode(v); err != nil {
			return nil, newError("VectorizeService.UpsertVectors", "failed to encode vector", err)
		}
	}

	url := fmt.Sprintf("%s/upsert", s.indexURL(indexName))
	var result UpsertResult
	err := s.doRaw(ctx, "POST", url, "application/x-ndjson", &buf, &result)
	if err != nil {
		return nil, newError("VectorizeService.UpsertVectors", "failed to upsert vectors", err)
	}
	return &result, nil
}

// GetVector fetches a single vector by ID. A missing vector returns an
// error satisfying isNotFound.
func (s *VectorizeService) GetVector(ctx context.Context, indexName, id string) (*VectorizeVector, error) {
	if indexName == "" {
		return nil, validationError("VectorizeService.GetVector", "index name is required")
	}
	if id == "" {
		return nil, validationError("VectorizeService.GetVector", "vector id is required")
	}

	body := map[string]interface{}{"ids": []string{id}}
	url := fmt.Sprintf("%s/get_by_ids", s.indexURL(indexName))
	var results []VectorizeVector
	err := s.doJSON(ctx, "POST", url, body, &results)
	if err != nil {
		if isNotFound(err) {
			return nil, notFound("VectorizeService.GetVector", indexName, id, err)
		}
		return nil, newError("VectorizeService.GetVector", fmt.Sprintf("failed to get vector %q", id), err)
	}
	if len(results) == 0 {
		return nil, notFound("VectorizeService.GetVector", indexName, id, nil)
	}
	return &results[0], nil
}

// DeleteVectors deletes vectors from an index by ID.
func (s *VectorizeService) DeleteVectors(ctx context.Context, indexName string, ids []string) (*DeleteResult, error) {
	if indexName == "" {
		return nil, validationError("VectorizeService.DeleteVectors", "index name is required")
	}
	if len(ids) == 0 {
		return nil, validationError("VectorizeService.DeleteVectors", "at least one id is required")
	}

	body := map[string]interface{}{"ids": ids}
	url := fmt.Sprintf("%s/delete_by_ids", s.indexURL(indexName))
	var result DeleteResult
	err := s.doJSON(ctx, "POST", url, body, &result)
	if err != nil {
		return nil, newError("VectorizeService.DeleteVectors", "failed to delete vectors", err)
	}
	return &result, nil
}

// ListNamespaces lists the distinct vector namespaces present in an index.
// NOTE: Cloudflare's public docs do not enumerate a dedicated namespaces
// endpoint at the time of writing; this follows the index-info endpoint
// convention (GET .../indexes/{name}/info) and reads a "namespaces" field
// from the result. Verify against live docs before relying on this in
// production; adjust the path/shape if the API differs.
func (s *VectorizeService) ListNamespaces(ctx context.Context, indexName string) ([]string, error) {
	if indexName == "" {
		return nil, validationError("VectorizeService.ListNamespaces", "index name is required")
	}

	url := fmt.Sprintf("%s/info", s.indexURL(indexName))
	var result struct {
		Namespaces []string `json:"namespaces"`
	}
	err := s.doJSON(ctx, "GET", url, nil, &result)
	if err != nil {
		return nil, newError("VectorizeService.ListNamespaces", fmt.Sprintf("failed to list namespaces for index %q", indexName), err)
	}
	if result.Namespaces == nil {
		result.Namespaces = []string{}
	}
	return result.Namespaces, nil
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

	resp, err := controlPlaneClient().Do(req)
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
