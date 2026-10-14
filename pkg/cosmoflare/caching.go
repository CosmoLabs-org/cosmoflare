package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HTML caching-trap detection (FEAT-069 wave 2): Cloudflare does not cache
// HTML by extension default, so an origin emitting s-maxage on HTML produces
// cf-cache-status: DYNAMIC — a caching fix that looks shipped and does
// nothing. MyCarGuide verified this live: origin s-maxage alone left every
// HTML response DYNAMIC until a zone Cache Rule existed.

// ProbeResult is the header evidence from one HTTPS probe of a zone origin.
type ProbeResult struct {
	StatusCode    int
	ContentType   string
	CacheControl  string
	CFCacheStatus string
}

// CachingService probes zone origins and reads zone Cache Rules. The probe
// fetches the public zone apex (no credentials); the rules call is a
// best-effort REST read — invisible rules (403/404) never mask probe
// evidence, the finding just says the rules were not visible.
type CachingService struct {
	apiToken string
	baseURL  string
	client   *http.Client
	probe    func(ctx context.Context, url string) (*ProbeResult, error)
}

// CachingOption configures a CachingService (tests inject baseURL and probe).
type CachingOption func(*CachingService)

// WithCachingBaseURL overrides the REST base URL (tests).
func WithCachingBaseURL(u string) CachingOption {
	return func(s *CachingService) { s.baseURL = u }
}

// WithCachingHTTPClient overrides the HTTP client (tests).
func WithCachingHTTPClient(c *http.Client) CachingOption {
	return func(s *CachingService) { s.client = c }
}

// WithCachingProbe replaces the HTTPS origin probe (tests).
func WithCachingProbe(p func(ctx context.Context, url string) (*ProbeResult, error)) CachingOption {
	return func(s *CachingService) { s.probe = p }
}

// NewCachingService creates a CachingService from an API token.
func NewCachingService(apiToken string, opts ...CachingOption) *CachingService {
	s := &CachingService{
		apiToken: apiToken,
		baseURL:  "https://api.cloudflare.com/client/v4",
		client:   &http.Client{Timeout: 10 * time.Second},
	}
	s.probe = func(ctx context.Context, url string) (*ProbeResult, error) {
		return s.defaultProbe(ctx, url)
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *CachingService) defaultProbe(ctx context.Context, url string) (*ProbeResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "cosmoflare-efficiency/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
	return &ProbeResult{
		StatusCode:    res.StatusCode,
		ContentType:   res.Header.Get("Content-Type"),
		CacheControl:  res.Header.Get("Cache-Control"),
		CFCacheStatus: res.Header.Get("CF-Cache-Status"),
	}, nil
}

// zoneCacheRuleCount reads the zone's Cache Rules and returns how many
// exist. Any transport/API failure returns (0, err) — callers treat error
// as "rules not visible", never as "no rules".
func (s *CachingService) zoneCacheRuleCount(ctx context.Context, zoneID string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/zones/"+zoneID+"/cache/rules?per_page=5", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	res, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	var body struct {
		Success bool             `json:"success"`
		Result  []map[string]any `json:"result"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&body); err != nil {
		return 0, fmt.Errorf("HTTP %d: %w", res.StatusCode, err)
	}
	if !body.Success {
		msg := "unknown error"
		if len(body.Errors) > 0 {
			msg = body.Errors[0].Message
		}
		return 0, fmt.Errorf("HTTP %d: %s", res.StatusCode, msg)
	}
	return len(body.Result), nil
}

// minCacheableMaxAgeSec is the floor for "the origin is trying to cache":
// shorter max-ages are not a caching attempt worth flagging.
const minCacheableMaxAgeSec = 60

// assessHTMLCacheTrap is the pure trap verdict. A trap requires all three:
// HTML content, a public cacheable header at or above the floor, and a
// cf-cache-status that proves the edge did NOT serve it from cache (DYNAMIC
// or absent). rulesVisible=false keeps the trap but says so in the signals.
func assessHTMLCacheTrap(p *ProbeResult, rulesVisible bool, ruleCount int) (bool, string) {
	if p == nil || !strings.Contains(strings.ToLower(p.ContentType), "text/html") {
		return false, ""
	}
	cc := strings.ToLower(p.CacheControl)
	if !strings.Contains(cc, "public") {
		return false, ""
	}
	cacheable := strings.Contains(cc, "s-maxage")
	if !cacheable {
		for _, part := range strings.Split(cc, ",") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "max-age=") {
				if secs, err := strconv.Atoi(strings.TrimPrefix(part, "max-age=")); err == nil && secs >= minCacheableMaxAgeSec {
					cacheable = true
				}
			}
		}
	}
	if !cacheable {
		return false, ""
	}
	status := p.CFCacheStatus
	if status == "HIT" || status == "STALE" || status == "REVALIDATED" || status == "UPDATING" {
		return false, "" // the edge IS caching it
	}
	var signals []string
	signals = append(signals, fmt.Sprintf("origin %q on HTML", p.CacheControl))
	if status == "" {
		signals = append(signals, "no cf-cache-status header")
	} else {
		signals = append(signals, fmt.Sprintf("cf-cache-status %s", status))
	}
	if !rulesVisible {
		signals = append(signals, "zone Cache Rules not visible with this token")
	} else if ruleCount == 0 {
		signals = append(signals, "no zone Cache Rule exists")
	} else {
		signals = append(signals, fmt.Sprintf("%d zone Cache Rule(s) exist — check they cover HTML paths", ruleCount))
	}
	return true, strings.Join(signals, "; ")
}

// DetectHTMLCacheTrap probes the zone apex and, when the origin's headers
// prove a caching attempt the edge is not honoring, returns a finding.
// A clean zone returns (nil, nil). Probe transport errors return an error —
// an unreachable apex is not a trap verdict.
func (s *CachingService) DetectHTMLCacheTrap(ctx context.Context, zone Zone) (*EfficiencyFinding, error) {
	probe, err := s.probe(ctx, "https://"+zone.Name+"/")
	if err != nil {
		return nil, fmt.Errorf("probe %s: %w", zone.Name, err)
	}
	ruleCount, rulesErr := s.zoneCacheRuleCount(ctx, zone.ID)
	rulesVisible := rulesErr == nil
	trapped, signals := assessHTMLCacheTrap(probe, rulesVisible, ruleCount)
	if !trapped {
		return nil, nil
	}
	return &EfficiencyFinding{
		Resource: "zone:" + zone.Name,
		Check:    "html-cache-trap",
		Severity: "warning",
		Evidence: signals,
		Fix:      "origin s-maxage does not cache HTML at Cloudflare by default — add a zone Cache Rule covering the HTML paths (dash: Caching > Cache Rules), or emit the cache header at the edge (Worker fetch handler / Cache API)",
	}, nil
}
