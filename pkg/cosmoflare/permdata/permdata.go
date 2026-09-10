/*
Package permdata provides the embedded Cloudflare API token permission
manifest — the single source of truth for which token permissions a
command group requires.

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/
package permdata

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed permissions.json
var permissionsJSON []byte

// Family describes one Cloudflare API token permission family.
type Family struct {
	ID     string   `json:"id"`
	Scope  string   `json:"scope"`
	Name   string   `json:"name"`
	Read   bool     `json:"read"`
	Edit   bool     `json:"edit"`
	UsedBy []string `json:"used_by"`
}

// LeastPrivilegeEntry describes the minimum permissions a command group needs.
type LeastPrivilegeEntry struct {
	CommandGroup string   `json:"command_group"`
	Permissions  []string `json:"permissions"`
}

// Manifest is the full permission manifest.
type Manifest struct {
	SchemaVersion   int                   `json:"schema_version"`
	ManifestVersion string                `json:"manifest_version"`
	SourceURL       string                `json:"source_url"`
	VerifiedOn      string                `json:"verified_on"`
	Families        []Family              `json:"families"`
	LeastPrivilege  []LeastPrivilegeEntry `json:"least_privilege"`
}

var loadOnce = sync.OnceValues(func() (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(permissionsJSON, &m); err != nil {
		return nil, fmt.Errorf("permdata: parse embedded permissions.json: %w", err)
	}
	return &m, nil
})

// Load returns the parsed permission manifest, parsing the embedded JSON
// exactly once.
func Load() (*Manifest, error) {
	return loadOnce()
}

// Families returns families for the given scope ("account", "zone", or
// "user"). An empty scope returns all families.
func Families(scope string) []Family {
	m, err := Load()
	if err != nil {
		return nil
	}
	if scope == "" {
		return m.Families
	}
	var out []Family
	for _, f := range m.Families {
		if f.Scope == scope {
			out = append(out, f)
		}
	}
	return out
}

// LeastPrivilege returns the minimum permission set for the given command
// group. The second return value reports whether the command group was
// found.
func LeastPrivilege(commandGroup string) ([]string, bool) {
	m, err := Load()
	if err != nil {
		return nil, false
	}
	for _, lp := range m.LeastPrivilege {
		if lp.CommandGroup == commandGroup {
			return lp.Permissions, true
		}
	}
	return nil, false
}
