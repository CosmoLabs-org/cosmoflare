package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"testing"
)

// workerSecretRunGlobals snapshots and restores the package-level flag
// variables the worker secret runners read, so tests cannot leak state.
func workerSecretRunGlobals(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	oldValue, oldForce := workerSecretValue, workerSecretForce
	t.Cleanup(func() { workerSecretValue, workerSecretForce = oldValue, oldForce })
}

// workerSecretRunResetFlags returns the secret flag variables to their
// zero values so each subtest starts from a pristine command state.
func workerSecretRunResetFlags() {
	workerSecretValue = ""
	workerSecretForce = false
}

// TestRunWorkerSecretPut_RequiresArgs verifies the argument guards fire
// before any service construction or stdin access.
func TestRunWorkerSecretPut_RequiresArgs(t *testing.T) {
	workerSecretRunGlobals(t)
	workerSecretRunResetFlags()

	err := runWorkerSecretPut(workerSecretPutCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker name error, got %v", err)
	}

	err = runWorkerSecretPut(workerSecretPutCmd, []string{"my-worker"})
	if err == nil || !strings.Contains(err.Error(), "secret name is required") {
		t.Fatalf("expected secret name error, got %v", err)
	}
}

// TestRunWorkerSecretPut_RequiresValue verifies that a blank stdin value
// is rejected offline (command input stubbed with an empty reader).
func TestRunWorkerSecretPut_RequiresValue(t *testing.T) {
	workerSecretRunGlobals(t)
	workerSecretRunResetFlags()
	workerSecretPutCmd.SetIn(strings.NewReader("\n"))
	workerSecretPutCmd.SetErr(io.Discard)
	t.Cleanup(func() {
		workerSecretPutCmd.SetIn(nil)
		workerSecretPutCmd.SetErr(nil)
	})

	err := runWorkerSecretPut(workerSecretPutCmd, []string{"my-worker", "API_TOKEN"})
	if err == nil || !strings.Contains(err.Error(), "secret value is required") {
		t.Fatalf("expected secret value error, got %v", err)
	}
}

// TestRunWorkerSecretDelete_RequiresArgs verifies the argument guards.
func TestRunWorkerSecretDelete_RequiresArgs(t *testing.T) {
	workerSecretRunGlobals(t)
	workerSecretRunResetFlags()

	err := runWorkerSecretDelete(workerSecretDeleteCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker name error, got %v", err)
	}

	err = runWorkerSecretDelete(workerSecretDeleteCmd, []string{"my-worker"})
	if err == nil || !strings.Contains(err.Error(), "secret name is required") {
		t.Fatalf("expected secret name error, got %v", err)
	}
}

// TestRunWorkerSecretList_RequiresWorker verifies the argument guard.
func TestRunWorkerSecretList_RequiresWorker(t *testing.T) {
	workerSecretRunGlobals(t)

	err := runWorkerSecretList(workerSecretListCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker name error, got %v", err)
	}
}

// TestRunWorkerSecretBulk_RequiresArgs verifies the argument guards.
func TestRunWorkerSecretBulk_RequiresArgs(t *testing.T) {
	workerSecretRunGlobals(t)

	err := runWorkerSecretBulk(workerSecretBulkCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected worker name error, got %v", err)
	}

	err = runWorkerSecretBulk(workerSecretBulkCmd, []string{"my-worker"})
	if err == nil || !strings.Contains(err.Error(), "secrets file is required") {
		t.Fatalf("expected secrets file error, got %v", err)
	}
}

// TestRunWorkerSecretBulk_MissingFile verifies the read error surfaces
// before any credentials or service construction.
func TestRunWorkerSecretBulk_MissingFile(t *testing.T) {
	workerSecretRunGlobals(t)
	APIToken = ""

	missing := filepath.Join(t.TempDir(), "does-not-exist.json")
	err := runWorkerSecretBulk(workerSecretBulkCmd, []string{"my-worker", missing})
	if err == nil || !strings.Contains(err.Error(), "failed to read secrets file") {
		t.Fatalf("expected read error, got %v", err)
	}
}

// TestRunWorkerSecretBulk_InvalidJSON verifies malformed JSON is rejected
// offline with the parse error.
func TestRunWorkerSecretBulk_InvalidJSON(t *testing.T) {
	workerSecretRunGlobals(t)
	APIToken = ""

	file := filepath.Join(t.TempDir(), "secrets.json")
	if err := os.WriteFile(file, []byte(`{"API_TOKEN": missing-quote}`), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	err := runWorkerSecretBulk(workerSecretBulkCmd, []string{"my-worker", file})
	if err == nil || !strings.Contains(err.Error(), "failed to parse secrets file") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

// TestRunWorkerSecretBulk_EmptyObject verifies an empty JSON object is
// rejected rather than performing a no-op API round trip.
func TestRunWorkerSecretBulk_EmptyObject(t *testing.T) {
	workerSecretRunGlobals(t)
	APIToken = ""

	file := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(file, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	err := runWorkerSecretBulk(workerSecretBulkCmd, []string{"my-worker", file})
	if err == nil || !strings.Contains(err.Error(), "secrets file contains no entries") {
		t.Fatalf("expected empty-file error, got %v", err)
	}
}

// TestParseSecretsBulkJSON exercises the pure JSON helper directly:
// valid objects round-trip and every malformed shape is rejected.
func TestParseSecretsBulkJSON(t *testing.T) {
	got, err := parseSecretsBulkJSON([]byte(`{"API_TOKEN":"tok","DB_PASSWORD":"pw"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got["API_TOKEN"] != "tok" || got["DB_PASSWORD"] != "pw" {
		t.Errorf("unexpected parse result: %v", got)
	}

	// Empty values are allowed at parse time; the service validates them.
	got, err = parseSecretsBulkJSON([]byte(`{"EMPTY":""}`))
	if err != nil {
		t.Fatalf("unexpected error for empty value: %v", err)
	}
	if got["EMPTY"] != "" {
		t.Errorf("expected empty value preserved, got %q", got["EMPTY"])
	}

	// An empty object parses cleanly to an empty map; the runner rejects
	// it later with its own message.
	got, err = parseSecretsBulkJSON([]byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error for empty object: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}

	cases := []struct {
		name string
		data string
		want string
	}{
		{"malformed", `{"KEY": nope}`, "invalid JSON"},
		{"array", `["API_TOKEN"]`, "invalid JSON"},
		{"string", `"API_TOKEN"`, "invalid JSON"},
		{"number", `42`, "invalid JSON"},
		{"blank name", `{"  ":"value"}`, "secret names must not be empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseSecretsBulkJSON([]byte(tc.data))
			if err == nil {
				t.Fatalf("expected error for %s", tc.data)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

// TestRegisterWorkerSecretCmds verifies registration wires the group and
// all four subcommands onto a parent without touching cmd/worker.go.
func TestRegisterWorkerSecretCmds(t *testing.T) {
	parent := &cobra.Command{Use: "worker-stub", Run: func(*cobra.Command, []string) {}}
	registerWorkerSecretCmds(parent)

	subs := parent.Commands()
	if len(subs) != 1 || subs[0].Name() != "secret" {
		t.Fatalf("expected exactly one 'secret' group on parent, got %d command(s)", len(subs))
	}

	names := map[string]bool{}
	for _, sub := range subs[0].Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"put", "delete", "list", "bulk"} {
		if !names[want] {
			t.Errorf("expected 'secret %s' registered, got %v", want, names)
		}
	}
}
