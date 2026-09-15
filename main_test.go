/*
Unit tests for the cosmoflare entry point (main)

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT (https://opensource.org/licenses/MIT)

main() terminates the process via os.Exit(1) when cmd.Execute() fails, so it
cannot be called in-process without killing the test run. These tests use the
standard re-exec pattern: TestMain detects a sentinel environment variable in
the child process and routes control into main() with a caller-supplied argv,
while the parent test asserts on the child's exit code and captured output.
*/

package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const (
	// mainTestArgvEnv hands the argv under test from the parent test process
	// to the re-exec'd child. Arguments are joined with the ASCII unit
	// separator so arguments containing spaces survive the round trip.
	mainTestArgvEnv = "COSMOFLARE_MAIN_TEST_ARGV"

	// mainTestArgvSep separates the argv entries carried in mainTestArgvEnv.
	mainTestArgvSep = "\x1f"
)

// credentialEnvVars lists the environment variables that satisfy the root
// command's credential checks. Subtests that exercise the "missing
// credentials" error paths remove them from the child environment.
var credentialEnvVars = []string{
	"CLOUDFLARE_API_TOKEN",
	"CLOUDFLARE_ACCOUNT_ID",
}

// TestMain either runs the package's tests (parent process) or, when the
// sentinel variable mainTestArgvEnv is set, acts as a stand-in for the real
// binary: it replaces os.Args and calls main(). main() itself calls
// os.Exit(1) on failure; if it returns normally the child exits 0.
func TestMain(m *testing.M) {
	if raw, ok := os.LookupEnv(mainTestArgvEnv); ok {
		os.Args = strings.Split(raw, mainTestArgvSep)
		main() // may os.Exit(1)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// runMainInSubprocess re-executes the test binary with argv applied to main()
// and returns the resulting exit code, stdout and stderr. When dropCreds is
// true the Cloudflare credential variables are stripped from the child
// environment; extraEnv entries are appended after that filtering so callers
// can inject specific credential values.
func runMainInSubprocess(t *testing.T, argv []string, dropCreds bool, extraEnv []string) (int, string, string) {
	t.Helper()

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test binary: %v", err)
	}

	// "-test.run=^$" matches no tests, so the child performs no work besides
	// the TestMain branch above.
	cmd := exec.Command(exe, "-test.run=^$")

	env := os.Environ()
	if dropCreds {
		filtered := env[:0:0]
		for _, kv := range env {
			drop := false
			for _, cred := range credentialEnvVars {
				if strings.HasPrefix(kv, cred+"=") {
					drop = true
					break
				}
			}
			if !drop {
				filtered = append(filtered, kv)
			}
		}
		env = filtered
	}
	cmd.Env = append(env, extraEnv...)
	cmd.Env = append(cmd.Env, mainTestArgvEnv+"="+strings.Join(argv, mainTestArgvSep))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("re-exec main(): %v (stderr: %q)", err, stderr.String())
		}
		return exitErr.ExitCode(), stdout.String(), stderr.String()
	}
	return 0, stdout.String(), stderr.String()
}

// TestMainFunction validates main() end-to-end: successful invocations (help,
// version, bare CLI) must exit 0 and print to stdout, while failures (unknown
// command, unknown flag, missing API token, missing account ID) must exit 1.
// Success here means cmd.Execute() returned nil and main() returned normally;
// failure paths cover both the error branch of main() and the credential
// guards enforced before command execution.
// mainFunctionCase describes one scenario exercised by TestMainFunction:
// the argv to run main() with in a subprocess, optional credential
// stripping/injection, and the expected exit code and output fragments.
type mainFunctionCase struct {
	name             string
	argv             []string
	dropCreds        bool
	extraEnv         []string
	wantExit         int
	wantStdout       string
	wantStderr       string
	minStderrRepeats map[string]int
}

// mainFunctionCases returns the scenario table for TestMainFunction: the
// version/help/no-arg banners, unknown command and flag errors, and the
// missing-credential guards.
func mainFunctionCases() []mainFunctionCase {
	const unknownCommand = "definitely-not-a-real-cosmoflare-command"

	return []mainFunctionCase{
		{
			name:       "version flag prints banner and exits zero",
			argv:       []string{"cosmoflare", "--version"},
			wantExit:   0,
			wantStdout: "Cosmoflare version",
		},
		{
			name:       "help flag prints usage and exits zero",
			argv:       []string{"cosmoflare", "--help"},
			wantExit:   0,
			wantStdout: "Usage:",
		},
		{
			name:       "no arguments prints help and exits zero",
			argv:       []string{"cosmoflare"},
			wantExit:   0,
			wantStdout: "Cosmoflare manages the full Cloudflare developer platform",
		},
		{
			name:     "unknown command exits one and prints error on stderr",
			argv:     []string{"cosmoflare", unknownCommand},
			wantExit: 1,
			// Cobra prints the error once and main() prints it a second time
			// via its "Error: %v" branch, so the message must appear twice.
			wantStderr:       `unknown command "` + unknownCommand + `" for "cosmoflare"`,
			minStderrRepeats: map[string]int{`unknown command "` + unknownCommand + `"`: 2},
		},
		{
			name:       "unknown flag exits one and prints error on stderr",
			argv:       []string{"cosmoflare", "--definitely-not-a-flag"},
			wantExit:   1,
			wantStderr: "Error: unknown flag: --definitely-not-a-flag",
		},
		{
			name:       "command without API token exits one",
			argv:       []string{"cosmoflare", "bucket", "list"},
			dropCreds:  true,
			wantExit:   1,
			wantStdout: "Cloudflare API token is required",
		},
		{
			name:      "command with token but no account ID exits one",
			argv:      []string{"cosmoflare", "bucket", "list"},
			dropCreds: true,
			extraEnv: []string{
				// Long enough to pass the basic token length check so the
				// failure comes from the missing account ID instead.
				"CLOUDFLARE_API_TOKEN=test-token-0123456789",
			},
			wantExit:   1,
			wantStdout: "Cloudflare Account ID is required",
		},
	}
}

func TestMainFunction(t *testing.T) {
	for _, tt := range mainFunctionCases() {
		t.Run(tt.name, func(t *testing.T) {
			exitCode, stdout, stderr := runMainInSubprocess(t, tt.argv, tt.dropCreds, tt.extraEnv)

			if exitCode != tt.wantExit {
				t.Errorf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s",
					exitCode, tt.wantExit, stdout, stderr)
			}
			if tt.wantStdout != "" && !strings.Contains(stdout, tt.wantStdout) {
				t.Errorf("stdout does not contain %q; got:\n%s", tt.wantStdout, stdout)
			}
			if tt.wantStderr != "" && !strings.Contains(stderr, tt.wantStderr) {
				t.Errorf("stderr does not contain %q; got:\n%s", tt.wantStderr, stderr)
			}
			for needle, want := range tt.minStderrRepeats {
				if got := strings.Count(stderr, needle); got < want {
					t.Errorf("stderr contains %q %d times, want at least %d (cobra prints it once and main() prints it again); got:\n%s",
						needle, got, want, stderr)
				}
			}
		})
	}
}

// TestMainVersionIsDeterministic validates that the version banner written by
// the success path of main() always starts with the expected prefix and ends
// with a newline, regardless of the value injected at build time.
func TestMainVersionIsDeterministic(t *testing.T) {
	exitCode, stdout, stderr := runMainInSubprocess(t, []string{"cosmoflare", "--version"}, false, nil)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr:\n%s)", exitCode, stderr)
	}
	if !strings.HasPrefix(stdout, "Cosmoflare version ") {
		t.Errorf("stdout = %q, want prefix %q", stdout, "Cosmoflare version ")
	}
	if !strings.HasSuffix(stdout, "\n") {
		t.Errorf("stdout = %q, want trailing newline", stdout)
	}
}
