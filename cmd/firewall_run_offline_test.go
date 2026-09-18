package cmd

import (
	"strings"
	"testing"
)

// buckFwGlobals snapshots the package globals the firewall runners read.
func buckFwGlobals(t *testing.T) {
	t.Helper()
	oldAcct, oldTok := AccountID, APIToken
	oldDry, oldJSON := DryRun, JSONOutput
	oldForce := fwForce
	t.Cleanup(func() {
		AccountID, APIToken = oldAcct, oldTok
		DryRun, JSONOutput = oldDry, oldJSON
		fwForce = oldForce
	})
}

// TestRunFirewallList_ServiceErrorNoCreds verifies listing rules fails at
// service construction when credentials are absent.
func TestRunFirewallList_ServiceErrorNoCreds(t *testing.T) {
	buckFwGlobals(t)
	AccountID, APIToken = "", ""

	err := runFirewallList(firewallListCmd, []string{"zone-123"})
	if err == nil || !strings.Contains(err.Error(), "failed to create firewall service") {
		t.Fatalf("expected firewall service error, got %v", err)
	}
}

// TestRunFirewallGet_ServiceErrorNoCreds verifies getting one rule fails at
// service construction when credentials are absent.
func TestRunFirewallGet_ServiceErrorNoCreds(t *testing.T) {
	buckFwGlobals(t)
	AccountID, APIToken = "", ""

	err := runFirewallGet(firewallGetCmd, []string{"zone-123", "rule-456"})
	if err == nil || !strings.Contains(err.Error(), "failed to create firewall service") {
		t.Fatalf("expected firewall service error, got %v", err)
	}
}

// TestRunFirewallUpdate_ServiceErrorNoCreds verifies the update runner
// builds the service before the dry-run short circuit, so missing
// credentials surface as a service error even with --dry-run.
func TestRunFirewallUpdate_ServiceErrorNoCreds(t *testing.T) {
	buckFwGlobals(t)
	AccountID, APIToken = "", ""
	DryRun = true

	err := runFirewallUpdate(firewallUpdateCmd, []string{"zone-123", "rule-456"})
	if err == nil || !strings.Contains(err.Error(), "failed to create firewall service") {
		t.Fatalf("expected firewall service error, got %v", err)
	}
}

// TestRunFirewallCreate_ServiceErrorNoCreds verifies the create runner
// fails at service construction when credentials are absent.
func TestRunFirewallCreate_ServiceErrorNoCreds(t *testing.T) {
	buckFwGlobals(t)
	AccountID, APIToken = "", ""
	DryRun = false

	err := runFirewallCreate(firewallCreateCmd, []string{"zone-123"})
	if err == nil || !strings.Contains(err.Error(), "failed to create firewall service") {
		t.Fatalf("expected firewall service error, got %v", err)
	}
}

// TestRunFirewallDelete_CancelPrompt verifies a negative confirmation
// cancels rule deletion and returns nil without building the service.
func TestRunFirewallDelete_CancelPrompt(t *testing.T) {
	buckFwGlobals(t)
	AccountID, APIToken = "", "" // service creation would fail if reached
	DryRun, fwForce = false, false
	buckWithStdin(t, "n\n")

	if err := runFirewallDelete(firewallDeleteCmd, []string{"zone-123", "rule-456"}); err != nil {
		t.Fatalf("cancelled deletion should return nil, got %v", err)
	}
}

// TestRunFirewallDelete_ConfirmPromptReachesService verifies an affirmative
// answer proceeds past the prompt to the (failing) service construction.
func TestRunFirewallDelete_ConfirmPromptReachesService(t *testing.T) {
	buckFwGlobals(t)
	AccountID, APIToken = "", ""
	DryRun, fwForce = false, false
	buckWithStdin(t, "y\n")

	err := runFirewallDelete(firewallDeleteCmd, []string{"zone-123", "rule-456"})
	if err == nil || !strings.Contains(err.Error(), "failed to create firewall service") {
		t.Fatalf("expected firewall service error after confirming, got %v", err)
	}
}
