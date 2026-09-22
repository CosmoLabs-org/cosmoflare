package cmd

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// wafRateLimitRunSnapshot snapshots the WAF ratelimit flag variables on top
// of the shared run globals so tests cannot leak state into each other.
func wafRateLimitRunSnapshot(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	savedRequests, savedPeriod := wafRateLimitRequests, wafRateLimitPeriod
	savedAction := wafRateLimitAction
	t.Cleanup(func() {
		wafRateLimitRequests, wafRateLimitPeriod = savedRequests, savedPeriod
		wafRateLimitAction = savedAction
	})
}

// wafRateLimitResetVars clears every variable the runner reads.
func wafRateLimitResetVars() {
	wafRateLimitRequests, wafRateLimitPeriod = 10, 10
	wafRateLimitAction = ""
}

// TestBuildWAFRatelimitInput verifies the pure rule builder: expression
// string, default action, flag action, threshold numbers, and the mandatory
// characteristics.
func TestBuildWAFRatelimitInput(t *testing.T) {
	in, err := buildWAFRatelimitInput("z1", "/login", "", 30, 60)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if in.Expression != `starts_with(http.request.uri.path, "/login")` {
		t.Errorf("expression: got %q", in.Expression)
	}
	if in.Action != "challenge" {
		t.Errorf("default action: want challenge, got %q", in.Action)
	}
	if in.RequestsPerPeriod != 30 || in.Period != 60 {
		t.Errorf("threshold: want 30/60, got %d/%d", in.RequestsPerPeriod, in.Period)
	}
	if in.ZoneID != "z1" {
		t.Errorf("zone ID: got %q", in.ZoneID)
	}
	if !containsStr(in.Characteristics, "cf.colo.id") {
		t.Errorf("characteristics missing mandatory cf.colo.id: %v", in.Characteristics)
	}

	blockedIn, err := buildWAFRatelimitInput("z1", "/api/search", "block", 100, 60)
	if err != nil {
		t.Fatalf("build block: %v", err)
	}
	if blockedIn.Action != "block" {
		t.Errorf("explicit action: want block, got %q", blockedIn.Action)
	}
}

// TestValidateWAFRatelimitSpec covers every contract violation with its
// agent-readable message.
func TestValidateWAFRatelimitSpec(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		action   string
		requests int
		period   int
		wantErr  string
	}{
		{"relative path", "login", "", 30, 10, `must be an absolute URL path`},
		{"bad action", "/login", "managed_challenge", 30, 10, `invalid --action`},
		{"zero requests", "/login", "", 0, 10, "--requests must be a positive"},
		{"negative requests", "/login", "", -1, 10, "--requests must be a positive"},
		{"zero period", "/login", "", 30, 0, "--period must be a positive"},
		{"valid", "/login", "block", 30, 10, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWAFRatelimitSpec(tc.path, tc.action, tc.requests, tc.period)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunWAFRatelimit_Validation verifies the runner rejects malformed args
// before building a service.
func TestRunWAFRatelimit_Validation(t *testing.T) {
	wafRateLimitRunSnapshot(t)
	cases := []struct {
		name    string
		args    []string
		path    string
		action  string
		wantErr string
	}{
		{"missing args", []string{"z1"}, "/login", "", "zone ID and path are required"},
		{"no args", nil, "/login", "", "zone ID and path are required"},
		{"relative path", []string{"z1", "login"}, "login", "", "must be an absolute URL path"},
		{"bad action", []string{"z1", "/login"}, "/login", "drop", "invalid --action"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wafRateLimitResetVars()
			wafRateLimitAction = tc.action
			err := runWAFRatelimit(wafRateLimitCmd, tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunWAFRatelimit_MissingCreds verifies the runner aborts with a wrapped
// service error before any network call when credentials are empty.
func TestRunWAFRatelimit_MissingCreds(t *testing.T) {
	wafRateLimitRunSnapshot(t)
	wafRateLimitResetVars()

	err := runWAFRatelimit(wafRateLimitCmd, []string{"z1", "/login"})
	if err == nil || !strings.Contains(err.Error(), "failed to create WAF rate-limit service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
}

// TestRunWAFRatelimit_DryRun verifies dry-run short-circuits before any
// service creation (no credentials needed) and prints the full would-be
// rule in both output modes.
func TestRunWAFRatelimit_DryRun(t *testing.T) {
	wafRateLimitRunSnapshot(t)
	cases := []struct {
		name string
		json bool
	}{
		{"human output", false},
		{"json output", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wafRateLimitResetVars()
			DryRun = true
			JSONOutput = tc.json
			wafRateLimitRequests, wafRateLimitPeriod = 30, 60
			wafRateLimitAction = "block"

			r, restore := captureStdout(t)
			err := runWAFRatelimit(wafRateLimitCmd, []string{"z1", "/login"})
			restore()
			if err != nil {
				t.Fatalf("dry-run returned error: %v", err)
			}
			out, readErr := io.ReadAll(r)
			if readErr != nil {
				t.Fatalf("reading captured stdout: %v", readErr)
			}
			text := string(out)
			for _, want := range []string{
				`starts_with(http.request.uri.path, "/login")`,
				"block",
				"z1",
				"30",
				"60",
			} {
				if !strings.Contains(text, want) {
					t.Errorf("output missing %q, got: %s", want, text)
				}
			}
			if tc.json {
				var envelope struct {
					Success bool           `json:"success"`
					Message string         `json:"message"`
					Data    map[string]any `json:"data"`
					DryRun  bool           `json:"dry_run"`
				}
				if err := json.Unmarshal(out, &envelope); err != nil {
					t.Fatalf("JSON envelope decode: %v\n%s", err, text)
				}
				if !envelope.Success || !envelope.DryRun {
					t.Errorf("envelope success/dry_run flags wrong: %+v", envelope)
				}
				if envelope.Data["expression"] != `starts_with(http.request.uri.path, "/login")` {
					t.Errorf("envelope data expression wrong: %v", envelope.Data["expression"])
				}
				if envelope.Data["zone_id"] != "z1" {
					t.Errorf("envelope data zone wrong: %v", envelope.Data["zone_id"])
				}
			}
		})
	}
}
