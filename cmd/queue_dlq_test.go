package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// queueDlqTestEnv sets the globals the queue dlq/consumer commands need and
// restores them when the test ends.
func queueDlqTestEnv(t *testing.T) {
	t.Helper()
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origSettingsFile := queueConsumerSettingsFile
	origConsumerDLQ := queueDLQConsumerName
	origProducerDLQ := queueDLQProducerName
	origClear := queueDLQClear
	t.Cleanup(func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		queueConsumerSettingsFile = origSettingsFile
		queueDLQConsumerName = origConsumerDLQ
		queueDLQProducerName = origProducerDLQ
		queueDLQClear = origClear
	})

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
}

func queueDlqWriteFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

func queueDlqCaptureStdout(t *testing.T, fn func()) string {
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

func TestQueueDlqCmd_RegisteredOnQueue(t *testing.T) {
	for _, name := range []string{"consumer", "dlq"} {
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

func TestQueueConsumerCmd_RegisteredSubcommands(t *testing.T) {
	for _, name := range []string{"update", "remove"} {
		found := false
		for _, sub := range queueConsumerCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("queue consumer subcommand %q not registered", name)
		}
	}
}

func TestQueueDlqCmd_RunE(t *testing.T) {
	if queueConsumerUpdateCmd.RunE == nil {
		t.Error("queueConsumerUpdateCmd.RunE is nil")
	}
	if queueConsumerRemoveCmd.RunE == nil {
		t.Error("queueConsumerRemoveCmd.RunE is nil")
	}
	if queueDlqCmd.RunE == nil {
		t.Error("queueDlqCmd.RunE is nil")
	}
}

func TestQueueDlqCmd_Args(t *testing.T) {
	if queueConsumerUpdateCmd.Args == nil {
		t.Error("queueConsumerUpdateCmd.Args is nil, expected cobra.ExactArgs(2)")
	}
	if queueConsumerRemoveCmd.Args == nil {
		t.Error("queueConsumerRemoveCmd.Args is nil, expected cobra.ExactArgs(2)")
	}
	if queueDlqCmd.Args == nil {
		t.Error("queueDlqCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

// --- Flag registration ---

func TestQueueConsumerUpdate_Flags(t *testing.T) {
	if queueConsumerUpdateCmd.Flags().Lookup("settings-file") == nil {
		t.Error("--settings-file flag not registered on queueConsumerUpdateCmd")
	}
}

func TestQueueDlq_Flags(t *testing.T) {
	for _, name := range []string{"consumer-dlq", "producer-dlq", "clear"} {
		if queueDlqCmd.Flags().Lookup(name) == nil {
			t.Errorf("--%s flag not registered on queueDlqCmd", name)
		}
	}
}

// --- Validation ---

func TestQueueDlq_ClearExclusivity(t *testing.T) {
	queueDlqTestEnv(t)

	queueDLQClear = true
	queueDLQConsumerName = "my-dlq"

	err := runQueueDlq(queueDlqCmd, []string{"my-queue"})
	if err == nil {
		t.Error("expected error when --clear is combined with --consumer-dlq")
	}
}

func TestQueueDlq_ClearExclusivityProducer(t *testing.T) {
	queueDlqTestEnv(t)

	queueDLQClear = true
	queueDLQProducerName = "my-dlq"

	err := runQueueDlq(queueDlqCmd, []string{"my-queue"})
	if err == nil {
		t.Error("expected error when --clear is combined with --producer-dlq")
	}
}

func TestQueueConsumerUpdate_InvalidSettingsFile(t *testing.T) {
	queueDlqTestEnv(t)

	queueConsumerSettingsFile = queueDlqWriteFile(t, "not json")

	err := runQueueConsumerUpdate(queueConsumerUpdateCmd, []string{"my-queue", "my-consumer"})
	if err == nil {
		t.Error("expected error when settings file is not valid JSON")
	}
}

func TestQueueConsumerUpdate_MissingSettingsFile(t *testing.T) {
	queueDlqTestEnv(t)

	queueConsumerSettingsFile = filepath.Join(t.TempDir(), "does-not-exist.json")

	err := runQueueConsumerUpdate(queueConsumerUpdateCmd, []string{"my-queue", "my-consumer"})
	if err == nil {
		t.Error("expected error when settings file does not exist")
	}
}

// --- Dry run ---

func TestQueueConsumerUpdate_DryRun(t *testing.T) {
	queueDlqTestEnv(t)

	queueConsumerSettingsFile = queueDlqWriteFile(t, `{"batch_size":10,"max_retries":3,"max_wait_time_ms":5000}`)

	output := queueDlqCaptureStdout(t, func() {
		if err := runQueueConsumerUpdate(queueConsumerUpdateCmd, []string{"my-queue", "my-consumer"}); err != nil {
			t.Errorf("runQueueConsumerUpdate(DryRun) returned error: %v", err)
		}
	})
	if !strings.Contains(output, "DRY RUN") {
		t.Errorf("output should contain 'DRY RUN', got: %q", output)
	}
}

func TestQueueConsumerUpdate_DryRunJSON(t *testing.T) {
	queueDlqTestEnv(t)
	JSONOutput = true

	queueConsumerSettingsFile = queueDlqWriteFile(t, `{"batch_size":10}`)

	output := queueDlqCaptureStdout(t, func() {
		if err := runQueueConsumerUpdate(queueConsumerUpdateCmd, []string{"my-queue", "my-consumer"}); err != nil {
			t.Errorf("runQueueConsumerUpdate(DryRun+JSON) returned error: %v", err)
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
	if !res.Success || !res.DryRun {
		t.Errorf("expected success=true, dry_run=true, got %+v", res)
	}
}

func TestQueueConsumerRemove_DryRun(t *testing.T) {
	queueDlqTestEnv(t)

	output := queueDlqCaptureStdout(t, func() {
		if err := runQueueConsumerRemove(queueConsumerRemoveCmd, []string{"my-queue", "my-consumer"}); err != nil {
			t.Errorf("runQueueConsumerRemove(DryRun) returned error: %v", err)
		}
	})
	if !strings.Contains(output, "DRY RUN") {
		t.Errorf("output should contain 'DRY RUN', got: %q", output)
	}
}

func TestQueueConsumerRemove_DryRunJSON(t *testing.T) {
	queueDlqTestEnv(t)
	JSONOutput = true

	output := queueDlqCaptureStdout(t, func() {
		if err := runQueueConsumerRemove(queueConsumerRemoveCmd, []string{"my-queue", "my-consumer"}); err != nil {
			t.Errorf("runQueueConsumerRemove(DryRun+JSON) returned error: %v", err)
		}
	})

	var res struct {
		Success bool `json:"success"`
		DryRun  bool `json:"dry_run"`
	}
	if err := json.Unmarshal([]byte(output), &res); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, output)
	}
	if !res.Success || !res.DryRun {
		t.Errorf("expected success=true, dry_run=true, got %+v", res)
	}
}

func TestQueueDlq_ConfigureDryRun(t *testing.T) {
	queueDlqTestEnv(t)

	queueDLQConsumerName = "my-dlq"

	output := queueDlqCaptureStdout(t, func() {
		if err := runQueueDlq(queueDlqCmd, []string{"my-queue"}); err != nil {
			t.Errorf("runQueueDlq(DryRun) returned error: %v", err)
		}
	})
	if !strings.Contains(output, "DRY RUN") {
		t.Errorf("output should contain 'DRY RUN', got: %q", output)
	}
}

func TestQueueDlq_ConfigureDryRunJSON(t *testing.T) {
	queueDlqTestEnv(t)
	JSONOutput = true

	queueDLQProducerName = "my-dlq"

	output := queueDlqCaptureStdout(t, func() {
		if err := runQueueDlq(queueDlqCmd, []string{"my-queue"}); err != nil {
			t.Errorf("runQueueDlq(DryRun+JSON) returned error: %v", err)
		}
	})

	var res struct {
		Success bool `json:"success"`
		DryRun  bool `json:"dry_run"`
	}
	if err := json.Unmarshal([]byte(output), &res); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, output)
	}
	if !res.Success || !res.DryRun {
		t.Errorf("expected success=true, dry_run=true, got %+v", res)
	}
}

func TestQueueDlq_ClearDryRun(t *testing.T) {
	queueDlqTestEnv(t)

	queueDLQClear = true

	output := queueDlqCaptureStdout(t, func() {
		if err := runQueueDlq(queueDlqCmd, []string{"my-queue"}); err != nil {
			t.Errorf("runQueueDlq(DryRun, clear) returned error: %v", err)
		}
	})
	if !strings.Contains(output, "DRY RUN") || !strings.Contains(output, "clear") {
		t.Errorf("output should mention DRY RUN and clear, got: %q", output)
	}
}
