package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestPagesDeploymentCmd_RegisteredUnderPages(t *testing.T) {
	found := false
	for _, sub := range pagesCmd.Commands() {
		if sub.Name() == "deployment" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("pagesDeploymentCmd not registered under pagesCmd")
	}
}

func TestPagesDeploymentCmd_Subcommands(t *testing.T) {
	expected := []string{"view", "retry", "logs"}
	for _, name := range expected {
		found := false
		for _, sub := range pagesDeploymentCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pages deployment subcommand %q not registered", name)
		}
	}
}

func TestPagesDeploymentCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{pagesDeploymentViewCmd, pagesDeploymentRetryCmd, pagesDeploymentLogsCmd}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

func TestPagesDeploymentView_Args(t *testing.T) {
	if pagesDeploymentViewCmd.Args == nil {
		t.Error("pagesDeploymentViewCmd.Args is nil, expected cobra.ExactArgs(2)")
	}
}

func TestPagesDeploymentRetry_Args(t *testing.T) {
	if pagesDeploymentRetryCmd.Args == nil {
		t.Error("pagesDeploymentRetryCmd.Args is nil, expected cobra.ExactArgs(2)")
	}
}

func TestPagesDeploymentLogs_Args(t *testing.T) {
	if pagesDeploymentLogsCmd.Args == nil {
		t.Error("pagesDeploymentLogsCmd.Args is nil, expected cobra.ExactArgs(2)")
	}
}

// --- DryRun ---

func TestPagesDeploymentRetry_DryRun(t *testing.T) {
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

	err := runPagesDeploymentRetry(pagesDeploymentRetryCmd, []string{"my-site", "dep-abc123"})
	if err != nil {
		t.Errorf("runPagesDeploymentRetry(DryRun) returned error: %v", err)
	}
}
