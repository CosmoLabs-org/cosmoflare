package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestCHCmd_RegisteredUnderSSL(t *testing.T) {
	found := false
	for _, sub := range sslCmd.Commands() {
		if sub.Name() == "custom-hostname" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("custom-hostname command not registered under ssl")
	}
}

func TestCHCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "create", "get", "update", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range sslCustomHostnameCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("custom-hostname subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestCHCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{chListCmd, chCreateCmd, chGetCmd, chUpdateCmd, chDeleteCmd}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration and defaults ---

func TestCHList_Flags(t *testing.T) {
	if chListCmd.Flags().Lookup("hostname") == nil {
		t.Error("flag --hostname not registered on chListCmd")
	}
}

func TestCHCreate_Flags(t *testing.T) {
	f := chCreateCmd.Flags().Lookup("origin")
	if f == nil {
		t.Fatal("flag --origin not registered on chCreateCmd")
	}
	if f.DefValue != "" {
		t.Errorf("flag --origin default = %q, want empty", f.DefValue)
	}
}

func TestCHUpdate_Flags(t *testing.T) {
	expected := []string{"hostname", "origin"}
	for _, name := range expected {
		if chUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on chUpdateCmd", name)
		}
	}
}

// --- Arg validation ---

func TestCHList_NoZoneID(t *testing.T) {
	err := runCHList(chListCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !strings.Contains(err.Error(), "zone ID") {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

func TestCHCreate_MissingArgs(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"no args", nil, "zone ID and hostname are required"},
		{"zone only", []string{"zone123"}, "zone ID and hostname are required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runCHCreate(chCreateCmd, tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestCHGet_MissingArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"no args", nil},
		{"zone only", []string{"zone123"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runCHGet(chGetCmd, tc.args)
			if err == nil || !strings.Contains(err.Error(), "zone ID and hostname ID are required") {
				t.Fatalf("expected arg error, got %v", err)
			}
		})
	}
}

func TestCHUpdate_MissingArgs(t *testing.T) {
	err := runCHUpdate(chUpdateCmd, []string{"zone123"})
	if err == nil || !strings.Contains(err.Error(), "zone ID and hostname ID are required") {
		t.Fatalf("expected arg error, got %v", err)
	}
}

func TestCHUpdate_NoFlags(t *testing.T) {
	chRunGlobals(t)
	chFlagReset(t, chUpdateCmd, "hostname", "origin")

	err := runCHUpdate(chUpdateCmd, []string{"zone123", "ch-1"})
	if err == nil {
		t.Fatal("expected error when no update flags provided")
	}
	if !strings.Contains(err.Error(), "update flag") {
		t.Errorf("error = %q, want it to mention 'update flag'", err.Error())
	}
}

func TestCHDelete_MissingArgs(t *testing.T) {
	err := runCHDelete(chDeleteCmd, []string{"zone123"})
	if err == nil || !strings.Contains(err.Error(), "zone ID and hostname ID are required") {
		t.Fatalf("expected arg error, got %v", err)
	}
}

// --- DryRun mode ---

func TestCHCreate_DryRun(t *testing.T) {
	chRunGlobals(t)

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"

	err := runCHCreate(chCreateCmd, []string{"zone123", "app.customer.com"})
	if err != nil {
		t.Errorf("runCHCreate(DryRun) returned error: %v", err)
	}
}

func TestCHCreate_DryRunJSON(t *testing.T) {
	chRunGlobals(t)

	DryRun = true
	JSONOutput = true
	APIToken = "test-token"

	err := runCHCreate(chCreateCmd, []string{"zone123", "app.customer.com"})
	if err != nil {
		t.Errorf("runCHCreate(DryRun+JSON) returned error: %v", err)
	}
}

func TestCHUpdate_DryRun(t *testing.T) {
	chRunGlobals(t)
	chFlagReset(t, chUpdateCmd, "hostname", "origin")

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	if err := chUpdateCmd.Flags().Set("origin", "origin.example.com"); err != nil {
		t.Fatal(err)
	}

	err := runCHUpdate(chUpdateCmd, []string{"zone123", "ch-1"})
	if err != nil {
		t.Errorf("runCHUpdate(DryRun) returned error: %v", err)
	}
}

func TestCHDelete_DryRun(t *testing.T) {
	chRunGlobals(t)

	DryRun = true
	JSONOutput = true
	APIToken = "test-token"

	err := runCHDelete(chDeleteCmd, []string{"zone123", "ch-1"})
	if err != nil {
		t.Errorf("runCHDelete(DryRun+JSON) returned error: %v", err)
	}
}

// --- Offline service creation failure ---

func TestCHList_ServiceFailsOffline(t *testing.T) {
	chRunGlobals(t)

	APIToken = ""
	err := runCHList(chListCmd, []string{"zone123"})
	if err == nil || !strings.Contains(err.Error(), "failed to create SSL custom hostname service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
}

// --- Flag parsing ---

func TestCHFlagsParsing(t *testing.T) {
	cmd := &cobra.Command{}
	var hostname, origin string
	cmd.Flags().StringVar(&hostname, "hostname", "", "")
	cmd.Flags().StringVar(&origin, "origin", "", "")

	if hostname != "" || origin != "" {
		t.Error("flags should default to empty")
	}
	if err := cmd.Flags().Set("hostname", "customer.com"); err != nil {
		t.Fatalf("failed to set --hostname: %v", err)
	}
	if err := cmd.Flags().Set("origin", "origin.example.com"); err != nil {
		t.Fatalf("failed to set --origin: %v", err)
	}
	if hostname != "customer.com" {
		t.Errorf("hostname = %q, want customer.com", hostname)
	}
	if origin != "origin.example.com" {
		t.Errorf("origin = %q, want origin.example.com", origin)
	}
}

// --- Shared helpers ---

// chRunGlobals snapshots and restores the global state the custom hostname
// runners read, so tests cannot leak state.
func chRunGlobals(t *testing.T) {
	t.Helper()
	oldDry, oldJSON, oldToken := DryRun, JSONOutput, APIToken
	t.Cleanup(func() {
		DryRun, JSONOutput, APIToken = oldDry, oldJSON, oldToken
	})
}

// chFlagReset restores the given flags to their default values and clears
// their "changed" marks once the test finishes.
func chFlagReset(t *testing.T, cmd *cobra.Command, names ...string) {
	t.Helper()
	for _, name := range names {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			continue
		}
		t.Cleanup(func() {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	}
}
