package cmdmanifest

import (
	"encoding/json"
	"os"
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

// TestPermissionsCompleteForAPICommands pins the FEAT-020 completion
// invariant: every command that touches a CF API endpoint must have its
// permission column authored (at least one scope) OR be explicitly marked
// NoPermsRequired (verified none required, e.g. token verify). Local-only
// commands (no APIOps) are exempt — they never need token permissions.
func TestPermissionsCompleteForAPICommands(t *testing.T) {
	for _, c := range Load().Commands() {
		if len(c.APIOps) == 0 {
			continue
		}
		if len(c.Permissions.Account) > 0 || len(c.Permissions.Zone) > 0 || len(c.Permissions.User) > 0 {
			continue
		}
		if c.NoPermsRequired {
			continue
		}
		t.Errorf("%q: touches %q but declares no permissions and NoPermsRequired=false (sparse = not yet authored)", c.ID, c.APIOps[0])
	}
}

// TestWorkerDomainPermissionsFilled pins the FEAT-020 fill: the three
// worker.domain.* commands call the Workers custom-domains API, which the
// Cloudflare docs gate with the "Workers Scripts" permission group
// (developers.cloudflare.com/api/resources/workers/subresources/domains —
// "Workers Scripts Read"/"Workers Scripts Write"). The expected name is
// cross-checked against the permdata catalog family with id
// account.workers_scripts so a rename upstream fails this test loudly.
func TestWorkerDomainPermissionsFilled(t *testing.T) {
	want := permdataFamilyName(t, "account.workers_scripts")

	m := Load()
	for _, id := range []string{"worker.domain.list", "worker.domain.attach", "worker.domain.detach"} {
		c, ok := m.Get(id)
		if !ok {
			t.Fatalf("%s not registered", id)
		}
		if len(c.Permissions.Account) != 1 {
			t.Errorf("%s: Permissions.Account = %v, want exactly one entry %q", id, c.Permissions.Account, want)
			continue
		}
		if got := c.Permissions.Account[0]; got != want {
			t.Errorf("%s: account permission = %q, want permdata family name %q (id account.workers_scripts)", id, got, want)
		}
		if c.NoPermsRequired {
			t.Errorf("%s: NoPermsRequired must be false — the domains API does check permissions", id)
		}
	}
}

// TestAccountVerifyNoPermsRequired pins the explicit marker: account.verify
// calls GET /user/tokens/verify, which accepts any valid token regardless of
// scopes — empty permissions there mean "verified none required", not
// "not yet authored".
func TestAccountVerifyNoPermsRequired(t *testing.T) {
	c, ok := Load().Get("account.verify")
	if !ok {
		t.Fatal("account.verify not registered")
	}
	if !c.NoPermsRequired {
		t.Error("account.verify: NoPermsRequired = false, want true (GET /user/tokens/verify accepts any valid token)")
	}
	if len(c.Permissions.Account)+len(c.Permissions.Zone)+len(c.Permissions.User) != 0 {
		t.Errorf("account.verify: must not declare permissions, got %+v", c.Permissions)
	}
}

// permdataFamilyName reads the family display name for the given id from
// pkg/cosmoflare/permdata/permissions.json (the FEAT-011 source of truth for
// token-UI permission spellings). Path is relative to this package dir.
func permdataFamilyName(t *testing.T, id string) string {
	t.Helper()
	raw, err := os.ReadFile("../../pkg/cosmoflare/permdata/permissions.json")
	if err != nil {
		t.Fatalf("read permdata catalog: %v", err)
	}
	var cat struct {
		Families []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"families"`
	}
	if err := json.Unmarshal(raw, &cat); err != nil {
		t.Fatalf("parse permdata catalog: %v", err)
	}
	for _, f := range cat.Families {
		if f.ID == id {
			return f.Name
		}
	}
	t.Fatalf("permdata catalog has no family with id %q", id)
	return ""
}

// TestSpectrumWebAnalyticsPermissionsFilled pins the FEAT-011 dataset fill:
// spectrum (needs DNS + Zone Settings — no dedicated Spectrum permission
// exists) and web-analytics (Account Analytics) must no longer have sparse
// permission columns.
func TestSpectrumWebAnalyticsPermissionsFilled(t *testing.T) {
	for _, c := range Load().Commands() {
		if c.Service != "spectrum" && c.Service != "web-analytics" {
			continue
		}
		if len(c.Permissions.Account) == 0 && len(c.Permissions.Zone) == 0 {
			t.Errorf("%q: sparse permissions — spectrum needs Zone DNS + Zone Settings, web-analytics needs Account Analytics", c.ID)
		}
	}
}
