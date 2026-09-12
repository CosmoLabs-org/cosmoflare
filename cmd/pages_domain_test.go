package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestPagesDomainCmd_RegisteredUnderPages(t *testing.T) {
	found := false
	for _, sub := range pagesCmd.Commands() {
		if sub.Name() == "domain" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("pagesDomainCmd not registered under pagesCmd")
	}
}

func TestPagesDomainCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "attach", "detach"}
	for _, name := range expected {
		found := false
		for _, sub := range pagesDomainCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pages domain subcommand %q not registered", name)
		}
	}
}

func TestPagesDomainCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{pagesDomainListCmd, pagesDomainAttachCmd, pagesDomainDetachCmd}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

func TestPagesDomainAttach_Args(t *testing.T) {
	if pagesDomainAttachCmd.Args == nil {
		t.Error("pagesDomainAttachCmd.Args is nil, expected cobra.ExactArgs(2)")
	}
}

func TestPagesDomainDetach_Args(t *testing.T) {
	if pagesDomainDetachCmd.Args == nil {
		t.Error("pagesDomainDetachCmd.Args is nil, expected cobra.ExactArgs(2)")
	}
}

func TestPagesDomainList_Args(t *testing.T) {
	if pagesDomainListCmd.Args == nil {
		t.Error("pagesDomainListCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

// --- DryRun ---

func TestPagesDomainAttach_DryRun(t *testing.T) {
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

	err := runPagesDomainAttach(pagesDomainAttachCmd, []string{"my-site", "example.com"})
	if err != nil {
		t.Errorf("runPagesDomainAttach(DryRun) returned error: %v", err)
	}
}

func TestPagesDomainDetach_DryRun(t *testing.T) {
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

	err := runPagesDomainDetach(pagesDomainDetachCmd, []string{"my-site", "example.com"})
	if err != nil {
		t.Errorf("runPagesDomainDetach(DryRun) returned error: %v", err)
	}
}
