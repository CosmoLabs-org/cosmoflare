package cosmoflare

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ValidationSeverity indicates the severity of a validation finding.
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "error"
	SeverityWarning ValidationSeverity = "warning"
	SeverityInfo    ValidationSeverity = "info"
)

// ValidationFinding represents a single validation issue found in the config.
type ValidationFinding struct {
	Severity ValidationSeverity `json:"severity"`
	Field    string             `json:"field"`
	Message  string             `json:"message"`
	Fixable  bool               `json:"fixable,omitempty"`
	FixHint  string             `json:"fix_hint,omitempty"`
}

// ValidationResult holds the complete result of config validation.
type ValidationResult struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationFinding `json:"errors,omitempty"`
	Warnings []ValidationFinding `json:"warnings,omitempty"`
	Info     []ValidationFinding `json:"info,omitempty"`
}

// TotalFindings returns the total number of findings across all severities.
func (r *ValidationResult) TotalFindings() int {
	return len(r.Errors) + len(r.Warnings) + len(r.Info)
}

// HasErrors returns true if there are any error-level findings.
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if there are any warning-level findings.
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// ConfigValidator validates a CosmoflareConfig against Cloudflare API constraints.
type ConfigValidator struct {
	// baseDir is the directory containing the config file, used for
	// resolving relative paths (e.g., worker script paths).
	baseDir string
}

// NewConfigValidator creates a new ConfigValidator.
// baseDir is the directory where the .cosmoflare.yaml lives, used to resolve
// relative file paths in the config (e.g., worker script paths).
func NewConfigValidator(baseDir string) *ConfigValidator {
	return &ConfigValidator{baseDir: baseDir}
}

// Validate runs all validation rules against the given config and returns
// a ValidationResult summarizing every finding.
func (v *ConfigValidator) Validate(cfg *CosmoflareConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	v.validateRequiredFields(cfg, result)
	v.validateWorkers(cfg, result)
	v.validateR2(cfg, result)
	v.validateKV(cfg, result)
	v.validateDNS(cfg, result)
	v.validateZoneIDs(cfg, result)
	v.validateSSLModes(cfg, result)

	// Valid is true only if there are zero errors
	result.Valid = !result.HasErrors()

	return result
}

// addFinding appends a finding to the appropriate severity list.
func (v *ConfigValidator) addFinding(result *ValidationResult, severity ValidationSeverity, field, message string) {
	finding := ValidationFinding{
		Severity: severity,
		Field:    field,
		Message:  message,
	}
	switch severity {
	case SeverityError:
		result.Errors = append(result.Errors, finding)
	case SeverityWarning:
		result.Warnings = append(result.Warnings, finding)
	case SeverityInfo:
		result.Info = append(result.Info, finding)
	}
}

// addFixableFinding appends a fixable finding with a hint.
func (v *ConfigValidator) addFixableFinding(result *ValidationResult, severity ValidationSeverity, field, message, fixHint string) {
	finding := ValidationFinding{
		Severity: severity,
		Field:    field,
		Message:  message,
		Fixable:  true,
		FixHint:  fixHint,
	}
	switch severity {
	case SeverityError:
		result.Errors = append(result.Errors, finding)
	case SeverityWarning:
		result.Warnings = append(result.Warnings, finding)
	case SeverityInfo:
		result.Info = append(result.Info, finding)
	}
}

// --- Validation Rules ---

// validateRequiredFields checks that name and type are present.
func (v *ConfigValidator) validateRequiredFields(cfg *CosmoflareConfig, result *ValidationResult) {
	if cfg.Name == "" {
		v.addFinding(result, SeverityError, "name", "required field 'name' is missing")
	}
	if cfg.Type == "" {
		v.addFinding(result, SeverityError, "type", "required field 'type' is missing")
	}
}

// validateWorkers checks worker script paths exist on disk.
func (v *ConfigValidator) validateWorkers(cfg *CosmoflareConfig, result *ValidationResult) {
	if len(cfg.Workers) == 0 {
		return
	}

	names := make(map[string]bool)
	for name, w := range cfg.Workers {
		// Duplicate name check (map keys are inherently unique in Go,
		// but we track for completeness and test coverage)
		if names[name] {
			v.addFinding(result, SeverityError, fmt.Sprintf("workers.%s", name),
				fmt.Sprintf("duplicate worker name %q", name))
		}
		names[name] = true

		// Script path must be set
		if w.Script == "" {
			v.addFinding(result, SeverityError, fmt.Sprintf("workers.%s.script", name),
				"worker script path is required")
			continue
		}

		// Resolve relative to baseDir and check existence
		scriptPath := w.Script
		if v.baseDir != "" && !isAbsPath(scriptPath) {
			scriptPath = v.baseDir + "/" + scriptPath
		}
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			v.addFinding(result, SeverityError, fmt.Sprintf("workers.%s.script", name),
				fmt.Sprintf("script file %q does not exist", w.Script))
		}
	}
}

// r2BucketNameRegex matches valid S3/R2 bucket names:
// 3-63 chars, lowercase letters, numbers, and hyphens, must start/end with letter or number.
var r2BucketNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{1,61}[a-z0-9]$`)

// validateR2 checks R2 bucket naming rules (S3-compatible).
func (v *ConfigValidator) validateR2(cfg *CosmoflareConfig, result *ValidationResult) {
	if len(cfg.R2.Buckets) == 0 {
		return
	}

	names := make(map[string]bool)
	for i, b := range cfg.R2.Buckets {
		field := fmt.Sprintf("r2.buckets[%d]", i)

		if b.Name == "" {
			v.addFinding(result, SeverityError, field+".name", "bucket name is required")
			continue
		}

		// Duplicate check
		if names[b.Name] {
			v.addFinding(result, SeverityError, field+".name",
				fmt.Sprintf("duplicate bucket name %q", b.Name))
		}
		names[b.Name] = true

		// Length check
		if len(b.Name) < 3 || len(b.Name) > 63 {
			v.addFinding(result, SeverityError, field+".name",
				fmt.Sprintf("bucket name %q must be 3-63 characters (got %d)", b.Name, len(b.Name)))
			continue
		}

		// No dots (Cloudflare R2 differs from S3 here — dots cause issues with SSL)
		if strings.Contains(b.Name, ".") {
			v.addFixableFinding(result, SeverityError, field+".name",
				fmt.Sprintf("bucket name %q must not contain dots", b.Name),
				"Replace dots with hyphens")
			continue
		}

		// Uppercase check
		if b.Name != strings.ToLower(b.Name) {
			v.addFixableFinding(result, SeverityError, field+".name",
				fmt.Sprintf("bucket name %q must be lowercase", b.Name),
				fmt.Sprintf("Use %q instead", strings.ToLower(b.Name)))
			continue
		}

		// Full regex validation
		if !r2BucketNameRegex.MatchString(b.Name) {
			v.addFinding(result, SeverityError, field+".name",
				fmt.Sprintf("bucket name %q is invalid — must be 3-63 chars, lowercase alphanumeric and hyphens, must start/end with letter or number", b.Name))
		}
	}
}

// kvNamespaceRegex matches valid KV namespace titles: alphanumeric and hyphens.
var kvNamespaceRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-]*$`)

// validateKV checks KV namespace names are valid.
func (v *ConfigValidator) validateKV(cfg *CosmoflareConfig, result *ValidationResult) {
	if len(cfg.KV.Namespaces) == 0 {
		return
	}

	titles := make(map[string]bool)
	for i, ns := range cfg.KV.Namespaces {
		field := fmt.Sprintf("kv.namespaces[%d]", i)

		if ns.Title == "" {
			v.addFinding(result, SeverityError, field+".title", "namespace title is required")
			continue
		}

		// Duplicate check
		if titles[ns.Title] {
			v.addFinding(result, SeverityError, field+".title",
				fmt.Sprintf("duplicate namespace title %q", ns.Title))
		}
		titles[ns.Title] = true

		if !kvNamespaceRegex.MatchString(ns.Title) {
			v.addFinding(result, SeverityError, field+".title",
				fmt.Sprintf("namespace title %q is invalid — must be alphanumeric and hyphens, starting with a letter or number", ns.Title))
		}
	}
}

// validDNSTypes lists the DNS record types accepted by the Cloudflare API.
var validDNSTypes = map[string]bool{
	"A":     true,
	"AAAA":  true,
	"CNAME": true,
	"MX":    true,
	"TXT":   true,
	"NS":    true,
	"SRV":   true,
	"LOC":   true,
	"CAA":   true,
	"CERT":  true,
	"DNSKEY": true,
	"DS":    true,
	"NAPTR": true,
	"SMIMEA": true,
	"SSHFP": true,
	"TLSA":  true,
	"URI":   true,
	"PTR":   true,
	"SPF":   true,
	"HTTPS": true,
	"SVCB":  true,
}

// validateDNS checks DNS records for valid types, required fields, and duplicates.
func (v *ConfigValidator) validateDNS(cfg *CosmoflareConfig, result *ValidationResult) {
	if len(cfg.DNS.Records) == 0 {
		return
	}

	// Warn if zone_id is missing but records are defined
	if cfg.DNS.ZoneID == "" && len(cfg.DNS.Records) > 0 {
		v.addFinding(result, SeverityWarning, "dns.zone_id",
			"DNS records defined but zone_id is not set — records cannot be applied without a zone_id")
	}

	type recordKey struct {
		Type string
		Name string
	}
	seen := make(map[recordKey]bool)

	for i, rec := range cfg.DNS.Records {
		field := fmt.Sprintf("dns.records[%d]", i)

		// Required: type
		if rec.Type == "" {
			v.addFinding(result, SeverityError, field+".type", "DNS record type is required")
		} else {
			upper := strings.ToUpper(rec.Type)
			if !validDNSTypes[upper] {
				v.addFinding(result, SeverityError, field+".type",
					fmt.Sprintf("invalid DNS record type %q — valid types: A, AAAA, CNAME, MX, TXT, NS, SRV, CAA, PTR, etc.", rec.Type))
			} else if rec.Type != upper {
				v.addFixableFinding(result, SeverityWarning, field+".type",
					fmt.Sprintf("DNS record type %q should be uppercase", rec.Type),
					fmt.Sprintf("Use %q instead", upper))
			}
		}

		// Required: name
		if rec.Name == "" {
			v.addFinding(result, SeverityError, field+".name", "DNS record name is required")
		}

		// Required: content
		if rec.Content == "" {
			v.addFinding(result, SeverityError, field+".content", "DNS record content is required")
		}

		// Duplicate check (same type + name, excluding types that allow multiples)
		if rec.Type != "" && rec.Name != "" {
			key := recordKey{Type: strings.ToUpper(rec.Type), Name: rec.Name}
			// These types legitimately allow multiple records with the same name
			allowMultiple := key.Type == "MX" || key.Type == "TXT" || key.Type == "NS" || key.Type == "SRV"
			if !allowMultiple {
				if seen[key] {
					v.addFinding(result, SeverityError, field,
						fmt.Sprintf("duplicate DNS record: %s %s", rec.Type, rec.Name))
				}
				seen[key] = true
			}
		}

		// TTL validation: must be 1 (auto) or 60-86400
		if rec.TTL != 0 && rec.TTL != 1 && (rec.TTL < 60 || rec.TTL > 86400) {
			v.addFinding(result, SeverityError, field+".ttl",
				fmt.Sprintf("TTL %d is out of range — must be 1 (auto) or 60-86400", rec.TTL))
		}
	}
}

// zoneIDRegex matches a 32-character lowercase hex string.
var zoneIDRegex = regexp.MustCompile(`^[0-9a-f]{32}$`)

// validateZoneIDs checks that zone IDs are valid 32-char hex strings.
func (v *ConfigValidator) validateZoneIDs(cfg *CosmoflareConfig, result *ValidationResult) {
	if cfg.DNS.ZoneID == "" {
		return
	}
	if !zoneIDRegex.MatchString(cfg.DNS.ZoneID) {
		v.addFinding(result, SeverityError, "dns.zone_id",
			fmt.Sprintf("zone ID %q is invalid — must be a 32-character lowercase hex string", cfg.DNS.ZoneID))
	}
}

// validSSLModes lists the SSL modes supported by Cloudflare.
var validSSLModes = map[string]bool{
	"off":      true,
	"flexible": true,
	"full":     true,
	"strict":   true,
}

// validateSSLModes checks SSL mode fields if present. Since the
// CosmoflareConfig doesn't currently have an SSL section, this is
// forward-looking — it validates the ssl_mode field if it ever appears
// in a worker or DNS context. For now it's a no-op placeholder that
// allows the rule engine to be extended.
func (v *ConfigValidator) validateSSLModes(_ *CosmoflareConfig, _ *ValidationResult) {
	// SSL section will be added to CosmoflareConfig when the SSL
	// service is integrated into the declarative config. This method
	// is pre-registered so the rule engine is ready.
}

// ValidateSSLMode checks a single SSL mode string and returns an error if invalid.
// This is exported for use by other packages (e.g., cmd layer) that may validate
// SSL mode flags independently.
func ValidateSSLMode(mode string) error {
	if !validSSLModes[strings.ToLower(mode)] {
		return fmt.Errorf("invalid SSL mode %q — valid modes: off, flexible, full, strict", mode)
	}
	return nil
}

// ValidatePortNumber checks whether a port number is in the valid range (1-65535).
func ValidatePortNumber(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d is out of range — must be 1-65535", port)
	}
	return nil
}

// isAbsPath checks if a path is absolute (starts with /).
func isAbsPath(path string) bool {
	return strings.HasPrefix(path, "/")
}
