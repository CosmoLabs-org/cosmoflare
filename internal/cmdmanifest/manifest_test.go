package cmdmanifest

import (
	"strings"
	"testing"
)

// TestManifestIDsAndPathsUnique pins the registry invariants: every command
// has a unique id and a unique CLI path (FEAT-020).
func TestManifestIDsAndPathsUnique(t *testing.T) {
	m := Load()
	seenID := map[string]bool{}
	seenPath := map[string]bool{}
	for _, c := range m.Commands() {
		if c.ID == "" {
			t.Fatalf("command with empty id: %+v", c)
		}
		if seenID[c.ID] {
			t.Errorf("duplicate id %q", c.ID)
		}
		seenID[c.ID] = true

		if len(c.CLIPath) == 0 {
			t.Fatalf("command %q has no CLI path", c.ID)
		}
		key := strings.Join(c.CLIPath, " ")
		if seenPath[key] {
			t.Errorf("duplicate CLI path %q (id %q)", key, c.ID)
		}
		seenPath[key] = true
	}
}

// TestManifestRequiredFields pins descriptor completeness: id, service, verb,
// and danger level must always be present — a half-described command is the
// drift the registry exists to prevent.
func TestManifestRequiredFields(t *testing.T) {
	for _, c := range Load().Commands() {
		if c.Service == "" {
			t.Errorf("%q: empty service", c.ID)
		}
		switch c.Verb {
		case "read", "write", "delete", "list":
		default:
			t.Errorf("%q: verb %q not one of read|write|delete|list", c.ID, c.Verb)
		}
		switch c.DangerLevel {
		case "low", "medium", "high":
		default:
			t.Errorf("%q: danger level %q not one of low|medium|high", c.ID, c.DangerLevel)
		}
	}
}

// TestManifestDestructiveImpliesHighDanger pins the consistency rule the
// audit consumer relies on: a destructive command is never labeled low/medium.
func TestManifestDestructiveImpliesHighDanger(t *testing.T) {
	for _, c := range Load().Commands() {
		if c.Destructive && c.DangerLevel != "high" {
			t.Errorf("%q: destructive but danger=%q, want high", c.ID, c.DangerLevel)
		}
	}
}

// TestResolveCLI covers path resolution: exact hit, miss, and the id-first
// segment matching used by the audit consumer.
func TestResolveCLI(t *testing.T) {
	m := Load()

	c, ok := m.ResolveCLI("bucket", "delete")
	if !ok {
		t.Fatal("r2 bucket delete not registered")
	}
	if c.ID != "r2.bucket.delete" {
		t.Errorf("resolved id = %q, want r2.bucket.delete", c.ID)
	}
	if !c.Destructive {
		t.Error("r2.bucket.delete must be marked destructive")
	}

	if _, ok := m.ResolveCLI("r2", "nope"); ok {
		t.Error("unknown path resolved")
	}
	if _, ok := m.ResolveCLI(); ok {
		t.Error("empty path resolved")
	}
}

// TestGetByID covers direct id lookup.
func TestGetByID(t *testing.T) {
	m := Load()
	c, ok := m.Get("r2.bucket.create")
	if !ok {
		t.Fatal("r2.bucket.create not registered")
	}
	if len(c.Permissions.Account) == 0 {
		t.Error("r2.bucket.create should declare its account permission")
	}
	if _, ok := m.Get("not.an.id"); ok {
		t.Error("unknown id resolved")
	}
}

// TestWranglerEquivalentsParsable guards the consumer convention: an empty
// wrangler_equivalent means "no wrangler counterpart"; a non-empty one starts
// with "wrangler ".
func TestWranglerEquivalentsParsable(t *testing.T) {
	for _, c := range Load().Commands() {
		if c.WranglerEquivalent == "" {
			continue
		}
		if !strings.HasPrefix(c.WranglerEquivalent, "wrangler ") {
			t.Errorf("%q: wrangler equivalent %q must start with \"wrangler \"", c.ID, c.WranglerEquivalent)
		}
	}
}
