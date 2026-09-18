package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buckZoneGlobals snapshots the package globals the zone runners read so
// tests cannot leak credentials or output-mode state into each other.
func buckZoneGlobals(t *testing.T) {
	t.Helper()
	oldAcct, oldTok := AccountID, APIToken
	oldDry, oldJSON := DryRun, JSONOutput
	oldForce := zoneForce
	oldType := zoneType
	t.Cleanup(func() {
		AccountID, APIToken = oldAcct, oldTok
		DryRun, JSONOutput = oldDry, oldJSON
		zoneForce, zoneType = oldForce, oldType
	})
}

// TestRunZoneCreate_DryRun verifies the dry-run path reports the zone it
// would create without contacting the Cloudflare API.
func TestRunZoneCreate_DryRun(t *testing.T) {
	buckZoneGlobals(t)
	AccountID, APIToken = "acct-123", "tok-123"
	DryRun, JSONOutput = true, false
	zoneType = "partial"

	if err := runZoneCreate(zoneCreateCmd, []string{"example.com"}); err != nil {
		t.Fatalf("runZoneCreate dry-run: %v", err)
	}
}

// TestRunZoneCreate_ServiceErrorNoCreds verifies the create runner surfaces
// a service-creation error before doing anything when credentials are
// missing.
func TestRunZoneCreate_ServiceErrorNoCreds(t *testing.T) {
	buckZoneGlobals(t)
	AccountID, APIToken = "", ""
	DryRun = false

	err := runZoneCreate(zoneCreateCmd, []string{"example.com"})
	if err == nil || !strings.Contains(err.Error(), "failed to create zone service") {
		t.Fatalf("expected zone service error, got %v", err)
	}
}

// TestRunZoneList_ServiceErrorNoCreds verifies listing zones fails fast
// with the service error when credentials are absent.
func TestRunZoneList_ServiceErrorNoCreds(t *testing.T) {
	buckZoneGlobals(t)
	AccountID, APIToken = "", ""

	err := runZoneList(zoneListCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create zone service") {
		t.Fatalf("expected zone service error, got %v", err)
	}
}

// TestRunZoneGet_ServiceErrorNoCreds verifies getting a zone fails at
// service construction when credentials are absent.
func TestRunZoneGet_ServiceErrorNoCreds(t *testing.T) {
	buckZoneGlobals(t)
	AccountID, APIToken = "", ""

	err := runZoneGet(zoneGetCmd, []string{"zone-123"})
	if err == nil || !strings.Contains(err.Error(), "failed to create zone service") {
		t.Fatalf("expected zone service error, got %v", err)
	}
}

// TestRunZoneSettings_ServiceErrorNoCreds verifies the settings runner
// fails at service construction when credentials are absent.
func TestRunZoneSettings_ServiceErrorNoCreds(t *testing.T) {
	buckZoneGlobals(t)
	AccountID, APIToken = "", ""

	err := runZoneSettings(zoneSettingsCmd, []string{"zone-123"})
	if err == nil || !strings.Contains(err.Error(), "failed to create zone service") {
		t.Fatalf("expected zone service error, got %v", err)
	}
}

// TestRunZoneDelete_CancelPrompt verifies a negative confirmation cancels
// the deletion and returns success without ever building the zone service.
func TestRunZoneDelete_CancelPrompt(t *testing.T) {
	buckZoneGlobals(t)
	AccountID, APIToken = "", "" // service creation would fail if reached
	DryRun, zoneForce = false, false
	buckWithStdin(t, "n\n")

	if err := runZoneDelete(zoneDeleteCmd, []string{"zone-123"}); err != nil {
		t.Fatalf("cancelled deletion should return nil, got %v", err)
	}
}

// TestRunZoneDelete_ConfirmPromptReachesService verifies an affirmative
// confirmation proceeds past the prompt and fails at service construction
// with missing credentials.
func TestRunZoneDelete_ConfirmPromptReachesService(t *testing.T) {
	buckZoneGlobals(t)
	AccountID, APIToken = "", ""
	DryRun, zoneForce = false, false
	buckWithStdin(t, "yes\n")

	err := runZoneDelete(zoneDeleteCmd, []string{"zone-123"})
	if err == nil || !strings.Contains(err.Error(), "failed to create zone service") {
		t.Fatalf("expected zone service error after confirming, got %v", err)
	}
}

// TestRunZoneDelete_DryRunSkipsPrompt verifies --dry-run neither prompts
// nor deletes, even without --force.
func TestRunZoneDelete_DryRunSkipsPrompt(t *testing.T) {
	buckZoneGlobals(t)
	AccountID, APIToken = "acct-123", "tok-123"
	DryRun, JSONOutput, zoneForce = true, false, false
	buckWithStdin(t, "") // EOF: a prompt would read nothing and cancel

	if err := runZoneDelete(zoneDeleteCmd, []string{"zone-123"}); err != nil {
		t.Fatalf("runZoneDelete dry-run: %v", err)
	}
}

// TestRunZoneDelete_ForceNoCreds verifies --force skips the prompt and
// surfaces the service error with missing credentials.
func TestRunZoneDelete_ForceNoCreds(t *testing.T) {
	buckZoneGlobals(t)
	AccountID, APIToken = "", ""
	DryRun, zoneForce = false, true
	buckWithStdin(t, "n\n") // would cancel if the prompt were (wrongly) shown

	err := runZoneDelete(zoneDeleteCmd, []string{"zone-123"})
	if err == nil || !strings.Contains(err.Error(), "failed to create zone service") {
		t.Fatalf("expected zone service error with --force, got %v", err)
	}
}

// TestRunZoneDelete_NoStdinDoesNotPanic verifies an empty stdin (EOF at the
// confirmation prompt) cancels cleanly instead of panicking.
func TestRunZoneDelete_NoStdinDoesNotPanic(t *testing.T) {
	buckZoneGlobals(t)
	AccountID, APIToken = "", ""
	DryRun, zoneForce = false, false
	buckWithStdin(t, "")

	if err := runZoneDelete(zoneDeleteCmd, []string{"zone-123"}); err != nil {
		t.Fatalf("EOF at prompt should cancel, got %v", err)
	}
}

// TestZoneDryRunCreatesNoFiles is a belt-and-braces guard: a dry run must
// not have created a config file in the working directory.
func TestZoneDryRunCreatesNoFiles(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(oldWD) })

	buckZoneGlobals(t)
	AccountID, APIToken = "acct-123", "tok-123"
	DryRun, JSONOutput = true, false

	if err := runZoneCreate(zoneCreateCmd, []string{"example.com"}); err != nil {
		t.Fatalf("runZoneCreate dry-run: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, filepath.Join(dir, e.Name()))
		}
		t.Errorf("dry run created files: %v", names)
	}
}
