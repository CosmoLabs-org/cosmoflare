package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	knowledge "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"
)

// Probe verdicts.
const (
	VerdictTripped      = "tripped"
	VerdictNotCounted   = "not-counted"
	VerdictInconclusive = "inconclusive"
)

// MaxProbeRequests is the hard ceiling on one probe burst, regardless of
// caller input (issue constraint: capped, opt-in live traffic).
const MaxProbeRequests = 60

// cfKnowledgeProduct is the knowledge pack the prober explains verdicts with.
const cfKnowledgeProduct = "ratelimit"

// RateLimitProbeResult is one trip-probe outcome. Statuses maps HTTP status
// to count; -1 accumulates transport errors.
type RateLimitProbeResult struct {
	URL         string      `json:"url"`
	Requests    int         `json:"requests"`
	Statuses    map[int]int `json:"statuses"`
	Tripped     bool        `json:"tripped"`
	Verdict     string      `json:"verdict"`
	Explanation string      `json:"explanation,omitempty"`
}

// classifyProbe derives a verdict from a probe's status histogram. Pure —
// tests call it directly; do not re-derive verdict rules in test bodies.
func classifyProbe(statuses map[int]int, sent, requests int) (verdict, explanation string) {
	blocked := statuses[http.StatusTooManyRequests] + statuses[http.StatusForbidden]
	if blocked > 0 {
		return VerdictTripped, fmt.Sprintf("%d of %d requests were blocked (429/403) — the rule sees this traffic class", blocked, sent)
	}
	errs := statuses[-1]
	ok := 0
	for status, n := range statuses {
		if status >= 200 && status < 400 {
			ok += n
		}
	}
	if sent+errs < requests || errs > sent || ok < sent {
		return VerdictInconclusive, fmt.Sprintf("%d of %d requests errored, %d completed (non-2xx/3xx responses present) — cannot judge; re-run when the target is reachable", errs, requests, sent)
	}
	var skipped []string
	for _, tc := range knowledge.SkippedTrafficClasses(cfKnowledgeProduct) {
		skipped = append(skipped, tc.Class)
	}
	return VerdictNotCounted, fmt.Sprintf(
		"all %d requests passed unblocked — the rule is live but this traffic class is likely not counted (documented skipped classes: %s); verify with a cache-busted URL or an origin-only path",
		sent, strings.Join(skipped, ", "))
}

// ExpressionPath extracts the literal path from a `path eq "/x"` or
// `http.request.uri.path eq "/x"` expression. Empty when the expression is
// not a bare literal path equality.
func ExpressionPath(expr string) string {
	expr = strings.TrimSpace(expr)
	for _, prefix := range []string{"path eq ", "http.request.uri.path eq "} {
		rest, ok := strings.CutPrefix(expr, prefix)
		if !ok {
			continue
		}
		rest = strings.TrimSpace(rest)
		if len(rest) >= 2 && rest[0] == '"' && rest[len(rest)-1] == '"' {
			return rest[1 : len(rest)-1]
		}
	}
	return ""
}

// RateLimitProber fires a bounded burst of GETs at a live URL and reports
// whether a zone rate-limiting rule trips. Stdlib only, no Cloudflare
// credentials (RedirectProber pattern). Explicit invocation only — never on
// default paths.
type RateLimitProber struct {
	httpClient  *http.Client
	timeout     time.Duration
	concurrency int
}

// RateLimitProbeOption configures a RateLimitProber.
type RateLimitProbeOption func(*RateLimitProber)

// WithRateLimitProbeHTTPClient overrides the probe HTTP client.
func WithRateLimitProbeHTTPClient(c *http.Client) RateLimitProbeOption {
	return func(p *RateLimitProber) {
		if c != nil {
			p.httpClient = c
		}
	}
}

// WithRateLimitProbeTimeout sets the per-request budget.
func WithRateLimitProbeTimeout(d time.Duration) RateLimitProbeOption {
	return func(p *RateLimitProber) {
		if d > 0 {
			p.timeout = d
		}
	}
}

// WithRateLimitProbeConcurrency caps parallel requests inside one burst.
func WithRateLimitProbeConcurrency(n int) RateLimitProbeOption {
	return func(p *RateLimitProber) {
		if n > 0 {
			p.concurrency = n
		}
	}
}

// NewRateLimitProber builds a prober with defaults: 10s per request, 5
// concurrent requests.
func NewRateLimitProber(opts ...RateLimitProbeOption) *RateLimitProber {
	p := &RateLimitProber{
		httpClient:  &http.Client{},
		timeout:     10 * time.Second,
		concurrency: 5,
	}
	for _, opt := range opts {
		opt(p)
	}
	p.httpClient.Timeout = p.timeout
	return p
}

// Probe sends `requests` GETs to rawURL — even indices plain, odd indices
// with a unique `cfprobe` query (cache-buster, so cached vs origin classes
// are distinguishable) — and classifies the outcome.
func (p *RateLimitProber) Probe(ctx context.Context, rawURL string, requests int) RateLimitProbeResult {
	if requests < 1 {
		requests = 1
	}
	if requests > MaxProbeRequests {
		requests = MaxProbeRequests
	}

	var mu sync.Mutex
	statuses := map[int]int{}
	sent := 0

	var wg sync.WaitGroup
	sem := make(chan struct{}, p.concurrency)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			u := rawURL
			if i%2 == 1 {
				sep := "?"
				if strings.Contains(u, "?") {
					sep = "&"
				}
				u = fmt.Sprintf("%s%scfprobe=%d-%d", u, sep, i, time.Now().UnixNano())
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
			if err != nil {
				mu.Lock()
				statuses[-1]++
				mu.Unlock()
				return
			}
			req.Header.Set("User-Agent", "Cosmoflare-RateLimitProbe/1.0")
			resp, err := p.httpClient.Do(req)
			if err != nil {
				mu.Lock()
				statuses[-1]++
				mu.Unlock()
				return
			}
			resp.Body.Close()
			mu.Lock()
			statuses[resp.StatusCode]++
			sent++
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	verdict, explanation := classifyProbe(statuses, sent, requests)
	return RateLimitProbeResult{
		URL:         rawURL,
		Requests:    requests,
		Statuses:    statuses,
		Tripped:     verdict == VerdictTripped,
		Verdict:     verdict,
		Explanation: explanation,
	}
}
