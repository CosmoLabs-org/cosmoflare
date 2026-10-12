package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// deadFunctionsFixture builds a fake project root in a t.TempDir() and
// creates the given slash-relative files inside it.
func deadFunctionsFixture(t *testing.T, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range files {
		p := filepath.Join(dir, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", f, err)
		}
		if err := os.WriteFile(p, []byte("export default { fetch() {} };"), 0o644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}
	return dir
}

func TestAuditDeadFunctionsBothDirsShadowed(t *testing.T) {
	dir := deadFunctionsFixture(t,
		"functions/api/subscribe.ts",
		"functions/api/status.js",
		"dist/_worker.js",
	)
	report, err := auditDeadFunctionsScan(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(report.Shadowed) != 2 {
		t.Fatalf("want 2 shadowed routes, got %d: %+v", len(report.Shadowed), report.Shadowed)
	}
	routes := map[string]bool{}
	for _, s := range report.Shadowed {
		routes[s.Route] = true
	}
	if !routes["/api/subscribe"] || !routes["/api/status"] {
		t.Fatalf("expected /api/subscribe and /api/status in %+v", routes)
	}
	if report.Worker == nil || !strings.HasSuffix(*report.Worker, "dist/_worker.js") {
		t.Fatalf("expected worker dist/_worker.js, got %v", report.Worker)
	}
}

func TestAuditDeadFunctionsFunctionsOnly(t *testing.T) {
	dir := deadFunctionsFixture(t, "functions/api/subscribe.ts")
	report, err := auditDeadFunctionsScan(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(report.Shadowed) != 0 {
		t.Fatalf("expected no shadowed routes, got %+v", report.Shadowed)
	}
	if report.Worker != nil {
		t.Fatalf("expected worker null, got %q", *report.Worker)
	}
}

func TestAuditDeadFunctionsWorkerOnly(t *testing.T) {
	dir := deadFunctionsFixture(t, "_worker.js")
	report, err := auditDeadFunctionsScan(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(report.Shadowed) != 0 {
		t.Fatalf("expected no shadowed routes, got %+v", report.Shadowed)
	}
	if report.Worker == nil || !strings.HasSuffix(*report.Worker, "_worker.js") {
		t.Fatalf("expected worker _worker.js, got %v", report.Worker)
	}
}

func TestAuditDeadFunctionsNeither(t *testing.T) {
	dir := deadFunctionsFixture(t, "src/pages/index.astro")
	report, err := auditDeadFunctionsScan(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(report.Shadowed) != 0 {
		t.Fatalf("expected no shadowed routes, got %+v", report.Shadowed)
	}
	if report.Worker != nil {
		t.Fatalf("expected worker null, got %q", *report.Worker)
	}
}

func TestAuditDeadFunctionsWorkerPathPrecedence(t *testing.T) {
	// All three documented locations present: root wins, then dist, then build.
	dir := deadFunctionsFixture(t, "_worker.js", "dist/_worker.js", "build/_worker.js")
	report, err := auditDeadFunctionsScan(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if report.Worker == nil || filepath.Dir(*report.Worker) != dir || !strings.HasSuffix(*report.Worker, "_worker.js") {
		t.Fatalf("expected root _worker.js, got %v", report.Worker)
	}
}

func TestAuditDeadFunctionsBuildWorkerDetected(t *testing.T) {
	dir := deadFunctionsFixture(t, "functions/api/ping.ts", "build/_worker.js")
	report, err := auditDeadFunctionsScan(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(report.Shadowed) != 1 {
		t.Fatalf("expected 1 shadowed route, got %+v", report.Shadowed)
	}
	if report.Worker == nil || !strings.HasSuffix(*report.Worker, "build/_worker.js") {
		t.Fatalf("expected worker build/_worker.js, got %v", report.Worker)
	}
}

func TestAuditDeadFunctionsRouteMapping(t *testing.T) {
	// Shadowing needs BOTH sides present: an index handler and a worker.
	dir := deadFunctionsFixture(t, "functions/index.ts", "_worker.js")
	report, err := auditDeadFunctionsScan(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(report.Shadowed) != 1 || report.Shadowed[0].Route != "/" {
		t.Fatalf("expected functions/index.ts → /, got %+v", report.Shadowed)
	}
}

func TestAuditDeadFunctionsExitCodes(t *testing.T) {
	oldExit := deadFunctionsExit
	defer func() { deadFunctionsExit = oldExit }()

	// Both dirs: exit 1.
	code := 0
	deadFunctionsExit = func(i int) { code = i }
	dir := deadFunctionsFixture(t, "functions/api/subscribe.ts", "dist/_worker.js")
	cmd := auditDeadFunctionsCmd
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	if err := runAuditDeadFunctions(cmd, []string{dir}); err != nil {
		t.Fatalf("RunE (shadowed): %v", err)
	}
	if code != 1 {
		t.Fatalf("expected exit code 1 when shadowed, got %d", code)
	}

	// Neither: exit 0.
	code = 0
	dir = deadFunctionsFixture(t)
	if err := runAuditDeadFunctions(cmd, []string{dir}); err != nil {
		t.Fatalf("RunE (clean): %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0 when clean, got %d", code)
	}

	// Missing project dir: RunE errors (cobra exits non-zero upstream).
	if err := runAuditDeadFunctions(cmd, []string{filepath.Join(dir, "nope")}); err == nil {
		t.Fatal("expected error for missing project dir")
	}
}
