package cosmoflare

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// analyticsCapture records the last request seen by a test server.
type analyticsCapture struct {
	Method string
	Path   string
	Auth   string
	Body   string
}

// analyticsServer starts an httptest server responding with body for every
// request and recording requests in cap.
func analyticsServer(t *testing.T, body string, cap *analyticsCapture) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cap != nil {
			data, _ := io.ReadAll(r.Body)
			cap.Method = r.Method
			cap.Path = r.URL.Path
			cap.Auth = r.Header.Get("Authorization")
			cap.Body = string(data)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
}

func analyticsWindow() AnalyticsWindow {
	return AnalyticsWindow{
		Start: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
	}
}

func TestAnalyticsZoneHTTP(t *testing.T) {
	var cap analyticsCapture
	resp := `{"data":{"viewer":{"zones":[{"httpRequestsAdaptiveGroups":[` +
		`{"sum":{"count":10,"edgeResponseBytes":2048,"visits":3},"dimensions":{"datetimeHour":"2026-08-01T00:00:00Z"}},` +
		`{"sum":{"count":5,"edgeResponseBytes":512,"visits":1},"dimensions":{"datetimeHour":"2026-08-01T01:00:00Z"}}` +
		`]}]}}}`
	srv := analyticsServer(t, resp, &cap)
	defer srv.Close()

	s := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))
	got, err := s.ZoneHTTP(context.Background(), "zone123", analyticsWindow())
	if err != nil {
		t.Fatalf("ZoneHTTP: %v", err)
	}

	if cap.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", cap.Method)
	}
	if cap.Path != "/graphql" {
		t.Errorf("path = %s, want /graphql", cap.Path)
	}
	if cap.Auth != "Bearer tok" {
		t.Errorf("authorization = %q, want %q", cap.Auth, "Bearer tok")
	}
	if !strings.Contains(cap.Body, "httpRequestsAdaptiveGroups") {
		t.Errorf("body missing httpRequestsAdaptiveGroups: %s", cap.Body)
	}
	if !strings.Contains(cap.Body, "2026-08-01T00:00:00Z") {
		t.Errorf("body missing RFC3339 start datetime: %s", cap.Body)
	}
	var reqBody map[string]any
	if err := json.Unmarshal([]byte(cap.Body), &reqBody); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if reqBody["query"] == "" {
		t.Errorf("body missing query field: %s", cap.Body)
	}

	want := ZoneHTTPSummary{Requests: 15, Bytes: 2560, Visits: 4}
	if *got != want {
		t.Errorf("summary = %+v, want %+v", *got, want)
	}
}

func TestAnalyticsR2StorageGroups(t *testing.T) {
	resp := `{"data":{"viewer":{"accounts":[{"r2StorageAdaptiveGroups":[` +
		`{"max":{"objectCount":100,"uploadCount":2,"payloadSize":1000,"metadataSize":10},"dimensions":{"bucketName":"media"}},` +
		`{"max":{"objectCount":200,"uploadCount":1,"payloadSize":900,"metadataSize":20},"dimensions":{"bucketName":"media"}},` +
		`{"max":{"objectCount":50,"uploadCount":0,"payloadSize":500,"metadataSize":5},"dimensions":{"bucketName":"backups"}}` +
		`]}]}}}`
	srv := analyticsServer(t, resp, &analyticsCapture{})
	defer srv.Close()

	s := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))
	got, err := s.R2Storage(context.Background(), analyticsWindow())
	if err != nil {
		t.Fatalf("R2Storage: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d buckets, want 2: %+v", len(got), got)
	}
	byBucket := map[string]R2BucketStorage{}
	for _, b := range got {
		byBucket[b.Bucket] = b
	}
	media := byBucket["media"]
	if media.ObjectCount != 200 || media.UploadCount != 2 || media.PayloadSize != 1000 || media.MetadataSize != 20 {
		t.Errorf("media = %+v, want max-aggregated values", media)
	}
	backups := byBucket["backups"]
	if backups.ObjectCount != 50 || backups.PayloadSize != 500 {
		t.Errorf("backups = %+v", backups)
	}
}

func TestAnalyticsR2OperationsAggregates(t *testing.T) {
	resp := `{"data":{"viewer":{"accounts":[{"r2OperationsAdaptiveGroups":[` +
		`{"sum":{"requests":10},"dimensions":{"actionType":"GetObject","actionStatus":"success","bucketName":"media"}},` +
		`{"sum":{"requests":5},"dimensions":{"actionType":"GetObject","actionStatus":"success","bucketName":"media"}},` +
		`{"sum":{"requests":2},"dimensions":{"actionType":"GetObject","actionStatus":"userError","bucketName":"media"}},` +
		`{"sum":{"requests":7},"dimensions":{"actionType":"PutObject","actionStatus":"success","bucketName":"backups"}}` +
		`]}]}}}`
	srv := analyticsServer(t, resp, &analyticsCapture{})
	defer srv.Close()

	s := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))
	got, err := s.R2Operations(context.Background(), analyticsWindow())
	if err != nil {
		t.Fatalf("R2Operations: %v", err)
	}
	type key struct{ bucket, action, status string }
	counts := map[key]uint64{}
	for _, r := range got {
		counts[key{r.Bucket, r.Action, r.Status}] += r.Requests
	}
	if len(got) != 3 {
		t.Fatalf("got %d rows, want 3: %+v", len(got), got)
	}
	if counts[key{"media", "GetObject", "success"}] != 15 {
		t.Errorf("media GetObject success = %d, want 15", counts[key{"media", "GetObject", "success"}])
	}
	if counts[key{"media", "GetObject", "userError"}] != 2 {
		t.Errorf("media GetObject userError = %d, want 2", counts[key{"media", "GetObject", "userError"}])
	}
	if counts[key{"backups", "PutObject", "success"}] != 7 {
		t.Errorf("backups PutObject success = %d, want 7", counts[key{"backups", "PutObject", "success"}])
	}
}

func TestAnalyticsWorkers(t *testing.T) {
	resp := `{"data":{"viewer":{"accounts":[{"workersInvocationsAdaptive":[` +
		`{"sum":{"requests":100,"errors":3,"subrequests":40},"quantiles":{"cpuTimeP50":1.5,"cpuTimeP99":9.25},"dimensions":{"scriptName":"api"}},` +
		`{"sum":{"requests":50,"errors":0,"subrequests":10},"quantiles":{"cpuTimeP50":0.5,"cpuTimeP99":2.0},"dimensions":{"scriptName":"cron"}}` +
		`]}]}}}`
	srv := analyticsServer(t, resp, &analyticsCapture{})
	defer srv.Close()

	s := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))
	got, err := s.Workers(context.Background(), analyticsWindow())
	if err != nil {
		t.Fatalf("Workers: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d scripts, want 2: %+v", len(got), got)
	}
	byScript := map[string]WorkersSummary{}
	for _, w := range got {
		byScript[w.Script] = w
	}
	api := byScript["api"]
	if api.Requests != 100 || api.Errors != 3 || api.Subrequests != 40 {
		t.Errorf("api = %+v", api)
	}
	if api.CPUP50 != 1.5 || api.CPUP99 != 9.25 {
		t.Errorf("api quantiles = %+v", api)
	}
	if byScript["cron"].CPUP99 != 2.0 {
		t.Errorf("cron = %+v", byScript["cron"])
	}
}

func TestAnalyticsGraphQLErrors(t *testing.T) {
	srv := analyticsServer(t, `{"errors":[{"message":"boom"}]}`, &analyticsCapture{})
	defer srv.Close()

	s := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))
	_, err := s.Workers(context.Background(), analyticsWindow())
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error %q missing boom", err)
	}
}

func TestAnalyticsWindowValidation(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	s := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))
	ctx := context.Background()

	// 40-day window exceeds R2 retention (31 days).
	w40 := AnalyticsWindow{
		Start: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
	}
	if _, err := s.R2Storage(ctx, w40); err == nil {
		t.Error("40-day R2 window: want error, got nil")
	} else if _, ok := err.(*R2ValidationError); !ok {
		t.Errorf("40-day R2 window: want *R2ValidationError, got %T", err)
	}
	if hit {
		t.Error("server was hit despite validation failure")
	}

	// Start >= End is invalid.
	wBad := AnalyticsWindow{
		Start: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}
	if _, err := s.ZoneHTTP(ctx, "zone1", wBad); err == nil {
		t.Error("Start>=End: want error, got nil")
	}
	if hit {
		t.Error("server was hit despite validation failure")
	}
}
