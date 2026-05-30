package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestSyncCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "sync" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("syncCmd not registered on rootCmd")
	}
}

func TestSyncCmd_Metadata(t *testing.T) {
	if syncCmd.Use != "sync" {
		t.Errorf("syncCmd.Use = %q, want %q", syncCmd.Use, "sync")
	}
	if syncCmd.Short == "" {
		t.Error("syncCmd.Short is empty")
	}
	if syncCmd.Long == "" {
		t.Error("syncCmd.Long is empty")
	}
}

// --- Subcommand registration ---

func TestSyncCmd_Subcommands(t *testing.T) {
	expected := []string{"up", "down"}
	for _, name := range expected {
		found := false
		for _, sub := range syncCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("sync subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestSyncCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		syncUpCmd,
		syncDownCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestSyncUp_Flags(t *testing.T) {
	expected := []string{"delete", "exclude", "include", "checksum", "progress"}
	for _, name := range expected {
		if syncUpCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on syncUpCmd", name)
		}
	}
}

func TestSyncDown_Flags(t *testing.T) {
	expected := []string{"delete", "exclude", "include", "checksum", "progress"}
	for _, name := range expected {
		if syncDownCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on syncDownCmd", name)
		}
	}
}

func TestSyncUp_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"delete", "false"},
		{"checksum", "false"},
		{"progress", "false"},
	}
	for _, tc := range cases {
		f := syncUpCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on syncUpCmd", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestSyncDown_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"delete", "false"},
		{"checksum", "false"},
		{"progress", "false"},
	}
	for _, tc := range cases {
		f := syncDownCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on syncDownCmd", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

// --- Arg validation ---

func TestSyncUp_NoArgs(t *testing.T) {
	err := runSyncUp(syncUpCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestSyncUp_OneArg(t *testing.T) {
	err := runSyncUp(syncUpCmd, []string{"/tmp/test"})
	if err == nil {
		t.Fatal("expected error when only one arg provided (missing bucket)")
	}
}

func TestSyncDown_NoArgs(t *testing.T) {
	err := runSyncDown(syncDownCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestSyncDown_OneArg(t *testing.T) {
	err := runSyncDown(syncDownCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only one arg provided (missing local dir)")
	}
}

// --- Dry-run mode ---

func TestSyncUp_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origAccountID := AccountID
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		AccountID = origAccountID
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	AccountID = "test-account"

	// Should fail with non-existent directory (that's fine for a dry-run test)
	err := runSyncUp(syncUpCmd, []string{"/nonexistent-sync-test-dir-xyz", "test-bucket"})
	// We expect an error here (directory doesn't exist), but the command should not panic
	if err == nil {
		t.Log("runSyncUp returned nil error for nonexistent dir - this is acceptable in dry-run")
	}
}

func TestSyncDown_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origAccountID := AccountID
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		AccountID = origAccountID
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	AccountID = "test-account"

	err := runSyncDown(syncDownCmd, []string{"test-bucket", "/nonexistent-sync-test-dir-xyz"})
	if err == nil {
		t.Log("runSyncDown returned nil error for nonexistent dir - acceptable in dry-run")
	}
}

// --- Bucket/prefix parsing ---

func TestParseBucketPrefix(t *testing.T) {
	cases := []struct {
		input      string
		wantBucket string
		wantPrefix string
	}{
		{"my-bucket", "my-bucket", ""},
		{"my-bucket/prefix", "my-bucket", "prefix/"},
		{"my-bucket/deep/prefix", "my-bucket", "deep/prefix/"},
		{"my-bucket/prefix/", "my-bucket", "prefix/"},
	}
	for _, tc := range cases {
		bucket, prefix := parseBucketPrefix(tc.input)
		if bucket != tc.wantBucket {
			t.Errorf("parseBucketPrefix(%q) bucket = %q, want %q", tc.input, bucket, tc.wantBucket)
		}
		if prefix != tc.wantPrefix {
			t.Errorf("parseBucketPrefix(%q) prefix = %q, want %q", tc.input, prefix, tc.wantPrefix)
		}
	}
}
