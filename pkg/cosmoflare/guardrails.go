package cosmoflare

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GuardrailResult contains the outcome of a guardrail check.
type GuardrailResult struct {
	Allowed bool     `json:"allowed"`
	Reasons []string `json:"reasons,omitempty"`
}

// GuardrailChecker validates operations against project rules.
type GuardrailChecker struct {
	cfg *ProjectConfig
}

// NewGuardrailChecker creates a new checker from project config.
func NewGuardrailChecker(cfg *ProjectConfig) *GuardrailChecker {
	if cfg == nil {
		cfg = &ProjectConfig{}
	}
	return &GuardrailChecker{cfg: cfg}
}

// CheckUpload validates an upload against project guardrails.
func (g *GuardrailChecker) CheckUpload(bucket, key string, size int64) *GuardrailResult {
	var reasons []string

	// Check bucket allowlist
	if len(g.cfg.Guardrails.AllowedBuckets) > 0 {
		found := false
		for _, b := range g.cfg.Guardrails.AllowedBuckets {
			if b == bucket {
				found = true
				break
			}
		}
		if !found {
			reasons = append(reasons, fmt.Sprintf("bucket %q not in allowed_buckets list", bucket))
		}
	}

	// Check max file size
	if g.cfg.Guardrails.MaxFileSize > 0 && size > g.cfg.Guardrails.MaxFileSize {
		reasons = append(reasons, fmt.Sprintf("file size %d exceeds max_file_size %d", size, g.cfg.Guardrails.MaxFileSize))
	}
	if g.cfg.MaxFileSize > 0 && size > g.cfg.MaxFileSize {
		reasons = append(reasons, fmt.Sprintf("file size %d exceeds max_file_size %d", size, g.cfg.MaxFileSize))
	}

	// Check blocked key patterns
	for _, pattern := range g.cfg.Guardrails.BlockedKeys {
		if matchBlockedKey(key, pattern) {
			reasons = append(reasons, fmt.Sprintf("key %q matches blocked pattern %q", key, pattern))
		}
	}

	return &GuardrailResult{
		Allowed: len(reasons) == 0,
		Reasons: reasons,
	}
}

// enforceUploadGuardrails rejects an upload when the attached project config
// denies it. A nil checker (no project config attached to the client)
// enforces nothing. The check runs before any data is read or sent, so a
// blocked upload never performs a partial transfer.
func (c *client) enforceUploadGuardrails(op, bucket, key string, size int64) error {
	if c.guardrails == nil {
		return nil
	}
	res := c.guardrails.CheckUpload(bucket, key, size)
	if res.Allowed {
		return nil
	}
	return validationError(op, fmt.Sprintf(
		"upload of %q to bucket %q blocked by project guardrails: %s — remove or adjust the guardrail rules in .cosmoflare.yaml if this upload is intentional",
		key, bucket, strings.Join(res.Reasons, "; ")))
}

// CheckBucketAccess verifies that a bucket is in the declared allowlist.
func (g *GuardrailChecker) CheckBucketAccess(bucket string) *GuardrailResult {
	// If no allowed_buckets specified, all buckets are allowed
	if len(g.cfg.AllowedBuckets) == 0 && len(g.cfg.Guardrails.AllowedBuckets) == 0 {
		return &GuardrailResult{Allowed: true}
	}

	allowed := g.cfg.AllowedBuckets
	if len(allowed) == 0 {
		allowed = g.cfg.Guardrails.AllowedBuckets
	}

	for _, b := range allowed {
		if b == bucket {
			return &GuardrailResult{Allowed: true}
		}
	}

	return &GuardrailResult{
		Allowed: false,
		Reasons: []string{fmt.Sprintf("bucket %q not in declared allowed_buckets", bucket)},
	}
}

// ValidateBucketName checks R2 bucket naming rules.
func ValidateBucketName(name string) error {
	return validateBucketName(name)
}

// matchBlockedKey checks if a key matches a blocked pattern (glob or prefix).
func matchBlockedKey(key, pattern string) bool {
	// Exact match
	if key == pattern {
		return true
	}
	// Prefix match (pattern ends with /)
	if strings.HasSuffix(pattern, "/") && strings.HasPrefix(key, pattern) {
		return true
	}
	// Extension match (pattern starts with *.)
	if strings.HasPrefix(pattern, "*.") {
		ext := pattern[1:] // e.g., ".env"
		return strings.EqualFold(filepath.Ext(key), ext)
	}
	// Glob match
	if matched, _ := filepath.Match(pattern, key); matched {
		return true
	}
	return false
}
