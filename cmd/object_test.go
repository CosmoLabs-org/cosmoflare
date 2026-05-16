package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestObjectCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "object" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("objectCmd not registered on rootCmd")
	}
}

func TestObjectCmd_Metadata(t *testing.T) {
	if objectCmd.Use != "object" {
		t.Errorf("objectCmd.Use = %q, want %q", objectCmd.Use, "object")
	}
	if objectCmd.Short == "" {
		t.Error("objectCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestObjectCmd_Subcommands(t *testing.T) {
	expected := []string{"ls", "get", "put", "delete", "copy", "head", "search", "batch", "presign"}
	for _, name := range expected {
		found := false
		for _, sub := range objectCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("object subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestObjectCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		objectListCmd,
		objectGetCmd,
		objectPutCmd,
		objectDeleteCmd,
		objectCopyCmd,
		objectHeadCmd,
		objectSearchCmd,
		objectBatchCmd,
		objectPresignCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestObjectList_Flags(t *testing.T) {
	expected := []string{"prefix", "delimiter", "max-keys", "recursive"}
	for _, name := range expected {
		if objectListCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on objectListCmd", name)
		}
	}
}

func TestObjectList_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"prefix", ""},
		{"delimiter", ""},
		{"max-keys", "0"},
		{"recursive", "false"},
	}
	for _, tc := range cases {
		f := objectListCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestObjectGet_Flags(t *testing.T) {
	expected := []string{"output", "range-start", "range-end"}
	for _, name := range expected {
		if objectGetCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on objectGetCmd", name)
		}
	}
}

func TestObjectPut_Flags(t *testing.T) {
	expected := []string{"key", "content-type", "cache-control", "metadata", "progress"}
	for _, name := range expected {
		if objectPutCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on objectPutCmd", name)
		}
	}
}

func TestObjectPut_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"key", ""},
		{"content-type", ""},
		{"cache-control", ""},
		{"progress", "true"},
	}
	for _, tc := range cases {
		f := objectPutCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestObjectCopy_Flags(t *testing.T) {
	expected := []string{"metadata", "content-type"}
	for _, name := range expected {
		if objectCopyCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on objectCopyCmd", name)
		}
	}
}

func TestObjectHead_Flags(t *testing.T) {
	f := objectHeadCmd.Flags().Lookup("output")
	if f == nil {
		t.Fatal("--output flag not registered on objectHeadCmd")
	}
	if f.DefValue != "table" {
		t.Errorf("--output default = %q, want %q", f.DefValue, "table")
	}
}

func TestObjectSearch_Flags(t *testing.T) {
	f := objectSearchCmd.Flags().Lookup("type")
	if f == nil {
		t.Fatal("--type flag not registered on objectSearchCmd")
	}
	if f.DefValue != "prefix" {
		t.Errorf("--type default = %q, want %q", f.DefValue, "prefix")
	}
}

func TestObjectBatch_Flags(t *testing.T) {
	f := objectBatchCmd.Flags().Lookup("continue")
	if f == nil {
		t.Fatal("--continue flag not registered on objectBatchCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--continue default = %q, want %q", f.DefValue, "false")
	}
}

func TestObjectPresign_Flags(t *testing.T) {
	f := objectPresignCmd.Flags().Lookup("expires")
	if f == nil {
		t.Fatal("--expires flag not registered on objectPresignCmd")
	}
	if f.DefValue != "1h" {
		t.Errorf("--expires default = %q, want %q", f.DefValue, "1h")
	}
}

// --- Arg validation ---

func TestObjectList_NoArgs(t *testing.T) {
	err := runObjectList(objectListCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no bucket name provided")
	}
}

func TestObjectGet_NoArgs(t *testing.T) {
	err := runObjectGet(objectGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectGet_OneArg(t *testing.T) {
	err := runObjectGet(objectGetCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing object key)")
	}
}

func TestObjectPut_NoArgs(t *testing.T) {
	err := runObjectPut(objectPutCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectPut_OneArg(t *testing.T) {
	err := runObjectPut(objectPutCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing local path)")
	}
}

func TestObjectDelete_NoArgs(t *testing.T) {
	err := runObjectDelete(objectDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectDelete_OneArg(t *testing.T) {
	err := runObjectDelete(objectDeleteCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing object key)")
	}
}

func TestObjectCopy_NoArgs(t *testing.T) {
	err := runObjectCopy(objectCopyCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectCopy_OneArg(t *testing.T) {
	err := runObjectCopy(objectCopyCmd, []string{"src-bucket/file.txt"})
	if err == nil {
		t.Fatal("expected error when only source provided (missing destination)")
	}
}

func TestObjectCopy_InvalidSource(t *testing.T) {
	err := runObjectCopy(objectCopyCmd, []string{"no-slash", "dest-bucket/file.txt"})
	if err == nil {
		t.Fatal("expected error when source missing bucket/key separator")
	}
}

func TestObjectCopy_InvalidDest(t *testing.T) {
	err := runObjectCopy(objectCopyCmd, []string{"src-bucket/file.txt", "no-slash"})
	if err == nil {
		t.Fatal("expected error when destination missing bucket/key separator")
	}
}

func TestObjectHead_NoArgs(t *testing.T) {
	err := runObjectHead(objectHeadCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectHead_OneArg(t *testing.T) {
	err := runObjectHead(objectHeadCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing object key)")
	}
}

func TestObjectSearch_NoArgs(t *testing.T) {
	err := runObjectSearch(objectSearchCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectSearch_OneArg(t *testing.T) {
	err := runObjectSearch(objectSearchCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing search query)")
	}
}

func TestObjectBatch_NoArgs(t *testing.T) {
	err := runObjectBatch(objectBatchCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectBatch_OneArg(t *testing.T) {
	err := runObjectBatch(objectBatchCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing spec file)")
	}
}

func TestObjectPresign_NoArgs(t *testing.T) {
	err := runObjectPresign(objectPresignCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectPresign_OneArg(t *testing.T) {
	err := runObjectPresign(objectPresignCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing object key)")
	}
}
