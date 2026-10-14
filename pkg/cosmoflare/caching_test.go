package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- pure trap assessment (assessHTMLCacheTrap) ---

func TestAssessHTMLCacheTrap_Trap(t *testing.T) {
	probe := &ProbeResult{
		StatusCode:    200,
		ContentType:   "text/html; charset=utf-8",
		CacheControl:  "public, s-maxage=600",
		CFCacheStatus: "DYNAMIC",
	}
	trapped, signals := assessHTMLCacheTrap(probe, true, 0)
	if !trapped {
		t.Fatalf("expected trap, signals: %s", signals)
	}
	for _, want := range []string{"s-maxage=600", "DYNAMIC"} {
		if !strings.Contains(signals, want) {
			t.Errorf("signals %q missing %q", signals, want)
		}
	}
	if !strings.Contains(signals, "no zone Cache Rule") {
		t.Errorf("signals should note the missing Cache Rule: %q", signals)
	}
}

func TestAssessHTMLCacheTrap_CleanWhenRuleExists(t *testing.T) {
	probe := &ProbeResult{
		StatusCode:    200,
		ContentType:   "text/html",
		CacheControl:  "public, s-maxage=600",
		CFCacheStatus: "HIT",
	}
	trapped, _ := assessHTMLCacheTrap(probe, true, 3)
	if trapped {
		t.Fatal("zone with 3 cache rules and cf-cache-status HIT must not be a trap")
	}
}

func TestAssessHTMLCacheTrap_NonHTML(t *testing.T) {
	probe := &ProbeResult{
		StatusCode:    200,
		ContentType:   "application/json",
		CacheControl:  "public, s-maxage=600",
		CFCacheStatus: "DYNAMIC",
	}
	trapped, _ := assessHTMLCacheTrap(probe, true, 0)
	if trapped {
		t.Fatal("non-HTML content must never be a trap")
	}
}

func TestAssessHTMLCacheTrap_NoCacheableHeader(t *testing.T) {
	probe := &ProbeResult{
		StatusCode:    200,
		ContentType:   "text/html",
		CacheControl:  "no-store",
		CFCacheStatus: "DYNAMIC",
	}
	trapped, _ := assessHTMLCacheTrap(probe, true, 0)
	if trapped {
		t.Fatal("no-store on HTML is honest, not a trap")
	}
}

func TestAssessHTMLCacheTrap_ShortMaxAge(t *testing.T) {
	probe := &ProbeResult{
		StatusCode:    200,
		ContentType:   "text/html",
		CacheControl:  "public, max-age=30",
		CFCacheStatus: "DYNAMIC",
	}
	trapped, _ := assessHTMLCacheTrap(probe, true, 0)
	if trapped {
		t.Fatal("max-age below the 60s floor is not a caching attempt")
	}
}

// --- CachingService.DetectHTMLCacheTrap (probe + rules wired) ---

func newCachingTestService(t *testing.T, rulesBody string, rulesStatus int, probe *ProbeResult) *CachingService {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cache/rules") {
			t.Errorf("unexpected REST call %s", r.URL.Path)
		}
		w.WriteHeader(rulesStatus)
		fmt.Fprint(w, rulesBody)
	}))
	t.Cleanup(srv.Close)
	return NewCachingService("tok", WithCachingBaseURL(srv.URL), WithCachingProbe(func(ctx context.Context, url string) (*ProbeResult, error) {
		return probe, nil
	}))
}

func TestDetectHTMLCacheTrap_FindsTrap(t *testing.T) {
	svc := newCachingTestService(t,
		`{"success":true,"result":[]}`, 200,
		&ProbeResult{StatusCode: 200, ContentType: "text/html", CacheControl: "public, s-maxage=300", CFCacheStatus: "DYNAMIC"})
	f, err := svc.DetectHTMLCacheTrap(context.Background(), Zone{ID: "z1", Name: "example.com"})
	if err != nil {
		t.Fatalf("DetectHTMLCacheTrap: %v", err)
	}
	if f == nil {
		t.Fatal("expected a finding for a trapped zone")
	}
	if f.Check != "html-cache-trap" || f.Severity != "warning" || f.Resource != "zone:example.com" {
		t.Fatalf("finding = %+v", f)
	}
	if f.Evidence == "" || f.Fix == "" {
		t.Fatal("finding must carry evidence and fix")
	}
}

func TestDetectHTMLCacheTrap_CleanZoneNil(t *testing.T) {
	svc := newCachingTestService(t,
		`{"success":true,"result":[{"id":"r1"}]}`, 200,
		&ProbeResult{StatusCode: 200, ContentType: "text/html", CacheControl: "public, s-maxage=300", CFCacheStatus: "HIT"})
	f, err := svc.DetectHTMLCacheTrap(context.Background(), Zone{ID: "z1", Name: "example.com"})
	if err != nil {
		t.Fatalf("DetectHTMLCacheTrap: %v", err)
	}
	if f != nil {
		t.Fatalf("clean zone must return nil finding, got %+v", f)
	}
}

func TestDetectHTMLCacheTrap_RulesNotVisibleIsStillATrap(t *testing.T) {
	// 403 on cache/rules = rules not visible; the probe evidence stands alone.
	svc := newCachingTestService(t,
		`{"success":false,"errors":[{"message":"forbidden"}]}`, 403,
		&ProbeResult{StatusCode: 200, ContentType: "text/html", CacheControl: "public, s-maxage=300", CFCacheStatus: "DYNAMIC"})
	f, err := svc.DetectHTMLCacheTrap(context.Background(), Zone{ID: "z1", Name: "example.com"})
	if err != nil {
		t.Fatalf("DetectHTMLCacheTrap: %v", err)
	}
	if f == nil {
		t.Fatal("trap evidence stands even when rules are not visible")
	}
	if !strings.Contains(f.Evidence, "not visible") {
		t.Errorf("evidence should say rules not visible: %q", f.Evidence)
	}
}

// --- EfficiencyService wiring (zone above floor gets probed) ---

func TestEfficiencyHTMLCacheTrapWiring(t *testing.T) {
	gql := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := readBody(t, r)
		switch {
		case strings.Contains(body, `"zoneTag":"z-trap"`):
			// Enough traffic to pass the 1,000-request coverage floor.
			fmt.Fprint(w, `{"data":{"viewer":{"zones":[{"httpRequests1dGroups":[`+
				`{"sum":{"requests":5000,"cachedRequests":10},"dimensions":{"date":"2026-10-08"}}]}]}}}`)
		case strings.Contains(body, `"zoneTag":"z-quiet"`):
			fmt.Fprint(w, `{"data":{"viewer":{"zones":[{"httpRequests1dGroups":[`+
				`{"sum":{"requests":50,"cachedRequests":0},"dimensions":{"date":"2026-10-08"}}]}]}}}`)
		default:
			fmt.Fprint(w, `{"data":{"viewer":{"zones":[]}}}`)
		}
	}))
	defer gql.Close()
	analytics := NewAnalyticsService("acct-eff", "tok", WithAnalyticsBaseURL(gql.URL))
	var probed []string
	caching := NewCachingService("tok",
		WithCachingProbe(func(ctx context.Context, url string) (*ProbeResult, error) {
			probed = append(probed, url)
			return &ProbeResult{StatusCode: 200, ContentType: "text/html", CacheControl: "public, s-maxage=300", CFCacheStatus: "DYNAMIC"}, nil
		}))
	svc := NewEfficiencyServiceWithZones(nil, nil, analytics, nil, stubEffZoneLister{zones: []*Zone{
		{ID: "z-trap", Name: "trapped.example"},
		{ID: "z-quiet", Name: "quiet.example"},
	}})
	svc.caching = caching
	rep, err := svc.Report(context.Background(), effWindow())
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	var trap *EfficiencyFinding
	for i, f := range rep.Findings {
		if f.Check == "html-cache-trap" {
			trap = &rep.Findings[i]
			break
		}
	}
	if trap == nil {
		t.Fatalf("expected html-cache-trap finding, got %+v", rep.Findings)
	}
	if trap.Resource != "zone:trapped.example" {
		t.Fatalf("trap resource = %q", trap.Resource)
	}
	// The quiet zone is below the coverage floor and must never be probed.
	for _, url := range probed {
		if strings.Contains(url, "quiet.example") {
			t.Errorf("quiet zone was probed: %v", probed)
		}
	}
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	buf := make([]byte, 4096)
	n, _ := r.Body.Read(buf)
	return string(buf[:n])
}
