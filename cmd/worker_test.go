package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
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

// --- parseWorkerBindings edge cases ---

func TestParseWorkerBindings_TwoParts(t *testing.T) {
	_, err := parseWorkerBindings([]string{"NAME:type"})
	if err == nil {
		t.Fatal("expected error for binding with only 2 parts")
	}
}

func TestParseWorkerBindings_Empty(t *testing.T) {
	bindings, err := parseWorkerBindings([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bindings) != 0 {
		t.Errorf("expected 0 bindings, got %d", len(bindings))
	}
}

// --- Deploy error paths ---

func TestWorkerDeploy_NonexistentScript(t *testing.T) {
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

	workerScript = "/nonexistent/script.js"
	err := runWorkerDeploy(workerDeployCmd, []string{"my-worker"})
	if err == nil {
		t.Fatal("expected error for nonexistent script file")
	}
}

// --- Deploy with options DryRun ---

func TestWorkerDeploy_DryRunWithOptions(t *testing.T) {
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

	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/test.js"
	if err := os.WriteFile(scriptPath, []byte("export default {}"), 0644); err != nil {
		t.Fatal(err)
	}
	workerScript = scriptPath
	workerCompatDate = "2024-01-01"
	workerBindings = []string{"MY_KV:kv:ns-123"}
	workerTags = []string{"production"}
	workerModule = true
	defer func() {
		workerCompatDate = ""
		workerBindings = nil
		workerTags = nil
		workerModule = false
	}()

	err := runWorkerDeploy(workerDeployCmd, []string{"my-worker"})
	if err != nil {
		t.Errorf("runWorkerDeploy(DryRun+options) returned error: %v", err)
	}
}

func TestWorkerDeploy_DryRunInvalidBindings(t *testing.T) {
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

	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/test.js"
	if err := os.WriteFile(scriptPath, []byte("export default {}"), 0644); err != nil {
		t.Fatal(err)
	}
	workerScript = scriptPath
	workerBindings = []string{"invalid-format"}
	defer func() { workerBindings = nil }()

	err := runWorkerDeploy(workerDeployCmd, []string{"my-worker"})
	if err == nil {
		t.Fatal("expected error for invalid binding format")
	}
}

// --- Worker settings DryRun JSON ---

func TestWorkerSettings_DryRunJSON(t *testing.T) {
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
	workerCompatDate = "2024-01-01"
	defer func() { workerCompatDate = "" }()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWorkerSettings(workerSettingsCmd, []string{"my-worker"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runWorkerSettings(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

func TestWorkerSettings_DryRunWithBindings(t *testing.T) {
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
	workerCompatDate = ""
	workerUsageModel = ""
	workerBindings = []string{"MY_KV:kv:ns-123"}
	defer func() { workerBindings = nil }()

	err := runWorkerSettings(workerSettingsCmd, []string{"my-worker"})
	if err != nil {
		t.Errorf("runWorkerSettings(DryRun+bindings) returned error: %v", err)
	}
}

// --- Worker delete DryRun JSON ---

func TestWorkerDelete_DryRunJSON(t *testing.T) {
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

	err := runWorkerDelete(workerDeleteCmd, []string{"my-worker"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runWorkerDelete(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

// --- logs --follow flags ---

func TestWorkerLogs_FollowFlags(t *testing.T) {
	flags := []struct {
		name     string
		defValue string
	}{
		{"follow", "false"},
		{"interval", "2"},
		{"level", ""},
		{"since", ""},
	}

	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			flag := workerLogsCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Fatalf("flag --%s not found on workerLogsCmd", f.name)
			}
			if flag.DefValue != f.defValue {
				t.Errorf("flag --%s default = %q, want %q", f.name, flag.DefValue, f.defValue)
			}
		})
	}
}

func TestWorkerLogs_FollowShorthand(t *testing.T) {
	f := workerLogsCmd.Flags().Lookup("follow")
	if f == nil {
		t.Fatal("--follow flag not found")
	}
	if f.Shorthand != "f" {
		t.Errorf("--follow shorthand = %q, want %q", f.Shorthand, "f")
	}
}

// --- Additional flag coverage ---

func TestWorkerDeploy_ModuleFlag(t *testing.T) {
	f := workerDeployCmd.Flags().Lookup("module")
	if f == nil {
		t.Fatal("--module flag not registered on workerDeployCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--module default = %q, want %q", f.DefValue, "false")
	}
}

func TestWorkerDeploy_BindingsDefault(t *testing.T) {
	f := workerDeployCmd.Flags().Lookup("bindings")
	if f == nil {
		t.Fatal("--bindings flag not registered on workerDeployCmd")
	}
	// StringSlice default is "[]"
	if f.DefValue != "[]" {
		t.Errorf("--bindings default = %q, want %q", f.DefValue, "[]")
	}
}

func TestWorkerDeploy_TagsDefault(t *testing.T) {
	f := workerDeployCmd.Flags().Lookup("tags")
	if f == nil {
		t.Fatal("--tags flag not registered on workerDeployCmd")
	}
	if f.DefValue != "[]" {
		t.Errorf("--tags default = %q, want %q", f.DefValue, "[]")
	}
}

func TestWorkerSettings_CompatibilityDateDefault(t *testing.T) {
	f := workerSettingsCmd.Flags().Lookup("compatibility-date")
	if f == nil {
		t.Fatal("--compatibility-date flag not registered on workerSettingsCmd")
	}
	if f.DefValue != "" {
		t.Errorf("--compatibility-date default = %q, want empty string", f.DefValue)
	}
}

func TestWorkerSettings_UsageModelDefault(t *testing.T) {
	f := workerSettingsCmd.Flags().Lookup("usage-model")
	if f == nil {
		t.Fatal("--usage-model flag not registered on workerSettingsCmd")
	}
	if f.DefValue != "" {
		t.Errorf("--usage-model default = %q, want empty string", f.DefValue)
	}
}

func TestWorkerDeploy_CompatibilityDateDefault(t *testing.T) {
	f := workerDeployCmd.Flags().Lookup("compatibility-date")
	if f == nil {
		t.Fatal("--compatibility-date flag not registered on workerDeployCmd")
	}
	if f.DefValue != "" {
		t.Errorf("--compatibility-date default = %q, want empty string", f.DefValue)
	}
}

// --- Command Use fields ---

func TestWorkerSubcmdUseFields(t *testing.T) {
	cases := []struct {
		cmd  *cobra.Command
		want string
	}{
		{workerDeployCmd, "deploy [name]"},
		{workerListCmd, "list"},
		{workerGetCmd, "get [name]"},
		{workerDeleteCmd, "delete [name]"},
		{workerLogsCmd, "logs [name]"},
		{workerSettingsCmd, "settings [name]"},
	}
	for _, tc := range cases {
		if tc.cmd.Use != tc.want {
			t.Errorf("%s.Use = %q, want %q", tc.cmd.Name(), tc.cmd.Use, tc.want)
		}
	}
}

// --- Long description content ---

func TestWorkerCmd_LongContainsExamples(t *testing.T) {
	if !bytes.Contains([]byte(workerCmd.Long), []byte("deploy")) {
		t.Error("workerCmd.Long should mention 'deploy'")
	}
	if !bytes.Contains([]byte(workerCmd.Long), []byte("cosmoflare worker")) {
		t.Error("workerCmd.Long should contain example with 'cosmoflare worker'")
	}
}

func TestWorkerDeployCmd_LongContainsExamples(t *testing.T) {
	if !bytes.Contains([]byte(workerDeployCmd.Long), []byte("--script")) {
		t.Error("workerDeployCmd.Long should mention '--script'")
	}
}

func TestWorkerLogsCmd_LongContainsFollow(t *testing.T) {
	if !bytes.Contains([]byte(workerLogsCmd.Long), []byte("--follow")) {
		t.Error("workerLogsCmd.Long should mention '--follow'")
	}
}

// --- parseWorkerBindings: colon in ID (SplitN 3 parts) ---

func TestParseWorkerBindings_ColonInID(t *testing.T) {
	// SplitN with n=3 means the ID part can contain colons
	bindings, err := parseWorkerBindings([]string{"MY_DO:d1:db:extra"})
	if err != nil {
		t.Fatalf("unexpected error for binding with colon in ID: %v", err)
	}
	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(bindings))
	}
	if bindings[0].ID != "db:extra" {
		t.Errorf("ID = %q, want %q", bindings[0].ID, "db:extra")
	}
}

// --- runWorkerDeploy: DryRun=false exits early at service creation with bad creds ---

func TestWorkerDelete_DryRunFalseForceTrue(t *testing.T) {
	origDryRun := DryRun
	origForce := workerForce
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		workerForce = origForce
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = false
	workerForce = true
	AccountID = ""
	APIToken = ""

	err := runWorkerDelete(workerDeleteCmd, []string{"my-worker"})
	// Should fail at service creation, not at arg validation
	if err == nil {
		t.Fatal("expected error when no credentials provided")
	}
}

// --- workerLogsCmd interval default ---

func TestWorkerLogs_IntervalDefault(t *testing.T) {
	f := workerLogsCmd.Flags().Lookup("interval")
	if f == nil {
		t.Fatal("--interval flag not found on workerLogsCmd")
	}
	if f.DefValue != "2" {
		t.Errorf("--interval default = %q, want %q", f.DefValue, "2")
	}
}

// --- worker delete Short description ---

func TestWorkerDeleteCmd_ShortNotEmpty(t *testing.T) {
	if workerDeleteCmd.Short == "" {
		t.Error("workerDeleteCmd.Short is empty")
	}
	if !bytes.Contains([]byte(workerDeleteCmd.Short), []byte("Delete")) {
		t.Errorf("workerDeleteCmd.Short = %q, want it to mention 'Delete'", workerDeleteCmd.Short)
	}
}

// --- All subcommand Short fields non-empty ---

func TestWorkerSubcmds_ShortNotEmpty(t *testing.T) {
	cmds := []*cobra.Command{
		workerDeployCmd,
		workerListCmd,
		workerGetCmd,
		workerDeleteCmd,
		workerLogsCmd,
		workerSettingsCmd,
	}
	for _, c := range cmds {
		if c.Short == "" {
			t.Errorf("%q Short is empty", c.Use)
		}
	}
}

// --- Profile prefix scoping (FEAT-026) ---

// workerPrefixTestEnv snapshots the globals the prefix-scoping tests touch
// and installs a staging profile with resource prefix "stg-".
func workerPrefixTestEnv(profile *config.Profile) func() {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origProfile := ActiveProfile
	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	ActiveProfile = profile
	return func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		ActiveProfile = origProfile
	}
}

func TestWorkerDeploy_PrefixScoping(t *testing.T) {
	script := filepath.Join(t.TempDir(), "worker.js")
	if err := os.WriteFile(script, []byte("export default {}"), 0o644); err != nil {
		t.Fatalf("writing script: %v", err)
	}

	tests := []struct {
		name    string
		profile *config.Profile
		arg     string
		want    string
	}{
		{"prefix applied", &config.Profile{Name: "staging", ResourcePrefix: "stg-"}, "my-worker", "stg-my-worker"},
		{"already prefixed passes through", &config.Profile{Name: "staging", ResourcePrefix: "stg-"}, "stg-my-worker", "stg-my-worker"},
		{"nil profile is a no-op", nil, "my-worker", "my-worker"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := workerPrefixTestEnv(tt.profile)
			defer restore()
			workerScript = script
			defer func() { workerScript = "" }()

			out := capturePrint(t, func() {
				if err := runWorkerDeploy(workerDeployCmd, []string{tt.arg}); err != nil {
					t.Errorf("runWorkerDeploy returned error: %v", err)
				}
			})
			if !strings.Contains(out, tt.want) {
				t.Errorf("dry-run output should mention %q, got: %q", tt.want, out)
			}
		})
	}
}

func TestWorkerDelete_PrefixScoping(t *testing.T) {
	restore := workerPrefixTestEnv(&config.Profile{Name: "staging", ResourcePrefix: "stg-"})
	defer restore()

	out := capturePrint(t, func() {
		if err := runWorkerDelete(workerDeleteCmd, []string{"my-worker"}); err != nil {
			t.Errorf("runWorkerDelete returned error: %v", err)
		}
	})
	if !strings.Contains(out, "stg-my-worker") {
		t.Errorf("dry-run output should mention %q, got: %q", "stg-my-worker", out)
	}
}

func TestFilterWorkersByProfile(t *testing.T) {
	workers := []*cosmoflare.Worker{
		{Name: "stg-api"},
		{Name: "prod-api"},
	}

	ActiveProfile = &config.Profile{Name: "staging", ResourcePrefix: "stg-"}
	got := filterWorkersByProfile(workers)
	if len(got) != 1 || got[0].Name != "stg-api" {
		t.Errorf("filterWorkersByProfile with prefix = %v, want only stg-api", got)
	}

	ActiveProfile = nil
	got = filterWorkersByProfile(workers)
	if len(got) != 2 {
		t.Errorf("filterWorkersByProfile with nil profile kept %d, want 2", len(got))
	}
}
