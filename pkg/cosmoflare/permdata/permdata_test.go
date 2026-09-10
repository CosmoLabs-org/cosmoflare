package permdata

import (
	"encoding/json"
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
