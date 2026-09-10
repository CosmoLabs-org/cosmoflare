package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// executeDecodeCommand runs `cosmoflare decode <args>` through the root
// command (cobra's Execute always resolves via the root, so args must be set
// there — same pattern as executeAlertsCommand in alerts_test.go).
func executeDecodeCommand(args ...string) (string, error) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(append([]string{"decode"}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestDecodeCommandPrintsCauseAndFix(t *testing.T) {
	out, err := executeDecodeCommand("10405")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out, "endpoint does not exist") || !strings.Contains(out, "fix") {
		t.Fatalf("decode output must carry cause and fix: %q", out)
	}
}

func TestDecodeCommandUnknownCodeFails(t *testing.T) {
	if _, err := executeDecodeCommand("99999"); err == nil {
		t.Fatal("unknown code must error")
	}
}

func TestKnowledgeListCommandRegistered(t *testing.T) {
	if knowledgeCmd == nil || knowledgeCmd.Name() != "knowledge" {
		t.Fatal("knowledge command not registered")
	}
}

func TestRateLimitCommandsRegistered(t *testing.T) {
	for _, name := range []string{"list", "create"} {
		found := false
		for _, c := range rateLimitCmd.Commands() {
			if c.Name() == name {
				found = true
			}
		}
		if !found {
			t.Fatalf("ratelimit subcommand %q missing", name)
		}
	}
}
