package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// workerRouteRunGlobals snapshots and restores the package-level state that
// the worker-route runners read, so tests cannot leak state.
func workerRouteRunGlobals(t *testing.T) {
	t.Helper()
	oldPattern, oldScript, oldForce := workerRoutePattern, workerRouteScript, workerRouteForce
	oldAccountID, oldToken := AccountID, APIToken
	oldDry, oldJSON := DryRun, JSONOutput
	t.Cleanup(func() {
		workerRoutePattern, workerRouteScript, workerRouteForce = oldPattern, oldScript, oldForce
		AccountID, APIToken = oldAccountID, oldToken
		DryRun, JSONOutput = oldDry, oldJSON
	})
}

// workerRouteRunResetFlags clears the shared route flag variables so each
// subtest starts from a pristine command state.
func workerRouteRunResetFlags() {
	workerRoutePattern = ""
	workerRouteScript = ""
	workerRouteForce = false
}

// --- registration ---

func TestWorkerRouteCmd_RegisteredOnWorker(t *testing.T) {
	found := false
	for _, c := range workerCmd.Commands() {
		if c.Name() == "route" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected 'route' subcommand on worker command")
	}
}

func TestWorkerRouteCmd_Subcommands(t *testing.T) {
	want := []string{"list", "create", "update", "delete"}
	for _, name := range want {
		sub, _, err := workerRouteCmd.Find([]string{name})
		if err != nil {
			t.Fatalf("missing subcommand %s: %v", name, err)
		}
		if sub.Name() != name {
			t.Fatalf("expected subcommand %s, got %s", name, sub.Name())
		}
	}
}

func TestWorkerRouteCmd_Metadata(t *testing.T) {
	if workerRouteCmd.Short == "" {
		t.Error("expected non-empty Short")
	}
	if !strings.Contains(workerRouteCmd.Long, "Examples:") {
		t.Error("expected Long to contain examples")
	}
}

func TestWorkerRouteListCmd_Flags(t *testing.T) {
	if workerRouteListCmd.Flags().Lookup("json") != nil {
		t.Error("list should not define a local --json flag")
	}
}

func TestWorkerRouteCreateCmd_RequiredFlags(t *testing.T) {
	if workerRouteCreateCmd.Flags().Lookup("pattern") == nil {
		t.Error("create is missing --pattern flag")
	}
	if workerRouteCreateCmd.Flags().Lookup("script") == nil {
		t.Error("create is missing --script flag")
	}
}

func TestWorkerRouteUpdateCmd_RequiredFlags(t *testing.T) {
	if workerRouteUpdateCmd.Flags().Lookup("pattern") == nil {
		t.Error("update is missing --pattern flag")
	}
	if workerRouteUpdateCmd.Flags().Lookup("script") == nil {
		t.Error("update is missing --script flag")
	}
}

func TestWorkerRouteDeleteCmd_ForceFlag(t *testing.T) {
	f := workerRouteDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("delete is missing --force flag")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("expected bool --force, got %s", f.Value.Type())
	}
}

func TestWorkerRouteCmd_UseFields(t *testing.T) {
	cases := map[string]string{
		"list":   "list [zone-id]",
		"create": "create [zone-id]",
		"update": "update [zone-id] [route-id]",
		"delete": "delete [zone-id] [route-id]",
	}
	for name, use := range cases {
		var cmd = workerRouteCmd
		sub, _, err := cmd.Find([]string{name})
		if err != nil {
			t.Fatalf("missing subcommand %s: %v", name, err)
		}
		if sub.Use != use {
			t.Errorf("%s Use = %q, want %q", name, sub.Use, use)
		}
	}
}

// --- offline validation paths ---

func TestRunWorkerRouteList_RequiresZoneID(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()

	err := runWorkerRouteList(workerRouteListCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "zone ID is required") {
		t.Fatalf("expected zone ID error, got %v", err)
	}
}

func TestRunWorkerRouteList_MissingToken(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	APIToken = ""
	AccountID = ""

	err := runWorkerRouteList(workerRouteListCmd, []string{"zone123"})
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
}

func TestRunWorkerRouteCreate_RequiresZoneID(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	workerRoutePattern = "example.com/*"
	workerRouteScript = "my-worker"

	err := runWorkerRouteCreate(workerRouteCreateCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "zone ID is required") {
		t.Fatalf("expected zone ID error, got %v", err)
	}
}

func TestRunWorkerRouteCreate_RequiresPattern(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	workerRoutePattern = ""
	workerRouteScript = "my-worker"

	err := runWorkerRouteCreate(workerRouteCreateCmd, []string{"zone123"})
	if err == nil || !strings.Contains(err.Error(), "pattern is required") {
		t.Fatalf("expected pattern error, got %v", err)
	}
}

func TestRunWorkerRouteCreate_RequiresScript(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	workerRoutePattern = "example.com/*"
	workerRouteScript = ""

	err := runWorkerRouteCreate(workerRouteCreateCmd, []string{"zone123"})
	if err == nil || !strings.Contains(err.Error(), "script is required") {
		t.Fatalf("expected script error, got %v", err)
	}
}

func TestRunWorkerRouteUpdate_RequiresZoneID(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	workerRoutePattern = "example.com/*"
	workerRouteScript = "my-worker"

	err := runWorkerRouteUpdate(workerRouteUpdateCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "zone ID is required") {
		t.Fatalf("expected zone ID error, got %v", err)
	}
}

func TestRunWorkerRouteUpdate_RequiresRouteID(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	workerRoutePattern = "example.com/*"
	workerRouteScript = "my-worker"

	err := runWorkerRouteUpdate(workerRouteUpdateCmd, []string{"zone123"})
	if err == nil || !strings.Contains(err.Error(), "route ID is required") {
		t.Fatalf("expected route ID error, got %v", err)
	}
}

func TestRunWorkerRouteUpdate_RequiresPattern(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	workerRoutePattern = ""
	workerRouteScript = "my-worker"

	err := runWorkerRouteUpdate(workerRouteUpdateCmd, []string{"zone123", "route1"})
	if err == nil || !strings.Contains(err.Error(), "pattern is required") {
		t.Fatalf("expected pattern error, got %v", err)
	}
}

func TestRunWorkerRouteUpdate_RequiresScript(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	workerRoutePattern = "example.com/*"
	workerRouteScript = ""

	err := runWorkerRouteUpdate(workerRouteUpdateCmd, []string{"zone123", "route1"})
	if err == nil || !strings.Contains(err.Error(), "script is required") {
		t.Fatalf("expected script error, got %v", err)
	}
}

func TestRunWorkerRouteDelete_RequiresZoneID(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	workerRouteForce = true

	err := runWorkerRouteDelete(workerRouteDeleteCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "zone ID is required") {
		t.Fatalf("expected zone ID error, got %v", err)
	}
}

func TestRunWorkerRouteDelete_RequiresRouteID(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	workerRouteForce = true

	err := runWorkerRouteDelete(workerRouteDeleteCmd, []string{"zone123"})
	if err == nil || !strings.Contains(err.Error(), "route ID is required") {
		t.Fatalf("expected route ID error, got %v", err)
	}
}

func TestRunWorkerRouteDelete_MissingToken(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	workerRouteForce = true
	APIToken = ""
	AccountID = ""

	err := runWorkerRouteDelete(workerRouteDeleteCmd, []string{"zone123", "route1"})
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
}

// --- dry-run (offline) success paths ---

func TestRunWorkerRouteCreate_DryRun(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	DryRun = true
	JSONOutput = false
	workerRoutePattern = "example.com/api/*"
	workerRouteScript = "api-gateway"

	if err := runWorkerRouteCreate(workerRouteCreateCmd, []string{"zone123"}); err != nil {
		t.Fatalf("dry-run create returned error: %v", err)
	}
}

func TestRunWorkerRouteCreate_DryRunJSON(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	DryRun = true
	JSONOutput = true
	workerRoutePattern = "example.com/api/*"
	workerRouteScript = "api-gateway"

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWorkerRouteCreate(workerRouteCreateCmd, []string{"zone123"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("dry-run create (json) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	for _, want := range []string{"DRY RUN", "example.com/api/*", "api-gateway"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q: %s", want, out)
		}
	}
}

func TestRunWorkerRouteUpdate_DryRun(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	DryRun = true
	JSONOutput = false
	workerRoutePattern = "example.com/v2/*"
	workerRouteScript = "api-gateway"

	if err := runWorkerRouteUpdate(workerRouteUpdateCmd, []string{"zone123", "route1"}); err != nil {
		t.Fatalf("dry-run update returned error: %v", err)
	}
}

func TestRunWorkerRouteDelete_DryRun(t *testing.T) {
	workerRouteRunGlobals(t)
	workerRouteRunResetFlags()
	DryRun = true
	JSONOutput = false
	workerRouteForce = false

	// Dry-run skips the confirmation prompt, so no stdin is consumed.
	if err := runWorkerRouteDelete(workerRouteDeleteCmd, []string{"zone123", "route1"}); err != nil {
		t.Fatalf("dry-run delete returned error: %v", err)
	}
}
