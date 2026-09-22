package cosmoflare

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"time"
)

// d1UsageCapture records every request the stub server saw.
type d1UsageCapture struct {
	Path string
	Auth string
	Body string
}

// d1UsageStubServer spins up one server serving BOTH seams the D1 usage
// service talks to: POST /graphql for analytics and GET
// /accounts/{id}/d1/database/{db} for database facts. gqlBody is the raw
// GraphQL envelope; the REST endpoint replies with the fixed database
// payload below.
func d1UsageStubServer(t *testing.T, gqlBody string, cap *d1UsageCapture) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if cap != nil {
			cap.Path = r.URL.Path
			cap.Auth = r.Header.Get("Authorization")
			cap.Body = string(data)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/graphql":
			_, _ = w.Write([]byte(gqlBody))
		case strings.HasPrefix(r.URL.Path, "/accounts/acct-1/d1/database/"):
			_, _ = w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":{` +
				`"uuid":"db-1","name":"prod","version":"production","num_tables":7,"file_size":2048,` +
				`"created_at":"2026-01-01T00:00:00Z"}}`))
		default:
			http.Error(w, `{"success":false,"errors":[{"code":9999,"message":"unexpected path"}]}`, http.StatusNotFound)
		}
	}))
}

// d1UsageFixtureNow pins the test seam's clock so window math is exact.
func d1UsageFixtureNow() time.Time {
	return time.Date(2026, 9, 22, 12, 30, 0, 0, time.UTC)
}

// d1UsageService builds the service pair against the stub server.
func d1UsageService(t *testing.T, srv *httptest.Server) *D1UsageService {
	t.Helper()
	cf, err := cloudflare.NewWithAPIToken("tok", cloudflare.BaseURL(srv.URL))
	if err != nil {
		t.Fatalf("cloudflare client: %v", err)
	}
	d1, err := NewD1Service(cf, "acct-1")
	if err != nil {
		t.Fatalf("D1 service: %v", err)
	}
	analytics := NewAnalyticsService("acct-1", "tok", WithAnalyticsBaseURL(srv.URL))
	svc, err := NewD1UsageService(analytics, d1)
	if err != nil {
		t.Fatalf("D1 usage service: %v", err)
	}
	svc.nowFn = d1UsageFixtureNow
	return svc
}

// TestD1UsageDailyUsage verifies DailyUsage POSTs the d1AnalyticsAdaptiveGroups
// GraphQL document with bearer auth, bounds the window at (today-(days-1))
// to tomorrow exclusive, sums duplicate groups per date, and returns days
// sorted ascending.
func TestD1UsageDailyUsage(t *testing.T) {
	t.Parallel()
	var cap d1UsageCapture
	gql := `{"data":{"viewer":{"accounts":[{"d1AnalyticsAdaptiveGroups":[` +
		`{"sum":{"rowsRead":100,"rowsWritten":10},"dimensions":{"date":"2026-09-22","databaseId":"db-1"}},` +
		`{"sum":{"rowsRead":50,"rowsWritten":5},"dimensions":{"date":"2026-09-22","databaseId":"db-1"}},` +
		`{"sum":{"rowsRead":900,"rowsWritten":90},"dimensions":{"date":"2026-09-21","databaseId":"db-1"}}` +
		`]}]}}}`
	srv := d1UsageStubServer(t, gql, &cap)
	defer srv.Close()

	svc := d1UsageService(t, srv)
	usage, err := svc.DailyUsage(context.Background(), "db-1", 2)
	if err != nil {
		t.Fatalf("DailyUsage: %v", err)
	}

	if !strings.Contains(cap.Path, "/graphql") {
		t.Errorf("path = %s, want /graphql", cap.Path)
	}
	if !strings.Contains(cap.Auth, "Bearer tok") {
		t.Errorf("authorization = %q, want bearer tok", cap.Auth)
	}
	for _, want := range []string{"d1AnalyticsAdaptiveGroups", `"databaseId":"db-1"`, "2026-09-21", "2026-09-23"} {
		if !strings.Contains(cap.Body, want) {
			t.Errorf("request body missing %q: %s", want, cap.Body)
		}
	}

	if len(usage) != 2 {
		t.Fatalf("days = %d, want 2 (%+v)", len(usage), usage)
	}
	if usage[0].Date != "2026-09-21" || usage[0].RowsRead != 900 || usage[0].RowsWritten != 90 {
		t.Errorf("first day = %+v, want 2026-09-21 900/90", usage[0])
	}
	if usage[1].Date != "2026-09-22" || usage[1].RowsRead != 150 || usage[1].RowsWritten != 15 {
		t.Errorf("today = %+v, want 2026-09-22 with summed 150/15", usage[1])
	}
}

// TestD1UsageReport verifies Report combines the daily rows with the REST
// database facts (file size, table count) and computes window totals.
func TestD1UsageReport(t *testing.T) {
	t.Parallel()
	gql := `{"data":{"viewer":{"accounts":[{"d1AnalyticsAdaptiveGroups":[` +
		`{"sum":{"rowsRead":1000,"rowsWritten":100},"dimensions":{"date":"2026-09-22","databaseId":"db-1"}}` +
		`]}]}}}`
	srv := d1UsageStubServer(t, gql, nil)
	defer srv.Close()

	svc := d1UsageService(t, srv)
	report, err := svc.Report(context.Background(), "db-1", 1)
	if err != nil {
		t.Fatalf("Report: %v", err)
	}

	if report.TotalRowsRead != 1000 || report.TotalRowsWritten != 100 {
		t.Errorf("totals = %d/%d, want 1000/100", report.TotalRowsRead, report.TotalRowsWritten)
	}
	if report.FileSizeBytes != 2048 || report.NumTables != 7 {
		t.Errorf("db facts = %d bytes / %d tables, want 2048 / 7", report.FileSizeBytes, report.NumTables)
	}
	if today := report.Today(); today.RowsRead != 1000 {
		t.Errorf("Today() = %+v, want today's 1000 reads", today)
	}
}

// TestD1UsageGraphQLErrors verifies a GraphQL errors array (malformed or
// rejected query) surfaces as an error rather than an empty report.
func TestD1UsageGraphQLErrors(t *testing.T) {
	t.Parallel()
	srv := d1UsageStubServer(t, `{"errors":[{"message":"cannot query field"}]}`, nil)
	defer srv.Close()

	svc := d1UsageService(t, srv)
	if _, err := svc.DailyUsage(context.Background(), "db-1", 1); err == nil {
		t.Fatal("expected error for GraphQL errors array, got nil")
	}
}

// TestD1UsageMalformedEnvelope verifies a non-JSON body fails loudly.
func TestD1UsageMalformedEnvelope(t *testing.T) {
	t.Parallel()
	srv := d1UsageStubServer(t, `not-json`, nil)
	defer srv.Close()

	svc := d1UsageService(t, srv)
	if _, err := svc.DailyUsage(context.Background(), "db-1", 1); err == nil {
		t.Fatal("expected error for malformed response, got nil")
	}
}

// TestD1UsageValidation covers constructor and argument validation without
// touching the network.
func TestD1UsageValidation(t *testing.T) {
	t.Parallel()

	if _, err := NewD1UsageService(nil, nil); err == nil {
		t.Error("expected error for nil analytics service")
	}
	analytics := NewAnalyticsService("acct-1", "tok")
	if _, err := NewD1UsageService(analytics, nil); err == nil {
		t.Error("expected error for nil D1 service")
	}
	if _, err := NewD1UsageServiceFromCreds("", "tok"); err == nil {
		t.Error("expected error for empty account ID")
	}
	if _, err := NewD1UsageServiceFromCreds("acct-1", ""); err == nil {
		t.Error("expected error for empty API token")
	}

	svc, err := NewD1UsageService(analytics, &D1Service{})
	if err != nil {
		t.Fatalf("constructor: %v", err)
	}
	svc.nowFn = d1UsageFixtureNow

	if _, err := svc.DailyUsage(context.Background(), "", 1); err == nil {
		t.Error("expected error for empty database ID")
	}
	if _, err := svc.DailyUsage(context.Background(), "db-1", 0); err == nil {
		t.Error("expected error for days=0")
	}
	if _, err := svc.DailyUsage(context.Background(), "db-1", D1UsageMaxDays+1); err == nil {
		t.Error("expected error for days beyond retention")
	}
}

// TestD1UsageRequestIsJSON ensures the GraphQL request body is a valid
// {query, variables} JSON document — the shape Cloudflare's /graphql
// endpoint requires.
func TestD1UsageRequestIsJSON(t *testing.T) {
	t.Parallel()
	var cap d1UsageCapture
	srv := d1UsageStubServer(t, `{"data":{}}`, &cap)
	defer srv.Close()

	svc := d1UsageService(t, srv)
	if _, err := svc.DailyUsage(context.Background(), "db-1", 1); err != nil {
		t.Fatalf("DailyUsage: %v", err)
	}

	var req struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	if err := json.Unmarshal([]byte(cap.Body), &req); err != nil {
		t.Fatalf("body not JSON: %v (%s)", err, cap.Body)
	}
	if req.Query == "" || len(req.Variables) == 0 {
		t.Errorf("body missing query/variables: %s", cap.Body)
	}
	if req.Variables["accountTag"] != "acct-1" || req.Variables["databaseId"] != "db-1" {
		t.Errorf("variables = %v, want accountTag=acct-1 databaseId=db-1", req.Variables)
	}
}
