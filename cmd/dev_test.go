package cmd

import (
	"testing"
)

func TestDevCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "dev" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("devCmd not registered on rootCmd")
	}
}

func TestDevCmd_Metadata(t *testing.T) {
	if devCmd.Use != "dev" {
		t.Errorf("devCmd.Use = %q, want %q", devCmd.Use, "dev")
	}
	if devCmd.Short == "" {
		t.Error("devCmd.Short is empty")
	}
}

func TestDevCmd_Flags(t *testing.T) {
	flags := []struct {
		name     string
		defValue string
	}{
		{"port", "8787"},
		{"watch", "true"},
		{"services", ""},
		{"profile", ""},
	}

	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			flag := devCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Fatalf("flag --%s not found on devCmd", f.name)
			}
			if flag.DefValue != f.defValue {
				t.Errorf("flag --%s default = %q, want %q", f.name, flag.DefValue, f.defValue)
			}
		})
	}
}

func TestDevCmd_RequiresNoArgs(t *testing.T) {
	// dev command should work with zero args (reads config from cwd)
	if devCmd.Args != nil {
		// cobra.NoArgs is the expected validator, but if Args is nil that's also fine
		err := devCmd.Args(devCmd, []string{"unexpected"})
		if err == nil {
			t.Error("expected error when passing unexpected args")
		}
	}
}
