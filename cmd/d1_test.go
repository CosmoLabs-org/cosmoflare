package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestD1Cmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "d1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("d1Cmd not registered on rootCmd")
	}
}

func TestD1Cmd_Metadata(t *testing.T) {
	if d1Cmd.Use != "d1" {
		t.Errorf("d1Cmd.Use = %q, want %q", d1Cmd.Use, "d1")
	}
	if d1Cmd.Short == "" {
		t.Error("d1Cmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestD1Cmd_Subcommands(t *testing.T) {
	expected := []string{"create", "list", "get", "delete", "query"}
	for _, name := range expected {
		found := false
		for _, sub := range d1Cmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("d1 subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestD1Cmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		d1CreateCmd,
		d1ListCmd,
		d1GetCmd,
		d1DeleteCmd,
		d1QueryCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestD1Delete_ForceFlag(t *testing.T) {
	f := d1DeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on d1DeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

func TestD1Query_Flags(t *testing.T) {
	expected := []string{"sql", "param"}
	for _, name := range expected {
		if d1QueryCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on d1QueryCmd", name)
		}
	}
}

func TestD1Query_SQLRequired(t *testing.T) {
	f := d1QueryCmd.Flags().Lookup("sql")
	if f == nil {
		t.Fatal("--sql flag not found on d1QueryCmd")
	}
}

// --- Arg validation via cobra.ExactArgs ---

func TestD1Create_Args(t *testing.T) {
	if d1CreateCmd.Args == nil {
		t.Error("d1CreateCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestD1Get_Args(t *testing.T) {
	if d1GetCmd.Args == nil {
		t.Error("d1GetCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestD1Delete_Args(t *testing.T) {
	if d1DeleteCmd.Args == nil {
		t.Error("d1DeleteCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

func TestD1Query_Args(t *testing.T) {
	if d1QueryCmd.Args == nil {
		t.Error("d1QueryCmd.Args is nil, expected cobra.ExactArgs(1)")
	}
}

// --- DryRun mode ---

func TestD1Create_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"

	err := runD1Create(d1CreateCmd, []string{"my-database"})
	if err != nil {
		t.Errorf("runD1Create(DryRun) returned error: %v", err)
	}
}

func TestD1Create_DryRunJSON(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = true
	AccountID = "test-account"
	APIToken = "test-token"

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runD1Create(d1CreateCmd, []string{"my-database"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runD1Create(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

func TestD1Delete_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"

	err := runD1Delete(d1DeleteCmd, []string{"480f4f69-1a28-4fdd-9240-1ed29f0ac1df"})
	if err != nil {
		t.Errorf("runD1Delete(DryRun) returned error: %v", err)
	}
}

func TestD1Query_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	d1SQL = "SELECT 1"

	err := runD1Query(d1QueryCmd, []string{"480f4f69-1a28-4fdd-9240-1ed29f0ac1df"})
	if err != nil {
		t.Errorf("runD1Query(DryRun) returned error: %v", err)
	}
}

// --- D1 query validation ---

func TestD1Query_NoSQL(t *testing.T) {
	d1SQL = ""
	err := runD1Query(d1QueryCmd, []string{"db-123"})
	if err == nil {
		t.Fatal("expected error when --sql flag is empty")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("sql")) {
		t.Errorf("error = %q, want it to mention 'sql'", err.Error())
	}
}

// --- formatBytes helper ---

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
	}
	for _, tc := range cases {
		got := formatBytes(tc.input)
		if got != tc.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- FEAT-058 wave 2: stable error codes on d1 failure paths ---

// TestD1Runners_ErrorCodes asserts that the d1 runners' main failure paths
// (missing args/flags, missing token, missing account ID) surface stable
// machine-readable error codes in the --json error envelope (FEAT-058
// wave 2).
func TestD1Runners_ErrorCodes(t *testing.T) {
	savedJSON, savedDry := JSONOutput, DryRun
	savedToken, savedAcct := APIToken, AccountID
	savedSQL, savedLocal, savedRemote := d1SQL, d1Local, d1Remote
	defer func() {
		JSONOutput, DryRun = savedJSON, savedDry
		APIToken, AccountID = savedToken, savedAcct
		d1SQL, d1Local, d1Remote = savedSQL, savedLocal, savedRemote
	}()
	JSONOutput = true
	DryRun = false
	d1Local, d1Remote = false, false

	cases := []struct {
		name      string
		apiToken  string
		accountID string
		sql       string
		local     bool
		remote    bool
		run       func() error
		wantCode  string
	}{
		{
			name:      "query missing --sql",
			apiToken:  "test-token",
			accountID: "acct123",
			run:       func() error { return runD1Query(d1QueryCmd, []string{"db123"}) },
			wantCode:  "VALIDATION_ERROR",
		},
		{
			name:      "query local and remote conflict",
			apiToken:  "test-token",
			accountID: "acct123",
			sql:       "SELECT 1",
			local:     true,
			remote:    true,
			run:       func() error { return runD1Query(d1QueryCmd, []string{"db123"}) },
			wantCode:  "VALIDATION_ERROR",
		},
		{
			name:      "list missing token",
			apiToken:  "",
			accountID: "acct123",
			run:       func() error { return runD1List(d1ListCmd, nil) },
			wantCode:  "AUTH_MISSING_TOKEN",
		},
		{
			name:      "list missing account id",
			apiToken:  "test-token",
			accountID: "",
			run:       func() error { return runD1List(d1ListCmd, nil) },
			wantCode:  "CONFIG_MISSING",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			APIToken, AccountID = tc.apiToken, tc.accountID
			d1SQL, d1Local, d1Remote = tc.sql, tc.local, tc.remote

			r, restore := captureStdout(t)
			err := tc.run()
			restore()

			out, readErr := io.ReadAll(r)
			if readErr != nil {
				t.Fatalf("reading captured stdout failed: %v", readErr)
			}
			if err == nil {
				t.Fatalf("expected error, got nil; stdout: %q", out)
			}
			if code := codedErrorCode(t, out); code != tc.wantCode {
				t.Errorf("error_code = %q, want %q", code, tc.wantCode)
			}
		})
	}
}
