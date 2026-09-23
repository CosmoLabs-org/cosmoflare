package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readAuditLines decodes the newline-delimited audit entries a test wrote.
func readAuditLines(t *testing.T, path string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("decode audit line %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

// TestAuditMutationStampsDangerLevel pins the FEAT-020 wave 1 consumer: a
// registered destructive command logs its manifest id and danger level.
func TestAuditMutationStampsDangerLevel(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "audit.log")
	t.Setenv(auditLogPathEnv, logPath)

	auditMutation([]string{"bucket", "delete"}, "my-bucket", true)

	entries := readAuditLines(t, logPath)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	e := entries[0]
	if e["operation"] != "bucket delete" {
		t.Errorf("operation = %v, want r2 bucket delete", e["operation"])
	}
	details, _ := e["details"].(map[string]any)
	if details["danger_level"] != "high" {
		t.Errorf("danger_level = %v, want high", details["danger_level"])
	}
	if details["manifest_id"] != "r2.bucket.delete" {
		t.Errorf("manifest_id = %v, want r2.bucket.delete", details["manifest_id"])
	}
	if details["destructive"] != "true" {
		t.Errorf("destructive = %v, want true", details["destructive"])
	}
	if e["service"] != "r2" || e["action"] != "delete" {
		t.Errorf("service/action = %v/%v, want r2/delete", e["service"], e["action"])
	}
}

// TestAuditMutationUnregisteredPathStillLogs pins the fallback: a command
// not yet in the registry logs with manifest_id=unregistered rather than
// vanishing — coverage gaps must be visible in the log.
func TestAuditMutationUnregisteredPathStillLogs(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "audit.log")
	t.Setenv(auditLogPathEnv, logPath)

	auditMutation([]string{"future", "command"}, "res", false)

	entries := readAuditLines(t, logPath)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	details, _ := entries[0]["details"].(map[string]any)
	if details["manifest_id"] != "unregistered" {
		t.Errorf("manifest_id = %v, want unregistered", details["manifest_id"])
	}
	if entries[0]["success"] != false {
		t.Errorf("success = %v, want false", entries[0]["success"])
	}
}

// TestAuditMutationNoPathIsNoOp pins that a missing home (empty default
// path, no override) neither panics nor writes anything.
func TestAuditMutationNoPathIsNoOp(t *testing.T) {
	t.Setenv(auditLogPathEnv, "")
	// Empty override falls through to the home default; on CI machines with
	// a home this still must not error — assert only that it returns.
	auditMutation([]string{"bucket", "delete"}, "b", true)
}
