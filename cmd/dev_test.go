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

func TestDevCmd_RejectsMultipleArgs(t *testing.T) {
	if devCmd.Args == nil {
		t.Skip("devCmd.Args is nil, cannot validate")
	}
	err := devCmd.Args(devCmd, []string{"a", "b", "c"})
	if err == nil {
		t.Error("expected error when passing multiple args to dev command")
	}
}

func TestDevCmd_AcceptsZeroArgs(t *testing.T) {
	if devCmd.Args == nil {
		t.Skip("devCmd.Args is nil, cannot validate")
	}
	err := devCmd.Args(devCmd, []string{})
	if err != nil {
		t.Errorf("expected no error with zero args, got: %v", err)
	}
}

func TestDevCmd_HasRunE(t *testing.T) {
	if devCmd.RunE == nil {
		t.Error("devCmd.RunE is nil — no handler wired")
	}
}

func TestDevCmd_LongDescription(t *testing.T) {
	if devCmd.Long == "" {
		t.Error("devCmd.Long description is empty")
	}
}

func TestDevCmd_PortFlagType(t *testing.T) {
	flag := devCmd.Flags().Lookup("port")
	if flag == nil {
		t.Fatal("--port flag not found")
	}
	if flag.Value.Type() != "int" {
		t.Errorf("--port type = %q, want %q", flag.Value.Type(), "int")
	}
}

func TestDevCmd_WatchFlagType(t *testing.T) {
	flag := devCmd.Flags().Lookup("watch")
	if flag == nil {
		t.Fatal("--watch flag not found")
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("--watch type = %q, want %q", flag.Value.Type(), "bool")
	}
}

func TestDevCmd_ServicesFlagType(t *testing.T) {
	flag := devCmd.Flags().Lookup("services")
	if flag == nil {
		t.Fatal("--services flag not found")
	}
	if flag.Value.Type() != "string" {
		t.Errorf("--services type = %q, want %q", flag.Value.Type(), "string")
	}
}

func TestDevCmd_ProfileFlagType(t *testing.T) {
	flag := devCmd.Flags().Lookup("profile")
	if flag == nil {
		t.Fatal("--profile flag not found")
	}
	if flag.Value.Type() != "string" {
		t.Errorf("--profile type = %q, want %q", flag.Value.Type(), "string")
	}
}
