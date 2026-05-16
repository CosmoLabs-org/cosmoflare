package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestWorkerCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "worker" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("workerCmd not registered on rootCmd")
	}
}

func TestWorkerCmd_Metadata(t *testing.T) {
	if workerCmd.Use != "worker" {
		t.Errorf("workerCmd.Use = %q, want %q", workerCmd.Use, "worker")
	}
	if workerCmd.Short == "" {
		t.Error("workerCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestWorkerCmd_Subcommands(t *testing.T) {
	expected := []string{"deploy", "list", "get", "delete", "logs", "settings"}
	for _, name := range expected {
		found := false
		for _, sub := range workerCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("worker subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestWorkerCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		workerDeployCmd,
		workerListCmd,
		workerGetCmd,
		workerDeleteCmd,
		workerLogsCmd,
		workerSettingsCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestWorkerDeploy_Flags(t *testing.T) {
	expected := []string{"script", "compatibility-date", "bindings", "tags", "module"}
	for _, name := range expected {
		if workerDeployCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on workerDeployCmd", name)
		}
	}
}

func TestWorkerDeploy_ScriptFlagShorthand(t *testing.T) {
	f := workerDeployCmd.Flags().Lookup("script")
	if f == nil {
		t.Fatal("--script flag not found")
	}
	if f.Shorthand != "s" {
		t.Errorf("--script shorthand = %q, want %q", f.Shorthand, "s")
	}
}

func TestWorkerDelete_ForceFlag(t *testing.T) {
	f := workerDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on workerDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

func TestWorkerLogs_LimitFlag(t *testing.T) {
	f := workerLogsCmd.Flags().Lookup("limit")
	if f == nil {
		t.Fatal("--limit flag not registered on workerLogsCmd")
	}
	if f.DefValue != "100" {
		t.Errorf("--limit default = %q, want %q", f.DefValue, "100")
	}
}

func TestWorkerSettings_Flags(t *testing.T) {
	expected := []string{"compatibility-date", "usage-model", "bindings"}
	for _, name := range expected {
		if workerSettingsCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on workerSettingsCmd", name)
		}
	}
}

// --- Arg validation ---

func TestWorkerDeploy_NoArgs(t *testing.T) {
	err := runWorkerDeploy(workerDeployCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no worker name provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("name")) {
		t.Errorf("error = %q, want it to mention 'name'", err.Error())
	}
}

func TestWorkerDeploy_NoScript(t *testing.T) {
	workerScript = ""
	err := runWorkerDeploy(workerDeployCmd, []string{"my-worker"})
	if err == nil {
		t.Fatal("expected error when no script file provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("script")) {
		t.Errorf("error = %q, want it to mention 'script'", err.Error())
	}
}

func TestWorkerGet_NoArgs(t *testing.T) {
	err := runWorkerGet(workerGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no worker name provided")
	}
}

func TestWorkerDelete_NoArgs(t *testing.T) {
	err := runWorkerDelete(workerDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no worker name provided")
	}
}

func TestWorkerLogs_NoArgs(t *testing.T) {
	err := runWorkerLogs(workerLogsCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no worker name provided")
	}
}

func TestWorkerSettings_NoArgs(t *testing.T) {
	err := runWorkerSettings(workerSettingsCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no worker name provided")
	}
}

func TestWorkerSettings_NoSettings(t *testing.T) {
	workerCompatDate = ""
	workerUsageModel = ""
	workerBindings = nil
	err := runWorkerSettings(workerSettingsCmd, []string{"my-worker"})
	if err == nil {
		t.Fatal("expected error when no settings provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("setting")) {
		t.Errorf("error = %q, want it to mention 'setting'", err.Error())
	}
}

// --- DryRun mode ---

func TestWorkerDeploy_DryRun(t *testing.T) {
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

	// Worker deploy opens the script file before DryRun check, so create a temp file
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(scriptPath, []byte("export default {}"), 0644); err != nil {
		t.Fatalf("failed to create temp script: %v", err)
	}
	workerScript = scriptPath

	err := runWorkerDeploy(workerDeployCmd, []string{"my-worker"})
	if err != nil {
		t.Errorf("runWorkerDeploy(DryRun) returned error: %v", err)
	}
}

func TestWorkerDeploy_DryRunJSON(t *testing.T) {
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

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(scriptPath, []byte("export default {}"), 0644); err != nil {
		t.Fatalf("failed to create temp script: %v", err)
	}
	workerScript = scriptPath

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWorkerDeploy(workerDeployCmd, []string{"my-worker"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runWorkerDeploy(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

func TestWorkerDelete_DryRun(t *testing.T) {
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

	err := runWorkerDelete(workerDeleteCmd, []string{"my-worker"})
	if err != nil {
		t.Errorf("runWorkerDelete(DryRun) returned error: %v", err)
	}
}

func TestWorkerSettings_DryRun(t *testing.T) {
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
	workerCompatDate = "2024-01-01"

	err := runWorkerSettings(workerSettingsCmd, []string{"my-worker"})
	if err != nil {
		t.Errorf("runWorkerSettings(DryRun) returned error: %v", err)
	}
}

// --- parseWorkerBindings ---

func TestParseWorkerBindings_Valid(t *testing.T) {
	bindings, err := parseWorkerBindings([]string{"MY_KV:kv:ns-123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(bindings))
	}
	if bindings[0].Name != "MY_KV" || bindings[0].Type != "kv" || bindings[0].ID != "ns-123" {
		t.Errorf("binding = %+v, want {Name:MY_KV Type:kv ID:ns-123}", bindings[0])
	}
}

func TestParseWorkerBindings_Invalid(t *testing.T) {
	_, err := parseWorkerBindings([]string{"invalid-format"})
	if err == nil {
		t.Fatal("expected error for invalid binding format")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("invalid binding format")) {
		t.Errorf("error = %q, want 'invalid binding format'", err.Error())
	}
}

func TestParseWorkerBindings_Multiple(t *testing.T) {
	bindings, err := parseWorkerBindings([]string{"A:kv:1", "B:r2:2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(bindings))
	}
}
