package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestPagesCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "pages" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("pagesCmd not registered on rootCmd")
	}
}

func TestPagesCmd_Metadata(t *testing.T) {
	if pagesCmd.Use != "pages" {
		t.Errorf("pagesCmd.Use = %q, want %q", pagesCmd.Use, "pages")
	}
	if pagesCmd.Short == "" {
		t.Error("pagesCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestPagesCmd_Subcommands(t *testing.T) {
	expected := []string{"create", "list", "get", "delete", "deployments"}
	for _, name := range expected {
		found := false
		for _, sub := range pagesCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pages subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestPagesCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		pagesCreateCmd,
		pagesListCmd,
		pagesGetCmd,
		pagesDeleteCmd,
		pagesDeploymentsCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestPagesCreate_BranchFlag(t *testing.T) {
	f := pagesCreateCmd.Flags().Lookup("branch")
	if f == nil {
		t.Fatal("--branch flag not registered on pagesCreateCmd")
	}
}

func TestPagesDelete_ForceFlag(t *testing.T) {
	f := pagesDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on pagesDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

// --- Arg validation via cobra.ExactArgs ---

func TestPagesCreate_Args(t *testing.T) {
	if pagesCreateCmd.Args == nil {
		t.Error("pagesCreateCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestPagesGet_Args(t *testing.T) {
	if pagesGetCmd.Args == nil {
		t.Error("pagesGetCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestPagesDelete_Args(t *testing.T) {
	if pagesDeleteCmd.Args == nil {
		t.Error("pagesDeleteCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestPagesDeployments_Args(t *testing.T) {
	if pagesDeploymentsCmd.Args == nil {
		t.Error("pagesDeploymentsCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

// --- DryRun mode ---

func TestPagesCreate_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origBranch := pagesBranch
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		pagesBranch = origBranch
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	pagesBranch = "main"

	err := runPagesCreate(pagesCreateCmd, []string{"my-site"})
	if err != nil {
		t.Errorf("runPagesCreate(DryRun) returned error: %v", err)
	}
}

func TestPagesCreate_DryRunJSON(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origBranch := pagesBranch
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		pagesBranch = origBranch
	}()

	DryRun = true
	JSONOutput = true
	AccountID = "test-account"
	APIToken = "test-token"
	pagesBranch = "main"

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runPagesCreate(pagesCreateCmd, []string{"my-site"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runPagesCreate(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

func TestPagesDelete_DryRun(t *testing.T) {
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

	err := runPagesDelete(pagesDeleteCmd, []string{"my-site"})
	if err != nil {
		t.Errorf("runPagesDelete(DryRun) returned error: %v", err)
	}
}
