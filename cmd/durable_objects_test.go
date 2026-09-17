package cmd

import (
	"strings"
	"testing"
)

// durableObjectsRunGlobals snapshots the credentials and output-mode globals
// the durable objects runners read, zeroing credentials so no test can leak
// state or reach the network.
func durableObjectsRunGlobals(t *testing.T) {
	t.Helper()
	oldAccount, oldToken := AccountID, APIToken
	oldDry, oldJSON := DryRun, JSONOutput
	AccountID, APIToken = "", ""
	t.Cleanup(func() {
		AccountID, APIToken = oldAccount, oldToken
		DryRun, JSONOutput = oldDry, oldJSON
	})
}

// TestDOCmd_Registration verifies the "do" command group is wired onto
// rootCmd with all three subcommands and RunE handlers attached.
func TestDOCmd_Registration(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub == doCmd {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("doCmd not registered on rootCmd")
	}

	for _, sub := range doCmd.Commands() {
		switch sub {
		case doNamespacesCmd, doObjectsCmd, doInspectCmd:
		default:
			t.Errorf("unexpected subcommand %q under do", sub.Name())
		}
	}
	if doNamespacesCmd.RunE == nil || doObjectsCmd.RunE == nil || doInspectCmd.RunE == nil {
		t.Error("every do subcommand must have a RunE handler")
	}
}

// TestDOObjectsCmd_Flags verifies the pagination flags exist on the objects
// subcommand with their documented defaults.
func TestDOObjectsCmd_Flags(t *testing.T) {
	limit := doObjectsCmd.Flags().Lookup("limit")
	if limit == nil {
		t.Fatal("--limit flag missing on do objects")
	}
	if limit.DefValue != "0" {
		t.Errorf("--limit default = %q, want 0", limit.DefValue)
	}
	cursor := doObjectsCmd.Flags().Lookup("cursor")
	if cursor == nil {
		t.Fatal("--cursor flag missing on do objects")
	}
	if cursor.DefValue != "" {
		t.Errorf("--cursor default = %q, want empty", cursor.DefValue)
	}
}

// TestGetDurableObjectsService validates credential handling in the service
// factory: both fields are required and valid values yield a service.
func TestGetDurableObjectsService(t *testing.T) {
	oldAccount, oldToken := AccountID, APIToken
	t.Cleanup(func() { AccountID, APIToken = oldAccount, oldToken })

	t.Run("missing account ID", func(t *testing.T) {
		AccountID, APIToken = "", "token"
		svc, err := getDurableObjectsService()
		if err == nil || !strings.Contains(err.Error(), "account ID is required") {
			t.Fatalf("expected account ID error, got %v", err)
		}
		if svc != nil {
			t.Error("expected nil service on error")
		}
	})

	t.Run("missing API token", func(t *testing.T) {
		AccountID, APIToken = "account", ""
		svc, err := getDurableObjectsService()
		if err == nil || !strings.Contains(err.Error(), "API token is required") {
			t.Fatalf("expected token error, got %v", err)
		}
		if svc != nil {
			t.Error("expected nil service on error")
		}
	})

	t.Run("valid credentials", func(t *testing.T) {
		AccountID, APIToken = "account", "token"
		svc, err := getDurableObjectsService()
		if err != nil {
			t.Fatalf("expected service, got error: %v", err)
		}
		if svc == nil {
			t.Fatal("expected non-nil service")
		}
	})
}

// TestRunDoNamespaces_OfflineNoCreds verifies the namespaces runner fails at
// service construction when credentials are absent — before any API call.
func TestRunDoNamespaces_OfflineNoCreds(t *testing.T) {
	durableObjectsRunGlobals(t)

	err := runDoNamespaces(doNamespacesCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create Durable Objects service") {
		t.Fatalf("expected service construction error, got %v", err)
	}
}

// TestRunDoObjects_OfflineNoCreds verifies the objects runner fails at
// service construction when credentials are absent.
func TestRunDoObjects_OfflineNoCreds(t *testing.T) {
	durableObjectsRunGlobals(t)

	err := runDoObjects(doObjectsCmd, []string{"ns-123"})
	if err == nil || !strings.Contains(err.Error(), "failed to create Durable Objects service") {
		t.Fatalf("expected service construction error, got %v", err)
	}
}

// TestRunDoInspect_OfflineNoCreds verifies the inspect runner fails at
// service construction when credentials are absent.
func TestRunDoInspect_OfflineNoCreds(t *testing.T) {
	durableObjectsRunGlobals(t)

	err := runDoInspect(doInspectCmd, []string{"ns-123", "obj-456"})
	if err == nil || !strings.Contains(err.Error(), "failed to create Durable Objects service") {
		t.Fatalf("expected service construction error, got %v", err)
	}
}
