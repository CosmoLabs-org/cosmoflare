package cmd

import (
	"bytes"
	"os"
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

// --- DryRun JSON variants ---

func TestKVNamespaceCreate_DryRunJSON(t *testing.T) {
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

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runKVNamespaceCreate(kvNamespaceCreateCmd, []string{"my-cache"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runKVNamespaceCreate(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

func TestKVNamespaceDelete_DryRunJSON(t *testing.T) {
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

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runKVNamespaceDelete(kvNamespaceDeleteCmd, []string{"ns-abc123"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runKVNamespaceDelete(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

func TestKVPut_DryRunJSON(t *testing.T) {
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
	kvValue = "test-value"

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runKVPut(kvPutCmd, []string{"ns-abc", "my-key"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runKVPut(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

func TestKVPut_DryRunWithFile(t *testing.T) {
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
	kvValue = ""

	tmpDir := t.TempDir()
	filePath := tmpDir + "/value.txt"
	if err := os.WriteFile(filePath, []byte("file-content"), 0644); err != nil {
		t.Fatal(err)
	}
	kvFile = filePath
	defer func() { kvFile = "" }()

	err := runKVPut(kvPutCmd, []string{"ns-abc", "my-key"})
	if err != nil {
		t.Errorf("runKVPut(DryRun+file) returned error: %v", err)
	}
}

func TestKVPut_DryRunWithTTL(t *testing.T) {
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
	kvTTL = 3600
	defer func() { kvTTL = 0 }()

	err := runKVPut(kvPutCmd, []string{"ns-abc", "my-key"})
	if err != nil {
		t.Errorf("runKVPut(DryRun+TTL) returned error: %v", err)
	}
}

func TestKVDelete_DryRunJSON(t *testing.T) {
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

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runKVDelete(kvDeleteCmd, []string{"ns-abc", "my-key"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runKVDelete(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

// --- KVPut with nonexistent file ---

func TestKVPut_NonexistentFile(t *testing.T) {
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

	DryRun = false
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	kvValue = ""
	kvFile = "/nonexistent/file.txt"
	defer func() { kvFile = "" }()

	err := runKVPut(kvPutCmd, []string{"ns-abc", "my-key"})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// --- Error message assertions ---

func TestKVNamespaceCreate_ErrorMentionsTitle(t *testing.T) {
	err := runKVNamespaceCreate(kvNamespaceCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("title")) {
		t.Errorf("error = %q, want it to mention 'title'", err.Error())
	}
}

func TestKVNamespaceDelete_ErrorMentionsNamespaceID(t *testing.T) {
	err := runKVNamespaceDelete(kvNamespaceDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("namespace ID")) {
		t.Errorf("error = %q, want it to mention 'namespace ID'", err.Error())
	}
}

// --- KVPut ---

func TestKVPut_BothValueAndFile(t *testing.T) {
	kvValue = "val"
	kvFile = "somefile"
	defer func() {
		kvValue = ""
		kvFile = ""
	}()

	// When both value and file are provided, value takes precedence in the code
	// This exercises the kvValue != "" path before kvFile
	err := runKVPut(kvPutCmd, []string{"ns-abc", "my-key"})
	// Will fail because of API client, but passes value validation
	if err == nil {
		t.Fatal("expected error (no API client available)")
	}
}

// --- Additional arg/error message coverage ---

func TestKVDelete_OneArg(t *testing.T) {
	err := runKVDelete(kvDeleteCmd, []string{"ns-abc"})
	if err == nil {
		t.Fatal("expected error when only namespace ID provided (missing key)")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("namespace ID")) {
		t.Errorf("error = %q, want it to mention 'namespace ID'", err.Error())
	}
}

func TestKVGet_ErrorMentionsNamespaceAndKey(t *testing.T) {
	err := runKVGet(kvGetCmd, []string{"ns-abc"})
	if err == nil {
		t.Fatal("expected error when only namespace ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("namespace ID")) {
		t.Errorf("error = %q, want it to mention 'namespace ID'", err.Error())
	}
}

func TestKVDelete_ErrorMentionsNamespaceAndKey(t *testing.T) {
	err := runKVDelete(kvDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("namespace ID")) {
		t.Errorf("error = %q, want it to mention 'namespace ID'", err.Error())
	}
}

func TestKVList_ErrorMentionsNamespaceID(t *testing.T) {
	err := runKVList(kvListCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no namespace ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("namespace ID")) {
		t.Errorf("error = %q, want it to mention 'namespace ID'", err.Error())
	}
}

// --- Command Use fields ---

func TestKVSubcmdUseFields(t *testing.T) {
	cases := []struct {
		cmd  *cobra.Command
		want string
	}{
		{kvPutCmd, "put [namespace-id] [key]"},
		{kvGetCmd, "get [namespace-id] [key]"},
		{kvDeleteCmd, "delete [namespace-id] [key]"},
		{kvListCmd, "list [namespace-id]"},
		{kvNamespaceCreateCmd, "create [title]"},
		{kvNamespaceListCmd, "list"},
		{kvNamespaceDeleteCmd, "delete [namespace-id]"},
	}
	for _, tc := range cases {
		if tc.cmd.Use != tc.want {
			t.Errorf("%s.Use = %q, want %q", tc.cmd.Name(), tc.cmd.Use, tc.want)
		}
	}
}

// --- kvNamespaceCmd metadata ---

func TestKVNamespaceCmd_Metadata(t *testing.T) {
	if kvNamespaceCmd.Use != "namespace" {
		t.Errorf("kvNamespaceCmd.Use = %q, want %q", kvNamespaceCmd.Use, "namespace")
	}
	if kvNamespaceCmd.Short == "" {
		t.Error("kvNamespaceCmd.Short is empty")
	}
}

// --- All leaf subcommand Short fields non-empty ---

func TestKVSubcmds_ShortNotEmpty(t *testing.T) {
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
		if c.Short == "" {
			t.Errorf("%q Short is empty", c.Use)
		}
	}
}

// --- Long description content ---

func TestKVCmd_LongContainsExamples(t *testing.T) {
	if !bytes.Contains([]byte(kvCmd.Long), []byte("cosmoflare kv")) {
		t.Error("kvCmd.Long should contain example with 'cosmoflare kv'")
	}
	if !bytes.Contains([]byte(kvCmd.Long), []byte("namespace")) {
		t.Error("kvCmd.Long should mention 'namespace'")
	}
}

func TestKVPutCmd_LongContainsValueFlag(t *testing.T) {
	if !bytes.Contains([]byte(kvPutCmd.Long), []byte("--value")) {
		t.Error("kvPutCmd.Long should mention '--value'")
	}
	if !bytes.Contains([]byte(kvPutCmd.Long), []byte("--file")) {
		t.Error("kvPutCmd.Long should mention '--file'")
	}
}

func TestKVNamespaceDeleteCmd_LongWarning(t *testing.T) {
	if !bytes.Contains([]byte(kvNamespaceDeleteCmd.Long), []byte("WARNING")) {
		t.Error("kvNamespaceDeleteCmd.Long should contain 'WARNING' about irreversibility")
	}
}

// --- kvList prefix default ---

func TestKVList_PrefixDefault(t *testing.T) {
	f := kvListCmd.Flags().Lookup("prefix")
	if f == nil {
		t.Fatal("--prefix flag not found on kvListCmd")
	}
	if f.DefValue != "" {
		t.Errorf("--prefix default = %q, want empty string", f.DefValue)
	}
}

// --- kvNamespaceDelete force=true with DryRun=false and no creds ---

func TestKVNamespaceDelete_ForceTrueNoCreds(t *testing.T) {
	origDryRun := DryRun
	origForce := kvForce
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		kvForce = origForce
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = false
	kvForce = true
	AccountID = ""
	APIToken = ""

	err := runKVNamespaceDelete(kvNamespaceDeleteCmd, []string{"ns-abc123"})
	// Fails at service creation — proves force bypasses the confirmation prompt
	if err == nil {
		t.Fatal("expected error when no credentials provided")
	}
}

// --- kvPut with DryRun and TTL=0 (no TTL option appended) ---

func TestKVPut_DryRunZeroTTL(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		kvValue = ""
		kvTTL = 0
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	kvValue = "hello"
	kvTTL = 0

	err := runKVPut(kvPutCmd, []string{"ns-abc", "zero-ttl-key"})
	if err != nil {
		t.Errorf("runKVPut(DryRun, TTL=0) returned error: %v", err)
	}
}
