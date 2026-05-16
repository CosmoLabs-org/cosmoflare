package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestBucketCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "bucket" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("bucketCmd not registered on rootCmd")
	}
}

func TestBucketCmd_Metadata(t *testing.T) {
	if bucketCmd.Use != "bucket" {
		t.Errorf("bucketCmd.Use = %q, want %q", bucketCmd.Use, "bucket")
	}
	if bucketCmd.Short == "" {
		t.Error("bucketCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestBucketCmd_Subcommands(t *testing.T) {
	expected := []string{"create", "list", "get", "update", "delete", "exists", "import"}
	for _, name := range expected {
		found := false
		for _, sub := range bucketCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("bucket subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestBucketCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		bucketCreateCmd,
		bucketListCmd,
		bucketGetCmd,
		bucketUpdateCmd,
		bucketDeleteCmd,
		bucketExistsCmd,
		bucketImportCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestBucketCreate_Flags(t *testing.T) {
	expected := []string{"location", "tags", "metadata"}
	for _, name := range expected {
		if bucketCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on bucketCreateCmd", name)
		}
	}
}

func TestBucketCreate_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"location", "auto"},
	}
	for _, tc := range cases {
		f := bucketCreateCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestBucketList_Flags(t *testing.T) {
	expected := []string{"format", "prefix", "tag"}
	for _, name := range expected {
		if bucketListCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on bucketListCmd", name)
		}
	}
}

func TestBucketList_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"format", "table"},
		{"prefix", ""},
	}
	for _, tc := range cases {
		f := bucketListCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestBucketGet_Flags(t *testing.T) {
	expected := []string{"output", "include-objects"}
	for _, name := range expected {
		if bucketGetCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on bucketGetCmd", name)
		}
	}
}

func TestBucketGet_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"output", "table"},
		{"include-objects", "false"},
	}
	for _, tc := range cases {
		f := bucketGetCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestBucketUpdate_Flags(t *testing.T) {
	expected := []string{"tags", "metadata", "add-tags", "remove-tags"}
	for _, name := range expected {
		if bucketUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on bucketUpdateCmd", name)
		}
	}
}

func TestBucketDelete_ForceFlag(t *testing.T) {
	f := bucketDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on bucketDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

func TestBucketImport_Flags(t *testing.T) {
	expected := []string{"spec", "continue"}
	for _, name := range expected {
		if bucketImportCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on bucketImportCmd", name)
		}
	}
}

// --- Arg validation ---

func TestBucketCreate_NoArgs(t *testing.T) {
	err := runBucketCreate(bucketCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no bucket name provided")
	}
}

func TestBucketGet_NoArgs(t *testing.T) {
	err := runBucketGet(bucketGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no bucket name provided")
	}
}

func TestBucketUpdate_NoArgs(t *testing.T) {
	err := runBucketUpdate(bucketUpdateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no bucket name provided")
	}
}

func TestBucketDelete_NoArgs(t *testing.T) {
	err := runBucketDelete(bucketDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no bucket name provided")
	}
}

func TestBucketExists_NoArgs(t *testing.T) {
	err := runBucketExists(bucketExistsCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no bucket name provided")
	}
}
