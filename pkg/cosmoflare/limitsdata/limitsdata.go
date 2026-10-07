// Package limitsdata embeds the Cloudflare limits catalog (schema
// cosmoflare-limits-catalog/v1) — the single source of truth for documented
// plan limits (BR-03, brainstorm docs/brainstorming/2026-09-10-cf-limits-awareness-layer.md
// D1). Limit values are data, not code: a limits change lands as a catalog
// JSON edit (with source_url + verified_on provenance), not a code release.
//
// The embedded catalog is a verbatim copy of the research corpus draft at
// docs/research/2026-09-10-cf-limits-corpus/catalog-draft.json (66 entries).
// Two resources the corpus is docs-silent on (r2.buckets_per_account,
// workers.scripts_per_account) are appended at load time as explicitly
// flagged local constants — never silently hardcoded (see corpus findings).
package limitsdata

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"
)

//go:embed catalog.json
var catalogJSON []byte

// Tiers holds the per-tier limit values of one entry. Keys follow the
// per-service tier vocabularies: free/paid/enterprise for account services
// (Workers, R2, D1, ...), free/pro/business/enterprise for zones. Values are
// numbers OR null; a nil pointer means "no published number" — callers
// report unknown, never fabricate. json.Number (not uint64) because some
// catalog values are fractional (e.g. r2.upload_multipart 4.995 GiB).
type Tiers struct {
	Free       *json.Number `json:"free"`
	Paid       *json.Number `json:"paid"`
	Pro        *json.Number `json:"pro"`
	Business   *json.Number `json:"business"`
	Enterprise *json.Number `json:"enterprise"`
}

// Entry mirrors one catalog row (schema v1). Every embedded entry carries
// SourceURL and VerifiedOn provenance; the two appended local constants
// carry nil VerifiedOn so the D3 stale check names them.
type Entry struct {
	ID             string          `json:"id"`
	Service        string          `json:"service"`
	Kind           string          `json:"kind"` // quota | size | rate
	Unit           string          `json:"unit"`
	Name           string          `json:"name"`
	Scope          string          `json:"scope"`
	Tiers          *Tiers          `json:"tiers"`
	Trackable      bool            `json:"trackable"`
	Enforceability string          `json:"enforceability"` // local | api-list | api-counter | opaque
	Soft           bool            `json:"soft"`
	Pricing        json.RawMessage `json:"pricing,omitempty"`
	SourceURL      string          `json:"source_url"`
	VerifiedOn     *time.Time      `json:"verified_on"`
	Notes          string          `json:"notes"`
}

// num builds a *json.Number from an integer literal.
func num(v uint64) *json.Number {
	n := json.Number(strconv.FormatUint(v, 10))
	return &n
}

// localNote is the shared marker for corpus docs-silent local constants.
const localNote = "local constant — corpus docs-silent; flagged unverified"

// localEntries returns the two LOCAL-ONLY entries appended at load time.
// They are NOT in catalog.json (they are unverified — no corpus source
// landed for them) but Lookup must return them like any other entry so the
// stale-entry warning names them (nil verified_on ⇒ always stale).
func localEntries() []Entry {
	workersTiers := &Tiers{Free: num(100), Paid: num(500)} // enterprise: nil (unpublished)
	return []Entry{
		{
			ID: "workers.scripts_per_account", Service: "workers", Kind: "quota",
			Unit: "count", Name: "Worker scripts per account", Scope: "account",
			Tiers: workersTiers, Trackable: true, Enforceability: "api-list", Soft: false,
			SourceURL:  "https://developers.cloudflare.com/workers/platform/limits/",
			VerifiedOn: nil,
			Notes:      localNote + " — value from the pre-catalog LimitsService map (free 100 / paid 500)",
		},
		{
			ID: "r2.buckets_per_account", Service: "r2", Kind: "quota",
			Unit: "count", Name: "Buckets per account", Scope: "account",
			// Plan-independent in the pre-catalog map → same value on every tier.
			Tiers:     &Tiers{Free: num(1000000), Paid: num(1000000), Enterprise: num(1000000)},
			Trackable: true, Enforceability: "api-list", Soft: false,
			SourceURL:  "https://developers.cloudflare.com/r2/platform/limits/",
			VerifiedOn: nil,
			Notes:      localNote + " — value from the pre-catalog LimitsService map (1,000,000)",
		},
	}
}

// loadOnce parses the embedded catalog exactly once and appends the local
// constants. Load is safe for concurrent use.
var loadOnce = sync.OnceValues(func() ([]Entry, error) {
	var doc struct {
		Schema  string  `json:"schema"`
		Entries []Entry `json:"entries"`
	}
	if err := json.Unmarshal(catalogJSON, &doc); err != nil {
		return nil, fmt.Errorf("limitsdata: parse embedded catalog.json: %w", err)
	}
	return append(doc.Entries, localEntries()...), nil
})

// indexOnce builds the id→Entry lookup table once.
var indexOnce = sync.OnceValues(func() (map[string]Entry, error) {
	entries, err := loadOnce()
	if err != nil {
		return nil, err
	}
	idx := make(map[string]Entry, len(entries))
	for _, e := range entries {
		idx[e.ID] = e
	}
	return idx, nil
})

// Load returns every catalog entry: the 66 embedded (verified) rows plus the
// 2 explicitly-unverified local constants.
func Load() ([]Entry, error) {
	return loadOnce()
}

// Lookup returns the entry for a catalog id. ok=false when the id is not in
// the catalog (including the local constants, which are returned normally).
func Lookup(id string) (Entry, bool) {
	idx, err := indexOnce()
	if err != nil {
		return Entry{}, false
	}
	e, ok := idx[id]
	return e, ok
}

// Stale returns the ids of entries whose verified_on is older than the
// cutoff — plus entries with nil verified_on (the unverified local
// constants), so the 90-day freshness warning (D3) names them too. Order is
// stable (catalog order, locals last).
func Stale(before time.Time) []string {
	entries, err := loadOnce()
	if err != nil {
		return nil
	}
	var ids []string
	for _, e := range entries {
		if e.VerifiedOn == nil || e.VerifiedOn.Before(before) {
			ids = append(ids, e.ID)
		}
	}
	return ids
}
