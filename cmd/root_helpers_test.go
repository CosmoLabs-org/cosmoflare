package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// printExitHelperEnv is the marker environment variable that re-runs the
// printErrorAndExit child process inside a test subprocess.
const printExitHelperEnv = "COSMOFLARE_TEST_PRINT_ERROR_AND_EXIT"

// setOutputMode snapshots the global output-mode flags and restores them on
// cleanup so a test cannot leak JSONOutput/DryRun state into other tests.
func setOutputMode(t *testing.T, jsonMode, dryRun bool) {
	t.Helper()
	savedJSON, savedDry := JSONOutput, DryRun
	JSONOutput, DryRun = jsonMode, dryRun
	t.Cleanup(func() { JSONOutput, DryRun = savedJSON, savedDry })
}

// TestPrintHelpers_TextModePrefixes verifies that the print helpers render
// their message with the expected status prefix in plain (non-JSON) mode.
func TestPrintHelpers_TextModePrefixes(t *testing.T) {
	setOutputMode(t, false, false)

	cases := []struct {
		name string
		fn   func(format string, args ...interface{})
		want string
	}{
		{"success uses checkmark", printSuccess, "✅ ok: file written"},
		{"warning uses warning sign", printWarning, "⚠️  careful now"},
		{"error uses cross mark", printError, "❌ something broke"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := capturePrint(t, func() { tc.fn("%s", tc.want) })
			if !strings.Contains(out, tc.want) {
				t.Errorf("output %q missing message %q", out, tc.want)
			}
		})
	}
}

// TestPrintHelpers_JSONModeSuppressed verifies the print helpers stay silent
// in --json mode so machine-readable output is not polluted.
func TestPrintHelpers_JSONModeSuppressed(t *testing.T) {
	setOutputMode(t, true, false)

	for _, fn := range []func(format string, args ...interface{}){printSuccess, printWarning, printError} {
		out := capturePrint(t, func() { fn("should not appear %d", 42) })
		if strings.TrimSpace(out) != "" {
			t.Errorf("JSON mode printed human output: %q", out)
		}
	}
}

// TestPrintHelpers_FormatArguments verifies printf-style formatting is
// applied to the message before printing.
func TestPrintHelpers_FormatArguments(t *testing.T) {
	setOutputMode(t, false, false)

	out := capturePrint(t, func() { printSuccess("created %s (%d bytes)", "a.txt", 12) })
	if !strings.Contains(out, "created a.txt (12 bytes)") {
		t.Errorf("formatted message wrong: %q", out)
	}
}

// TestGetRelativePath_RelativeToCWD verifies paths under the working
// directory are shortened for display, and that the working directory itself
// collapses to ".".
func TestGetRelativePath_RelativeToCWD(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	cases := []struct {
		name string
		path string
		want string
	}{
		{"file under cwd", filepath.Join(wd, "file.txt"), "file.txt"},
		{"nested file under cwd", filepath.Join(wd, "sub", "dir", "f.txt"), filepath.Join("sub", "dir", "f.txt")},
		{"cwd itself", wd, "."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := getRelativePath(tc.path); got != tc.want {
				t.Errorf("getRelativePath(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

// TestGetRelativePath_Fallthrough verifies inputs that cannot be made
// relative (relative paths against an absolute cwd) pass through unchanged.
func TestGetRelativePath_Fallthrough(t *testing.T) {
	if got := getRelativePath("bare/relative.txt"); got != "bare/relative.txt" {
		t.Errorf("relative input should pass through, got %q", got)
	}
	if got := getRelativePath(""); got != "" {
		t.Errorf("empty input should return empty, got %q", got)
	}
}

// TestExecute_NoArgsPrintsHelp verifies Execute on the bare root command
// renders the help text and returns nil (the cobra flag.ErrHelp contract).
func TestExecute_NoArgsPrintsHelp(t *testing.T) {
	savedArgs := os.Args
	savedOut, savedErr := rootCmd.OutOrStdout(), rootCmd.ErrOrStderr()
	rootCmd.SetArgs([]string{})
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(savedOut)
		rootCmd.SetErr(savedErr)
		os.Args = savedArgs
	})

	out := capturePrint(t, func() {
		// Hermetic: an earlier test may have redirected rootCmd's writers
		// to its own buffer — rebind to the (captured) stdout for this run.
		rootCmd.SetOut(os.Stdout)
		rootCmd.SetErr(os.Stderr)
		if err := Execute(); err != nil {
			t.Errorf("Execute with no args should print help and return nil, got %v", err)
		}
	})
	if !strings.Contains(out, "Usage:") && !strings.Contains(out, "Available Commands") {
		t.Errorf("expected help output, got %q", out)
	}
}

// TestExecute_UnknownCommandErrors verifies Execute surfaces an error for an
// unrecognized subcommand instead of silently succeeding.
func TestExecute_UnknownCommandErrors(t *testing.T) {
	rootCmd.SetArgs([]string{"definitely-not-a-real-command"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := Execute(); err == nil {
		t.Fatal("Execute with unknown command should return an error")
	} else if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected unknown-command error, got %v", err)
	}
}

// TestPrintErrorAndExit_ExitsWithStatusOne verifies printErrorAndExit
// terminates the process with exit code 1. The call is made in a subprocess
// so os.Exit does not kill the test runner.
func TestPrintErrorAndExit_ExitsWithStatusOne(t *testing.T) {
	if os.Getenv(printExitHelperEnv) == "text" {
		printErrorAndExit(errors.New("disk full"), "upload failed")
		return // unreachable on success; exit code would be 0 and the parent fails
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestPrintErrorAndExit_ExitsWithStatusOne$")
	cmd.Env = append(os.Environ(), printExitHelperEnv+"=text")
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit from printErrorAndExit, got err=%v (output: %q)", err, out)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
	}
}

// TestPrintErrorAndExit_JSONModeEmitsEnvelope verifies the JSON-mode branch
// of printErrorAndExit prints a parseable error envelope on stdout before
// exiting 1.
func TestPrintErrorAndExit_JSONModeEmitsEnvelope(t *testing.T) {
	if os.Getenv(printExitHelperEnv) == "json" {
		JSONOutput = true
		printErrorAndExit(errors.New("disk full"), "upload failed")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestPrintErrorAndExit_JSONModeEmitsEnvelope$")
	cmd.Env = append(os.Environ(), printExitHelperEnv+"=json")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit, got err=%v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
	}

	var resp OutputResponse
	if jerr := json.Unmarshal(stdout.Bytes(), &resp); jerr != nil {
		t.Fatalf("child stdout is not a JSON envelope: %v; raw: %q", jerr, stdout.String())
	}
	if resp.Success {
		t.Error("envelope success = true, want false")
	}
	if !strings.Contains(resp.Error, "upload failed") {
		t.Errorf("envelope error = %q, want it to contain the context", resp.Error)
	}
}
