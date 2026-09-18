package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFirstRunNudgeOnBareInvocation pins FEAT-017's onboarding promise: bare
// `cosmoflare` with no credential store appends the setup nudge; with a
// store present it stays silent.
func TestFirstRunNudgeOnBareInvocation(t *testing.T) {
	dir := t.TempDir()

	oldPath, oldJSON := firstRunConfigPath, JSONOutput
	savedOut, savedErr := rootCmd.OutOrStdout(), rootCmd.ErrOrStderr()
	t.Cleanup(func() {
		firstRunConfigPath, JSONOutput = oldPath, oldJSON
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(savedOut)
		rootCmd.SetErr(savedErr)
	})
	JSONOutput = false
	firstRunConfigPath = func() string { return filepath.Join(dir, "config.yaml") }

	runBare := func(t *testing.T) string {
		t.Helper()
		return capturePrint(t, func() {
			// Hermetic: a prior test may have left rootCmd writers bound to
			// its own buffer — rebind to the captured stdout for this run.
			rootCmd.SetOut(os.Stdout)
			rootCmd.SetErr(os.Stderr)
			rootCmd.SetArgs([]string{})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("bare invocation: %v", err)
			}
		})
	}

	out := runBare(t)
	if !strings.Contains(out, "cosmoflare setup") {
		t.Errorf("first run should nudge toward setup, got: %q", tail(out, 200))
	}

	// Configure the store: the nudge disappears.
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("profiles: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out = runBare(t)
	if strings.Contains(out, "First run?") {
		t.Errorf("configured store must not nag, got: %q", tail(out, 200))
	}
}

// tail returns the last n bytes of s for compact failure messages.
func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}
