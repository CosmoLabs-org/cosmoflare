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

// AnalyticsWindow bounds a query. Start inclusive, End exclusive.
type AnalyticsWindow struct {
	Start time.Time
	End   time.Time
}

// ZoneHTTPSummary is aggregated HTTP traffic for one zone over a window.
type ZoneHTTPSummary struct {
	Requests uint64 `json:"requests"`
	Bytes    uint64 `json:"bytes"` // edgeResponseBytes summed
	Visits   uint64 `json:"visits"`
}

// R2BucketStorage is point-in-time storage state for one bucket.
type R2BucketStorage struct {
	Bucket       string `json:"bucket"`
	ObjectCount  uint64 `json:"object_count"`
	UploadCount  uint64 `json:"upload_count"` // pending multipart uploads
	PayloadSize  uint64 `json:"payload_size"`
	MetadataSize uint64 `json:"metadata_size"`
}

// R2OperationCount is request volume for one action type on one bucket.
type R2OperationCount struct {
	Bucket   string `json:"bucket"`
	Action   string `json:"action"` // e.g. GetObject, PutObject, ListObjects
	Status   string `json:"status"` // success | userError | internalError
	Requests uint64 `json:"requests"`
}

// WorkersSummary is invocation volume for one script.
type WorkersSummary struct {
	Script      string  `json:"script"`
	Requests    uint64  `json:"requests"`
	Errors      uint64  `json:"errors"`
	Subrequests uint64  `json:"subrequests"`
	CPUP50      float64 `json:"cpu_time_p50"`
	CPUP99      float64 `json:"cpu_time_p99"`
}

// AnalyticsService queries Cloudflare's GraphQL Analytics API.
type AnalyticsService struct {
	accountID  string
	apiToken   string
	httpClient *http.Client
	baseURL    string // default https://api.cloudflare.com/client/v4
}

// NewAnalyticsService creates a service for querying GraphQL Analytics.
func NewAnalyticsService(accountID, apiToken string, opts ...AnalyticsOption) *AnalyticsService {
	s := &AnalyticsService{
		accountID:  accountID,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.cloudflare.com/client/v4",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// AnalyticsOption configures the service.
type AnalyticsOption func(*AnalyticsService)

// WithAnalyticsHTTPClient sets a custom HTTP client.
func WithAnalyticsHTTPClient(c *http.Client) AnalyticsOption {
	return func(s *AnalyticsService) { s.httpClient = c }
}

// WithAnalyticsBaseURL overrides the GraphQL API base URL.
func WithAnalyticsBaseURL(u string) AnalyticsOption {
	return func(s *AnalyticsService) { s.baseURL = u }
}

// analyticsGQLEnvelope is the GraphQL response wrapper.
type analyticsGQLEnvelope struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// query posts a GraphQL document with the given variables and decodes the
// "data" member of the response into out. A non-empty "errors" array is
// surfaced as an error carrying the first message.
func (s *AnalyticsService) query(ctx context.Context, op, gql string, vars map[string]any, out any) error {
	payload, err := json.Marshal(struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}{Query: gql, Variables: vars})
	if err != nil {
		return newError(op, "failed to encode request body", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/graphql", bytes.NewReader(payload))
	if err != nil {
		return newError(op, "failed to build request", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return newError(op, "request failed", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return newError(op, "failed to read response body", err)
	}

	var env analyticsGQLEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return newError(op, fmt.Sprintf("unexpected response (HTTP %d)", resp.StatusCode), err)
	}
	if len(env.Errors) > 0 {
		return newError(op, env.Errors[0].Message, nil)
	}
	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return newError(op, "failed to decode data", err)
		}
	}
	return nil
}

// validateWindow checks a window for a maximum span. R2 datasets retain 31
// days; Workers retains dates up to three months back (92 days here).
func validateAnalyticsWindow(op string, w AnalyticsWindow, max time.Duration) error {
	if !w.Start.Before(w.End) {
		return validationError(op, "window Start must be before End")
	}
	if max > 0 && w.End.Sub(w.Start) > max {
		return validationError(op, fmt.Sprintf("window spans %s, exceeding the %s analytics retention limit", w.End.Sub(w.Start), max))
	}
	return nil
}

// ZoneHTTP returns traffic for one zone over the window.
func (s *AnalyticsService) ZoneHTTP(ctx context.Context, zoneID string, w AnalyticsWindow) (*ZoneHTTPSummary, error) {
	const op = "AnalyticsZoneHTTP"
	if s.apiToken == "" {
		return nil, validationError(op, "API token is required")
	}
	if zoneID == "" {
		return nil, validationError(op, "zone ID is required")
	}
	if err := validateAnalyticsWindow(op, w, 0); err != nil {
		return nil, err
	}
	gql := `query($zoneTag: String!, $start: Time!, $end: Time!) {
  viewer {
    zones(filter: {zoneTag: $zoneTag}) {
      httpRequestsAdaptiveGroups(limit: 10000, filter: {datetime_geq: $start, datetime_leq: $end, requestSource: "eyeball"}) {
        sum { count edgeResponseBytes visits }
        dimensions { datetimeHour }
      }
    }
  }
}`
	var out struct {
		Viewer struct {
			Zones []struct {
				HTTPRequestsAdaptiveGroups []struct {
					Sum struct {
						Count             uint64 `json:"count"`
						EdgeResponseBytes uint64 `json:"edgeResponseBytes"`
						Visits            uint64 `json:"visits"`
					} `json:"sum"`
				} `json:"httpRequestsAdaptiveGroups"`
			} `json:"zones"`
		} `json:"viewer"`
	}
	vars := map[string]any{
		"zoneTag": zoneID,
		"start":   w.Start.Format(time.RFC3339),
		"end":     w.End.Format(time.RFC3339),
	}
	if err := s.query(ctx, op, gql, vars, &out); err != nil {
		return nil, err
	}
	summary := &ZoneHTTPSummary{}
	for _, z := range out.Viewer.Zones {
		for _, g := range z.HTTPRequestsAdaptiveGroups {
			summary.Requests += g.Sum.Count
			summary.Bytes += g.Sum.EdgeResponseBytes
			summary.Visits += g.Sum.Visits
		}
	}
	return summary, nil
}

// R2Storage returns per-bucket storage state (latest point per bucket).
func (s *AnalyticsService) R2Storage(ctx context.Context, w AnalyticsWindow) ([]R2BucketStorage, error) {
	const op = "AnalyticsR2Storage"
	if err := s.validateAccount(op); err != nil {
		return nil, err
	}
	if err := validateAnalyticsWindow(op, w, 31*24*time.Hour); err != nil {
		return nil, err
	}
	gql := `query($accountTag: String!, $start: Time!, $end: Time!) {
  viewer {
    accounts(filter: {accountTag: $accountTag}) {
      r2StorageAdaptiveGroups(limit: 10000, filter: {datetime_geq: $start, datetime_leq: $end}) {
        max { objectCount uploadCount payloadSize metadataSize }
        dimensions { datetime bucketName }
      }
    }
  }
}`
	var out struct {
		Viewer struct {
			Accounts []struct {
				R2StorageAdaptiveGroups []struct {
					Max struct {
						ObjectCount  uint64 `json:"objectCount"`
						UploadCount  uint64 `json:"uploadCount"`
						PayloadSize  uint64 `json:"payloadSize"`
						MetadataSize uint64 `json:"metadataSize"`
					} `json:"max"`
					Dimensions struct {
						BucketName string `json:"bucketName"`
					} `json:"dimensions"`
				} `json:"r2StorageAdaptiveGroups"`
			} `json:"accounts"`
		} `json:"viewer"`
	}
	if err := s.query(ctx, op, gql, s.windowVars(w), &out); err != nil {
		return nil, err
	}
	type acc struct {
		objectCount, uploadCount, payloadSize, metadataSize uint64
	}
	byBucket := map[string]*acc{}
	for _, a := range out.Viewer.Accounts {
		for _, g := range a.R2StorageAdaptiveGroups {
			name := g.Dimensions.BucketName
			cur, ok := byBucket[name]
			if !ok {
				cur = &acc{}
				byBucket[name] = cur
			}
			cur.objectCount = max(cur.objectCount, g.Max.ObjectCount)
			cur.uploadCount = max(cur.uploadCount, g.Max.UploadCount)
			cur.payloadSize = max(cur.payloadSize, g.Max.PayloadSize)
			cur.metadataSize = max(cur.metadataSize, g.Max.MetadataSize)
		}
	}
	buckets := make([]R2BucketStorage, 0, len(byBucket))
	for name, a := range byBucket {
		buckets = append(buckets, R2BucketStorage{
			Bucket:       name,
			ObjectCount:  a.objectCount,
			UploadCount:  a.uploadCount,
			PayloadSize:  a.payloadSize,
			MetadataSize: a.metadataSize,
		})
	}
	return buckets, nil
}

// R2Operations returns per-action request counts across buckets over the window.
func (s *AnalyticsService) R2Operations(ctx context.Context, w AnalyticsWindow) ([]R2OperationCount, error) {
	const op = "AnalyticsR2Operations"
	if err := s.validateAccount(op); err != nil {
		return nil, err
	}
	if err := validateAnalyticsWindow(op, w, 31*24*time.Hour); err != nil {
		return nil, err
	}
	gql := `query($accountTag: String!, $start: Time!, $end: Time!) {
  viewer {
    accounts(filter: {accountTag: $accountTag}) {
      r2OperationsAdaptiveGroups(limit: 10000, filter: {datetime_geq: $start, datetime_leq: $end}) {
        sum { requests }
        dimensions { datetime actionType actionStatus bucketName }
      }
    }
  }
}`
	var out struct {
		Viewer struct {
			Accounts []struct {
				R2OperationsAdaptiveGroups []struct {
					Sum struct {
						Requests uint64 `json:"requests"`
					} `json:"sum"`
					Dimensions struct {
						ActionType   string `json:"actionType"`
						ActionStatus string `json:"actionStatus"`
						BucketName   string `json:"bucketName"`
					} `json:"dimensions"`
				} `json:"r2OperationsAdaptiveGroups"`
			} `json:"accounts"`
		} `json:"viewer"`
	}
	if err := s.query(ctx, op, gql, s.windowVars(w), &out); err != nil {
		return nil, err
	}
	type key struct{ bucket, action, status string }
	totals := map[key]uint64{}
	for _, a := range out.Viewer.Accounts {
		for _, g := range a.R2OperationsAdaptiveGroups {
			k := key{g.Dimensions.BucketName, g.Dimensions.ActionType, g.Dimensions.ActionStatus}
			totals[k] += g.Sum.Requests
		}
	}
	rows := make([]R2OperationCount, 0, len(totals))
	for k, n := range totals {
		rows = append(rows, R2OperationCount{Bucket: k.bucket, Action: k.action, Status: k.status, Requests: n})
	}
	return rows, nil
}

// Workers returns per-script invocation summaries over the window.
func (s *AnalyticsService) Workers(ctx context.Context, w AnalyticsWindow) ([]WorkersSummary, error) {
	const op = "AnalyticsWorkers"
	if err := s.validateAccount(op); err != nil {
		return nil, err
	}
	if err := validateAnalyticsWindow(op, w, 92*24*time.Hour); err != nil {
		return nil, err
	}
	gql := `query($accountTag: String!, $start: Time!, $end: Time!) {
  viewer {
    accounts(filter: {accountTag: $accountTag}) {
      workersInvocationsAdaptive(limit: 10000, filter: {datetime_geq: $start, datetime_leq: $end}) {
        sum { requests errors subrequests }
        quantiles { cpuTimeP50 cpuTimeP99 }
        dimensions { datetime scriptName status }
      }
    }
  }
}`
	var out struct {
		Viewer struct {
			Accounts []struct {
				WorkersInvocationsAdaptive []struct {
					Sum struct {
						Requests    uint64 `json:"requests"`
						Errors      uint64 `json:"errors"`
						Subrequests uint64 `json:"subrequests"`
					} `json:"sum"`
					Quantiles struct {
						CPUTimeP50 float64 `json:"cpuTimeP50"`
						CPUTimeP99 float64 `json:"cpuTimeP99"`
					} `json:"quantiles"`
					Dimensions struct {
						ScriptName string `json:"scriptName"`
					} `json:"dimensions"`
				} `json:"workersInvocationsAdaptive"`
			} `json:"accounts"`
		} `json:"viewer"`
	}
	if err := s.query(ctx, op, gql, s.windowVars(w), &out); err != nil {
		return nil, err
	}
	type acc struct {
		requests, errors, subrequests uint64
		cpuP50, cpuP99                float64
	}
	byScript := map[string]*acc{}
	for _, a := range out.Viewer.Accounts {
		for _, g := range a.WorkersInvocationsAdaptive {
			name := g.Dimensions.ScriptName
			cur, ok := byScript[name]
			if !ok {
				cur = &acc{}
				byScript[name] = cur
			}
			cur.requests += g.Sum.Requests
			cur.errors += g.Sum.Errors
			cur.subrequests += g.Sum.Subrequests
			if g.Quantiles.CPUTimeP50 > cur.cpuP50 {
				cur.cpuP50 = g.Quantiles.CPUTimeP50
			}
			if g.Quantiles.CPUTimeP99 > cur.cpuP99 {
				cur.cpuP99 = g.Quantiles.CPUTimeP99
			}
		}
	}
	scripts := make([]WorkersSummary, 0, len(byScript))
	for name, a := range byScript {
		scripts = append(scripts, WorkersSummary{
			Script:      name,
			Requests:    a.requests,
			Errors:      a.errors,
			Subrequests: a.subrequests,
			CPUP50:      a.cpuP50,
			CPUP99:      a.cpuP99,
		})
	}
	return scripts, nil
}

// validateAccount ensures account-scoped queries have credentials.
func (s *AnalyticsService) validateAccount(op string) error {
	if s.accountID == "" {
		return validationError(op, "account ID is required")
	}
	if s.apiToken == "" {
		return validationError(op, "API token is required")
	}
	return nil
}

// windowVars builds the common GraphQL variables for an account-scoped window query.
func (s *AnalyticsService) windowVars(w AnalyticsWindow) map[string]any {
	return map[string]any{
		"accountTag": s.accountID,
		"start":      w.Start.Format(time.RFC3339),
		"end":        w.End.Format(time.RFC3339),
	}
}
