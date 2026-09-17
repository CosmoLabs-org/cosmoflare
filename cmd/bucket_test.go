package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestBucketCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "bucket" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("bucketCmd not registered on rootCmd")
	}
}

func TestBucketCmd_Metadata(t *testing.T) {
	if bucketCmd.Use != "bucket" {
		t.Errorf("bucketCmd.Use = %q, want %q", bucketCmd.Use, "bucket")
	}
	if bucketCmd.Short == "" {
		t.Error("bucketCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestBucketCmd_Subcommands(t *testing.T) {
	expected := []string{"create", "list", "get", "update", "delete", "exists", "import"}
	for _, name := range expected {
		found := false
		for _, sub := range bucketCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("bucket subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestBucketCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		bucketCreateCmd,
		bucketListCmd,
		bucketGetCmd,
		bucketUpdateCmd,
		bucketDeleteCmd,
		bucketExistsCmd,
		bucketImportCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestBucketCreate_Flags(t *testing.T) {
	expected := []string{"location", "tags", "metadata"}
	for _, name := range expected {
		if bucketCreateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on bucketCreateCmd", name)
		}
	}
}

func TestBucketCreate_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"location", "auto"},
	}
	for _, tc := range cases {
		f := bucketCreateCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestBucketList_Flags(t *testing.T) {
	expected := []string{"format", "prefix", "tag"}
	for _, name := range expected {
		if bucketListCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on bucketListCmd", name)
		}
	}
}

func TestBucketList_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"format", "table"},
		{"prefix", ""},
	}
	for _, tc := range cases {
		f := bucketListCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestBucketGet_Flags(t *testing.T) {
	expected := []string{"output", "include-objects"}
	for _, name := range expected {
		if bucketGetCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on bucketGetCmd", name)
		}
	}
}

func TestBucketGet_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"output", "table"},
		{"include-objects", "false"},
	}
	for _, tc := range cases {
		f := bucketGetCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestBucketUpdate_Flags(t *testing.T) {
	expected := []string{"tags", "metadata", "add-tags", "remove-tags"}
	for _, name := range expected {
		if bucketUpdateCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on bucketUpdateCmd", name)
		}
	}
}

func TestBucketDelete_ForceFlag(t *testing.T) {
	f := bucketDeleteCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("--force flag not registered on bucketDeleteCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

func TestBucketImport_Flags(t *testing.T) {
	expected := []string{"spec", "continue"}
	for _, name := range expected {
		if bucketImportCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on bucketImportCmd", name)
		}
	}
}

// --- Arg validation ---

func TestBucketCreate_NoArgs(t *testing.T) {
	err := runBucketCreate(bucketCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no bucket name provided")
	}
}

func TestBucketGet_NoArgs(t *testing.T) {
	err := runBucketGet(bucketGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no bucket name provided")
	}
}

func TestBucketUpdate_NoArgs(t *testing.T) {
	err := runBucketUpdate(bucketUpdateCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no bucket name provided")
	}
}

func TestBucketDelete_NoArgs(t *testing.T) {
	err := runBucketDelete(bucketDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no bucket name provided")
	}
}

func TestBucketExists_NoArgs(t *testing.T) {
	err := runBucketExists(bucketExistsCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no bucket name provided")
	}
}

func TestBucketImport_NoArgsNoSpec(t *testing.T) {
	bucketSpec = ""
	err := runBucketImport(bucketImportCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no spec file provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("specification file")) {
		t.Errorf("error = %q, want it to mention 'specification file'", err.Error())
	}
}

// --- DryRun mode ---

func TestBucketCreate_DryRun(t *testing.T) {
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

	err := runBucketCreate(bucketCreateCmd, []string{"my-bucket"})
	if err != nil {
		t.Errorf("runBucketCreate(DryRun) returned error: %v", err)
	}
}

func TestBucketUpdate_DryRun(t *testing.T) {
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

	err := runBucketUpdate(bucketUpdateCmd, []string{"my-bucket"})
	if err == nil {
		t.Error("runBucketUpdate should return error (metadata updates not supported by CF API)")
	}
}

func TestBucketUpdate_DryRunJSON(t *testing.T) {
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

	err := runBucketUpdate(bucketUpdateCmd, []string{"my-bucket"})

	w.Close()
	os.Stdout = old

	if err == nil {
		t.Error("runBucketUpdate(JSON) must return an error — JSON-mode failures exit non-zero (2026-09-14 exit-code contract)")
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("not supported")) {
		t.Errorf("JSON output should mention 'not supported', got: %q", output)
	}
}

func TestBucketDelete_DryRun(t *testing.T) {
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

	err := runBucketDelete(bucketDeleteCmd, []string{"my-bucket"})
	if err != nil {
		t.Errorf("runBucketDelete(DryRun) returned error: %v", err)
	}
}

func TestBucketDelete_DryRunJSON(t *testing.T) {
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

	err := runBucketDelete(bucketDeleteCmd, []string{"my-bucket"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runBucketDelete(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("DRY RUN")) {
		t.Errorf("JSON dry-run output should contain 'DRY RUN', got: %q", output)
	}
}

func TestBucketImport_DryRun(t *testing.T) {
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

	tmpDir := t.TempDir()
	specPath := tmpDir + "/buckets.json"
	specContent := `{"buckets":[{"name":"test-bucket-1"}]}`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}
	bucketSpec = ""

	err := runBucketImport(bucketImportCmd, []string{specPath})
	if err != nil {
		t.Errorf("runBucketImport(DryRun) returned error: %v", err)
	}
}

func TestBucketImport_YAMLSpec(t *testing.T) {
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

	tmpDir := t.TempDir()
	specPath := tmpDir + "/buckets.yaml"
	specContent := "buckets:\n  - name: test-yaml-bucket\n"
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}
	bucketSpec = ""

	err := runBucketImport(bucketImportCmd, []string{specPath})
	if err != nil {
		t.Errorf("runBucketImport(YAML) returned error: %v", err)
	}
}

func TestBucketImport_EmptySpec(t *testing.T) {
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

	tmpDir := t.TempDir()
	specPath := tmpDir + "/empty.json"
	if err := os.WriteFile(specPath, []byte(`{"buckets":[]}`), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}
	bucketSpec = ""

	err := runBucketImport(bucketImportCmd, []string{specPath})
	if err != nil {
		t.Errorf("runBucketImport(empty spec) returned error: %v", err)
	}
}

func TestBucketImport_InvalidSpecFile(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := tmpDir + "/bad.json"
	if err := os.WriteFile(specPath, []byte(`not-json-not-yaml`), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}
	bucketSpec = ""

	err := runBucketImport(bucketImportCmd, []string{specPath})
	if err == nil {
		t.Fatal("expected error for invalid spec file")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("parse spec file")) {
		t.Errorf("error = %q, want 'parse spec file'", err.Error())
	}
}

func TestBucketImport_NonexistentFile(t *testing.T) {
	bucketSpec = ""
	err := runBucketImport(bucketImportCmd, []string{"/nonexistent/file.json"})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// --- parseKeyValuePairs ---

func TestParseKeyValuePairs_Valid(t *testing.T) {
	result, err := parseKeyValuePairs([]string{"key1=value1", "key2=value2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["key1"] != "value1" {
		t.Errorf("key1 = %q, want %q", result["key1"], "value1")
	}
	if result["key2"] != "value2" {
		t.Errorf("key2 = %q, want %q", result["key2"], "value2")
	}
}

func TestParseKeyValuePairs_Invalid(t *testing.T) {
	_, err := parseKeyValuePairs([]string{"no-equals-sign"})
	if err == nil {
		t.Fatal("expected error for invalid key=value pair")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("invalid key=value pair")) {
		t.Errorf("error = %q, want 'invalid key=value pair'", err.Error())
	}
}

func TestParseKeyValuePairs_Empty(t *testing.T) {
	result, err := parseKeyValuePairs([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d items", len(result))
	}
}

func TestParseKeyValuePairs_ValueWithEquals(t *testing.T) {
	result, err := parseKeyValuePairs([]string{"key=val=ue"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["key"] != "val=ue" {
		t.Errorf("key = %q, want %q", result["key"], "val=ue")
	}
}

// --- parseSpecFile ---

func TestParseSpecFile_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/test.json"
	content := `{"buckets":[{"name":"my-bucket"}]}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var spec BucketSpec
	err := parseSpecFile(path, &spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spec.Buckets) != 1 || spec.Buckets[0].Name != "my-bucket" {
		t.Errorf("spec = %+v, want 1 bucket named 'my-bucket'", spec)
	}
}

func TestParseSpecFile_ValidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/test.yaml"
	content := "buckets:\n  - name: yaml-bucket\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var spec BucketSpec
	err := parseSpecFile(path, &spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spec.Buckets) != 1 || spec.Buckets[0].Name != "yaml-bucket" {
		t.Errorf("spec = %+v, want 1 bucket named 'yaml-bucket'", spec)
	}
}

func TestParseSpecFile_InvalidContent(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/bad.txt"
	if err := os.WriteFile(path, []byte(`!!!invalid!!!`), 0644); err != nil {
		t.Fatal(err)
	}

	var spec BucketSpec
	err := parseSpecFile(path, &spec)
	if err == nil {
		t.Fatal("expected error for invalid content")
	}
}

func TestParseSpecFile_Nonexistent(t *testing.T) {
	var spec BucketSpec
	err := parseSpecFile("/nonexistent/file.json", &spec)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// --- Error message content ---

func TestBucketCreate_ErrorMentionsBucketName(t *testing.T) {
	err := runBucketCreate(bucketCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bucket name") {
		t.Errorf("error = %q, want it to mention 'bucket name'", err.Error())
	}
}

// --- import with --spec flag ---

func TestBucketImport_SpecFlag(t *testing.T) {
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

	tmpDir := t.TempDir()
	specPath := tmpDir + "/spec.json"
	if err := os.WriteFile(specPath, []byte(`{"buckets":[{"name":"flag-bucket"}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	bucketSpec = specPath

	err := runBucketImport(bucketImportCmd, []string{})
	if err != nil {
		t.Errorf("runBucketImport(--spec) returned error: %v", err)
	}
}

// --- JSON import output ---

func TestBucketImport_DryRunJSON(t *testing.T) {
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

	tmpDir := t.TempDir()
	specPath := tmpDir + "/buckets.json"
	if err := os.WriteFile(specPath, []byte(`{"buckets":[{"name":"json-bucket"}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	bucketSpec = ""

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runBucketImport(bucketImportCmd, []string{specPath})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runBucketImport(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	// JSON import with DryRun should output JSON with results
	if !json.Valid([]byte(output)) && output != "" {
		t.Errorf("expected valid JSON output, got: %q", output)
	}
}

// --- Profile prefix scoping (FEAT-026) ---

// bucketPrefixTestEnv snapshots the globals the prefix-scoping tests touch
// and installs a staging profile with resource prefix "stg-".
func bucketPrefixTestEnv(profile *config.Profile) func() {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origProfile := ActiveProfile
	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	ActiveProfile = profile
	return func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		ActiveProfile = origProfile
	}
}

func TestBucketCreate_PrefixScoping(t *testing.T) {
	tests := []struct {
		name    string
		profile *config.Profile
		arg     string
		want    string
	}{
		{"prefix applied", &config.Profile{Name: "staging", ResourcePrefix: "stg-"}, "my-bucket", "stg-my-bucket"},
		{"already prefixed passes through", &config.Profile{Name: "staging", ResourcePrefix: "stg-"}, "stg-my-bucket", "stg-my-bucket"},
		{"nil profile is a no-op", nil, "my-bucket", "my-bucket"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := bucketPrefixTestEnv(tt.profile)
			defer restore()

			out := capturePrint(t, func() {
				if err := runBucketCreate(bucketCreateCmd, []string{tt.arg}); err != nil {
					t.Errorf("runBucketCreate returned error: %v", err)
				}
			})
			if !strings.Contains(out, tt.want) {
				t.Errorf("dry-run output should mention %q, got: %q", tt.want, out)
			}
		})
	}
}

func TestBucketDelete_PrefixScoping(t *testing.T) {
	restore := bucketPrefixTestEnv(&config.Profile{Name: "staging", ResourcePrefix: "stg-"})
	defer restore()

	out := capturePrint(t, func() {
		if err := runBucketDelete(bucketDeleteCmd, []string{"my-bucket"}); err != nil {
			t.Errorf("runBucketDelete returned error: %v", err)
		}
	})
	if !strings.Contains(out, "stg-my-bucket") {
		t.Errorf("dry-run output should mention %q, got: %q", "stg-my-bucket", out)
	}
}

func TestFilterBucketsByProfile(t *testing.T) {
	buckets := []*cosmoflare.Bucket{
		{Name: "stg-assets"},
		{Name: "prod-assets"},
	}

	ActiveProfile = &config.Profile{Name: "staging", ResourcePrefix: "stg-"}
	got := filterBuckets(buckets, "")
	if len(got) != 1 || got[0].Name != "stg-assets" {
		t.Errorf("filterBuckets with profile prefix = %v, want only stg-assets", got)
	}

	// The user --prefix flag combines with the profile prefix.
	ActiveProfile = &config.Profile{Name: "staging", ResourcePrefix: "stg-"}
	got = filterBuckets(buckets, "stg-a")
	if len(got) != 1 || got[0].Name != "stg-assets" {
		t.Errorf("filterBuckets with --prefix stg-a = %v, want only stg-assets", got)
	}

	ActiveProfile = nil
	got = filterBuckets(buckets, "")
	if len(got) != 2 {
		t.Errorf("filterBuckets with nil profile kept %d, want 2", len(got))
	}
}
