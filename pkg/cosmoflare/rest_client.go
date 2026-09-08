package cosmoflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// defaultRESTBaseURL is the Cloudflare v4 API root shared by the REST
// services that the cloudflare-go dependency does not wrap.
const defaultRESTBaseURL = "https://api.cloudflare.com/client/v4"

// restClient is the shared transport for the hand-rolled REST services
// (bucket domains, lifecycle, notifications): bearer auth, the optional
// cf-r2-jurisdiction header, and the standard response envelope. Services
// embed it and add only their paths, validation, and typed methods.
type restClient struct {
	apiToken     string
	httpClient   *http.Client
	baseURL      string
	jurisdiction string // optional cf-r2-jurisdiction: default|eu|us|fedramp
}

func newRESTClient(apiToken string) restClient {
	return restClient{
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultRESTBaseURL,
	}
}

// apiEnvelope is the standard Cloudflare API response wrapper.
type apiEnvelope struct {
	Success bool            `json:"success"`
	Errors  []apiErrorItem  `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

type apiErrorItem struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// do performs an authenticated request against the API and decodes the
// standard envelope. When out is non-nil the raw result is unmarshalled
// into it.
func (c *restClient) do(ctx context.Context, op, method, path string, body interface{}, out interface{}) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return newError(op, "failed to encode request body", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return newError(op, "failed to build request", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	if c.jurisdiction != "" {
		req.Header.Set("cf-r2-jurisdiction", c.jurisdiction)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return newError(op, "request failed", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return newError(op, "failed to read response body", err)
	}

	var env apiEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return newError(op, fmt.Sprintf("unexpected response (HTTP %d)", resp.StatusCode), err)
	}
	if !env.Success {
		msg := fmt.Sprintf("API returned errors (HTTP %d)", resp.StatusCode)
		if len(env.Errors) > 0 {
			msg = env.Errors[0].Message
		}
		return newError(op, msg, nil)
	}
	if out != nil && len(env.Result) > 0 {
		if err := json.Unmarshal(env.Result, out); err != nil {
			return newError(op, "failed to decode result", err)
		}
	}
	return nil
}
