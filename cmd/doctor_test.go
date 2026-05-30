package cmd

import (
	"bytes"
	"os"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

// TestDoctorCmdRegistration verifies that doctorCmd is registered on rootCmd
// and has the correct Use and Short fields.
func TestDoctorCmdRegistration(t *testing.T) {
	if doctorCmd == nil {
		t.Fatal("doctorCmd is nil")
	}

	if doctorCmd.Use != "doctor [domain]" {
		t.Errorf("doctorCmd.Use = %q, want %q", doctorCmd.Use, "doctor [domain]")
	}

	if doctorCmd.Short == "" {
		t.Error("doctorCmd.Short is empty")
	}

	// Verify it is registered on rootCmd.
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub == doctorCmd {
			found = true
			break
		}
	}
	if !found {
		t.Error("doctorCmd is not registered on rootCmd")
	}
}

// TestDoctorFlagDefaults verifies --fix and --all default to false.
func TestDoctorFlagDefaults(t *testing.T) {
	fixFlag := doctorCmd.Flags().Lookup("fix")
	if fixFlag == nil {
		t.Fatal("--fix flag not registered")
	}
	if fixFlag.DefValue != "false" {
		t.Errorf("--fix default = %q, want %q", fixFlag.DefValue, "false")
	}

	allFlag := doctorCmd.Flags().Lookup("all")
	if allFlag == nil {
		t.Fatal("--all flag not registered")
	}
	if allFlag.DefValue != "false" {
		t.Errorf("--all default = %q, want %q", allFlag.DefValue, "false")
	}
}

// TestDoctorFlagsParsing verifies that --fix and --all flags can be parsed.
func TestDoctorFlagsParsing(t *testing.T) {
	// Reset flag values before each sub-test.
	reset := func() {
		doctorFix = false
		doctorAll = false
	}

	t.Run("fix flag sets doctorFix", func(t *testing.T) {
		reset()
		cmd := &cobra.Command{}
		cmd.Flags().BoolVar(&doctorFix, "fix", false, "")
		if err := cmd.Flags().Set("fix", "true"); err != nil {
			t.Fatalf("failed to set --fix: %v", err)
		}
		if !doctorFix {
			t.Error("doctorFix should be true after --fix=true")
		}
	})

	t.Run("all flag sets doctorAll", func(t *testing.T) {
		reset()
		cmd := &cobra.Command{}
		cmd.Flags().BoolVar(&doctorAll, "all", false, "")
		if err := cmd.Flags().Set("all", "true"); err != nil {
			t.Fatalf("failed to set --all: %v", err)
		}
		if !doctorAll {
			t.Error("doctorAll should be true after --all=true")
		}
	})
}

// TestDoctorZoneIDPattern verifies the regex for Cloudflare zone ID format.
func TestDoctorZoneIDPattern(t *testing.T) {
	tests := []struct {
		input string
		match bool
	}{
		{"a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6", true},  // valid 32-char lowercase hex
		{"A1B2C3D4E5F6A7B8C9D0E1F2A3B4C5D6", false}, // uppercase — CF uses lowercase
		{"short", false},
		{"", false},
		{"a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6x", false}, // 33 chars — too long
		{"a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d", false},   // 31 chars — too short
		{"a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5g6", false},  // contains 'g' — not hex
		{"00000000000000000000000000000000", true},    // all zeros valid
		{"ffffffffffffffffffffffffffffffff", true},    // all f's valid
	}

	for _, tc := range tests {
		got := zoneIDPattern.MatchString(tc.input)
		if got != tc.match {
			t.Errorf("zoneIDPattern.MatchString(%q) = %v, want %v", tc.input, got, tc.match)
		}
	}
}

// TestDoctorRunNoArgs verifies that runDoctor returns an error when no args
// are provided and doctorAll is false.
func TestDoctorRunNoArgs(t *testing.T) {
	// Save and restore doctorAll.
	origAll := doctorAll
	defer func() { doctorAll = origAll }()

	doctorAll = false

	err := runDoctor(doctorCmd, []string{})
	if err == nil {
		t.Fatal("runDoctor with no args and doctorAll=false should return error")
	}

	want := "domain argument is required"
	if !bytes.Contains([]byte(err.Error()), []byte(want)) {
		t.Errorf("error message = %q, want it to contain %q", err.Error(), want)
	}
}

// TestDoctorPrintReportNoPanic verifies printDoctorReport does not panic with
// a minimal DiagnosticReport (nil sub-structs, empty issues).
func TestDoctorPrintReportNoPanic(t *testing.T) {
	report := &cosmoflare.DiagnosticReport{
		Domain:    "example.com",
		Timestamp: time.Now(),
		Score:     "healthy",
		Issues:    []cosmoflare.DiagnosticIssue{},
	}

	// Redirect stdout.
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	defer func() {
		if rec := recover(); rec != nil {
			w.Close()
			os.Stdout = old
			t.Fatalf("printDoctorReport panicked: %v", rec)
		}
	}()

	printDoctorReport(report)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("example.com")) {
		t.Errorf("output should contain domain name, got: %q", output)
	}
	if !bytes.Contains([]byte(output), []byte("healthy")) {
		t.Errorf("output should contain score, got: %q", output)
	}
}

// TestDoctorPrintReportScores verifies score variants don't panic.
func TestDoctorPrintReportScores(t *testing.T) {
	for _, score := range []string{"healthy", "warning", "critical"} {
		t.Run(score, func(t *testing.T) {
			report := &cosmoflare.DiagnosticReport{
				Domain:    "test.com",
				Timestamp: time.Now(),
				Score:     score,
				Issues:    []cosmoflare.DiagnosticIssue{},
			}

			old := os.Stdout
			_, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("failed to create pipe: %v", err)
			}
			os.Stdout = w

			defer func() {
				if rec := recover(); rec != nil {
					w.Close()
					os.Stdout = old
					t.Fatalf("printDoctorReport(%s) panicked: %v", score, rec)
				}
			}()

			printDoctorReport(report)
			w.Close()
			os.Stdout = old
		})
	}
}

// TestDoctorIssuesSectionEmpty verifies printIssuesSection prints nothing for an
// empty issues slice.
func TestDoctorIssuesSectionEmpty(t *testing.T) {
	report := &cosmoflare.DiagnosticReport{
		Domain:  "example.com",
		Score:   "healthy",
		Issues:  []cosmoflare.DiagnosticIssue{},
	}

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	printIssuesSection(report)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output != "" {
		t.Errorf("printIssuesSection with no issues should produce no output, got: %q", output)
	}
}

// TestDoctorIssuesSectionWithIssues verifies printIssuesSection renders issues
// and respects doctorFix for fix suggestions.
func TestDoctorIssuesSectionWithIssues(t *testing.T) {
	report := &cosmoflare.DiagnosticReport{
		Domain: "example.com",
		Score:  "warning",
		Issues: []cosmoflare.DiagnosticIssue{
			{
				Probe:    "ssl",
				Severity: "warning",
				Message:  "Certificate expires in 15 days",
				Fix:      "cosmoflare ssl update --zone example.com",
			},
			{
				Probe:    "dns",
				Severity: "critical",
				Message:  "DNS inconsistency detected",
				Fix:      "cosmoflare dns check --zone example.com",
			},
		},
	}

	captureOutput := func(fix bool) string {
		origFix := doctorFix
		doctorFix = fix
		defer func() { doctorFix = origFix }()

		old := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		os.Stdout = w

		printIssuesSection(report)

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		buf.ReadFrom(r)
		return buf.String()
	}

	t.Run("without fix flag", func(t *testing.T) {
		output := captureOutput(false)
		if !bytes.Contains([]byte(output), []byte("ISSUES")) {
			t.Errorf("output should contain ISSUES header, got: %q", output)
		}
		if !bytes.Contains([]byte(output), []byte("Certificate expires")) {
			t.Errorf("output should contain issue message, got: %q", output)
		}
		if bytes.Contains([]byte(output), []byte("FIX:")) {
			t.Errorf("output should NOT contain FIX: when doctorFix=false, got: %q", output)
		}
	})

	t.Run("with fix flag", func(t *testing.T) {
		output := captureOutput(true)
		if !bytes.Contains([]byte(output), []byte("ISSUES")) {
			t.Errorf("output should contain ISSUES header, got: %q", output)
		}
		if !bytes.Contains([]byte(output), []byte("FIX:")) {
			t.Errorf("output should contain FIX: when doctorFix=true, got: %q", output)
		}
		if !bytes.Contains([]byte(output), []byte("cosmoflare ssl update")) {
			t.Errorf("output should contain fix command, got: %q", output)
		}
	})

	t.Run("severity indicators", func(t *testing.T) {
		output := captureOutput(false)
		if !bytes.Contains([]byte(output), []byte("⚠")) {
			t.Errorf("output should contain warning indicator ⚠, got: %q", output)
		}
		if !bytes.Contains([]byte(output), []byte("✗")) {
			t.Errorf("output should contain critical indicator ✗, got: %q", output)
		}
	})
}

// TestDoctorIssuesSectionSeverityIndicators verifies all severity levels render
// the correct indicator character.
func TestDoctorIssuesSectionSeverityIndicators(t *testing.T) {
	cases := []struct {
		severity  string
		indicator string
	}{
		{"critical", "✗"},
		{"warning", "⚠"},
		{"info", "ℹ"},
		{"unknown", "?"},
	}

	for _, tc := range cases {
		t.Run(tc.severity, func(t *testing.T) {
			report := &cosmoflare.DiagnosticReport{
				Domain: "example.com",
				Score:  "warning",
				Issues: []cosmoflare.DiagnosticIssue{
					{Probe: "test", Severity: tc.severity, Message: "test issue"},
				},
			}

			old := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("failed to create pipe: %v", err)
			}
			os.Stdout = w

			origFix := doctorFix
			doctorFix = false
			printIssuesSection(report)
			doctorFix = origFix

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if !bytes.Contains([]byte(output), []byte(tc.indicator)) {
				t.Errorf("severity=%q: output should contain %q, got: %q", tc.severity, tc.indicator, output)
			}
		})
	}
}
