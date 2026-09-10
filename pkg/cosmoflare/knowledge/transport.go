package knowledge

import (
	"fmt"
	"net/http"
)

// Transport is an http.RoundTripper enforcing the endpoint registry on the
// request side. Routes inside a pack scope that match no registered
// endpoint never leave the process: CF would answer 10405 with wording
// that misreads as an auth-scope problem.
type Transport struct {
	Base http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (tr *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	verdict := CheckRoute(req.Method, NormalizePath(req.URL.Path))
	if verdict.Blocked {
		hint := ""
		if absent := absentEndpointHint(verdict.Pack, req.Method); absent != "" {
			hint = " " + absent
		}
		return nil, fmt.Errorf(
			"cosmoflare knowledge: %s %s is not a registered endpoint (pack %q).%s Cloudflare would return 10405 \"method not allowed for this authentication scheme\" — misleading: the route does not exist",
			req.Method, NormalizePath(req.URL.Path), verdict.Pack, hint)
	}
	base := tr.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// absentEndpointHint surfaces a pack note when the blocked method has a
// known-absent endpoint registered with one.
func absentEndpointHint(product, method string) string {
	loaded, err := Load()
	if err != nil {
		return ""
	}
	for _, p := range loaded {
		if p.Product != product {
			continue
		}
		for _, e := range p.Endpoints {
			if e.Method == method && e.Status != "" && e.Note != "" {
				return fmt.Sprintf("Note: %s.", e.Note)
			}
		}
	}
	return ""
}
