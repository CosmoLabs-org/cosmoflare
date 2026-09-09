package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// RedirectProbeResult is one redirect-destination probe outcome.
type RedirectProbeResult struct {
	Destination string `json:"destination"`
	Status      int    `json:"status"`            // final HTTP status; 0 on transport error or loop
	Loop        bool   `json:"loop,omitempty"`    // a hop was revisited
	Err         string `json:"err,omitempty"`     // transport error / hop-cap message
	Skipped     bool   `json:"skipped,omitempty"` // unprobeable ($n capture reference)
}

// RedirectProber probes redirect destinations with bounded redirects and
// visited-set loop detection. Stdlib only — no Cloudflare credentials
// (DoctorService pattern).
type RedirectProber struct {
	httpClient  *http.Client
	timeout     time.Duration // per-target budget
	hopCap      int           // max redirects followed per target
	concurrency int           // parallel probes in ProbeAll
}

// RedirectProbeOption configures the prober.
type RedirectProbeOption func(*RedirectProber)

// WithProbeHTTPClient sets a custom HTTP client (tests).
func WithProbeHTTPClient(c *http.Client) RedirectProbeOption {
	return func(p *RedirectProber) { p.httpClient = c }
}

// WithProbeTimeout sets the per-target budget (default 10s).
func WithProbeTimeout(d time.Duration) RedirectProbeOption {
	return func(p *RedirectProber) {
		if d > 0 {
			p.timeout = d
		}
	}
}

// WithProbeHopCap sets the max redirects followed (default 10).
func WithProbeHopCap(n int) RedirectProbeOption {
	return func(p *RedirectProber) {
		if n > 0 {
			p.hopCap = n
		}
	}
}

// WithProbeConcurrency sets ProbeAll parallelism (default 8).
func WithProbeConcurrency(n int) RedirectProbeOption {
	return func(p *RedirectProber) {
		if n > 0 {
			p.concurrency = n
		}
	}
}

// NewRedirectProber builds a prober with the documented defaults.
func NewRedirectProber(opts ...RedirectProbeOption) *RedirectProber {
	p := &RedirectProber{
		httpClient:  &http.Client{},
		timeout:     10 * time.Second,
		hopCap:      10,
		concurrency: 8,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Probe tests one destination: GET, then follow redirects up to hopCap with
// a visited-set; a revisit marks a loop. Destinations containing "$"
// capture references are unprobeable and reported as Skipped.
func (p *RedirectProber) Probe(ctx context.Context, destination string) RedirectProbeResult {
	if strings.Contains(destination, "$") {
		return RedirectProbeResult{Destination: destination, Skipped: true}
	}

	client := *p.httpClient
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse // manual hop control below
	}

	deadline := time.Now().Add(p.timeout)
	visited := map[string]bool{}
	url := destination
	for hop := 0; hop <= p.hopCap; hop++ {
		if visited[url] {
			return RedirectProbeResult{Destination: destination, Loop: true}
		}
		visited[url] = true

		if time.Now().After(deadline) {
			return RedirectProbeResult{Destination: destination, Err: "timeout"}
		}
		reqCtx, cancel := context.WithDeadline(ctx, deadline)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
		if err != nil {
			cancel()
			return RedirectProbeResult{Destination: destination, Err: err.Error()}
		}
		resp, err := client.Do(req)
		cancel()
		if err != nil {
			return RedirectProbeResult{Destination: destination, Err: err.Error()}
		}
		resp.Body.Close()

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			loc := resp.Header.Get("Location")
			if loc == "" {
				return RedirectProbeResult{Destination: destination, Status: resp.StatusCode}
			}
			next, err := resolveReference(url, loc)
			if err != nil {
				return RedirectProbeResult{Destination: destination, Err: err.Error()}
			}
			url = next
			continue
		}
		return RedirectProbeResult{Destination: destination, Status: resp.StatusCode}
	}
	return RedirectProbeResult{Destination: destination, Err: fmt.Sprintf("exceeded %d redirect hops", p.hopCap)}
}

// ProbeAll fans out over deduplicated destinations with a concurrency
// semaphore. Never issues more than p.concurrency requests at once.
func (p *RedirectProber) ProbeAll(ctx context.Context, destinations []string) map[string]RedirectProbeResult {
	seen := map[string]bool{}
	uniq := make([]string, 0, len(destinations))
	for _, d := range destinations {
		if d != "" && !seen[d] {
			seen[d] = true
			uniq = append(uniq, d)
		}
	}

	results := make(map[string]RedirectProbeResult, len(uniq))
	sem := make(chan struct{}, p.concurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, d := range uniq {
		wg.Add(1)
		go func(dest string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			res := p.Probe(ctx, dest)
			mu.Lock()
			results[dest] = res
			mu.Unlock()
		}(d)
	}
	wg.Wait()
	return results
}

// resolveReference resolves a Location header against the requesting URL.
func resolveReference(base, location string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(ref).String(), nil
}
