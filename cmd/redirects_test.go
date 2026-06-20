package cmd

import (
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// TestRedirectsCommandWiring asserts the redirects command group and its
// subcommands exist and are non-nil. No live credentials or network calls.
func TestRedirectsCommandWiring(t *testing.T) {
	if redirectsCmd == nil {
		t.Fatal("redirectsCmd is nil")
	}
	if redirectsCmd.Use != "redirects" {
		t.Errorf("redirectsCmd.Use = %q, want %q", redirectsCmd.Use, "redirects")
	}
	if redirectsListCmd == nil {
		t.Error("redirectsListCmd is nil")
	}
	if redirectsCreateCmd == nil {
		t.Error("redirectsCreateCmd is nil")
	}
	if redirectsDeleteCmd == nil {
		t.Error("redirectsDeleteCmd is nil")
	}
}

// TestRedirectsSubcommandsRegistered verifies the three subcommands are
// registered under redirectsCmd, and that redirectsCmd is registered on root.
func TestRedirectsSubcommandsRegistered(t *testing.T) {
	want := map[string]bool{"list": false, "create": false, "delete": false}
	for _, c := range redirectsCmd.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not registered under redirectsCmd", name)
		}
	}

	var onRoot bool
	for _, c := range rootCmd.Commands() {
		if c == redirectsCmd {
			onRoot = true
			break
		}
	}
	if !onRoot {
		t.Error("redirectsCmd is not registered on rootCmd")
	}
}

// TestRedirectsCreateFlags verifies the create subcommand registers its flags
// with the expected default for --status.
func TestRedirectsCreateFlags(t *testing.T) {
	for _, name := range []string{"when", "dest", "status", "preserve-query"} {
		if redirectsCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("create subcommand missing --%s flag", name)
		}
	}
	statusFlag := redirectsCreateCmd.Flags().Lookup("status")
	if statusFlag != nil && statusFlag.DefValue != "301" {
		t.Errorf("--status default = %q, want %q", statusFlag.DefValue, "301")
	}
}

// TestNewRedirectServiceOverridable confirms the factory var can be swapped for
// testing without invoking live Cloudflare credentials.
func TestNewRedirectServiceOverridable(t *testing.T) {
	orig := newRedirectService
	t.Cleanup(func() { newRedirectService = orig })

	called := false
	newRedirectService = func() (*cosmoflare.RedirectService, error) {
		called = true
		return nil, nil
	}

	if _, err := newRedirectService(); err != nil {
		t.Fatalf("overridden newRedirectService returned error: %v", err)
	}
	if !called {
		t.Error("overridden newRedirectService was not invoked")
	}
}
