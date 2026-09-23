package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// lbFlagSnapshot snapshots the loadbalancer run flags and restores them on
// cleanup, so one test's flag values cannot leak into the next.
func lbFlagSnapshot(t *testing.T) {
	t.Helper()
	saved := lbResetFlagsSnapshot()
	t.Cleanup(saved)
}

func lbResetFlagsSnapshot() func() {
	name, monitor, origins := lbPoolName, lbPoolMonitor, lbPoolOriginsJSON
	enabled, disable, force, health := lbPoolEnabled, lbPoolDisable, lbPoolForce, lbPoolHealth
	mType, mPath, mCodes := lbMonitorType, lbMonitorPath, lbMonitorExpectedCodes
	mPort, mInterval, mRetries, mTimeout := lbMonitorPort, lbMonitorInterval, lbMonitorRetries, lbMonitorTimeout
	mForce := lbMonitorForce
	return func() {
		lbPoolName, lbPoolMonitor, lbPoolOriginsJSON = name, monitor, origins
		lbPoolEnabled, lbPoolDisable, lbPoolForce, lbPoolHealth = enabled, disable, force, health
		lbMonitorType, lbMonitorPath, lbMonitorExpectedCodes = mType, mPath, mCodes
		lbMonitorPort, lbMonitorInterval, lbMonitorRetries, lbMonitorTimeout = mPort, mInterval, mRetries, mTimeout
		lbMonitorForce = mForce
	}
}

// TestRunLBMissingCreds verifies every loadbalancer runner aborts while
// creating the service — before any network call is possible — when
// credentials are empty.
func TestRunLBMissingCreds(t *testing.T) {
	runGlobalsSnapshot(t)
	lbFlagSnapshot(t)

	for _, tc := range []string{
		"pool create", "pool list", "pool get", "pool update",
		"monitor create", "monitor list", "monitor get", "monitor update",
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			switch tc.name {
			case "pool create":
				lbPoolName = "primary"
				lbPoolOriginsJSON = `[{"name":"web-1","address":"10.0.0.1:80"}]`
				err = runLBPoolCreate(lbPoolCreateCmd, tc.args)
			case "pool list":
				err = runLBPoolList(lbPoolListCmd, tc.args)
			case "pool get":
				err = runLBPoolGet(lbPoolGetCmd, tc.args)
			case "pool update":
				err = runLBPoolUpdate(lbPoolUpdateCmd, tc.args)
			case "monitor create":
				lbMonitorType = "http"
				err = runLBMonitorCreate(lbMonitorCreateCmd, tc.args)
			case "monitor list":
				err = runLBMonitorList(lbMonitorListCmd, tc.args)
			case "monitor get":
				err = runLBMonitorGet(lbMonitorGetCmd, tc.args)
			case "monitor update":
				err = runLBMonitorUpdate(lbMonitorUpdateCmd, tc.args)
			}
			if err == nil || !strings.Contains(err.Error(), "failed to create load balancer service") {
				t.Fatalf("expected service error, got %v", err)
			}
			// reset per-subtest flags so the next runner starts clean
			lbResetFlags()
		})
	}
}

// TestRunLBPoolDelete_DryRunByDefault verifies the registry-flagged
// destructive pool delete runs dry without --force, even offline.
func TestRunLBPoolDelete_DryRunByDefault(t *testing.T) {
	runGlobalsSnapshot(t)
	lbFlagSnapshot(t)
	lbPoolForce = false

	r, restore := captureStdout(t)
	err := runLBPoolDelete(lbPoolDeleteCmd, []string{"pool-1"})
	out := readAll(t, r)
	restore()

	if err != nil {
		t.Fatalf("dry-run delete returned error: %v", err)
	}
	if !strings.Contains(out, "DRY RUN") || !strings.Contains(out, "pool-1") {
		t.Fatalf("dry-run output missing pool id, got: %q", out)
	}
}

// TestRunLBMonitorDelete_DryRunByDefault verifies the registry-flagged
// destructive monitor delete runs dry without --force, even offline.
func TestRunLBMonitorDelete_DryRunByDefault(t *testing.T) {
	runGlobalsSnapshot(t)
	lbFlagSnapshot(t)
	lbMonitorForce = false

	r, restore := captureStdout(t)
	err := runLBMonitorDelete(lbMonitorDeleteCmd, []string{"mon-1"})
	out := readAll(t, r)
	restore()

	if err != nil {
		t.Fatalf("dry-run delete returned error: %v", err)
	}
	if !strings.Contains(out, "DRY RUN") || !strings.Contains(out, "mon-1") {
		t.Fatalf("dry-run output missing monitor id, got: %q", out)
	}
}

// TestRunLBDeletes_ForceWithoutCredsFails verifies --force escapes the
// dry-run default and then fails on the missing credentials guard instead
// of silently pretending success.
func TestRunLBDeletes_ForceWithoutCredsFails(t *testing.T) {
	runGlobalsSnapshot(t)
	lbFlagSnapshot(t)
	lbPoolForce = true
	lbMonitorForce = true

	if err := runLBPoolDelete(lbPoolDeleteCmd, []string{"pool-1"}); err == nil || !strings.Contains(err.Error(), "failed to create load balancer service") {
		t.Fatalf("expected service error on forced pool delete, got %v", err)
	}
	if err := runLBMonitorDelete(lbMonitorDeleteCmd, []string{"mon-1"}); err == nil || !strings.Contains(err.Error(), "failed to create load balancer service") {
		t.Fatalf("expected service error on forced monitor delete, got %v", err)
	}
}

// TestRunLBPoolCreate_OriginJSONParseErrors verifies malformed --origins
// values fail with an agent-readable message before any service is built.
func TestRunLBPoolCreate_OriginJSONParseErrors(t *testing.T) {
	runGlobalsSnapshot(t)
	lbFlagSnapshot(t)

	cases := []struct {
		name    string
		origins string
		wantErr string
	}{
		{"missing flag", "", "--origins is required"},
		{"not json", "web-1:10.0.0.1", "not valid JSON"},
		{"not an array", `{"name":"web-1"}`, "not valid JSON"},
		{"empty array", `[]`, "at least one origin"},
		{"origin missing address", `[{"name":"web-1"}]`, "missing an address"},
		{"origin missing name", `[{"address":"10.0.0.1:80"}]`, "missing a name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lbPoolName = "primary"
			lbPoolOriginsJSON = tc.origins
			err := runLBPoolCreate(lbPoolCreateCmd, nil)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunLBPoolCreate_MissingName verifies pool create rejects a missing
// --name before any service is built.
func TestRunLBPoolCreate_MissingName(t *testing.T) {
	runGlobalsSnapshot(t)
	lbFlagSnapshot(t)

	err := runLBPoolCreate(lbPoolCreateCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("expected name error, got %v", err)
	}
}

// TestRunLBMonitorCreate_InvalidType verifies monitor create rejects a
// probe type outside http/https before any service is built.
func TestRunLBMonitorCreate_InvalidType(t *testing.T) {
	runGlobalsSnapshot(t)
	lbFlagSnapshot(t)
	lbMonitorType = "tcp"

	err := runLBMonitorCreate(lbMonitorCreateCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--type must be http or https") {
		t.Fatalf("expected type error, got %v", err)
	}
}

// TestRunLBPoolUpdate_ExclusiveFlags verifies pool update rejects
// --enabled together with --disable.
func TestRunLBPoolUpdate_ExclusiveFlags(t *testing.T) {
	runGlobalsSnapshot(t)
	lbFlagSnapshot(t)
	lbPoolEnabled = true
	lbPoolDisable = true

	err := runLBPoolUpdate(lbPoolUpdateCmd, []string{"pool-1"})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected exclusivity error, got %v", err)
	}
}

// TestRunLBDryRun_JSONEnvelopes verifies the --dry-run JSON envelope shapes
// for the create commands carry the standard success wrapper with dry_run
// set and the typed payload under data.
func TestRunLBDryRun_JSONEnvelopes(t *testing.T) {
	runGlobalsSnapshot(t)
	setOutputMode(t, true, true)
	lbFlagSnapshot(t)

	t.Run("pool create", func(t *testing.T) {
		lbPoolName = "primary"
		lbPoolOriginsJSON = `[{"name":"web-1","address":"10.0.0.1:80"},{"name":"web-2","address":"10.0.0.2:80","enabled":false}]`
		lbPoolMonitor = "mon-1"

		r, restore := captureStdout(t)
		err := runLBPoolCreate(lbPoolCreateCmd, nil)
		out := readAll(t, r)
		restore()

		if err != nil {
			t.Fatalf("dry-run pool create returned error: %v", err)
		}
		var env struct {
			Success bool `json:"success"`
			DryRun  bool `json:"dry_run"`
			Data    struct {
				Name    string `json:"name"`
				Monitor string `json:"monitor"`
				Origins []struct {
					Name    string `json:"name"`
					Address string `json:"address"`
					Enabled bool   `json:"enabled"`
				} `json:"origins"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("pool create envelope not JSON: %v\n%s", err, out)
		}
		if !env.Success || !env.DryRun {
			t.Errorf("expected success+dry_run envelope, got: %s", out)
		}
		if env.Data.Name != "primary" || env.Data.Monitor != "mon-1" {
			t.Errorf("pool create payload not mapped: %s", out)
		}
		if len(env.Data.Origins) != 2 || !env.Data.Origins[0].Enabled || env.Data.Origins[1].Enabled {
			t.Errorf("origins not mapped with omitted-enabled=true default: %s", out)
		}
		lbResetFlags()
	})

	t.Run("monitor create", func(t *testing.T) {
		lbMonitorType = "https"
		lbMonitorPath = "/healthz"
		lbMonitorExpectedCodes = "2xx"
		lbMonitorInterval = 30
		lbMonitorRetries = 3
		lbMonitorTimeout = 2

		r, restore := captureStdout(t)
		err := runLBMonitorCreate(lbMonitorCreateCmd, nil)
		out := readAll(t, r)
		restore()

		if err != nil {
			t.Fatalf("dry-run monitor create returned error: %v", err)
		}
		var env struct {
			Success bool `json:"success"`
			DryRun  bool `json:"dry_run"`
			Data    struct {
				Type     string `json:"type"`
				Path     string `json:"path"`
				Interval int    `json:"interval"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("monitor create envelope not JSON: %v\n%s", err, out)
		}
		if !env.Success || !env.DryRun {
			t.Errorf("expected success+dry_run envelope, got: %s", out)
		}
		if env.Data.Type != "https" || env.Data.Path != "/healthz" || env.Data.Interval != 30 {
			t.Errorf("monitor create payload not mapped: %s", out)
		}
		lbResetFlags()
	})
}

// TestRunLBPoolUpdate_OriginJSONParseError verifies update surfaces the
// same agent-readable --origins parse errors as create.
func TestRunLBPoolUpdate_OriginJSONParseError(t *testing.T) {
	runGlobalsSnapshot(t)
	lbFlagSnapshot(t)
	lbPoolOriginsJSON = "not json"

	err := runLBPoolUpdate(lbPoolUpdateCmd, []string{"pool-1"})
	if err == nil || !strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("expected origins parse error, got %v", err)
	}
}

// readAll drains the captured stdout pipe and returns it as a string
// (shared helper for the loadbalancer run tests).
func readAll(t *testing.T, r interface{ Read([]byte) (int, error) }) string {
	t.Helper()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(readerFunc{r}); err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}
	return buf.String()
}

type readerFunc struct {
	r interface{ Read([]byte) (int, error) }
}

func (rf readerFunc) Read(p []byte) (int, error) { return rf.r.Read(p) }
