package cmd

import (
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// TestRateLimitProbeCmdRegistered verifies the probe subcommand exists.
func TestRateLimitProbeCmdRegistered(t *testing.T) {
	for _, c := range rateLimitCmd.Commands() {
		if c.Name() == "probe" {
			return
		}
	}
	t.Fatal("ratelimit probe subcommand missing")
}

// TestProbeExitCode verifies the scriptable exit-code mapping.
func TestProbeExitCode(t *testing.T) {
	tests := []struct {
		verdict string
		want    int
	}{
		{cosmoflare.VerdictTripped, 0},
		{cosmoflare.VerdictNotCounted, 2},
		{cosmoflare.VerdictInconclusive, 3},
		{"", 3},
	}
	for _, tt := range tests {
		if got := probeExitCode(tt.verdict); got != tt.want {
			t.Errorf("probeExitCode(%q) = %d, want %d", tt.verdict, got, tt.want)
		}
	}
}

// TestProbeAdvisory verifies the create advisory is pack-driven: it names
// the skipped classes from the embedded ratelimit pack.
func TestProbeAdvisory(t *testing.T) {
	got := probeAdvisory()
	if !strings.Contains(got, "cache-hit-static-asset") || !strings.Contains(got, "ratelimit probe") {
		t.Fatalf("advisory must list skipped classes and suggest the probe: %q", got)
	}
}
