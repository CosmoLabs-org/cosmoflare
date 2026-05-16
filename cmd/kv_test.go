package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestKVCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "kv" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("kvCmd not registered on rootCmd")
	}
}

func TestKVCmd_Metadata(t *testing.T) {
	if kvCmd.Use != "kv" {
		t.Errorf("kvCmd.Use = %q, want %q", kvCmd.Use, "kv")
	}
	if kvCmd.Short == "" {
		t.Error("kvCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestKVCmd_Subcommands(t *testing.T) {
	expected := []string{"namespace", "put", "get", "delete", "list"}
	for _, name := range expected {
		found := false
		for _, sub := range kvCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("kv subcommand %q not registered", name)
		}
	}
}

func TestKVNamespaceCmd_Subcommands(t *testing.T) {
	expected := []string{"create", "list", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range kvNamespaceCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("kv namespace subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestKVCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		kvNamespaceCreateCmd,
		kvNamespaceListCmd,
		kvNamespaceDeleteCmd,
		kvPutCmd,
		kvGetCmd,
		kvDeleteCmd,
		kvListCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestKVNamespaceDelete_ForceFlag(t *testing.T) {
	f := kvNamespaceDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on kvNamespaceDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

func TestKVPut_Flags(t *testing.T) {
	expected := []string{"value", "file", "ttl"}
	for _, name := range expected {
		if kvPutCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on kvPutCmd", name)
		}
	}
}

func TestKVPut_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"value", ""},
		{"file", ""},
		{"ttl", "0"},
	}
	for _, tc := range cases {
		f := kvPutCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestKVList_Flags(t *testing.T) {
	expected := []string{"prefix", "limit"}
	for _, name := range expected {
		if kvListCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on kvListCmd", name)
		}
	}
}

func TestKVList_LimitDefault(t *testing.T) {
	f := kvListCmd.Flags().Lookup("limit")
	if f == nil {
		t.Fatal("--limit flag not found on kvListCmd")
	}
	if f.DefValue != "1000" {
		t.Errorf("--limit default = %q, want %q", f.DefValue, "1000")
	}
}

// --- Arg validation ---

func TestKVNamespaceCreate_NoArgs(t *testing.T) {
	err := runKVNamespaceCreate(kvNamespaceCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no namespace title provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("title")) {
		t.Errorf("error = %q, want it to mention 'title'", err.Error())
	}
}

func TestKVNamespaceDelete_NoArgs(t *testing.T) {
	err := runKVNamespaceDelete(kvNamespaceDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no namespace ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("namespace ID")) {
		t.Errorf("error = %q, want it to mention 'namespace ID'", err.Error())
	}
}

func TestKVPut_NoArgs(t *testing.T) {
	err := runKVPut(kvPutCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestKVPut_OneArg(t *testing.T) {
	err := runKVPut(kvPutCmd, []string{"ns-abc"})
	if err == nil {
		t.Fatal("expected error when only namespace ID provided (missing key)")
	}
}

func TestKVPut_NoValue(t *testing.T) {
	kvValue = ""
	kvFile = ""
	err := runKVPut(kvPutCmd, []string{"ns-abc", "my-key"})
	if err == nil {
		t.Fatal("expected error when no value or file provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("value")) {
		t.Errorf("error = %q, want it to mention 'value'", err.Error())
	}
}

func TestKVGet_NoArgs(t *testing.T) {
	err := runKVGet(kvGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestKVGet_OneArg(t *testing.T) {
	err := runKVGet(kvGetCmd, []string{"ns-abc"})
	if err == nil {
		t.Fatal("expected error when only namespace ID provided (missing key)")
	}
}

func TestKVDelete_NoArgs(t *testing.T) {
	err := runKVDelete(kvDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestKVList_NoArgs(t *testing.T) {
	err := runKVList(kvListCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no namespace ID provided")
	}
}

// --- DryRun mode ---

func TestKVNamespaceCreate_DryRun(t *testing.T) {
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

	err := runKVNamespaceCreate(kvNamespaceCreateCmd, []string{"my-cache"})
	if err != nil {
		t.Errorf("runKVNamespaceCreate(DryRun) returned error: %v", err)
	}
}

func TestKVNamespaceDelete_DryRun(t *testing.T) {
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

	err := runKVNamespaceDelete(kvNamespaceDeleteCmd, []string{"ns-abc123"})
	if err != nil {
		t.Errorf("runKVNamespaceDelete(DryRun) returned error: %v", err)
	}
}

func TestKVPut_DryRun(t *testing.T) {
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
	kvValue = "test-value"

	err := runKVPut(kvPutCmd, []string{"ns-abc", "my-key"})
	if err != nil {
		t.Errorf("runKVPut(DryRun) returned error: %v", err)
	}
}

func TestKVDelete_DryRun(t *testing.T) {
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

	err := runKVDelete(kvDeleteCmd, []string{"ns-abc", "my-key"})
	if err != nil {
		t.Errorf("runKVDelete(DryRun) returned error: %v", err)
	}
}
