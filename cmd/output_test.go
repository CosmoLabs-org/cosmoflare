package cmd

import (
	"errors"
	"fmt"
	"testing"
)

// --- FEAT-040 slice: the output presenter collapses the 618 if-JSONOutput
// branches into one unconditional call site per command ---

func TestPresenter_ErrorPlainMode(t *testing.T) {
	old := JSONOutput
	JSONOutput = false
	t.Cleanup(func() { JSONOutput = old })

	p := NewPresenter()
	err := p.Error("bucket %q not found", "data")
	if err == nil {
		t.Fatal("Error returned nil in plain mode — must fail the command")
	}
	if got := err.Error(); got != `bucket "data" not found` {
		t.Errorf("plain-mode error = %q, want formatted message", got)
	}
}

func TestPresenter_ErrorJSONModeFailsTheCommand(t *testing.T) {
	// Decision 2026-09-14: JSON-mode errors print the envelope to stdout
	// (machine-parseable) AND return a non-nil error so cobra exits 1 —
	// deterministic exit codes apply to --json consumers too.
	old := JSONOutput
	JSONOutput = true
	t.Cleanup(func() { JSONOutput = old })

	p := NewPresenter()
	err := p.Error("bucket %q not found", "data")
	if err == nil {
		t.Fatal("JSON-mode Error returned nil — command would exit 0 on failure")
	}
	if got := err.Error(); got != `bucket "data" not found` {
		t.Errorf("JSON-mode error message = %q, want the formatted message", got)
	}
}

func TestPresenter_RespectsModeCapturedAtConstruction(t *testing.T) {
	old := JSONOutput
	JSONOutput = false
	t.Cleanup(func() { JSONOutput = old })

	plain := NewPresenter()
	JSONOutput = true
	jsonP := NewPresenter()

	if plain.IsJSON() || !jsonP.IsJSON() {
		t.Error("presenter captured the wrong mode at construction")
	}
}

func TestPresenter_ResultPlainInvokesHumanRenderer(t *testing.T) {
	old := JSONOutput
	JSONOutput = false
	t.Cleanup(func() { JSONOutput = old })

	p := NewPresenter()
	called := false
	err := p.Result(map[string]int{"n": 1}, func() { called = true })
	if err != nil {
		t.Fatalf("plain Result error: %v", err)
	}
	if !called {
		t.Error("plain mode must invoke the human renderer")
	}
}

func TestPresenter_ResultJSONSkipsHumanRenderer(t *testing.T) {
	old := JSONOutput
	JSONOutput = true
	t.Cleanup(func() { JSONOutput = old })

	p := NewPresenter()
	called := false
	_ = p.Result(map[string]int{"n": 1}, func() { called = true })
	if called {
		t.Error("JSON mode must not invoke the human renderer (tables would pollute stdout)")
	}
}

func TestPresenter_SuccessPayloadRoutesByMode(t *testing.T) {
	old := JSONOutput
	t.Cleanup(func() { JSONOutput = old })

	jsonBuilt, humanRan := false, false

	JSONOutput = true
	pj := NewPresenter()
	_ = pj.SuccessPayload("done", func() any { jsonBuilt = true; return 1 }, func() { humanRan = true })
	if !jsonBuilt || humanRan {
		t.Errorf("JSON mode: json closure ran=%v (want true), human ran=%v (want false)", jsonBuilt, humanRan)
	}

	jsonBuilt, humanRan = false, false
	JSONOutput = false
	pp := NewPresenter()
	_ = pp.SuccessPayload("done", func() any { jsonBuilt = true; return 1 }, func() { humanRan = true })
	if jsonBuilt || !humanRan {
		t.Errorf("plain mode: json closure ran=%v (want false), human ran=%v (want true)", jsonBuilt, humanRan)
	}
}

func TestPresenter_ErrorWrapPreservesWrapping(t *testing.T) {
	old := JSONOutput
	JSONOutput = false
	t.Cleanup(func() { JSONOutput = old })

	sentinel := fmt.Errorf("boom")
	p := NewPresenter()
	err := p.ErrorWrap("list failed", sentinel)
	if !errors.Is(err, sentinel) {
		t.Error("plain-mode ErrorWrap must preserve %w wrapping for errors.Is")
	}
}

func TestPrintErrorJSON_ReturnsNonNilAfterPrinting(t *testing.T) {
	// The envelope goes to stdout; the return value must be non-nil so the
	// CLI exits non-zero (agent-first contract: parseable body + exit code).
	err := printErrorJSON("kaboom")
	if err == nil {
		t.Fatal("printErrorJSON returned nil — exit code stays 0 on error")
	}
	if err.Error() != "kaboom" {
		t.Errorf("returned error = %q, want the envelope message", err.Error())
	}
}
