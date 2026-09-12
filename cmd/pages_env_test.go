package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestPagesEnvCmd_RegisteredUnderPages(t *testing.T) {
	found := false
	for _, sub := range pagesCmd.Commands() {
		if sub.Name() == "env" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("pagesEnvCmd not registered under pagesCmd")
	}
}

func TestPagesEnvCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "set", "delete"}
	for _, name := range expected {
		found := false
		for _, sub := range pagesEnvCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pages env subcommand %q not registered", name)
		}
	}
}

func TestPagesEnvCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{pagesEnvListCmd, pagesEnvSetCmd, pagesEnvDeleteCmd}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

func TestPagesEnvSet_SecretFlag(t *testing.T) {
	f := pagesEnvSetCmd.Flags().Lookup("secret")
	if f == nil {
		t.Fatal("--secret flag not registered on pagesEnvSetCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--secret default = %q, want %q", f.DefValue, "false")
	}
}

func TestPagesEnvList_EnvFlag(t *testing.T) {
	f := pagesEnvListCmd.Flags().Lookup("env")
	if f == nil {
		t.Fatal("--env flag not registered on pagesEnvListCmd")
	}
	if f.DefValue != "production" {
		t.Errorf("--env default = %q, want %q", f.DefValue, "production")
	}
}

func TestPagesEnvSet_Args(t *testing.T) {
	if pagesEnvSetCmd.Args == nil {
		t.Error("pagesEnvSetCmd.Args is nil, expected cobra.MinimumNArgs(2)")
	}
	if err := pagesEnvSetCmd.Args(pagesEnvSetCmd, []string{"my-site"}); err == nil {
		t.Error("expected error with only 1 arg (missing KEY=VALUE)")
	}
	if err := pagesEnvSetCmd.Args(pagesEnvSetCmd, []string{"my-site", "K=V"}); err != nil {
		t.Errorf("expected no error with 2 args, got: %v", err)
	}
}

func TestPagesEnvDelete_Args(t *testing.T) {
	if pagesEnvDeleteCmd.Args == nil {
		t.Error("pagesEnvDeleteCmd.Args is nil, expected cobra.ExactArgs(2)")
	}
}

// --- --env validation rejects unknown values ---

func TestPagesEnvList_RejectsUnknownEnv(t *testing.T) {
	origAccountID := AccountID
	origAPIToken := APIToken
	origEnv := pagesEnv
	defer func() {
		AccountID = origAccountID
		APIToken = origAPIToken
		pagesEnv = origEnv
	}()
	AccountID = "test-account"
	APIToken = "test-token"
	pagesEnv = "staging"

	err := runPagesEnvList(pagesEnvListCmd, []string{"my-site"})
	if err == nil {
		t.Error("expected error for unknown --env value")
	}
}

func TestPagesEnvSet_RejectsUnknownEnv(t *testing.T) {
	origAccountID := AccountID
	origAPIToken := APIToken
	origEnv := pagesEnv
	defer func() {
		AccountID = origAccountID
		APIToken = origAPIToken
		pagesEnv = origEnv
	}()
	AccountID = "test-account"
	APIToken = "test-token"
	pagesEnv = "staging"

	err := runPagesEnvSet(pagesEnvSetCmd, []string{"my-site", "K=V"})
	if err == nil {
		t.Error("expected error for unknown --env value")
	}
}

func TestPagesEnvDelete_RejectsUnknownEnv(t *testing.T) {
	origAccountID := AccountID
	origAPIToken := APIToken
	origEnv := pagesEnv
	defer func() {
		AccountID = origAccountID
		APIToken = origAPIToken
		pagesEnv = origEnv
	}()
	AccountID = "test-account"
	APIToken = "test-token"
	pagesEnv = "staging"

	err := runPagesEnvDelete(pagesEnvDeleteCmd, []string{"my-site", "K"})
	if err == nil {
		t.Error("expected error for unknown --env value")
	}
}

// --- DryRun ---

func TestPagesEnvSet_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origEnv := pagesEnv
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		pagesEnv = origEnv
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	pagesEnv = "production"

	err := runPagesEnvSet(pagesEnvSetCmd, []string{"my-site", "API_URL=https://example.com"})
	if err != nil {
		t.Errorf("runPagesEnvSet(DryRun) returned error: %v", err)
	}
}

func TestPagesEnvSet_InvalidPair(t *testing.T) {
	origAccountID := AccountID
	origAPIToken := APIToken
	origEnv := pagesEnv
	defer func() {
		AccountID = origAccountID
		APIToken = origAPIToken
		pagesEnv = origEnv
	}()
	AccountID = "test-account"
	APIToken = "test-token"
	pagesEnv = "production"

	err := runPagesEnvSet(pagesEnvSetCmd, []string{"my-site", "NOEQUALSSIGN"})
	if err == nil {
		t.Error("expected error for malformed KEY=VALUE pair")
	}
}

func TestPagesEnvDelete_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origEnv := pagesEnv
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		pagesEnv = origEnv
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	pagesEnv = "production"

	err := runPagesEnvDelete(pagesEnvDeleteCmd, []string{"my-site", "API_URL"})
	if err != nil {
		t.Errorf("runPagesEnvDelete(DryRun) returned error: %v", err)
	}
}
