package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// workerVersionRunGlobals snapshots and restores every package-level flag
// variable the versions runners read, so tests cannot leak state into each
// other or into the shared command tree.
func workerVersionRunGlobals(t *testing.T) {
	t.Helper()
	oldScript, oldCompat, oldModule := workerVersionScript, workerVersionCompatDate, workerVersionModule
	oldForce, oldBindings, oldTags := workerVersionForce, workerBindings, workerTags
	oldToken, oldAccount := APIToken, AccountID
	oldDry, oldJSON := DryRun, JSONOutput
	t.Cleanup(func() {
		workerVersionScript, workerVersionCompatDate, workerVersionModule = oldScript, oldCompat, oldModule
		workerVersionForce, workerBindings, workerTags = oldForce, oldBindings, oldTags
		APIToken, AccountID = oldToken, oldAccount
		DryRun, JSONOutput = oldDry, oldJSON
	})
}

// workerVersionResetFlags clears the versions flag variables and their
// "changed" marks so each subtest starts pristine.
func workerVersionResetFlags() {
	workerVersionScript = ""
	workerVersionCompatDate = ""
	workerVersionModule = false
	workerVersionForce = false
	workerBindings = nil
	workerTags = nil
	for _, c := range []*cobra.Command{workerVersionUploadCmd, workerVersionDeleteCmd} {
		for _, name := range []string{"script", "compatibility-date", "module", "force"} {
			if f := c.Flags().Lookup(name); f != nil {
				f.Changed = false
			}
		}
	}
}

// --- Command registration ---

func TestWorkerVersionsCmd_RegisteredUnderWorker(t *testing.T) {
	found := false
	for _, sub := range workerCmd.Commands() {
		if sub.Name() == "versions" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("workerVersionsCmd not registered under workerCmd")
	}
}

func TestWorkerVersionsCmd_Subcommands(t *testing.T) {
	expected := []string{"upload", "list", "view", "deploy", "delete", "rollback"}
	for _, name := range expected {
		found := false
		for _, sub := range workerVersionsCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("worker versions subcommand %q not registered", name)
		}
	}
}

func TestWorkerVersionsCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		workerVersionUploadCmd,
		workerVersionListCmd,
		workerVersionViewCmd,
		workerVersionDeployCmd,
		workerVersionDeleteCmd,
		workerVersionRollbackCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%s has no RunE wired", c.Name())
		}
	}
}

func TestWorkerVersionsCmd_MetadataAndExamples(t *testing.T) {
	cmds := []*cobra.Command{
		workerVersionsCmd,
		workerVersionUploadCmd,
		workerVersionListCmd,
		workerVersionViewCmd,
		workerVersionDeployCmd,
		workerVersionDeleteCmd,
		workerVersionRollbackCmd,
	}
	for _, c := range cmds {
		if c.Short == "" {
			t.Errorf("%s: Short is empty", c.Name())
		}
		if !strings.Contains(c.Long, "Example") {
			t.Errorf("%s: Long has no examples section", c.Name())
		}
	}
}

// --- Flag surface ---

func TestWorkerVersionUpload_Flags(t *testing.T) {
	if f := workerVersionUploadCmd.Flags().Lookup("script"); f == nil {
		t.Fatal("upload: --script flag missing")
	}
	if f := workerVersionUploadCmd.Flags().ShorthandLookup("s"); f == nil {
		t.Error("upload: --script shorthand -s missing")
	}
	if f := workerVersionUploadCmd.Flags().Lookup("compatibility-date"); f == nil {
		t.Error("upload: --compatibility-date flag missing")
	}
	if f := workerVersionUploadCmd.Flags().Lookup("module"); f == nil {
		t.Error("upload: --module flag missing")
	}
}

func TestWorkerVersionDelete_ForceFlag(t *testing.T) {
	if f := workerVersionDeleteCmd.Flags().Lookup("force"); f == nil {
		t.Fatal("delete: --force flag missing")
	}
	if workerVersionForce {
		t.Error("force should default to false")
	}
}

// --- Offline validation paths ---

func TestWorkerVersionUpload_NoArgs(t *testing.T) {
	workerVersionRunGlobals(t)
	workerVersionResetFlags()

	err := runWorkerVersionUpload(workerVersionUploadCmd, nil)
	if err == nil {
		t.Fatal("expected error when no worker name provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("name")) {
		t.Errorf("error = %q, want it to mention 'name'", err.Error())
	}
}

func TestWorkerVersionUpload_NoScript(t *testing.T) {
	workerVersionRunGlobals(t)
	workerVersionResetFlags()

	err := runWorkerVersionUpload(workerVersionUploadCmd, []string{"my-worker"})
	if err == nil {
		t.Fatal("expected error when no script file provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("script")) {
		t.Errorf("error = %q, want it to mention 'script'", err.Error())
	}
}

func TestWorkerVersionUpload_MissingScriptFile(t *testing.T) {
	workerVersionRunGlobals(t)
	workerVersionResetFlags()
	APIToken = "fake-token-1234567890"
	AccountID = "acct-123"
	workerVersionScript = filepath.Join(t.TempDir(), "does-not-exist.js")

	err := runWorkerVersionUpload(workerVersionUploadCmd, []string{"my-worker"})
	if err == nil {
		t.Fatal("expected error for missing script file")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("failed to open script file")) {
		t.Errorf("error = %q, want failed-to-open message", err.Error())
	}
}

func TestWorkerVersionList_NoArgs(t *testing.T) {
	workerVersionRunGlobals(t)

	if err := runWorkerVersionList(workerVersionListCmd, nil); err == nil {
		t.Fatal("expected error when no worker name provided")
	}
}

func TestWorkerVersionView_NoArgs(t *testing.T) {
	workerVersionRunGlobals(t)

	err := runWorkerVersionView(workerVersionViewCmd, nil)
	if err == nil {
		t.Fatal("expected error when no worker name provided")
	}
}

func TestWorkerVersionView_MissingVersionID(t *testing.T) {
	workerVersionRunGlobals(t)

	err := runWorkerVersionView(workerVersionViewCmd, []string{"my-worker"})
	if err == nil {
		t.Fatal("expected error when no version ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("version ID")) {
		t.Errorf("error = %q, want it to mention 'version ID'", err.Error())
	}
}

func TestWorkerVersionDeploy_MissingVersionID(t *testing.T) {
	workerVersionRunGlobals(t)

	err := runWorkerVersionDeploy(workerVersionDeployCmd, []string{"my-worker"})
	if err == nil {
		t.Fatal("expected error when no version ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("version ID")) {
		t.Errorf("error = %q, want it to mention 'version ID'", err.Error())
	}
}

func TestWorkerVersionDelete_MissingVersionID(t *testing.T) {
	workerVersionRunGlobals(t)

	err := runWorkerVersionDelete(workerVersionDeleteCmd, []string{"my-worker"})
	if err == nil {
		t.Fatal("expected error when no version ID provided")
	}
}

func TestWorkerVersionRollback_MissingVersionID(t *testing.T) {
	workerVersionRunGlobals(t)

	err := runWorkerVersionRollback(workerVersionRollbackCmd, []string{"my-worker"})
	if err == nil {
		t.Fatal("expected error when no version ID provided")
	}
}

func TestWorkerVersions_MissingToken(t *testing.T) {
	workerVersionRunGlobals(t)
	workerVersionResetFlags()
	APIToken = ""

	// Every runner with valid args must fail fast on service construction —
	// no network access, no confirmation prompt.
	if err := runWorkerVersionList(workerVersionListCmd, []string{"my-worker"}); err == nil {
		t.Error("list: expected service-creation error without token")
	}
	if err := runWorkerVersionView(workerVersionViewCmd, []string{"my-worker", "ver-1"}); err == nil {
		t.Error("view: expected service-creation error without token")
	}
	if err := runWorkerVersionDeploy(workerVersionDeployCmd, []string{"my-worker", "ver-1"}); err == nil {
		t.Error("deploy: expected service-creation error without token")
	}
	if err := runWorkerVersionRollback(workerVersionRollbackCmd, []string{"my-worker", "ver-1"}); err == nil {
		t.Error("rollback: expected service-creation error without token")
	}
}

// --- Dry-run happy paths (offline) ---

func TestWorkerVersionUpload_DryRun(t *testing.T) {
	workerVersionRunGlobals(t)
	workerVersionResetFlags()
	DryRun = true
	APIToken = "fake-token-1234567890"
	AccountID = "acct-123"

	dir := t.TempDir()
	script := filepath.Join(dir, "worker.js")
	if err := os.WriteFile(script, []byte("export default {};"), 0o600); err != nil {
		t.Fatalf("writing temp script: %v", err)
	}
	workerVersionScript = script

	if err := runWorkerVersionUpload(workerVersionUploadCmd, []string{"my-worker"}); err != nil {
		t.Fatalf("dry-run upload should succeed offline: %v", err)
	}
}

func TestWorkerVersionDeploy_DryRun(t *testing.T) {
	workerVersionRunGlobals(t)
	DryRun = true
	APIToken = "fake-token-1234567890"
	AccountID = "acct-123"

	if err := runWorkerVersionDeploy(workerVersionDeployCmd, []string{"my-worker", "ver-1"}); err != nil {
		t.Fatalf("dry-run deploy should succeed offline: %v", err)
	}
}

func TestWorkerVersionDelete_DryRun(t *testing.T) {
	workerVersionRunGlobals(t)
	workerVersionResetFlags()
	DryRun = true
	APIToken = "fake-token-1234567890"
	AccountID = "acct-123"

	// Dry-run must skip the confirmation prompt entirely.
	if err := runWorkerVersionDelete(workerVersionDeleteCmd, []string{"my-worker", "ver-1"}); err != nil {
		t.Fatalf("dry-run delete should succeed offline: %v", err)
	}
}

func TestWorkerVersionRollback_DryRunJSON(t *testing.T) {
	workerVersionRunGlobals(t)
	DryRun = true
	JSONOutput = true
	APIToken = "fake-token-1234567890"
	AccountID = "acct-123"

	if err := runWorkerVersionRollback(workerVersionRollbackCmd, []string{"my-worker", "ver-1"}); err != nil {
		t.Fatalf("dry-run JSON rollback should succeed offline: %v", err)
	}
}

// --- Helpers ---

func TestWorkerVersionArgs(t *testing.T) {
	name, ver, err := workerVersionArgs([]string{"w"}, false)
	if err != nil || name != "w" || ver != "" {
		t.Errorf("one-arg lookup = (%q, %q, %v)", name, ver, err)
	}

	name, ver, err = workerVersionArgs([]string{"w", "v"}, true)
	if err != nil || name != "w" || ver != "v" {
		t.Errorf("two-arg lookup = (%q, %q, %v)", name, ver, err)
	}

	if _, _, err := workerVersionArgs(nil, false); err == nil {
		t.Error("expected error for no args")
	}
	if _, _, err := workerVersionArgs([]string{"w"}, true); err == nil {
		t.Error("expected error for missing version ID")
	}
}
