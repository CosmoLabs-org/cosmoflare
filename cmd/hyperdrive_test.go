package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestHyperdriveCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "hyperdrive" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("hyperdriveCmd not registered on rootCmd")
	}
}

func TestHyperdriveCmd_Metadata(t *testing.T) {
	if hyperdriveCmd.Use != "hyperdrive" {
		t.Errorf("hyperdriveCmd.Use = %q, want %q", hyperdriveCmd.Use, "hyperdrive")
	}
	if hyperdriveCmd.Short == "" {
		t.Error("hyperdriveCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestHyperdriveCmd_Subcommands(t *testing.T) {
	expected := []string{"create", "list", "get", "update", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range hyperdriveCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("hyperdrive subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestHyperdriveCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		hyperdriveCreateCmd,
		hyperdriveListCmd,
		hyperdriveGetCmd,
		hyperdriveUpdateCmd,
		hyperdriveDeleteCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestHyperdriveCreate_Flags(t *testing.T) {
	expected := []string{"origin-host", "origin-port", "origin-scheme", "database", "user", "password"}
	for _, name := range expected {
		if hyperdriveCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on hyperdriveCreateCmd", name)
		}
	}
}

func TestHyperdriveUpdate_Flags(t *testing.T) {
	expected := []string{"name", "origin-host", "origin-port", "origin-scheme", "database", "user", "password"}
	for _, name := range expected {
		if hyperdriveUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on hyperdriveUpdateCmd", name)
		}
	}
}

func TestHyperdriveDelete_ForceFlag(t *testing.T) {
	f := hyperdriveDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on hyperdriveDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

// --- Arg validation ---

func TestHyperdriveCreate_Args(t *testing.T) {
	if hyperdriveCreateCmd.Args == nil {
		t.Error("hyperdriveCreateCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestHyperdriveGet_Args(t *testing.T) {
	if hyperdriveGetCmd.Args == nil {
		t.Error("hyperdriveGetCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestHyperdriveUpdate_Args(t *testing.T) {
	if hyperdriveUpdateCmd.Args == nil {
		t.Error("hyperdriveUpdateCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestHyperdriveDelete_Args(t *testing.T) {
	if hyperdriveDeleteCmd.Args == nil {
		t.Error("hyperdriveDeleteCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

// --- DryRun mode ---

func TestHyperdriveCreate_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"

	// Set required flags
	hyperdriveCreateCmd.Flags().Set("origin-host", "db.example.com")
	hyperdriveCreateCmd.Flags().Set("origin-port", "5432")
	hyperdriveCreateCmd.Flags().Set("origin-scheme", "postgres")
	hyperdriveCreateCmd.Flags().Set("database", "mydb")
	hyperdriveCreateCmd.Flags().Set("user", "admin")
	hyperdriveCreateCmd.Flags().Set("password", "secret")

	err := runHyperdriveCreate(hyperdriveCreateCmd, []string{"my-config"})
	if err != nil {
		t.Errorf("runHyperdriveCreate(DryRun) returned error: %v", err)
	}
}

func TestHyperdriveCreate_DryRunJSON(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = true
	AccountID = "test-account"
	APIToken = "test-token"

	hyperdriveCreateCmd.Flags().Set("origin-host", "db.example.com")
	hyperdriveCreateCmd.Flags().Set("origin-port", "5432")
	hyperdriveCreateCmd.Flags().Set("origin-scheme", "postgres")
	hyperdriveCreateCmd.Flags().Set("database", "mydb")
	hyperdriveCreateCmd.Flags().Set("user", "admin")
	hyperdriveCreateCmd.Flags().Set("password", "secret")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runHyperdriveCreate(hyperdriveCreateCmd, []string{"my-config"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runHyperdriveCreate(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

func TestHyperdriveDelete_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"

	err := runHyperdriveDelete(hyperdriveDeleteCmd, []string{"cfg-001"})
	if err != nil {
		t.Errorf("runHyperdriveDelete(DryRun) returned error: %v", err)
	}
}
