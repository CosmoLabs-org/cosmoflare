package cmd

import (
	"os"
	"strings"
	"testing"
)

// queueRunSnapshot snapshots the queue handler globals and restores them on
// cleanup.
func queueRunSnapshot(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	savedForce, savedName := queueForce, queueNewName
	t.Cleanup(func() { queueForce, queueNewName = savedForce, savedName })
}

// TestRunQueueServiceError verifies list, get, and consumers fail fast with
// the service-creation error when credentials are absent.
func TestRunQueueServiceError(t *testing.T) {
	queueRunSnapshot(t)
	JSONOutput = false
	DryRun = false
	AccountID = ""
	APIToken = ""

	cases := []struct {
		name string
		err  error
	}{
		{"list", runQueueList(nil, nil)},
		{"get", runQueueGet(nil, []string{"my-queue"})},
		{"consumers", runQueueConsumers(nil, []string{"my-queue"})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil || !strings.Contains(tc.err.Error(), "failed to create Queue service") {
				t.Errorf("expected service error, got %v", tc.err)
			}
		})
	}
}

// TestRunQueueDelete_ConfirmDeclined verifies a declined confirmation prompt
// cancels the deletion before any service is created.
func TestRunQueueDelete_ConfirmDeclined(t *testing.T) {
	queueRunSnapshot(t)
	JSONOutput = false
	DryRun = false
	queueForce = false
	AccountID = ""
	APIToken = ""

	savedStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := w.WriteString("no\n"); err != nil {
		t.Fatalf("seed stdin: %v", err)
	}
	w.Close()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = savedStdin })

	out := capturePrint(t, func() {
		if err := runQueueDelete(nil, []string{"my-queue"}); err != nil {
			t.Errorf("declined confirmation should cancel cleanly, got %v", err)
		}
	})

	if !strings.Contains(out, "Queue deletion cancelled") {
		t.Errorf("expected cancellation notice, got %q", out)
	}
}

// TestRunQueueUpdate_RequiresNewName verifies update refuses to run without
// the --name flag value.
func TestRunQueueUpdate_RequiresNewName(t *testing.T) {
	queueRunSnapshot(t)
	JSONOutput = false
	queueNewName = ""

	err := runQueueUpdate(nil, []string{"old-queue"})
	if err == nil || !strings.Contains(err.Error(), "--name flag is required") {
		t.Errorf("expected --name error, got %v", err)
	}
}
