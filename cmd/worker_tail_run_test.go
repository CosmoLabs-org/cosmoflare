package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

func TestWorkerTailFormatEntryText(t *testing.T) {
	e := &cosmoflare.LogEntry{
		Timestamp: time.Date(2026, 9, 16, 12, 30, 5, 0, time.UTC),
		Level:     "error",
		Message:   "fetch failed",
	}
	got := formatTailEntry(e, false)
	want := "2026-09-16T12:30:05Z error fetch failed"
	if got != want {
		t.Fatalf("text entry = %q, want %q", got, want)
	}
}

func TestWorkerTailFormatEntryJSON(t *testing.T) {
	e := &cosmoflare.LogEntry{
		Timestamp: time.Date(2026, 9, 16, 12, 30, 5, 0, time.UTC),
		Level:     "info",
		Message:   "hello",
		Event:     "request",
	}
	got := formatTailEntry(e, true)
	var decoded cosmoflare.LogEntry
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("json entry is not valid JSON: %v (%q)", err, got)
	}
	if decoded.Level != "info" || decoded.Message != "hello" || decoded.Event != "request" {
		t.Fatalf("json entry decoded wrong: %+v", decoded)
	}
	if !strings.Contains(got, `"timestamp":"2026-09-16T12:30:05Z"`) {
		t.Fatalf("json entry missing timestamp: %q", got)
	}
}

func TestWorkerTailFormatEntryNil(t *testing.T) {
	if got := formatTailEntry(nil, true); got != "" {
		t.Fatalf("nil entry should render empty, got %q", got)
	}
}

func TestWorkerTailValidation(t *testing.T) {
	runGlobalsSnapshot(t)
	if err := runWorkerTail(workerTailCmd, nil); err == nil || !strings.Contains(err.Error(), "worker name is required") {
		t.Fatalf("expected arg validation error, got %v", err)
	}
}

func TestWorkerTailInvalidFormat(t *testing.T) {
	runGlobalsSnapshot(t)
	old := workerTailFormat
	workerTailFormat = "yaml"
	t.Cleanup(func() { workerTailFormat = old })
	if err := runWorkerTail(workerTailCmd, []string{"api"}); err == nil || !strings.Contains(err.Error(), "invalid --format") {
		t.Fatalf("expected format validation error, got %v", err)
	}
}

func TestWorkerTailFailsOffline(t *testing.T) {
	runGlobalsSnapshot(t)
	old := workerTailFormat
	workerTailFormat = "text"
	t.Cleanup(func() { workerTailFormat = old })
	err := runWorkerTail(workerTailCmd, []string{"api"})
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected offline service error, got %v", err)
	}
}

func TestRegisterWorkerTailCmds(t *testing.T) {
	parent := workerCmd
	registerWorkerTailCmds(parent)
	registerWorkerTailCmds(parent) // idempotent: must not panic on double registration
	found := false
	for _, c := range parent.Commands() {
		if c.Name() == "tail" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("worker tree missing \"tail\" subcommand")
	}
}
