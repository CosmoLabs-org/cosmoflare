---
branch: master
created: "2026-05-16T00:00:00-03:00"
status: implemented
origin: "/brainstorming"
tags:
  - cors
  - transform-rules
  - rulesets
  - response-headers
  - zone
title: "CORS Management via Cloudflare Transform Rules (Response Header Modification)"
schema_version: 1
deliverables:
  - id: BR-01
    title: "CORSService struct with constructor and zone-scoped methods"
  - id: BR-02
    title: "GetCORSRules — list CORS rules from http_response_headers_transform entrypoint ruleset"
  - id: BR-03
    title: "SetCORSHeaders — upsert a named CORS rule (create or replace) in the entrypoint ruleset"
  - id: BR-04
    title: "RemoveCORSRule — delete a CORS rule by description/tag from the entrypoint ruleset"
  - id: BR-05
    title: "pkg/cosmoflare/cors.go — library implementation"
  - id: BR-06
    title: "cmd/cors.go — Cobra CLI (cors settings, cors set, cors remove)"
  - id: BR-07
    title: "cmd/cors_test.go + pkg/cosmoflare/cors_test.go — unit tests (table-driven)"
---

# CORS Management via Cloudflare Transform Rules

**Date**: 2026-05-16
**Status**: Implemented
**Related roadmap**: ROAD-039 (Page/Redirect Rules), Phase 4 — Zone-scoped services

---

## Problem

A previous attempt to implement CORS management for Cosmoflare targeted the Cloudflare Zone Settings API (zone-level "security headers" type settings). That approach is a dead end — **Cloudflare has no native CORS zone setting**. The Zone Settings API exposes things like `min_tls_version`, `always_use_https`, `browser_cache_ttl`, but nothing for `Access-Control-Allow-Origin` or any CORS header.

CORS response headers cannot be set at the zone settings level because they are header-level HTTP semantics that vary by request, path, and origin — not a single global toggle. Cloudflare intentionally does not model them as a zone setting.

### Where CORS Headers Actually Live

Cloudflare provides two mechanisms for injecting CORS response headers:

1. **Transform Rules — Response Header Modification** (preferred, no Worker required): Uses the Rulesets API with phase `http_response_headers_transform` and action `rewrite`. Rules in this phase can add/set/remove HTTP response headers using static values or dynamic expressions. This is exactly what the Cloudflare dashboard's "Transform Rules > Modify Response Header" UI uses under the hood.

2. **Workers**: A Worker script can manipulate response headers with full programmatic control. This is more powerful but requires deploying and billing for a Worker. Overkill for simple CORS configuration.

**Chosen approach: Transform Rules via the Rulesets API.** Zero Worker required, pure Cloudflare API, supported by the existing `cloudflare-go` SDK at `v0.116.0`.

---

## Solution: Transform Rules API

### SDK Functions to Use

All live in `github.com/cloudflare/cloudflare-go` (`rulesets.go`).

#### Key constants

```go
// Phase for response header modification rules
cloudflare.RulesetPhaseHTTPResponseHeadersTransform
// = "http_response_headers_transform"

// Action for modifying headers
cloudflare.RulesetRuleActionRewrite
// = "rewrite"

// Header operation types
cloudflare.RulesetRuleActionParametersHTTPHeaderOperationSet    // "set"
cloudflare.RulesetRuleActionParametersHTTPHeaderOperationAdd    // "add"
cloudflare.RulesetRuleActionParametersHTTPHeaderOperationRemove // "remove"
```

#### Key types

```go
// A rule inside a ruleset
cloudflare.RulesetRule{
    Action:     string,                          // "rewrite"
    Expression: string,                          // Wirefilter expression, e.g. "true"
    Description: string,                         // Human label, used as our CORS rule identifier
    Enabled:    *bool,
    ActionParameters: &cloudflare.RulesetRuleActionParameters{
        Headers: map[string]cloudflare.RulesetRuleActionParametersHTTPHeader{
            "Access-Control-Allow-Origin": {
                Operation: "set",
                Value:     "*",
            },
            "Access-Control-Allow-Methods": {
                Operation: "set",
                Value:     "GET, POST, OPTIONS",
            },
            // ... more headers
        },
    },
}
```

#### Entrypoint ruleset pattern (preferred)

Cloudflare uses an "entrypoint" ruleset per phase — one canonical ruleset per zone/phase pair. The right API flow is:

1. `GetEntrypointRuleset(ctx, ZoneIdentifier(zoneID), "http_response_headers_transform")` — get the zone's existing response header transform ruleset (may return 404 if none exists yet).
2. Append/replace a CORS rule in the `Rules` slice.
3. `UpdateEntrypointRuleset(ctx, ZoneIdentifier(zoneID), UpdateEntrypointRulesetParams{Phase: "http_response_headers_transform", Rules: updatedRules})` — write back the full rule list.

This is idempotent and non-destructive: other rules already in the entrypoint ruleset (e.g. existing security header rules) are preserved by reading the current state first, modifying only the CORS-tagged rules, then writing the full slice back.

#### ResourceContainer for zone-scoped calls

```go
rc := cloudflare.ZoneIdentifier(zoneID)
// This produces URL path: /zones/{zoneID}/rulesets/phases/http_response_headers_transform/entrypoint
```

#### Full API signatures

```go
// Read current entrypoint ruleset (may be empty / 404)
func (api *API) GetEntrypointRuleset(ctx context.Context, rc *ResourceContainer, phase string) (Ruleset, error)

// Write (full replace) the entrypoint ruleset rules
func (api *API) UpdateEntrypointRuleset(ctx context.Context, rc *ResourceContainer, params UpdateEntrypointRulesetParams) (Ruleset, error)

// UpdateEntrypointRulesetParams
type UpdateEntrypointRulesetParams struct {
    Phase       string        // "-" json tag (path param only)
    Description string        `json:"description,omitempty"`
    Rules       []RulesetRule `json:"rules"`
}
```

### CORS Rule Wire Format

A single CORS rule as sent to the API:

```json
{
  "action": "rewrite",
  "expression": "true",
  "description": "cosmoflare-cors",
  "enabled": true,
  "action_parameters": {
    "headers": {
      "Access-Control-Allow-Origin": {
        "operation": "set",
        "value": "https://example.com"
      },
      "Access-Control-Allow-Methods": {
        "operation": "set",
        "value": "GET, POST, PUT, DELETE, OPTIONS"
      },
      "Access-Control-Allow-Headers": {
        "operation": "set",
        "value": "Content-Type, Authorization, X-Requested-With"
      },
      "Access-Control-Max-Age": {
        "operation": "set",
        "value": "86400"
      },
      "Access-Control-Allow-Credentials": {
        "operation": "set",
        "value": "true"
      }
    }
  }
}
```

The `expression: "true"` means the rule applies to all requests. A more targeted expression could be `http.request.uri.path matches "^/api/"` to scope CORS only to API routes. The CLI should expose this as an optional `--path-pattern` flag.

We tag our managed rule with `description: "cosmoflare-cors"` (or a user-supplied name via `--rule-name`). This is the stable identifier we use to find/replace/delete our rule without touching other rules in the same entrypoint ruleset.

---

## CORSService Design

### Types

```go
// pkg/r2go2/cors.go

// CORSRule represents a single CORS configuration rule in a Transform Ruleset.
type CORSRule struct {
    ID          string            `json:"id,omitempty"`
    Name        string            `json:"name"`             // maps to RulesetRule.Description
    Enabled     bool              `json:"enabled"`
    Expression  string            `json:"expression"`       // Wirefilter, e.g. "true"
    AllowOrigins  []string        `json:"allow_origins"`    // parsed from header value
    AllowMethods  []string        `json:"allow_methods"`
    AllowHeaders  []string        `json:"allow_headers"`
    MaxAge        int             `json:"max_age,omitempty"`
    AllowCredentials bool         `json:"allow_credentials"`
}

// CORSOption is a functional option for building a CORS rule.
type CORSOption func(*corsConfig)

type corsConfig struct {
    name             string
    expression       string
    allowOrigins     []string
    allowMethods     []string
    allowHeaders     []string
    maxAge           int
    allowCredentials bool
}

// CORSService manages CORS response headers via Cloudflare Transform Rules.
// CORS is zone-scoped — requires a zone ID.
type CORSService struct {
    cf     *cloudflare.API
    zoneID string
}
```

### Constructor (mirrors SSLService pattern exactly)

```go
func NewCORSService(api *cloudflare.API, zoneID string) (*CORSService, error) {
    if api == nil {
        return nil, validationError("NewCORSService", "cloudflare API client is required")
    }
    if zoneID == "" {
        return nil, validationError("NewCORSService", "zone ID is required")
    }
    return &CORSService{cf: api, zoneID: zoneID}, nil
}

func NewCORSServiceFromCreds(zoneID, apiToken string) (*CORSService, error) {
    if zoneID == "" {
        return nil, validationError("NewCORSService", "zone ID is required")
    }
    if apiToken == "" {
        return nil, validationError("NewCORSService", "API token is required")
    }
    cf, err := cloudflare.NewWithAPIToken(apiToken)
    if err != nil {
        return nil, authError("NewCORSService", "failed to create Cloudflare API client", err)
    }
    return &CORSService{cf: cf, zoneID: zoneID}, nil
}
```

### Methods

#### GetCORSRules

```go
// GetCORSRules returns all CORS rules from the zone's response header transform ruleset.
// Returns an empty slice (not an error) if no transform ruleset exists yet.
func (s *CORSService) GetCORSRules(ctx context.Context) ([]*CORSRule, error)
```

Implementation sketch:
1. Call `GetEntrypointRuleset(ctx, ZoneIdentifier(s.zoneID), "http_response_headers_transform")`.
2. If 404 (no ruleset yet), return empty slice, nil.
3. Walk `ruleset.Rules`, parse each rule's `ActionParameters.Headers` map into a `CORSRule`. Filter to rules that contain at least one CORS header key (`Access-Control-Allow-Origin`, `Access-Control-Allow-Methods`, etc.).
4. Return the slice.

Note: we do NOT filter by description here — we return ALL rules that contain CORS headers, regardless of whether Cosmoflare created them. This is important for the "settings" display which shows the user what's actually active.

#### SetCORSHeaders

```go
// SetCORSHeaders creates or replaces a CORS rule in the entrypoint ruleset.
// If a rule with the given name (description) already exists it is replaced in-place.
// If no ruleset exists yet, one is created via UpdateEntrypointRuleset.
func (s *CORSService) SetCORSHeaders(ctx context.Context, opts ...CORSOption) (*CORSRule, error)
```

Functional options:

```go
func WithCORSName(name string) CORSOption           // default: "cosmoflare-cors"
func WithCORSOrigins(origins ...string) CORSOption  // e.g. "*", "https://example.com"
func WithCORSMethods(methods ...string) CORSOption  // e.g. "GET", "POST", "OPTIONS"
func WithCORSHeaders(headers ...string) CORSOption  // e.g. "Content-Type", "Authorization"
func WithCORSMaxAge(seconds int) CORSOption         // Access-Control-Max-Age value
func WithCORSCredentials(allow bool) CORSOption     // Access-Control-Allow-Credentials
func WithCORSExpression(expr string) CORSOption     // Wirefilter expression, default "true"
```

Implementation sketch:
1. Read existing entrypoint ruleset (handle 404 as empty).
2. Build a new `RulesetRule` from `corsConfig`.
3. Walk existing rules: if a rule with `Description == name` found, replace it at that index. Otherwise, append.
4. Call `UpdateEntrypointRuleset` with the full modified rules slice.
5. Parse the returned rule back into `*CORSRule` and return.

#### RemoveCORSRule

```go
// RemoveCORSRule removes a CORS rule from the entrypoint ruleset by name (description).
// Returns ErrCORSRuleNotFound if no rule with that name exists.
// If removing the rule leaves the ruleset empty, the ruleset itself is left in place
// (Cloudflare keeps the empty entrypoint — that is fine).
func (s *CORSService) RemoveCORSRule(ctx context.Context, name string) error
```

Implementation sketch:
1. Read current entrypoint ruleset.
2. Filter out any rule where `Description == name`.
3. If nothing was removed, return `ErrCORSRuleNotFound`.
4. Call `UpdateEntrypointRuleset` with the filtered rules slice.

### Sentinel Errors

```go
var (
    ErrCORSRuleNotFound = errors.New("CORS rule not found")
)
```

> **Note:** An `ErrNoCORSRuleset` was considered during design but was not implemented. A missing ruleset (404 from `GetEntrypointRuleset`) is treated as "empty, not an error" rather than a distinct sentinel.

### Internal Helper: parseCORSRule

```go
// parseCORSRule converts a cloudflare.RulesetRule into our CORSRule type.
// Returns (nil, false) if the rule has no CORS headers.
func parseCORSRule(r cloudflare.RulesetRule) (*CORSRule, bool)
```

This centralizes the header-key-to-struct mapping:
- `Access-Control-Allow-Origin` → split on `, ` → `AllowOrigins`
- `Access-Control-Allow-Methods` → split on `, ` → `AllowMethods`
- `Access-Control-Allow-Headers` → split on `, ` → `AllowHeaders`
- `Access-Control-Max-Age` → `strconv.Atoi` → `MaxAge`
- `Access-Control-Allow-Credentials` → `== "true"` → `AllowCredentials`

And its inverse `buildRulesetRule(cfg *corsConfig) cloudflare.RulesetRule`.

---

## CLI UX

### Command tree

```
r2go2 cors
  settings   [zone-id]                 # Show all CORS rules active on the zone
  set        [zone-id] [flags]         # Create or replace the named CORS rule
  remove     [zone-id] [flags]         # Remove a CORS rule by name
```

### cors settings

```
r2go2 cors settings ZONE_ID
r2go2 cors settings ZONE_ID --json
```

Output (table):
```
NAME               ORIGINS                    METHODS              CREDENTIALS  EXPRESSION
cosmoflare-cors    https://app.example.com    GET, POST, OPTIONS   true         true
```

JSON output: array of `CORSRule` objects.

If no CORS rules are active: "No CORS rules configured for this zone." (with exit 0).

### cors set

```
r2go2 cors set ZONE_ID --origins "https://app.example.com,https://admin.example.com" \
    --methods "GET,POST,PUT,DELETE,OPTIONS" \
    --headers "Content-Type,Authorization" \
    --max-age 86400 \
    --credentials \
    --rule-name cosmoflare-cors \
    --expression "true"

# Wildcard origin (simpler):
r2go2 cors set ZONE_ID --origins "*" --methods "GET,POST,OPTIONS"

# API path scoped:
r2go2 cors set ZONE_ID --origins "*" --expression 'http.request.uri.path matches "^/api/"'

# JSON output after applying:
r2go2 cors set ZONE_ID --origins "*" --json
```

Flags:
- `--origins` (required): comma-separated list of allowed origins, or `*`
- `--methods`: comma-separated HTTP methods (default: `GET, POST, OPTIONS`)
- `--headers`: comma-separated header names (default: `Content-Type, Authorization`)
- `--max-age`: integer seconds for preflight cache (default: `86400`)
- `--credentials`: boolean flag (presence = true; `--no-credentials` = false; default: false)
- `--rule-name`: identifies the rule for future updates/removal (default: `cosmoflare-cors`)
- `--expression`: Wirefilter expression (default: `true` = all requests)
- `--json`: machine-readable output

Output on success:
```
CORS rule "cosmoflare-cors" applied to zone abc123.
  Origins:      https://app.example.com
  Methods:      GET, POST, OPTIONS
  Headers:      Content-Type, Authorization
  Max-Age:      86400
  Credentials:  false
  Expression:   true
```

### cors remove

```
r2go2 cors remove ZONE_ID
r2go2 cors remove ZONE_ID --rule-name my-cors-rule
r2go2 cors remove ZONE_ID --json
```

Flags:
- `--rule-name`: which rule to remove (default: `cosmoflare-cors`)
- `--json`: machine-readable confirmation

Output on success:
```
CORS rule "cosmoflare-cors" removed from zone abc123.
```

Error if not found:
```
Error: CORS rule "cosmoflare-cors" not found on zone abc123.
       Use "r2go2 cors settings ZONE_ID" to list active CORS rules.
```

---

## Implementation Plan

### Files to Create

#### 1. `pkg/cosmoflare/cors.go`

Full library implementation. ~200 lines.

```go
package r2go2

import (
    "context"
    "errors"
    "strconv"
    "strings"

    "github.com/cloudflare/cloudflare-go"
)

const (
    corsPhase           = "http_response_headers_transform"
    corsDefaultRuleName = "cosmoflare-cors"
    corsDefaultExpr     = "true"
)

var (
    ErrCORSRuleNotFound = errors.New("CORS rule not found")
)

// --- Types ---
type CORSRule struct { ... }
type CORSOption func(*corsConfig)
type corsConfig struct { ... }
type CORSService struct { cf *cloudflare.API; zoneID string }

// --- Constructors ---
func NewCORSService(api *cloudflare.API, zoneID string) (*CORSService, error)
func NewCORSServiceFromCreds(zoneID, apiToken string) (*CORSService, error)

// --- Functional options ---
func WithCORSName(name string) CORSOption
func WithCORSOrigins(origins ...string) CORSOption
func WithCORSMethods(methods ...string) CORSOption
func WithCORSHeaders(headers ...string) CORSOption
func WithCORSMaxAge(seconds int) CORSOption
func WithCORSCredentials(allow bool) CORSOption
func WithCORSExpression(expr string) CORSOption

// --- Service methods ---
func (s *CORSService) GetCORSRules(ctx context.Context) ([]*CORSRule, error)
func (s *CORSService) SetCORSHeaders(ctx context.Context, opts ...CORSOption) (*CORSRule, error)
func (s *CORSService) RemoveCORSRule(ctx context.Context, name string) error

// --- Internal helpers ---
func parseCORSRule(r cloudflare.RulesetRule) (*CORSRule, bool)
func buildRulesetRule(cfg *corsConfig) cloudflare.RulesetRule
func isCORSHeader(key string) bool
```

Key implementation notes:
- `GetEntrypointRuleset` 404 must be treated as "empty ruleset, not an error". Check if the returned error string contains "not found" or check for empty ID on returned `Ruleset`.
- `UpdateEntrypointRuleset` is a full-replace PUT — always send the complete rules slice.
- `buildRulesetRule` sets `Enabled: cloudflare.BoolPtr(true)` explicitly.
- Header values for multi-value fields (origins, methods, headers) are stored as comma-space-separated strings: `"GET, POST, OPTIONS"` — this is the standard CORS header format.
- For `AllowOrigins` with a single `*`, store `"*"` directly; for multiple origins, Cloudflare Transform Rules support only one static value per header (the `value` field is a plain string, not an array). If multiple origins are given, the rule should use an expression-based value instead: `http.request.headers["Origin"]` matched against a list. Document this limitation clearly in the CLI help text. For the initial implementation: if `len(origins) > 1` and none is `*`, join with `, ` as a best-effort and warn the user that only browsers sending one of these exact values will match (or suggest using Workers for dynamic origin reflection).

#### 2. `pkg/cosmoflare/cors_test.go`

Unit tests (~150 lines). Table-driven. No network required — mock via `httptest.NewServer` returning pre-canned JSON responses (same pattern as other service tests in this package).

Tests to cover:
- `GetCORSRules` on empty zone (404 from API) → returns `[]`, nil
- `GetCORSRules` on zone with one CORS rule → parses correctly
- `SetCORSHeaders` on empty zone → creates rule, returns `*CORSRule`
- `SetCORSHeaders` on zone with existing rule of same name → replaces in-place, other rules preserved
- `SetCORSHeaders` on zone with existing rule of different name → appends, other rules preserved
- `RemoveCORSRule` on zone with matching rule → removes it
- `RemoveCORSRule` on zone without matching rule → returns `ErrCORSRuleNotFound`
- `NewCORSService` with nil API → returns validation error
- `NewCORSService` with empty zoneID → returns validation error

#### 3. `cmd/cors.go`

Cobra CLI. ~220 lines. Mirrors `cmd/ssl.go` structure exactly.

```go
package cmd

var corsCmd = &cobra.Command{Use: "cors", Short: "Manage CORS response headers via Transform Rules"}

var corsSettingsCmd  = &cobra.Command{Use: "settings [zone-id]", RunE: runCORSSettings}
var corsSetCmd       = &cobra.Command{Use: "set [zone-id]",      RunE: runCORSSet}
var corsRemoveCmd    = &cobra.Command{Use: "remove [zone-id]",   RunE: runCORSRemove}

// Flags
var (
    corsOrigins     string
    corsMethods     string
    corsHeaders     string
    corsMaxAge      int
    corsCredentials bool
    corsRuleName    string
    corsExpression  string
)

func init() {
    // Register subcommands
    corsCmd.AddCommand(corsSettingsCmd, corsSetCmd, corsRemoveCmd)

    // cors set flags
    corsSetCmd.Flags().StringVar(&corsOrigins, "origins", "", "Comma-separated allowed origins (required)")
    corsSetCmd.Flags().StringVar(&corsMethods, "methods", "GET, POST, OPTIONS", "Comma-separated HTTP methods")
    corsSetCmd.Flags().StringVar(&corsHeaders, "headers", "Content-Type, Authorization", "Comma-separated header names")
    corsSetCmd.Flags().IntVar(&corsMaxAge, "max-age", 86400, "Preflight cache max-age in seconds")
    corsSetCmd.Flags().BoolVar(&corsCredentials, "credentials", false, "Allow credentials")
    corsSetCmd.Flags().StringVar(&corsRuleName, "rule-name", "cosmoflare-cors", "Name/identifier for the rule")
    corsSetCmd.Flags().StringVar(&corsExpression, "expression", "true", "Wirefilter filter expression")
    corsSetCmd.MarkFlagRequired("origins")

    // cors remove flags
    corsRemoveCmd.Flags().StringVar(&corsRuleName, "rule-name", "cosmoflare-cors", "Name of rule to remove")

    // Register with root
    rootCmd.AddCommand(corsCmd)
}
```

#### 4. `cmd/cors_test.go`

Unit tests for CLI layer (~100 lines). Tests `runCORSSettings`, `runCORSSet`, `runCORSRemove` with `--json` flag using `executeCommand` helper (same pattern as `cmd/ssl_test.go`).

### Wire-up in root.go

Check `cmd/root.go` — `corsCmd` must be added to `rootCmd` in `init()` (already done via the `corsCmd.go` init block above, consistent with how ssl, cache, dns are wired).

### No changes needed to `pkg/cosmoflare/types.go` or `pkg/cosmoflare/options.go`

CORS types are self-contained in `cors.go`. No shared type additions needed.

---

## Edge Cases and Gotchas

### 404 on GetEntrypointRuleset

If the zone has never had a response header transform rule, `GetEntrypointRuleset` returns a 404. The SDK will return a non-nil error. The implementation must check whether the error is a "not found" error and treat it as "no rules exist yet" rather than a hard failure. On `SetCORSHeaders`, an empty/missing ruleset is fine — `UpdateEntrypointRuleset` will create the entrypoint ruleset implicitly.

Detection: `strings.Contains(err.Error(), "Could not find") || strings.Contains(err.Error(), "not found")` — or wrap in a helper `isNotFoundError(err)` consistent with how other services handle 404s.

### Multi-Origin Limitation

The `Headers` map in `RulesetRuleActionParameters` uses `value` (a static string) not an expression array. A single `Access-Control-Allow-Origin` header value can only be one thing per rule. For dynamic origin reflection (different value based on incoming `Origin` header), a Worker is required. Document this clearly. For the initial implementation:
- Single origin or `*` → works perfectly.
- Multiple specific origins → store first one, emit a `⚠️ Warning` that dynamic origin reflection requires a Worker.

Future enhancement (ROAD item): Generate a Worker snippet that does dynamic origin reflection and output it with `r2go2 cors set --origins "..." --generate-worker`.

### Rule Ordering in Entrypoint Ruleset

Rules in a ruleset are evaluated in order. CORS rules added by Cosmoflare via `SetCORSHeaders` are appended at the end of the existing rules. This is the safe default — security header rules placed earlier by other tools will not be displaced. If the user needs ordering control, they can use the Cloudflare dashboard or a future `--position` flag.

### Credentials + Wildcard Origin

The CORS spec forbids `Access-Control-Allow-Credentials: true` together with `Access-Control-Allow-Origin: *`. If the user passes both `--credentials` and `--origins "*"`, the CLI should emit an error:

```
Error: Cannot use --credentials with wildcard origin "*".
       Specify an explicit origin (e.g. --origins "https://example.com") when using --credentials.
```

### Cloudflare Plan Requirements

Transform Rules (response header modification) require a Cloudflare paid plan (Pro or higher) for zones. On Free plans, the API may return a permission error. The `RemoveCORSRule` and `GetCORSRules` errors should surface this cleanly with an actionable message:

```
Error: This zone's plan does not include Transform Rules (response header modification).
       Upgrade to Cloudflare Pro or higher to use CORS management.
       Alternatively, deploy a Worker for CORS header injection.
```

---

## Testing Strategy

### Unit tests (no network)

Use `httptest.NewServer` to mock the Cloudflare API. Return pre-canned JSON for `GET /zones/{id}/rulesets/phases/http_response_headers_transform/entrypoint` and `PUT /zones/{id}/rulesets/phases/http_response_headers_transform/entrypoint`.

Sample fixture (JSON file in `testdata/cors_ruleset.json`):

```json
{
  "result": {
    "id": "abc123",
    "name": "default",
    "phase": "http_response_headers_transform",
    "rules": [
      {
        "id": "rule001",
        "action": "rewrite",
        "expression": "true",
        "description": "cosmoflare-cors",
        "enabled": true,
        "action_parameters": {
          "headers": {
            "Access-Control-Allow-Origin": {"operation": "set", "value": "*"},
            "Access-Control-Allow-Methods": {"operation": "set", "value": "GET, POST, OPTIONS"}
          }
        }
      }
    ]
  },
  "success": true,
  "errors": [],
  "messages": []
}
```

### Integration tests

Mark with `//go:build integration` (consistent with existing real-API tests). Require `CF_API_TOKEN` and `CF_ZONE_ID` env vars. Run `GetCORSRules` → `SetCORSHeaders` → `GetCORSRules` (assert rule present) → `RemoveCORSRule` → `GetCORSRules` (assert empty). Place in `tests/integration/cors/` directory.

---

## Summary for GLM Dispatch

A GLM agent implementing this needs to:

1. Create `pkg/cosmoflare/cors.go` with `CORSService`, 2 constructors, 3 methods, internal helpers.
2. Create `pkg/cosmoflare/cors_test.go` with 9 table-driven test cases using httptest mock.
3. Create `cmd/cors.go` with `corsCmd`, `corsSettingsCmd`, `corsSetCmd`, `corsRemoveCmd` following `ssl.go` pattern exactly.
4. Create `cmd/cors_test.go` with CLI-layer tests for all 3 subcommands.
5. Run `go build -o build/r2go2 .` and `go test ./pkg/cosmoflare/... ./cmd/...` — both must pass clean.

No changes to any other file are required. The `init()` in `cmd/cors.go` wires the command to `rootCmd` automatically.

**Exact SDK call sequence for `SetCORSHeaders`:**
```go
rc := cloudflare.ZoneIdentifier(s.zoneID)

// 1. Read current state
existing, err := s.cf.GetEntrypointRuleset(ctx, rc, corsPhase)
// handle 404 as empty

// 2. Build new rule
newRule := buildRulesetRule(cfg)

// 3. Upsert (replace by description, or append)
rules := upsertRule(existing.Rules, newRule, cfg.name)

// 4. Write back
updated, err := s.cf.UpdateEntrypointRuleset(ctx, rc, cloudflare.UpdateEntrypointRulesetParams{
    Phase: corsPhase,
    Rules: rules,
})
```
