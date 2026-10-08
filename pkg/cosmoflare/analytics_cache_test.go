package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// churchesLive is the live churches.app breakdown captured 2026-10-08 (24h, eyeball).
func churchesLive() ZoneCacheSummary {
	return ZoneCacheSummary{ZoneID: "z1", Zone: "churches.app", ByStatus: map[string]uint64{
		"miss": 83, "bypass": 21, "dynamic": 939, "revalidated": 236, "expired": 4, "none": 197, "hit": 61,
	}}
}

func TestZoneCacheSummaryMissPct(t *testing.T) {
	t.Parallel()
	got, ok := churchesLive().MissPct(100)
	// eligible = hit 61 + miss 83 + expired 4 + revalidated 236 = 384; missed = 87
	if !ok || math.Abs(got-100*87.0/384.0) > 1e-9 {
		t.Fatalf("MissPct = %v, %v; want %v, true", got, ok, 100*87.0/384.0)
	}
	if _, ok := churchesLive().MissPct(385); ok {
		t.Error("MissPct must not be judgeable when eligible (384) < floor (385)")
	}
	if _, ok := (ZoneCacheSummary{ByStatus: map[string]uint64{"dynamic": 5000}}).MissPct(0); ok {
		t.Error("MissPct must not be judgeable with zero eligible requests")
	}
}

func TestZoneCacheSummaryUncachedPct(t *testing.T) {
	t.Parallel()
	got, ok := churchesLive().UncachedPct(100)
	// uncached = dynamic 939 + bypass 21 = 960; known = 384 + 960 = 1344; none excluded
	if !ok || math.Abs(got-100*960.0/1344.0) > 1e-9 {
		t.Fatalf("UncachedPct = %v, %v; want %v, true", got, ok, 100*960.0/1344.0)
	}
	if _, ok := churchesLive().UncachedPct(1345); ok {
		t.Error("UncachedPct must not be judgeable when known (1344) < floor (1345)")
	}
	only := ZoneCacheSummary{ByStatus: map[string]uint64{"none": 9000, "ignored": 5}}
	if _, ok := only.UncachedPct(0); ok {
		t.Error("none/ignored-only traffic must not be judgeable")
	}
}

// TestAnalyticsZoneCacheBatches: 12 zones → 2 requests (10 + 2), zoneTag_in
// variable, eyeball filter, cacheStatus grouping, results in input order,
// zones without rows returned with an empty map.
func TestAnalyticsZoneCacheBatches(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(data))
		mu.Unlock()
		var req struct {
			Variables struct {
				ZoneTags []string `json:"zoneTags"`
			} `json:"variables"`
		}
		_ = json.Unmarshal(data, &req)
		var zones []string
		for _, tag := range req.Variables.ZoneTags {
			if tag == "zone-03" { // one zone with no traffic rows at all
				continue
			}
			zones = append(zones, fmt.Sprintf(`{"zoneTag":%q,"httpRequestsAdaptiveGroups":[`+
				`{"count":7,"dimensions":{"cacheStatus":"hit"}},`+
				`{"count":3,"dimensions":{"cacheStatus":"miss"}},`+
				`{"count":2,"dimensions":{"cacheStatus":"hit"}}]}`, tag))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"data":{"viewer":{"zones":[%s]}}}`, strings.Join(zones, ","))
	}))
	defer srv.Close()

	refs := make([]ZoneRef, 12)
	for i := range refs {
		refs[i] = ZoneRef{ID: fmt.Sprintf("zone-%02d", i+1), Name: fmt.Sprintf("site%02d.example", i+1)}
	}
	s := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL(srv.URL))
	got, err := s.ZoneCache(context.Background(), refs, analyticsWindow())
	if err != nil {
		t.Fatalf("ZoneCache: %v", err)
	}
	if len(bodies) != 2 {
		t.Fatalf("requests = %d, want 2 (batches of 10)", len(bodies))
	}
	for _, b := range bodies {
		for _, want := range []string{"zoneTag_in", "cacheStatus", `requestSource: \"eyeball\"`, "zoneTag"} {
			if !strings.Contains(b, want) {
				t.Errorf("request body missing %s: %s", want, b)
			}
		}
	}
	if len(got) != 12 {
		t.Fatalf("summaries = %d, want 12", len(got))
	}
	if got[0].Zone != "site01.example" || got[0].ZoneID != "zone-01" {
		t.Errorf("got[0] = %+v, want input order with names", got[0])
	}
	if got[0].ByStatus["hit"] != 9 || got[0].ByStatus["miss"] != 3 {
		t.Errorf("got[0].ByStatus = %v, want hit 9 (7+2 summed) miss 3", got[0].ByStatus)
	}
	if got[2].ByStatus == nil || len(got[2].ByStatus) != 0 {
		t.Errorf("zone without rows: ByStatus = %v, want empty non-nil map", got[2].ByStatus)
	}
}

func TestAnalyticsZoneCacheValidation(t *testing.T) {
	t.Parallel()
	if _, err := NewAnalyticsService("acct", "", WithAnalyticsBaseURL("http://unused")).ZoneCache(context.Background(), []ZoneRef{{ID: "z"}}, analyticsWindow()); err == nil {
		t.Error("missing token must error")
	}
	got, err := NewAnalyticsService("acct", "tok", WithAnalyticsBaseURL("http://unused")).ZoneCache(context.Background(), nil, analyticsWindow())
	if err != nil || len(got) != 0 {
		t.Errorf("no zones = %v, %v; want empty, nil (no request)", got, err)
	}
}
