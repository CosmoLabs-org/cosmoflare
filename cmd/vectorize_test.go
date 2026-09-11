package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestVectorizeCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "vectorize" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("vectorizeCmd not registered on rootCmd")
	}
}

func TestVectorizeCmd_Metadata(t *testing.T) {
	if vectorizeCmd.Use != "vectorize" {
		t.Errorf("vectorizeCmd.Use = %q, want %q", vectorizeCmd.Use, "vectorize")
	}
	if vectorizeCmd.Short == "" {
		t.Error("vectorizeCmd.Short is empty")
	}
}

func TestVectorizeCmd_Subcommands(t *testing.T) {
	expected := []string{"create", "list", "get", "delete", "insert", "query"}
	subs := vectorizeCmd.Commands()
	nameSet := make(map[string]bool)
	for _, s := range subs {
		nameSet[s.Use] = true
	}
	for _, name := range expected {
		found := false
		for _, s := range subs {
			if s.Use == name || len(s.Use) > len(name) && s.Use[:len(name)] == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q on vectorizeCmd", name)
		}
	}
}

func TestVectorizeCmd_AllRunE(t *testing.T) {
	for _, sub := range vectorizeCmd.Commands() {
		if sub.RunE == nil {
			t.Errorf("vectorize subcommand %q has nil RunE", sub.Use)
		}
	}
}

func TestVectorizeCreate_Flags(t *testing.T) {
	flags := []struct {
		name     string
		defValue string
	}{
		{"dimensions", "0"},
		{"metric", "cosine"},
	}
	var createCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Use == "create [name]" || sub.Name() == "create" {
			createCmd = sub
			break
		}
	}
	if createCmd == nil {
		t.Fatal("create subcommand not found")
	}
	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			flag := createCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Fatalf("flag --%s not found", f.name)
			}
			if flag.DefValue != f.defValue {
				t.Errorf("flag --%s default = %q, want %q", f.name, flag.DefValue, f.defValue)
			}
		})
	}
}

func TestVectorizeDelete_ForceFlag(t *testing.T) {
	var deleteCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "delete" {
			deleteCmd = sub
			break
		}
	}
	if deleteCmd == nil {
		t.Fatal("delete subcommand not found")
	}
	f := deleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not found on delete")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

func TestVectorizeQuery_Flags(t *testing.T) {
	var queryCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "query" {
			queryCmd = sub
			break
		}
	}
	if queryCmd == nil {
		t.Fatal("query subcommand not found")
	}
	for _, name := range []string{"top-k", "values"} {
		if queryCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not found on query", name)
		}
	}
}

func TestVectorizeCmd_LongDescription(t *testing.T) {
	if vectorizeCmd.Long == "" {
		t.Error("vectorizeCmd.Long description is empty")
	}
}

func TestVectorizeCmd_SubcommandCount(t *testing.T) {
	subs := vectorizeCmd.Commands()
	if len(subs) != 10 {
		t.Errorf("vectorizeCmd has %d subcommands, want 10 (create, list, get, delete, insert, query, upsert, get-vector, delete-vectors, namespaces)", len(subs))
	}
}

func TestVectorizeInsert_Flags(t *testing.T) {
	var insertCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "insert" {
			insertCmd = sub
			break
		}
	}
	if insertCmd == nil {
		t.Fatal("insert subcommand not found")
	}
	flags := []struct {
		name     string
		defValue string
	}{
		{"file", ""},
		{"id", ""},
		{"values", ""},
	}
	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			flag := insertCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Fatalf("flag --%s not found on insert", f.name)
			}
			if flag.DefValue != f.defValue {
				t.Errorf("flag --%s default = %q, want %q", f.name, flag.DefValue, f.defValue)
			}
		})
	}
}

func TestVectorizeCreate_ArgsValidation(t *testing.T) {
	var createCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "create" {
			createCmd = sub
			break
		}
	}
	if createCmd == nil {
		t.Fatal("create subcommand not found")
	}
	if createCmd.Args == nil {
		t.Fatal("create subcommand has nil Args validator")
	}
	// No args should fail (ExactArgs(1))
	if err := createCmd.Args(createCmd, []string{}); err == nil {
		t.Error("expected error with no args for vectorize create")
	}
	// One arg should succeed
	if err := createCmd.Args(createCmd, []string{"my-index"}); err != nil {
		t.Errorf("expected no error with one arg, got: %v", err)
	}
	// Two args should fail
	if err := createCmd.Args(createCmd, []string{"a", "b"}); err == nil {
		t.Error("expected error with two args for vectorize create")
	}
}

func TestVectorizeQuery_TopKDefault(t *testing.T) {
	var queryCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "query" {
			queryCmd = sub
			break
		}
	}
	if queryCmd == nil {
		t.Fatal("query subcommand not found")
	}
	f := queryCmd.Flags().Lookup("top-k")
	if f == nil {
		t.Fatal("--top-k flag not found")
	}
	if f.DefValue != "10" {
		t.Errorf("--top-k default = %q, want %q", f.DefValue, "10")
	}
}

func TestVectorizeGet_ArgsValidation(t *testing.T) {
	var getCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "get" {
			getCmd = sub
			break
		}
	}
	if getCmd == nil {
		t.Fatal("get subcommand not found")
	}
	if getCmd.Args == nil {
		t.Fatal("get subcommand has nil Args validator")
	}
	if err := getCmd.Args(getCmd, []string{}); err == nil {
		t.Error("expected error with no args for vectorize get")
	}
	if err := getCmd.Args(getCmd, []string{"idx"}); err != nil {
		t.Errorf("expected no error with one arg, got: %v", err)
	}
}

func TestVectorizeDelete_ArgsValidation(t *testing.T) {
	var delCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "delete" {
			delCmd = sub
			break
		}
	}
	if delCmd == nil {
		t.Fatal("delete subcommand not found")
	}
	if delCmd.Args == nil {
		t.Fatal("delete subcommand has nil Args validator")
	}
	if err := delCmd.Args(delCmd, []string{}); err == nil {
		t.Error("expected error with no args for vectorize delete")
	}
	if err := delCmd.Args(delCmd, []string{"idx"}); err != nil {
		t.Errorf("expected no error with one arg, got: %v", err)
	}
	if err := delCmd.Args(delCmd, []string{"a", "b"}); err == nil {
		t.Error("expected error with two args for vectorize delete")
	}
}
