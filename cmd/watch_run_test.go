package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// watchRunGlobals snapshots and restores the package-level flag variables and
// credentials runWatch and its helpers read, so tests cannot leak state.
func watchRunGlobals(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	oldPrefix, oldExclude := watchPrefix, watchExclude
	oldInterval, oldDelete := watchInterval, watchDelete
	oldVerbose := Verbose
	t.Cleanup(func() {
		watchPrefix, watchExclude = oldPrefix, oldExclude
		watchInterval, watchDelete = oldInterval, oldDelete
		Verbose = oldVerbose
	})
}

// watchRunResetFlags restores watch flag defaults so each subtest starts from
// a pristine command state.
func watchRunResetFlags() {
	watchPrefix = ""
	watchExclude = nil
	watchInterval = time.Second
	watchDelete = false
}

// TestRunWatch_RequiresBucket verifies that invoking watch without a bucket
// argument fails fast with the usage error, before any watcher is built.
func TestRunWatch_RequiresBucket(t *testing.T) {
	watchRunGlobals(t)
	watchRunResetFlags()

	err := runWatch(watchCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "bucket name is required") {
		t.Fatalf("expected bucket-required error, got %v", err)
	}
}

// TestRunWatch_NonexistentDirectory verifies that pointing watch at a missing
// directory fails during watcher construction.
func TestRunWatch_NonexistentDirectory(t *testing.T) {
	watchRunGlobals(t)
	watchRunResetFlags()
	dir := filepath.Join(t.TempDir(), "does-not-exist")

	err := runWatch(watchCmd, []string{"my-bucket", dir})
	if err == nil || !strings.Contains(err.Error(), "failed to create watcher") {
		t.Fatalf("expected watcher error, got %v", err)
	}
}

// TestRunWatch_FileInsteadOfDirectory verifies that a regular file passed as
// the watch directory is rejected with the not-a-directory error.
func TestRunWatch_FileInsteadOfDirectory(t *testing.T) {
	watchRunGlobals(t)
	watchRunResetFlags()
	file := filepath.Join(t.TempDir(), "plain.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := runWatch(watchCmd, []string{"my-bucket", file})
	if err == nil || !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("expected not-a-directory error, got %v", err)
	}
}

// TestRunWatch_MissingCredentials verifies that without dry-run and with no
// configured credentials the run aborts at R2 client creation, before the
// event loop starts.
func TestRunWatch_MissingCredentials(t *testing.T) {
	watchRunGlobals(t)
	watchRunResetFlags()
	DryRun = false
	AccountID = ""
	APIToken = ""
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")

	err := runWatch(watchCmd, []string{"my-bucket", t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "failed to create R2 client") {
		t.Fatalf("expected R2 client error, got %v", err)
	}
}

// TestNewWatchR2Client_DryRunSkipsClient verifies that dry-run mode returns a
// nil client without touching credentials or the network.
func TestNewWatchR2Client_DryRunSkipsClient(t *testing.T) {
	watchRunGlobals(t)
	DryRun = true
	AccountID = ""
	APIToken = ""

	client, err := newWatchR2Client(t.TempDir())
	if err != nil {
		t.Fatalf("dry-run should not create a client or fail: %v", err)
	}
	if client != nil {
		t.Fatalf("dry-run should return a nil client, got %v", client)
	}
}

// TestNewWatchR2Client_MissingAccountID verifies the non-dry-run credential
// validation error surfaces from client construction.
func TestNewWatchR2Client_MissingAccountID(t *testing.T) {
	watchRunGlobals(t)
	DryRun = false
	AccountID = ""
	APIToken = ""
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")

	_, err := newWatchR2Client(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "failed to create R2 client") {
		t.Fatalf("expected R2 client error, got %v", err)
	}
}

// TestWatchPrintHeader_JSONModeIsSilent verifies the startup banner is
// suppressed in --json mode (no panic, nothing to assert beyond the call).
func TestWatchPrintHeader_JSONModeIsSilent(t *testing.T) {
	watchRunGlobals(t)
	watchRunResetFlags()
	JSONOutput = true

	watchPrintHeader("/tmp/some/dir", "my-bucket", 0)
}
