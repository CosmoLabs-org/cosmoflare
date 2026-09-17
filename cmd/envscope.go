package cmd

import "strings"

// Resource-name scoping for the active environment profile (FEAT-026 part 2).
//
// A profile with a resource_prefix separates environments inside one account:
// name references get the prefix applied and list output is filtered to it.
// Prefixing is idempotent — a name that already carries the prefix passes
// through unchanged, so `--env staging` with prefix "stg-" maps "users" and
// "stg-users" to the same resource. Server-generated IDs are absolute handles
// and are never prefixed.

// applyResourcePrefix returns name with the active profile's resource prefix
// prepended. No-op when no profile is active, the profile has no prefix, or
// the name already starts with it.
func applyResourcePrefix(name string) string {
	if ActiveProfile == nil || ActiveProfile.ResourcePrefix == "" {
		return name
	}
	return withResourcePrefix(ActiveProfile.ResourcePrefix, name)
}

// matchesResourcePrefix reports whether name carries the active profile's
// resource prefix. True when no prefix is active — nothing to filter on.
func matchesResourcePrefix(name string) bool {
	if ActiveProfile == nil || ActiveProfile.ResourcePrefix == "" {
		return true
	}
	return strings.HasPrefix(name, ActiveProfile.ResourcePrefix)
}

// withResourcePrefix prepends prefix to name unless name already has it.
// An empty name stays empty — it means "not provided", not a resource
// named after the bare prefix.
func withResourcePrefix(prefix, name string) string {
	if prefix == "" || name == "" || strings.HasPrefix(name, prefix) {
		return name
	}
	return prefix + name
}
