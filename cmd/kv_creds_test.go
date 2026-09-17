package cmd

import (
	"strings"
	"testing"
)

// kvCredsGlobals snapshots the credential globals, the output-mode flags, and
// every KV flag variable so the missing-credential tables cannot leak state.
func kvCredsGlobals(t *testing.T) {
	t.Helper()
	oldAcct, oldToken := AccountID, APIToken
	oldDry, oldJSON := DryRun, JSONOutput
	oldValue, oldFile, oldTTL := kvValue, kvFile, kvTTL
	oldPrefix, oldLimit := kvPrefix, kvLimit
	t.Cleanup(func() {
		AccountID, APIToken = oldAcct, oldToken
		DryRun, JSONOutput = oldDry, oldJSON
		kvValue, kvFile, kvTTL = oldValue, oldFile, oldTTL
		kvPrefix, kvLimit = oldPrefix, oldLimit
	})
}

// kvCredsReset installs the pristine flag state each subtest needs: a
// non-dry-run invocation with a usable --value so runKVPut passes its value
// guard and reaches the service-construction check.
func kvCredsReset() {
	kvValue = "payload"
	kvFile = ""
	kvTTL = 0
	kvPrefix = ""
	kvLimit = 1000
	DryRun = false
	JSONOutput = false
}

// kvCredsRunners returns every KV runner wired with valid positional
// arguments and --force on the namespace delete command, so the shared
// credential guards can be exercised uniformly.
func kvCredsRunners() map[string]func() error {
	return map[string]func() error{
		"namespace create": func() error { return runKVNamespaceCreate(kvNamespaceCreateCmd, []string{"my-cache"}) },
		"namespace list":   func() error { return runKVNamespaceList(kvNamespaceListCmd, nil) },
		"namespace delete": func() error { return runKVNamespaceDelete(kvNamespaceDeleteCmd, []string{"ns-abc123"}) },
		"put":              func() error { return runKVPut(kvPutCmd, []string{"ns-abc123", "my-key"}) },
		"get":              func() error { return runKVGet(kvGetCmd, []string{"ns-abc123", "my-key"}) },
		"delete":           func() error { return runKVDelete(kvDeleteCmd, []string{"ns-abc123", "my-key"}) },
		"list":             func() error { return runKVList(kvListCmd, []string{"ns-abc123"}) },
	}
}

// kvCredsNames keeps a stable subtest order for the credential tables.
var kvCredsNames = []string{
	"namespace create", "namespace list", "namespace delete",
	"put", "get", "delete", "list",
}

// TestRunKV_MissingAccountID verifies every KV command fails fast with the
// wrapped service-construction error when no account ID is configured.
func TestRunKV_MissingAccountID(t *testing.T) {
	fns := kvCredsRunners()
	for _, name := range kvCredsNames {
		t.Run(name, func(t *testing.T) {
			kvCredsGlobals(t)
			kvCredsReset()
			AccountID = ""
			APIToken = "token-abcdef1234567890"

			err := fns[name]()
			if err == nil || !strings.Contains(err.Error(), "failed to create KV service") {
				t.Fatalf("expected KV service creation error, got %v", err)
			}
			if !strings.Contains(err.Error(), "account ID is required") {
				t.Fatalf("expected wrapped account ID error, got %v", err)
			}
		})
	}
}

// TestRunKV_MissingAPIToken verifies every KV command fails fast with the
// wrapped token error when an account ID is present but the token is not.
func TestRunKV_MissingAPIToken(t *testing.T) {
	fns := kvCredsRunners()
	for _, name := range kvCredsNames {
		t.Run(name, func(t *testing.T) {
			kvCredsGlobals(t)
			kvCredsReset()
			AccountID = "test-account"
			APIToken = ""

			err := fns[name]()
			if err == nil || !strings.Contains(err.Error(), "API token is required") {
				t.Fatalf("expected API token error, got %v", err)
			}
		})
	}
}

// TestRunKVNamespaceDelete_DeclinedConfirmationCancels verifies the y/N
// prompt guards the destructive call: answering "n" cancels the deletion and
// returns success without touching the (unavailable) API.
func TestRunKVNamespaceDelete_DeclinedConfirmationCancels(t *testing.T) {
	kvCredsGlobals(t)
	kvCredsReset()
	AccountID = "test-account"
	APIToken = "test-token-abcdef1234567890"
	configRunWithStdin(t, "n\n")

	if err := runKVNamespaceDelete(kvNamespaceDeleteCmd, []string{"ns-abc123"}); err != nil {
		t.Fatalf("declined confirmation should cancel cleanly, got %v", err)
	}
}

// TestRunKVNamespaceDelete_EOFConfirmationCancels verifies a closed stdin
// (scripted pipeline) also cancels rather than deleting by default.
func TestRunKVNamespaceDelete_EOFConfirmationCancels(t *testing.T) {
	kvCredsGlobals(t)
	kvCredsReset()
	AccountID = "test-account"
	APIToken = "test-token-abcdef1234567890"
	configRunWithStdin(t, "")

	if err := runKVNamespaceDelete(kvNamespaceDeleteCmd, []string{"ns-abc123"}); err != nil {
		t.Fatalf("EOF confirmation should cancel cleanly, got %v", err)
	}
}
