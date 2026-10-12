/*
Package cli — TOKEN_SCOPES: per-command-family token scope requirements
(FEAT-072 permissions preflight).

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cli

import (
	"sort"
	"strings"
)

// TokenScope names one Cloudflare API token permission requirement in
// Scope:Object:Access form (e.g. Zone:Rulesets:Edit).
type TokenScope struct {
	Scope  string // zone | account
	Object string // Rulesets, DNS, R2, …
	Access string // Read | Edit
}

// String renders the canonical "Scope:Object:Access" token used in
// "missing scope: …" diagnostics.
func (s TokenScope) String() string {
	return s.Scope + ":" + s.Object + ":" + s.Access
}

// groups lists the Cloudflare permission-group names that satisfy this
// requirement. Cloudflare has renamed groups over time (e.g. "DNS Write"
// vs "DNS Edit"), so each requirement accepts its known spellings.
func (s TokenScope) groups() []string {
	return tokenScopeGroups[s]
}

// tokenScopeGroups maps each canonical requirement to the permission-group
// names (compared case- and punctuation-insensitively) that fulfil it.
var tokenScopeGroups = map[TokenScope][]string{
	{Scope: "Zone", Object: "Zones", Access: "Read"}: {"Zone Read"},
	// Write groups are supersets of their Read group on Cloudflare, so
	// DNS Write also satisfies a DNS Read requirement.
	{Scope: "Zone", Object: "DNS", Access: "Read"}: {"DNS Read", "DNS Write", "DNS Edit"},
	{Scope: "Zone", Object: "DNS", Access: "Edit"}: {"DNS Write", "DNS Edit"},

	{Scope: "Account", Object: "R2", Access: "Edit"}: {
		"Workers R2 Storage Edit", "Workers R2 Storage Edit Item",
		"Workers R2 Storage Bucket Item Edit", "R2 Edit", "R2 Write",
	},

	{Scope: "Account", Object: "WorkersKVStorage", Access: "Edit"}: {
		"Workers KV Storage Edit", "Workers KV Storage Write",
	},

	{Scope: "Account", Object: "D1", Access: "Read"}: {"D1 Read", "D1 Write", "D1 Edit"},
	{Scope: "Account", Object: "D1", Access: "Edit"}: {"D1 Write", "D1 Edit"},

	{Scope: "Account", Object: "WorkersScripts", Access: "Edit"}: {
		"Workers Scripts Edit", "Workers Scripts Write", "Workers Scripts Edit/Upload",
	},

	{Scope: "Account", Object: "Pages", Access: "Edit"}: {"Pages Edit", "Pages Write"},

	// Cache Rules and WAF rules both go through the Rulesets API — the
	// exact blind spot FB-30 reported.
	{Scope: "Zone", Object: "Rulesets", Access: "Edit"}: {
		"Zone Rulesets Edit", "Rulesets Edit", "Zone Rules Edit", "Zone Edit",
	},

	{Scope: "Account", Object: "GraphqlAnalytics", Access: "Read"}: {
		"GraphQL Analytics Read", "Graphql Analytics Read", "Analytics Read",
	},

	{Scope: "Zone", Object: "EmailRouting", Access: "Read"}: {
		"Email Routing Rules Read", "Email Routing Read", "Email Routing Settings Read",
	},

	{Scope: "Account", Object: "Billing", Access: "Read"}: {
		"Billing Read", "Billing Info Read",
	},
}

// TOKEN_SCOPES maps every command family cosmoflare drives to the token
// scopes it requires. Command families are the preflight keys:
//
//	cosmoflare auth preflight <family>
var TOKEN_SCOPES = map[string][]TokenScope{
	"zones": {{Scope: "Zone", Object: "Zones", Access: "Read"}},
	"dns":   {{Scope: "Zone", Object: "DNS", Access: "Read"}},
	"dns-edit": {
		{Scope: "Zone", Object: "DNS", Access: "Edit"},
	},
	"r2":        {{Scope: "Account", Object: "R2", Access: "Edit"}},
	"kv":        {{Scope: "Account", Object: "WorkersKVStorage", Access: "Edit"}},
	"d1":        {{Scope: "Account", Object: "D1", Access: "Read"}},
	"d1-edit":   {{Scope: "Account", Object: "D1", Access: "Edit"}},
	"workers":   {{Scope: "Account", Object: "WorkersScripts", Access: "Edit"}},
	"pages":     {{Scope: "Account", Object: "Pages", Access: "Edit"}},
	"cache":     {{Scope: "Zone", Object: "Rulesets", Access: "Edit"}},
	"waf":       {{Scope: "Zone", Object: "Rulesets", Access: "Edit"}},
	"analytics": {{Scope: "Account", Object: "GraphqlAnalytics", Access: "Read"}},
	"email":     {{Scope: "Zone", Object: "EmailRouting", Access: "Read"}},
	"billing":   {{Scope: "Account", Object: "Billing", Access: "Read"}},
}

// ScopeFamilies returns the known preflight family keys, sorted.
func ScopeFamilies() []string {
	out := make([]string, 0, len(TOKEN_SCOPES))
	for f := range TOKEN_SCOPES {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// RequiredScopes returns the canonical scope requirements for family.
func RequiredScopes(family string) ([]TokenScope, bool) {
	req, ok := TOKEN_SCOPES[family]
	return req, ok
}

// normalizeScope makes permission-group comparisons robust to case,
// spacing, and punctuation differences between manifest spellings.
func normalizeScope(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// scopePresent reports whether any of present (real permission-group names
// from /user/tokens/verify → token detail) satisfies the requirement.
func scopePresent(req TokenScope, present []string) bool {
	have := make(map[string]bool, len(present))
	for _, p := range present {
		have[normalizeScope(p)] = true
	}
	for _, g := range req.groups() {
		if have[normalizeScope(g)] {
			return true
		}
	}
	return false
}

// MissingScopes returns the canonical names of family's required scopes
// that are absent from present. Empty result means the token may proceed.
func MissingScopes(family string, present []string) []string {
	req, ok := TOKEN_SCOPES[family]
	if !ok {
		return nil
	}
	var missing []string
	for _, r := range req {
		if !scopePresent(r, present) {
			missing = append(missing, r.String())
		}
	}
	return missing
}
