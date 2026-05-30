package cmd

import (
	"encoding/json"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/cosmoflare"
)

// TestDomainsCmd_NotNil verifies domainsCmd is initialised.
func TestDomainsCmd_NotNil(t *testing.T) {
	if domainsCmd == nil {
		t.Fatal("domainsCmd must not be nil")
	}
}

// TestDomainsCmd_Metadata checks Use and Short fields.
func TestDomainsCmd_Metadata(t *testing.T) {
	if domainsCmd.Use != "domains" {
		t.Errorf("domainsCmd.Use = %q, want %q", domainsCmd.Use, "domains")
	}
	want := "List and inspect all domains in your Cloudflare account"
	if domainsCmd.Short != want {
		t.Errorf("domainsCmd.Short = %q, want %q", domainsCmd.Short, want)
	}
}

// TestDomainsCmd_RegisteredOnRoot confirms the command is a child of rootCmd.
func TestDomainsCmd_RegisteredOnRoot(t *testing.T) {
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "domains" {
			return
		}
	}
	t.Error("domainsCmd not found in rootCmd.Commands()")
}

// TestDomainsCmd_HasRunE verifies a RunE handler is wired up.
func TestDomainsCmd_HasRunE(t *testing.T) {
	if domainsCmd.RunE == nil {
		t.Error("domainsCmd.RunE must not be nil")
	}
}

// TestDomainsCmd_FlagDefaults verifies every flag default value.
func TestDomainsCmd_FlagDefaults(t *testing.T) {
	flags := domainsCmd.Flags()

	intCases := []struct {
		name string
		want string
	}{
		{"page", "1"},
		{"per-page", "50"},
	}
	for _, tc := range intCases {
		f := flags.Lookup(tc.name)
		if f == nil {
			t.Errorf("flag --%s not registered", tc.name)
			continue
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}

	stringCases := []struct {
		name string
		want string
	}{
		{"sort", "name"},
		{"filter", ""},
		{"name", ""},
	}
	for _, tc := range stringCases {
		f := flags.Lookup(tc.name)
		if f == nil {
			t.Errorf("flag --%s not registered", tc.name)
			continue
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}

	boolCases := []struct {
		name string
		want string
	}{
		{"detail", "false"},
		{"enrich", "false"},
	}
	for _, tc := range boolCases {
		f := flags.Lookup(tc.name)
		if f == nil {
			t.Errorf("flag --%s not registered", tc.name)
			continue
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

// TestDomainsCmd_FlagsRegistered verifies all expected flags exist.
func TestDomainsCmd_FlagsRegistered(t *testing.T) {
	expected := []string{"filter", "name", "page", "per-page", "sort", "detail", "enrich"}
	for _, name := range expected {
		if domainsCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on domainsCmd", name)
		}
	}
}

// TestDomainsCmd_DryRun verifies runDomains returns nil under DryRun without panicking.
func TestDomainsCmd_DryRun(t *testing.T) {
	// Save state, restore after test.
	origDryRun := DryRun
	origAccountID := AccountID
	origAPIToken := APIToken
	origJSON := JSONOutput
	defer func() {
		DryRun = origDryRun
		AccountID = origAccountID
		APIToken = origAPIToken
		JSONOutput = origJSON
	}()

	DryRun = true
	AccountID = "test-account"
	APIToken = "test-token"
	JSONOutput = false

	// Reset flag vars to defaults so the call is deterministic.
	domainsPage = 1
	domainsPerPage = 50
	domainsSort = "name"
	domainsFilter = ""
	domainsName = ""
	domainsDetail = false
	domainsEnrich = false

	err := runDomains(domainsCmd, nil)
	if err != nil {
		t.Errorf("runDomains(DryRun=true) returned error: %v", err)
	}
}

// TestDomainsResponse_JSONMarshal verifies the envelope marshals correctly.
func TestDomainsResponse_JSONMarshal(t *testing.T) {
	resp := domainsResponse{
		Domains: []*cosmoflare.DomainStatus{},
		Pagination: &cosmoflare.Pagination{
			Page:       1,
			PerPage:    50,
			Total:      0,
			TotalPages: 1,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal(domainsResponse) error: %v", err)
	}

	var out map[string]json.RawMessage
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	for _, key := range []string{"domains", "pagination"} {
		if _, ok := out[key]; !ok {
			t.Errorf("marshaled output missing key %q", key)
		}
	}
}

// TestDomainsResponse_DomainsKey verifies the "domains" key is an array.
func TestDomainsResponse_DomainsKey(t *testing.T) {
	resp := domainsResponse{
		Domains:    []*cosmoflare.DomainStatus{},
		Pagination: &cosmoflare.Pagination{},
	}

	data, _ := json.Marshal(resp)

	var out struct {
		Domains    []json.RawMessage `json:"domains"`
		Pagination json.RawMessage   `json:"pagination"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Empty slice must marshal as [] not null.
	if out.Domains == nil {
		t.Error(`"domains" key should be an array, got null`)
	}
}
