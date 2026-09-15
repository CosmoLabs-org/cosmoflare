package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVectorizeCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "vectorize" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("vectorizeCmd not registered on rootCmd")
	}
}

func TestVectorizeCmd_Metadata(t *testing.T) {
	if vectorizeCmd.Use != "vectorize" {
		t.Errorf("vectorizeCmd.Use = %q, want %q", vectorizeCmd.Use, "vectorize")
	}
	if vectorizeCmd.Short == "" {
		t.Error("vectorizeCmd.Short is empty")
	}
}

func TestVectorizeCmd_Subcommands(t *testing.T) {
	expected := []string{"create", "list", "get", "delete", "insert", "query"}
	subs := vectorizeCmd.Commands()
	nameSet := make(map[string]bool)
	for _, s := range subs {
		nameSet[s.Use] = true
	}
	for _, name := range expected {
		found := false
		for _, s := range subs {
			if s.Use == name || len(s.Use) > len(name) && s.Use[:len(name)] == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q on vectorizeCmd", name)
		}
	}
}

func TestVectorizeCmd_AllRunE(t *testing.T) {
	for _, sub := range vectorizeCmd.Commands() {
		if sub.RunE == nil {
			t.Errorf("vectorize subcommand %q has nil RunE", sub.Use)
		}
	}
}

func TestVectorizeCreate_Flags(t *testing.T) {
	flags := []struct {
		name     string
		defValue string
	}{
		{"dimensions", "0"},
		{"metric", "cosine"},
	}
	var createCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Use == "create [name]" || sub.Name() == "create" {
			createCmd = sub
			break
		}
	}
	if createCmd == nil {
		t.Fatal("create subcommand not found")
	}
	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			flag := createCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Fatalf("flag --%s not found", f.name)
			}
			if flag.DefValue != f.defValue {
				t.Errorf("flag --%s default = %q, want %q", f.name, flag.DefValue, f.defValue)
			}
		})
	}
}

func TestVectorizeDelete_ForceFlag(t *testing.T) {
	var deleteCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "delete" {
			deleteCmd = sub
			break
		}
	}
	if deleteCmd == nil {
		t.Fatal("delete subcommand not found")
	}
	f := deleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not found on delete")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

func TestVectorizeQuery_Flags(t *testing.T) {
	var queryCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "query" {
			queryCmd = sub
			break
		}
	}
	if queryCmd == nil {
		t.Fatal("query subcommand not found")
	}
	for _, name := range []string{"top-k", "values"} {
		if queryCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not found on query", name)
		}
	}
}

func TestVectorizeCmd_LongDescription(t *testing.T) {
	if vectorizeCmd.Long == "" {
		t.Error("vectorizeCmd.Long description is empty")
	}
}

func TestVectorizeCmd_SubcommandCount(t *testing.T) {
	subs := vectorizeCmd.Commands()
	if len(subs) != 10 {
		t.Errorf("vectorizeCmd has %d subcommands, want 10 (create, list, get, delete, insert, query, upsert, get-vector, delete-vectors, namespaces)", len(subs))
	}
}

func TestVectorizeInsert_Flags(t *testing.T) {
	var insertCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "insert" {
			insertCmd = sub
			break
		}
	}
	if insertCmd == nil {
		t.Fatal("insert subcommand not found")
	}
	flags := []struct {
		name     string
		defValue string
	}{
		{"file", ""},
		{"id", ""},
		{"values", ""},
	}
	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			flag := insertCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Fatalf("flag --%s not found on insert", f.name)
			}
			if flag.DefValue != f.defValue {
				t.Errorf("flag --%s default = %q, want %q", f.name, flag.DefValue, f.defValue)
			}
		})
	}
}

func TestVectorizeCreate_ArgsValidation(t *testing.T) {
	var createCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "create" {
			createCmd = sub
			break
		}
	}
	if createCmd == nil {
		t.Fatal("create subcommand not found")
	}
	if createCmd.Args == nil {
		t.Fatal("create subcommand has nil Args validator")
	}
	// No args should fail (ExactArgs(1))
	if err := createCmd.Args(createCmd, []string{}); err == nil {
		t.Error("expected error with no args for vectorize create")
	}
	// One arg should succeed
	if err := createCmd.Args(createCmd, []string{"my-index"}); err != nil {
		t.Errorf("expected no error with one arg, got: %v", err)
	}
	// Two args should fail
	if err := createCmd.Args(createCmd, []string{"a", "b"}); err == nil {
		t.Error("expected error with two args for vectorize create")
	}
}

func TestVectorizeQuery_TopKDefault(t *testing.T) {
	var queryCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "query" {
			queryCmd = sub
			break
		}
	}
	if queryCmd == nil {
		t.Fatal("query subcommand not found")
	}
	f := queryCmd.Flags().Lookup("top-k")
	if f == nil {
		t.Fatal("--top-k flag not found")
	}
	if f.DefValue != "10" {
		t.Errorf("--top-k default = %q, want %q", f.DefValue, "10")
	}
}

func TestVectorizeGet_ArgsValidation(t *testing.T) {
	var getCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "get" {
			getCmd = sub
			break
		}
	}
	if getCmd == nil {
		t.Fatal("get subcommand not found")
	}
	if getCmd.Args == nil {
		t.Fatal("get subcommand has nil Args validator")
	}
	if err := getCmd.Args(getCmd, []string{}); err == nil {
		t.Error("expected error with no args for vectorize get")
	}
	if err := getCmd.Args(getCmd, []string{"idx"}); err != nil {
		t.Errorf("expected no error with one arg, got: %v", err)
	}
}

func TestVectorizeDelete_ArgsValidation(t *testing.T) {
	var delCmd *cobra.Command
	for _, sub := range vectorizeCmd.Commands() {
		if sub.Name() == "delete" {
			delCmd = sub
			break
		}
	}
	if delCmd == nil {
		t.Fatal("delete subcommand not found")
	}
	if delCmd.Args == nil {
		t.Fatal("delete subcommand has nil Args validator")
	}
	if err := delCmd.Args(delCmd, []string{}); err == nil {
		t.Error("expected error with no args for vectorize delete")
	}
	if err := delCmd.Args(delCmd, []string{"idx"}); err != nil {
		t.Errorf("expected no error with one arg, got: %v", err)
	}
	if err := delCmd.Args(delCmd, []string{"a", "b"}); err == nil {
		t.Error("expected error with two args for vectorize delete")
	}
}

// vectorizeRunGlobals snapshots and restores the package globals the
// vectorize runners read so tests cannot leak state between each other.
func vectorizeRunGlobals(t *testing.T) {
	t.Helper()
	oldDims, oldMetric, oldForce := vectorizeDimensions, vectorizeMetric, vectorizeForce
	oldFile, oldID, oldVals := vectorizeFile, vectorizeID, vectorizeValues
	oldTopK, oldUpsert := vectorizeTopK, vectorizeUpsertFile
	oldGetID, oldDelIDs := vectorizeGetVectorID, vectorizeDeleteIDsCSV
	oldAccount, oldToken := AccountID, APIToken
	oldDry, oldJSON := DryRun, JSONOutput
	t.Cleanup(func() {
		vectorizeDimensions, vectorizeMetric, vectorizeForce = oldDims, oldMetric, oldForce
		vectorizeFile, vectorizeID, vectorizeValues = oldFile, oldID, oldVals
		vectorizeTopK, vectorizeUpsertFile = oldTopK, oldUpsert
		vectorizeGetVectorID, vectorizeDeleteIDsCSV = oldGetID, oldDelIDs
		AccountID, APIToken = oldAccount, oldToken
		DryRun, JSONOutput = oldDry, oldJSON
	})
}

// vectorizeResetFlags clears the flag-bound globals to their zero/default
// values so each subtest starts pristine.
func vectorizeResetFlags() {
	vectorizeDimensions = 0
	vectorizeMetric = "cosine"
	vectorizeForce = false
	vectorizeFile = ""
	vectorizeID = ""
	vectorizeValues = ""
	vectorizeTopK = 10
	vectorizeUpsertFile = ""
	vectorizeGetVectorID = ""
	vectorizeDeleteIDsCSV = ""
}

// vectorizeWriteVectors writes content to a temp file and returns its path.
func vectorizeWriteVectors(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// vectorizeWantErr asserts the runner returns an error containing want.
func vectorizeWantErr(t *testing.T, name, want string, run func() error) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		err := run()
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error containing %q, got %v", want, err)
		}
	})
}

// TestRunVectorizeCreate_DryRun verifies the create dry-run branch reports
// the would-be index creation in both human and JSON output modes.
func TestRunVectorizeCreate_DryRun(t *testing.T) {
	for _, tc := range []struct {
		name string
		json bool
	}{
		{"human", false},
		{"json", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vectorizeRunGlobals(t)
			vectorizeResetFlags()
			vectorizeDimensions = 768
			vectorizeMetric = "dot-product"
			DryRun = true
			JSONOutput = tc.json
			APIToken = "fake-token-1234567890"
			AccountID = "acct123"

			if err := runVectorizeCreate(vectorizeCreateCmd, []string{"my-index"}); err != nil {
				t.Fatalf("dry-run create should succeed: %v", err)
			}
		})
	}
}

// TestRunVectorizeCreate_MissingCreds verifies the service-construction
// error path when account ID or API token is absent.
func TestRunVectorizeCreate_MissingCreds(t *testing.T) {
	vectorizeRunGlobals(t)
	vectorizeResetFlags()
	DryRun = false
	cases := []struct {
		name      string
		accountID string
		token     string
	}{
		{"no-account", "", "fake-token-1234567890"},
		{"no-token", "acct123", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			AccountID, APIToken = tc.accountID, tc.token
			err := runVectorizeCreate(vectorizeCreateCmd, []string{"my-index"})
			if err == nil || !strings.Contains(err.Error(), "failed to create vectorize service") {
				t.Fatalf("expected service error, got %v", err)
			}
		})
	}
}

// TestRunVectorizeReadCmds_MissingCreds verifies the list, get and namespaces
// runners surface the service-construction error before any request.
func TestRunVectorizeReadCmds_MissingCreds(t *testing.T) {
	vectorizeRunGlobals(t)
	vectorizeResetFlags()
	DryRun = false
	AccountID, APIToken = "", ""
	cases := []struct {
		name string
		run  func(cmd *cobra.Command, args []string) error
		cmd  *cobra.Command
	}{
		{"list", runVectorizeList, vectorizeListCmd},
		{"get", runVectorizeGet, vectorizeGetCmd},
		{"namespaces", runVectorizeNamespaces, vectorizeNamespacesCmd},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(tc.cmd, []string{"my-index"})
			if err == nil || !strings.Contains(err.Error(), "failed to create vectorize service") {
				t.Fatalf("expected service error, got %v", err)
			}
		})
	}
}

// TestRunVectorizeDelete_DryRun verifies the delete dry-run branch succeeds
// in both output modes.
func TestRunVectorizeDelete_DryRun(t *testing.T) {
	for _, tc := range []struct {
		name string
		json bool
	}{
		{"human", false},
		{"json", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vectorizeRunGlobals(t)
			vectorizeResetFlags()
			DryRun = true
			JSONOutput = tc.json

			if err := runVectorizeDelete(vectorizeDeleteCmd, []string{"my-index"}); err != nil {
				t.Fatalf("dry-run delete should succeed: %v", err)
			}
		})
	}
}

// TestRunVectorizeDelete_MissingCreds verifies the delete service error path.
func TestRunVectorizeDelete_MissingCreds(t *testing.T) {
	vectorizeRunGlobals(t)
	vectorizeResetFlags()
	DryRun = false
	AccountID, APIToken = "", ""

	err := runVectorizeDelete(vectorizeDeleteCmd, []string{"my-index"})
	if err == nil || !strings.Contains(err.Error(), "failed to create vectorize service") {
		t.Fatalf("expected service error, got %v", err)
	}
}

// TestRunVectorizeInsert_InputValidation verifies each malformed input mode
// is rejected before any network call.
func TestRunVectorizeInsert_InputValidation(t *testing.T) {
	vectorizeRunGlobals(t)
	vectorizeResetFlags()
	DryRun = true
	AccountID, APIToken = "acct123", "fake-token-1234567890"

	vectorizeWantErr(t, "no-input", "provide --file or both --id and --values", func() error {
		return runVectorizeInsert(vectorizeInsertCmd, []string{"my-index"})
	})
	vectorizeWantErr(t, "id-without-values", "provide --file or both --id and --values", func() error {
		vectorizeResetFlags()
		vectorizeID = "vec-1"
		return runVectorizeInsert(vectorizeInsertCmd, []string{"my-index"})
	})
	vectorizeWantErr(t, "invalid-values", "invalid --values", func() error {
		vectorizeResetFlags()
		vectorizeID, vectorizeValues = "vec-1", "0.1,nope"
		return runVectorizeInsert(vectorizeInsertCmd, []string{"my-index"})
	})
	vectorizeWantErr(t, "missing-file", "failed to read file", func() error {
		vectorizeResetFlags()
		vectorizeFile = filepath.Join(t.TempDir(), "does-not-exist.ndjson")
		return runVectorizeInsert(vectorizeInsertCmd, []string{"my-index"})
	})
	vectorizeWantErr(t, "bad-ndjson-line", "failed to parse vector line", func() error {
		vectorizeResetFlags()
		vectorizeFile = vectorizeWriteVectors(t, "bad.ndjson", "{\"id\":\"v1\"")
		return runVectorizeInsert(vectorizeInsertCmd, []string{"my-index"})
	})
}

// TestRunVectorizeInsert_DryRunSuccess verifies the dry-run branch succeeds
// for both the NDJSON file mode and the inline --id/--values mode.
func TestRunVectorizeInsert_DryRunSuccess(t *testing.T) {
	for _, tc := range []struct {
		name  string
		json  bool
		setup func(t *testing.T)
	}{
		{"file", false, func(t *testing.T) {
			vectorizeFile = vectorizeWriteVectors(t, "vectors.ndjson",
				"{\"id\":\"v1\",\"values\":[0.1,0.2]}\n\n{\"id\":\"v2\",\"values\":[0.3]}")
		}},
		{"inline-json", true, func(t *testing.T) {
			vectorizeID, vectorizeValues = "vec-1", "0.1, 0.2 ,0.3"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vectorizeRunGlobals(t)
			vectorizeResetFlags()
			DryRun = true
			JSONOutput = tc.json
			AccountID, APIToken = "acct123", "fake-token-1234567890"
			tc.setup(t)

			if err := runVectorizeInsert(vectorizeInsertCmd, []string{"my-index"}); err != nil {
				t.Fatalf("dry-run insert should succeed: %v", err)
			}
		})
	}
}

// TestRunVectorizeQuery_InputValidation verifies the query runner validates
// --values before constructing the service or issuing a request.
func TestRunVectorizeQuery_InputValidation(t *testing.T) {
	vectorizeRunGlobals(t)
	vectorizeResetFlags()
	AccountID, APIToken = "", ""

	vectorizeWantErr(t, "missing-values", "--values is required for query", func() error {
		vectorizeResetFlags()
		return runVectorizeQuery(vectorizeQueryCmd, []string{"my-index"})
	})
	vectorizeWantErr(t, "invalid-values", "invalid --values", func() error {
		vectorizeResetFlags()
		vectorizeValues = "1.0,abc"
		return runVectorizeQuery(vectorizeQueryCmd, []string{"my-index"})
	})
	vectorizeWantErr(t, "missing-creds", "failed to create vectorize service", func() error {
		vectorizeResetFlags()
		vectorizeValues = "0.1,0.2"
		return runVectorizeQuery(vectorizeQueryCmd, []string{"my-index"})
	})
}

// TestRunVectorizeUpsert_InputValidation verifies the upsert runner rejects
// a missing --file, an unreadable file and a malformed JSON array.
func TestRunVectorizeUpsert_InputValidation(t *testing.T) {
	vectorizeRunGlobals(t)
	vectorizeResetFlags()
	DryRun = true

	vectorizeWantErr(t, "missing-file-flag", "--file is required", func() error {
		vectorizeResetFlags()
		return runVectorizeUpsert(vectorizeUpsertCmd, []string{"my-index"})
	})
	vectorizeWantErr(t, "unreadable-file", "failed to read file", func() error {
		vectorizeResetFlags()
		vectorizeUpsertFile = filepath.Join(t.TempDir(), "missing.json")
		return runVectorizeUpsert(vectorizeUpsertCmd, []string{"my-index"})
	})
	vectorizeWantErr(t, "bad-json", "failed to parse vectors file", func() error {
		vectorizeResetFlags()
		vectorizeUpsertFile = vectorizeWriteVectors(t, "bad.json", "{\"not\":\"an array\"}")
		return runVectorizeUpsert(vectorizeUpsertCmd, []string{"my-index"})
	})
}

// TestRunVectorizeUpsert_DryRun verifies the dry-run branch succeeds for a
// valid JSON array in both output modes (no service is constructed).
func TestRunVectorizeUpsert_DryRun(t *testing.T) {
	for _, tc := range []struct {
		name string
		json bool
	}{
		{"human", false},
		{"json", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vectorizeRunGlobals(t)
			vectorizeResetFlags()
			DryRun = true
			JSONOutput = tc.json
			vectorizeUpsertFile = vectorizeWriteVectors(t, "vectors.json",
				`[{"id":"v1","values":[0.1,0.2]}]`)

			if err := runVectorizeUpsert(vectorizeUpsertCmd, []string{"my-index"}); err != nil {
				t.Fatalf("dry-run upsert should succeed: %v", err)
			}
		})
	}
}

// TestRunVectorizeGetVector_Validation verifies the --id requirement and the
// service-construction error path.
func TestRunVectorizeGetVector_Validation(t *testing.T) {
	vectorizeRunGlobals(t)
	vectorizeResetFlags()
	DryRun = false
	AccountID, APIToken = "acct123", ""

	vectorizeWantErr(t, "missing-id", "--id is required", func() error {
		vectorizeResetFlags()
		return runVectorizeGetVector(vectorizeGetVectorCmd, []string{"my-index"})
	})
	vectorizeWantErr(t, "missing-creds", "failed to create vectorize service", func() error {
		vectorizeResetFlags()
		vectorizeGetVectorID = "vec-1"
		return runVectorizeGetVector(vectorizeGetVectorCmd, []string{"my-index"})
	})
}

// TestRunVectorizeDeleteVectors_DryRun verifies the --ids requirement and the
// dry-run branch that trims whitespace around comma-separated IDs.
func TestRunVectorizeDeleteVectors_DryRun(t *testing.T) {
	vectorizeRunGlobals(t)
	vectorizeResetFlags()
	DryRun = true

	vectorizeWantErr(t, "missing-ids", "--ids is required", func() error {
		vectorizeResetFlags()
		return runVectorizeDeleteVectors(vectorizeDeleteVectorsCmd, []string{"my-index"})
	})

	for _, tc := range []struct {
		name string
		json bool
	}{
		{"human", false},
		{"json", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vectorizeResetFlags()
			JSONOutput = tc.json
			vectorizeDeleteIDsCSV = " vec-1 , vec-2 "

			if err := runVectorizeDeleteVectors(vectorizeDeleteVectorsCmd, []string{"my-index"}); err != nil {
				t.Fatalf("dry-run delete-vectors should succeed: %v", err)
			}
		})
	}
}

// TestParseFloatSlice verifies exact parsing behavior: values, whitespace
// tolerance, and error text for invalid tokens.
func TestParseFloatSlice(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    []float64
		wantErr string
	}{
		{"single", "0.5", []float64{0.5}, ""},
		{"multiple", "0.1,0.2,0.3", []float64{0.1, 0.2, 0.3}, ""},
		{"whitespace", " 1 , -2 , 3e2 ", []float64{1, -2, 300}, ""},
		{"invalid-token", "0.1,abc", nil, `invalid float "abc"`},
		{"empty-string", "", nil, `invalid float ""`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseFloatSlice(tc.in)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}
