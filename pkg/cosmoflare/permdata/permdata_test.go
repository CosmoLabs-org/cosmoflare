package permdata

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	m, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(m.Families) == 0 {
		t.Fatal("Load() returned no families")
	}
	if len(m.LeastPrivilege) == 0 {
		t.Fatal("Load() returned no least_privilege rows")
	}
	if m.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", m.SchemaVersion)
	}
}

func TestLoad_FamilyCountsMatchJSON(t *testing.T) {
	var raw struct {
		Families       []json.RawMessage `json:"families"`
		LeastPrivilege []json.RawMessage `json:"least_privilege"`
	}
	if err := json.Unmarshal(permissionsJSON, &raw); err != nil {
		t.Fatalf("unmarshal raw json: %v", err)
	}

	m, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(m.Families) != len(raw.Families) {
		t.Errorf("Families count = %d, want %d", len(m.Families), len(raw.Families))
	}
	if len(m.LeastPrivilege) != len(raw.LeastPrivilege) {
		t.Errorf("LeastPrivilege count = %d, want %d", len(m.LeastPrivilege), len(raw.LeastPrivilege))
	}
}

func TestFamilies_ScopeFilter(t *testing.T) {
	cases := []string{"account", "zone", "user"}
	for _, scope := range cases {
		fams := Families(scope)
		if len(fams) == 0 {
			t.Errorf("Families(%q) returned none", scope)
		}
		for _, f := range fams {
			if f.Scope != scope {
				t.Errorf("Families(%q) returned family with scope %q", scope, f.Scope)
			}
		}
	}
}

func TestFamilies_EmptyScopeReturnsAll(t *testing.T) {
	all := Families("")
	m, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(all) != len(m.Families) {
		t.Errorf("Families(\"\") returned %d, want %d", len(all), len(m.Families))
	}
}

func TestFamilies_UnknownScope(t *testing.T) {
	fams := Families("nonexistent")
	if len(fams) != 0 {
		t.Errorf("Families(\"nonexistent\") = %d families, want 0", len(fams))
	}
}

func TestLeastPrivilege_Hit(t *testing.T) {
	perms, ok := LeastPrivilege("deploy")
	if !ok {
		t.Fatal("LeastPrivilege(\"deploy\") not found")
	}
	if len(perms) == 0 {
		t.Error("LeastPrivilege(\"deploy\") returned no permissions")
	}
}

func TestLeastPrivilege_Miss(t *testing.T) {
	perms, ok := LeastPrivilege("nonexistent-group")
	if ok {
		t.Error("LeastPrivilege(\"nonexistent-group\") = true, want false")
	}
	if perms != nil {
		t.Errorf("LeastPrivilege(\"nonexistent-group\") permissions = %v, want nil", perms)
	}
}

// TestQwenIngestion_CatalogExpanded pins the FEAT-011 criterion-4 ingestion:
// the Qwen dataset (docs/research/2026-09-17-cf-perms-next-waves-tiers)
// grew the catalog well past the 76-family seed.
func TestQwenIngestion_CatalogExpanded(t *testing.T) {
	m, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(m.Families) < 150 {
		t.Errorf("Families count = %d, want >= 150 after Qwen dataset ingestion", len(m.Families))
	}
	if m.ManifestVersion < "2026-09-18" {
		t.Errorf("ManifestVersion = %q, want >= 2026-09-18 (Qwen verification date)", m.ManifestVersion)
	}
}

// TestQwenIngestion_UnlocksPreserved pins the drift folklore the catalog
// exists for: Zone WAF Edit unlocks the modern rulesets phases, and account
// Logs Edit unlocks Logpush job writes.
func TestQwenIngestion_UnlocksPreserved(t *testing.T) {
	m, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := map[string]string{
		"zone.waf":     "PUT /zones/{zone_id}/rulesets/phases/http_ratelimit/entrypoint",
		"account.logs": "POST /accounts/{account_id}/logpush/jobs",
	}
	got := map[string][]string{}
	for _, f := range m.Families {
		got[f.ID] = f.Unlocks
	}
	for id, unlock := range want {
		if !slices.Contains(got[id], unlock) {
			t.Errorf("family %q: unlocks missing %q (got %v)", id, unlock, got[id])
		}
	}
}

// TestQwenIngestion_VerifiedDatesParse ensures any family carrying a
// verified date uses a real ISO date — a malformed date means the merge
// script broke a row.
func TestQwenIngestion_VerifiedDatesParse(t *testing.T) {
	m, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for _, f := range m.Families {
		if f.Verified == "" {
			continue
		}
		if len(f.Verified) != 10 || f.Verified[4] != '-' || f.Verified[7] != '-' {
			t.Errorf("family %q: verified %q is not an ISO date", f.ID, f.Verified)
		}
	}
}

// TestQwenIngestion_PageShieldRenamePinned records the token-UI rename the
// Qwen pass surfaced: the Page Shield permission family is now "Client-side
// security".
func TestQwenIngestion_PageShieldRenamePinned(t *testing.T) {
	m, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for _, f := range m.Families {
		if f.ID == "zone.page_shield" {
			if !strings.Contains(f.Notes, "Client-side security") {
				t.Errorf("zone.page_shield notes = %q, want mention of Client-side security rename", f.Notes)
			}
			return
		}
	}
	t.Error("zone.page_shield family missing")
}
