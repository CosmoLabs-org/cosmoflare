package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// queueSendTestEnv sets the globals the queue send commands need and
// restores them when the test ends.
func queueSendTestEnv(t *testing.T) {
	t.Helper()
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origBody := queueSendBody
	origFile := queueSendFile
	origContentType := queueSendContentType
	origDelay := queueSendDelaySeconds
	t.Cleanup(func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		queueSendBody = origBody
		queueSendFile = origFile
		queueSendContentType = origContentType
		queueSendDelaySeconds = origDelay
	})

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
}

// queueSendWriteFile writes content to a temp file and returns its path.
func queueSendWriteFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "messages.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

// queueSendCaptureStdout runs fn with os.Stdout redirected to a pipe
// and returns everything printed.
func queueSendCaptureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// --- Command registration ---

func TestQueueSendCmd_RegisteredOnQueue(t *testing.T) {
	for _, name := range []string{"send", "send-batch"} {
		found := false
		for _, sub := range queueCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("queue subcommand %q not registered", name)
		}
	}
}

func TestQueueSendCmd_RunE(t *testing.T) {
	if queueSendCmd.RunE == nil {
		t.Error("queueSendCmd.RunE is nil")
	}
	if queueSendBatchCmd.RunE == nil {
		t.Error("queueSendBatchCmd.RunE is nil")
	}
}

func TestQueueSendCmd_Args(t *testing.T) {
	if queueSendCmd.Args == nil {
		t.Error("queueSendCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
	if queueSendBatchCmd.Args == nil {
		t.Error("queueSendBatchCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

// --- Flag registration ---

func TestQueueSend_Flags(t *testing.T) {
	for _, name := range []string{"body", "file", "content-type", "delay-seconds"} {
		if queueSendCmd.Flags().Lookup(name) == nil {
			t.Errorf("--%s flag not registered on queueSendCmd", name)
		}
	}
}

func TestQueueSendBatch_FileFlagRequired(t *testing.T) {
	f := queueSendBatchCmd.Flags().Lookup("file")
	if f == nil {
		t.Fatal("--file flag not registered on queueSendBatchCmd")
	}
	if _, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; !ok {
		t.Error("--file flag should be marked required on queueSendBatchCmd")
	}
}

// --- Validation ---

func TestQueueSend_BodyAndFileConflict(t *testing.T) {
	queueSendTestEnv(t)

	queueSendBody = "inline"
	queueSendFile = "some-file.json"

	err := runQueueSend(queueSendCmd, []string{"my-queue"})
	if err == nil {
		t.Fatal("expected error when both --body and --file are set")
	}
	if !strings.Contains(err.Error(), "--body") || !strings.Contains(err.Error(), "--file") {
		t.Errorf("error should mention both flags: %q", err.Error())
	}
}

func TestQueueSend_MissingArgs(t *testing.T) {
	queueSendTestEnv(t)

	queueSendBody = ""
	queueSendFile = ""

	err := runQueueSend(queueSendCmd, []string{"my-queue"})
	if err == nil {
		t.Fatal("expected error when neither --body nor --file is set")
	}
	if !strings.Contains(err.Error(), "--body") || !strings.Contains(err.Error(), "--file") {
		t.Errorf("error should mention the flags: %q", err.Error())
	}
}

func TestQueueSendBatch_MissingFile(t *testing.T) {
	queueSendTestEnv(t)

	queueSendFile = ""

	err := runQueueSendBatch(queueSendBatchCmd, []string{"my-queue"})
	if err == nil {
		t.Fatal("expected error when --file is empty")
	}
}

func TestQueueSendBatch_InvalidJSON(t *testing.T) {
	queueSendTestEnv(t)

	queueSendFile = queueSendWriteFile(t, "not json")

	err := runQueueSendBatch(queueSendBatchCmd, []string{"my-queue"})
	if err == nil {
		t.Fatal("expected error for invalid JSON file")
	}
}

func TestQueueSendBatch_EmptyArray(t *testing.T) {
	queueSendTestEnv(t)

	queueSendFile = queueSendWriteFile(t, "[]")

	err := runQueueSendBatch(queueSendBatchCmd, []string{"my-queue"})
	if err == nil {
		t.Fatal("expected error for empty message array")
	}
}

// --- Over-100 batch rejection ---

func TestQueueSendBatch_Over100Rejected(t *testing.T) {
	queueSendTestEnv(t)
	DryRun = false

	msgs := make([]string, 101)
	for i := range msgs {
		msgs[i] = `{"body":"m"}`
	}
	queueSendFile = queueSendWriteFile(t, "["+strings.Join(msgs, ",")+"]")

	err := runQueueSendBatch(queueSendBatchCmd, []string{"my-queue"})
	if err == nil {
		t.Fatal("expected error when batch exceeds 100 messages")
	}
	msg := err.Error()
	if !strings.Contains(msg, "100") {
		t.Errorf("error should state the 100-message cap: %q", msg)
	}
	if !strings.Contains(msg, "split") {
		t.Errorf("error should explain how to split the batch: %q", msg)
	}
}

// --- Happy paths (DryRun) ---

func TestQueueSend_DryRun(t *testing.T) {
	queueSendTestEnv(t)

	queueSendBody = "hello"
	queueSendContentType = "application/json"
	queueSendDelaySeconds = 30

	output := queueSendCaptureStdout(t, func() {
		if err := runQueueSend(queueSendCmd, []string{"my-queue"}); err != nil {
			t.Errorf("runQueueSend(DryRun) returned error: %v", err)
		}
	})
	if !strings.Contains(output, "DRY RUN") {
		t.Errorf("output should contain 'DRY RUN', got: %q", output)
	}
}

func TestQueueSend_DryRunFromFile(t *testing.T) {
	queueSendTestEnv(t)

	queueSendFile = queueSendWriteFile(t, `{"event":"signup"}`)

	output := queueSendCaptureStdout(t, func() {
		if err := runQueueSend(queueSendCmd, []string{"my-queue"}); err != nil {
			t.Errorf("runQueueSend(DryRun, file) returned error: %v", err)
		}
	})
	if !strings.Contains(output, "DRY RUN") {
		t.Errorf("output should contain 'DRY RUN', got: %q", output)
	}
}

func TestQueueSend_DryRunJSON(t *testing.T) {
	queueSendTestEnv(t)
	JSONOutput = true

	queueSendBody = "hello"

	output := queueSendCaptureStdout(t, func() {
		if err := runQueueSend(queueSendCmd, []string{"my-queue"}); err != nil {
			t.Errorf("runQueueSend(DryRun+JSON) returned error: %v", err)
		}
	})

	var res struct {
		Success bool        `json:"success"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
		DryRun  bool        `json:"dry_run"`
	}
	if err := json.Unmarshal([]byte(output), &res); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, output)
	}
	if !res.Success {
		t.Errorf("expected success=true, got %v", res.Success)
	}
	if !res.DryRun {
		t.Errorf("expected dry_run=true, got %v", res.DryRun)
	}
	if !strings.Contains(res.Message, "DRY RUN") {
		t.Errorf("message should mention DRY RUN: %q", res.Message)
	}
	if res.Data == nil {
		t.Error("expected non-nil data in JSON output")
	}
}

func TestQueueSendBatch_DryRunJSON(t *testing.T) {
	queueSendTestEnv(t)
	JSONOutput = true

	queueSendFile = queueSendWriteFile(t, `[{"body":"one"},{"body":"two"}]`)

	output := queueSendCaptureStdout(t, func() {
		if err := runQueueSendBatch(queueSendBatchCmd, []string{"my-queue"}); err != nil {
			t.Errorf("runQueueSendBatch(DryRun+JSON) returned error: %v", err)
		}
	})

	var res struct {
		Success bool `json:"success"`
		DryRun  bool `json:"dry_run"`
		Data    struct {
			Queue    string `json:"queue"`
			Messages int    `json:"messages"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &res); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, output)
	}
	if !res.Success || !res.DryRun {
		t.Errorf("expected success=true dry_run=true, got %v/%v", res.Success, res.DryRun)
	}
	if res.Data.Messages != 2 {
		t.Errorf("expected data.messages=2, got %d", res.Data.Messages)
	}
}

// --- Oversized message warning ---

func TestQueueSend_OversizedWarning(t *testing.T) {
	queueSendTestEnv(t)

	queueSendBody = strings.Repeat("a", queueSendMaxMessageBytes)

	output := queueSendCaptureStdout(t, func() {
		if err := runQueueSend(queueSendCmd, []string{"my-queue"}); err != nil {
			t.Errorf("oversized message should still send (soft warning), got error: %v", err)
		}
	})
	if !strings.Contains(output, "128,000") {
		t.Errorf("warning should name the 128,000-byte limit, got: %q", output)
	}
	if !strings.Contains(output, "DRY RUN") {
		t.Errorf("command should continue after the warning, got: %q", output)
	}
}

func TestQueueSend_NoWarningUnderLimit(t *testing.T) {
	queueSendTestEnv(t)

	queueSendBody = strings.Repeat("a", queueSendMaxMessageBytes-queueSendMetadataBytes)

	output := queueSendCaptureStdout(t, func() {
		if err := runQueueSend(queueSendCmd, []string{"my-queue"}); err != nil {
			t.Errorf("runQueueSend returned error: %v", err)
		}
	})
	if strings.Contains(output, "exceeds") {
		t.Errorf("no warning expected under the limit, got: %q", output)
	}
}
