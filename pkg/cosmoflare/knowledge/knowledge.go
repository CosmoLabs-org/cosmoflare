// Package knowledge encodes Cloudflare API tribal knowledge — endpoint
// registries, plan caps, field invariants, and error decodes — as embedded
// JSON data packs. Knowledge is advisory when absent, authoritative when
// present: routes outside every pack scope pass through untouched; routes
// inside a scope that match no registered endpoint are blocked before send.
package knowledge

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed packs/*.json
var packFS embed.FS

// Endpoint registers one API route. A PathTemplate segment written as
// "{name}" matches any single non-empty path segment; a trailing "*" glued
// to a segment (or a standalone "*" segment) prefix-matches the rest.
type Endpoint struct {
	Method       string `json:"method"`
	PathTemplate string `json:"path"`
	Status       string `json:"status,omitempty"` // "" = exists; "absent" = known-nonexistent; "deprecated-off" = API disabled
	Note         string `json:"note,omitempty"`
}

// ErrorDecode maps one CF error code (optionally refined by context) to a
// human/agent-actionable cause and fix.
type ErrorDecode struct {
	Code    int    `json:"code"`
	Context string `json:"context,omitempty"` // "" = applies globally
	Cause   string `json:"cause"`
	Fix     string `json:"fix"`
}

// PlanCap is one plan's payload-parameter caps. Cap keys are op-prefixed:
// "max:<field>" (numeric ceiling), "allow_set:<field>" (permitted string set).
type PlanCap struct {
	Plan string              `json:"plan"`
	Caps map[string]int      `json:"caps"`
	Sets map[string][]string `json:"sets,omitempty"`
}

// Invariant is a mandatory-field rule checked pre-send.
type Invariant struct {
	Field string `json:"field"`
	Op    string `json:"op"` // "must_include"
	Value string `json:"value"`
}

// TrafficClass documents whether WAF rate limiting counts one traffic class.
// Source carries the evidence link (CF docs or field evidence) so drift is
// auditable.
type TrafficClass struct {
	Class   string `json:"class"`
	Counted bool   `json:"counted"`
	Note    string `json:"note,omitempty"`
	Source  string `json:"source,omitempty"`
}

// Pack is one product's knowledge.
type Pack struct {
	Product    string        `json:"product"`
	Scopes     []string      `json:"scopes"`
	Endpoints  []Endpoint    `json:"endpoints"`
	Errors     []ErrorDecode `json:"errors"`
	PlanCaps   []PlanCap     `json:"plan_caps"`
	Invariants []Invariant   `json:"invariants"`

	TrafficClasses []TrafficClass `json:"traffic_classes,omitempty"`
}

var (
	loadOnce sync.Once
	packs    []*Pack
	loadErr  error
)

// Load parses every embedded pack exactly once. Malformed pack JSON fails
// loud — corrupted knowledge must never silently pass.
func Load() ([]*Pack, error) {
	loadOnce.Do(func() {
		entries, err := packFS.ReadDir("packs")
		if err != nil {
			loadErr = fmt.Errorf("knowledge: read packs dir: %w", err)
			return
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			data, err := packFS.ReadFile("packs/" + e.Name())
			if err != nil {
				loadErr = fmt.Errorf("knowledge: read pack %s: %w", e.Name(), err)
				return
			}
			var p Pack
			if err := json.Unmarshal(data, &p); err != nil {
				loadErr = fmt.Errorf("knowledge: parse pack %s: %w", e.Name(), err)
				return
			}
			packs = append(packs, &p)
		}
	})
	return packs, loadErr
}

// NormalizePath strips one leading "/client/v4" (present when the default
// cloudflare-go BaseURL is used; absent under httptest BaseURLs).
func NormalizePath(path string) string {
	return strings.TrimPrefix(path, "/client/v4")
}

// matchPath reports whether a concrete path matches a template.
func matchPath(tmpl, path string) bool {
	if tmpl == "" || path == "" {
		return false
	}
	t := strings.Split(strings.TrimPrefix(tmpl, "/"), "/")
	p := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i, seg := range t {
		if i >= len(p) {
			return false
		}
		if strings.HasSuffix(seg, "*") {
			return strings.HasPrefix(p[i], strings.TrimSuffix(seg, "*"))
		}
		if strings.HasPrefix(seg, "{") {
			if p[i] == "" {
				return false
			}
			continue
		}
		if seg != p[i] {
			return false
		}
	}
	return len(t) == len(p)
}

// RouteVerdict is the outcome of a route check.
type RouteVerdict struct {
	InScope  bool
	Blocked  bool
	Pack     string
	Endpoint *Endpoint
}

// CheckRoute classifies a method+path against every loaded pack.
func CheckRoute(method, path string) RouteVerdict {
	loaded, err := Load()
	if err != nil {
		return RouteVerdict{} // loader errors surface at Load() call sites
	}
	for _, p := range loaded {
		scoped := false
		for _, s := range p.Scopes {
			if matchPath(s, path) {
				scoped = true
				break
			}
		}
		if !scoped {
			continue
		}
		for i := range p.Endpoints {
			e := p.Endpoints[i]
			if e.Method == method && matchPath(e.PathTemplate, path) {
				if e.Status == "" {
					return RouteVerdict{InScope: true, Pack: p.Product, Endpoint: &e}
				}
				// Registered as known-absent or API-disabled → blocked too
				// (the Endpoint is kept so callers can surface the note).
				return RouteVerdict{InScope: true, Blocked: true, Pack: p.Product, Endpoint: &e}
			}
		}
		// In scope, no endpoint match → blocked.
		return RouteVerdict{InScope: true, Blocked: true, Pack: p.Product}
	}
	return RouteVerdict{}
}

// LookupDecode finds the best decode for a code: exact (code+context) first,
// then the code's context-free entry. Nil when nothing is registered.
func LookupDecode(code int, context string) *ErrorDecode {
	loaded, err := Load()
	if err != nil {
		return nil
	}
	var generic *ErrorDecode
	for _, p := range loaded {
		for _, d := range p.Errors {
			if d.Code != code {
				continue
			}
			if d.Context == context && context != "" {
				return &d
			}
			if d.Context == "" && generic == nil {
				generic = &d
			}
		}
	}
	return generic
}

// SkippedTrafficClasses returns the pack's documented not-counted classes
// for one product. Advisory when absent: unknown products and absent blocks
// return nil.
func SkippedTrafficClasses(product string) []TrafficClass {
	loaded, err := Load()
	if err != nil {
		return nil
	}
	var out []TrafficClass
	for _, p := range loaded {
		if p.Product != product {
			continue
		}
		for _, tc := range p.TrafficClasses {
			if !tc.Counted {
				out = append(out, tc)
			}
		}
	}
	return out
}
