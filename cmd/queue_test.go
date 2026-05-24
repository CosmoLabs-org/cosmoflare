package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestQueueCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "queue" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("queueCmd not registered on rootCmd")
	}
}

func TestQueueCmd_Metadata(t *testing.T) {
	if queueCmd.Use != "queue" {
		t.Errorf("queueCmd.Use = %q, want %q", queueCmd.Use, "queue")
	}
	if queueCmd.Short == "" {
		t.Error("queueCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestQueueCmd_Subcommands(t *testing.T) {
	expected := []string{"create", "list", "get", "update", "delete", "consumers"}
	for _, name := range expected {
		found := false
		for _, sub := range queueCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("queue subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestQueueCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		queueCreateCmd,
		queueListCmd,
		queueGetCmd,
		queueUpdateCmd,
		queueDeleteCmd,
		queueConsumersCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestQueueDelete_ForceFlag(t *testing.T) {
	f := queueDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on queueDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

func TestQueueUpdate_NameFlag(t *testing.T) {
	f := queueUpdateCmd.Flags().Lookup("name")
	if f == nil {
		t.Fatal("--name flag not registered on queueUpdateCmd")
	}
}

// --- Arg validation via cobra.ExactArgs ---

func TestQueueCreate_Args(t *testing.T) {
	if queueCreateCmd.Args == nil {
		t.Error("queueCreateCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestQueueGet_Args(t *testing.T) {
	if queueGetCmd.Args == nil {
		t.Error("queueGetCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestQueueUpdate_Args(t *testing.T) {
	if queueUpdateCmd.Args == nil {
		t.Error("queueUpdateCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestQueueDelete_Args(t *testing.T) {
	if queueDeleteCmd.Args == nil {
		t.Error("queueDeleteCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestQueueConsumers_Args(t *testing.T) {
	if queueConsumersCmd.Args == nil {
		t.Error("queueConsumersCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

// --- DryRun mode ---

func TestQueueCreate_DryRun(t *testing.T) {
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

	err := runQueueCreate(queueCreateCmd, []string{"my-queue"})
	if err != nil {
		t.Errorf("runQueueCreate(DryRun) returned error: %v", err)
	}
}

func TestQueueCreate_DryRunJSON(t *testing.T) {
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

	err := runQueueCreate(queueCreateCmd, []string{"my-queue"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runQueueCreate(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

func TestQueueDelete_DryRun(t *testing.T) {
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

	err := runQueueDelete(queueDeleteCmd, []string{"my-queue"})
	if err != nil {
		t.Errorf("runQueueDelete(DryRun) returned error: %v", err)
	}
}

func TestQueueUpdate_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origNewName := queueNewName
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		queueNewName = origNewName
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	queueNewName = "new-name"

	err := runQueueUpdate(queueUpdateCmd, []string{"old-name"})
	if err != nil {
		t.Errorf("runQueueUpdate(DryRun) returned error: %v", err)
	}
}

// --- Queue update validation ---

func TestQueueUpdate_NoName(t *testing.T) {
	origNewName := queueNewName
	defer func() { queueNewName = origNewName }()

	queueNewName = ""
	err := runQueueUpdate(queueUpdateCmd, []string{"old-name"})
	if err == nil {
		t.Fatal("expected error when --name flag is empty")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("name")) {
		t.Errorf("error = %q, want it to mention 'name'", err.Error())
	}
}
