package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestTemplatesCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "templates" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("templatesCmd not registered on rootCmd")
	}
}

func TestTemplatesCmd_Metadata(t *testing.T) {
	if templatesCmd.Use != "templates" {
		t.Errorf("templatesCmd.Use = %q, want %q", templatesCmd.Use, "templates")
	}
	if templatesCmd.Short == "" {
		t.Error("templatesCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestTemplatesCmd_Subcommands(t *testing.T) {
	expected := []string{"list", "info", "create"}
	for _, name := range expected {
		found := false
		for _, sub := range templatesCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("templates subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestTemplatesCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		templatesListCmd,
		templatesInfoCmd,
		templatesCreateCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestTemplatesCreate_Flags(t *testing.T) {
	expected := []string{"force"}
	for _, name := range expected {
		if templatesCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on templatesCreateCmd", name)
		}
	}
}

func TestTemplatesCreate_ForceDefault(t *testing.T) {
	f := templatesCreateCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not found")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

// --- Templates list output ---

func TestTemplatesList_Output(t *testing.T) {
	buf := new(bytes.Buffer)
	templatesListCmd.SetOut(buf)
	templatesListCmd.SetErr(buf)

	// Reset JSONOutput for this test
	origJSON := JSONOutput
	JSONOutput = false
	defer func() { JSONOutput = origJSON }()

	err := templatesListCmd.RunE(templatesListCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	templates := []string{"api", "static", "fullstack", "cron", "queue"}
	for _, name := range templates {
		if !bytes.Contains([]byte(output), []byte(name)) {
			t.Errorf("list output should contain template %q", name)
		}
	}
}

// --- Templates info validation ---

func TestTemplatesInfo_RequiresArg(t *testing.T) {
	buf := new(bytes.Buffer)
	templatesInfoCmd.SetOut(buf)
	templatesInfoCmd.SetErr(buf)

	err := templatesInfoCmd.RunE(templatesInfoCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no template name provided")
	}
}

func TestTemplatesInfo_UnknownTemplate(t *testing.T) {
	buf := new(bytes.Buffer)
	templatesInfoCmd.SetOut(buf)
	templatesInfoCmd.SetErr(buf)

	err := templatesInfoCmd.RunE(templatesInfoCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
}
