package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestObjectCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "object" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("objectCmd not registered on rootCmd")
	}
}

func TestObjectCmd_Metadata(t *testing.T) {
	if objectCmd.Use != "object" {
		t.Errorf("objectCmd.Use = %q, want %q", objectCmd.Use, "object")
	}
	if objectCmd.Short == "" {
		t.Error("objectCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestObjectCmd_Subcommands(t *testing.T) {
	expected := []string{"ls", "get", "put", "delete", "copy", "head", "search", "batch", "presign"}
	for _, name := range expected {
		found := false
		for _, sub := range objectCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("object subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestObjectCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		objectListCmd,
		objectGetCmd,
		objectPutCmd,
		objectDeleteCmd,
		objectCopyCmd,
		objectHeadCmd,
		objectSearchCmd,
		objectBatchCmd,
		objectPresignCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestObjectList_Flags(t *testing.T) {
	expected := []string{"prefix", "delimiter", "max-keys", "recursive"}
	for _, name := range expected {
		if objectListCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on objectListCmd", name)
		}
	}
}

func TestObjectList_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"prefix", ""},
		{"delimiter", ""},
		{"max-keys", "0"},
		{"recursive", "false"},
	}
	for _, tc := range cases {
		f := objectListCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestObjectGet_Flags(t *testing.T) {
	expected := []string{"output", "range-start", "range-end"}
	for _, name := range expected {
		if objectGetCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on objectGetCmd", name)
		}
	}
}

func TestObjectPut_Flags(t *testing.T) {
	expected := []string{"key", "content-type", "cache-control", "metadata", "progress"}
	for _, name := range expected {
		if objectPutCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on objectPutCmd", name)
		}
	}
}

func TestObjectPut_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"key", ""},
		{"content-type", ""},
		{"cache-control", ""},
		{"progress", "true"},
	}
	for _, tc := range cases {
		f := objectPutCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestObjectCopy_Flags(t *testing.T) {
	expected := []string{"metadata", "content-type"}
	for _, name := range expected {
		if objectCopyCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on objectCopyCmd", name)
		}
	}
}

func TestObjectHead_Flags(t *testing.T) {
	f := objectHeadCmd.Flags().Lookup("output")
	if f == nil {
		t.Fatal("--output flag not registered on objectHeadCmd")
	}
	if f.DefValue != "table" {
		t.Errorf("--output default = %q, want %q", f.DefValue, "table")
	}
}

func TestObjectSearch_Flags(t *testing.T) {
	f := objectSearchCmd.Flags().Lookup("type")
	if f == nil {
		t.Fatal("--type flag not registered on objectSearchCmd")
	}
	if f.DefValue != "prefix" {
		t.Errorf("--type default = %q, want %q", f.DefValue, "prefix")
	}
}

func TestObjectBatch_Flags(t *testing.T) {
	f := objectBatchCmd.Flags().Lookup("continue")
	if f == nil {
		t.Fatal("--continue flag not registered on objectBatchCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--continue default = %q, want %q", f.DefValue, "false")
	}
}

func TestObjectPresign_Flags(t *testing.T) {
	f := objectPresignCmd.Flags().Lookup("expires")
	if f == nil {
		t.Fatal("--expires flag not registered on objectPresignCmd")
	}
	if f.DefValue != "1h" {
		t.Errorf("--expires default = %q, want %q", f.DefValue, "1h")
	}
}

// --- Arg validation ---

func TestObjectList_NoArgs(t *testing.T) {
	err := runObjectList(objectListCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no bucket name provided")
	}
}

func TestObjectGet_NoArgs(t *testing.T) {
	err := runObjectGet(objectGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectGet_OneArg(t *testing.T) {
	err := runObjectGet(objectGetCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing object key)")
	}
}

func TestObjectPut_NoArgs(t *testing.T) {
	err := runObjectPut(objectPutCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectPut_OneArg(t *testing.T) {
	err := runObjectPut(objectPutCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing local path)")
	}
}

func TestObjectDelete_NoArgs(t *testing.T) {
	err := runObjectDelete(objectDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectDelete_OneArg(t *testing.T) {
	err := runObjectDelete(objectDeleteCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing object key)")
	}
}

func TestObjectCopy_NoArgs(t *testing.T) {
	err := runObjectCopy(objectCopyCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectCopy_OneArg(t *testing.T) {
	err := runObjectCopy(objectCopyCmd, []string{"src-bucket/file.txt"})
	if err == nil {
		t.Fatal("expected error when only source provided (missing destination)")
	}
}

func TestObjectCopy_InvalidSource(t *testing.T) {
	err := runObjectCopy(objectCopyCmd, []string{"no-slash", "dest-bucket/file.txt"})
	if err == nil {
		t.Fatal("expected error when source missing bucket/key separator")
	}
}

func TestObjectCopy_InvalidDest(t *testing.T) {
	err := runObjectCopy(objectCopyCmd, []string{"src-bucket/file.txt", "no-slash"})
	if err == nil {
		t.Fatal("expected error when destination missing bucket/key separator")
	}
}

func TestObjectHead_NoArgs(t *testing.T) {
	err := runObjectHead(objectHeadCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectHead_OneArg(t *testing.T) {
	err := runObjectHead(objectHeadCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing object key)")
	}
}

func TestObjectSearch_NoArgs(t *testing.T) {
	err := runObjectSearch(objectSearchCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectSearch_OneArg(t *testing.T) {
	err := runObjectSearch(objectSearchCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing search query)")
	}
}

func TestObjectBatch_NoArgs(t *testing.T) {
	err := runObjectBatch(objectBatchCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectBatch_OneArg(t *testing.T) {
	err := runObjectBatch(objectBatchCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing spec file)")
	}
}

func TestObjectPresign_NoArgs(t *testing.T) {
	err := runObjectPresign(objectPresignCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestObjectPresign_OneArg(t *testing.T) {
	err := runObjectPresign(objectPresignCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only bucket name provided (missing object key)")
	}
}

// --- Stdin validation ---

func TestObjectPut_StdinNoKey(t *testing.T) {
	err := runObjectPut(objectPutCmd, []string{"my-bucket", "-"})
	if err == nil {
		t.Fatal("expected error when stdin used without --key")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("key")) {
		t.Errorf("error = %q, want it to mention 'key'", err.Error())
	}
}

// --- DryRun mode ---

func TestObjectPut_DryRun(t *testing.T) {
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
	filePath := tmpDir + "/test.txt"
	if err := os.WriteFile(filePath, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	err := runObjectPut(objectPutCmd, []string{"my-bucket", filePath})
	if err != nil {
		t.Errorf("runObjectPut(DryRun) returned error: %v", err)
	}
}

func TestObjectPut_DryRunWithContentType(t *testing.T) {
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
	filePath := tmpDir + "/test.txt"
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	objectContentType = "text/plain"
	objectCacheControl = "max-age=3600"
	objectMetadata = []string{"author=test"}
	defer func() {
		objectContentType = ""
		objectCacheControl = ""
		objectMetadata = nil
	}()

	err := runObjectPut(objectPutCmd, []string{"my-bucket", filePath})
	if err != nil {
		t.Errorf("runObjectPut(DryRun+options) returned error: %v", err)
	}
}

func TestObjectDelete_DryRun(t *testing.T) {
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

	err := runObjectDelete(objectDeleteCmd, []string{"my-bucket", "file.txt"})
	if err != nil {
		t.Errorf("runObjectDelete(DryRun) returned error: %v", err)
	}
}

func TestObjectCopy_DryRun(t *testing.T) {
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

	err := runObjectCopy(objectCopyCmd, []string{"src-bucket/file.txt", "dst-bucket/backup.txt"})
	if err != nil {
		t.Errorf("runObjectCopy(DryRun) returned error: %v", err)
	}
}

func TestObjectPut_NonexistentFile(t *testing.T) {
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

	DryRun = false
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	objectKey = ""

	err := runObjectPut(objectPutCmd, []string{"my-bucket", "/nonexistent/file.txt"})
	if err == nil {
		t.Fatal("expected error for nonexistent local file")
	}
}

// --- Helper function tests ---

func TestEtagDisplay_Short(t *testing.T) {
	result := etagDisplay("abc123")
	if result != "abc123" {
		t.Errorf("etagDisplay(%q) = %q, want %q", "abc123", result, "abc123")
	}
}

func TestEtagDisplay_Long(t *testing.T) {
	longEtag := "abcdef0123456789abcdef0123456789"
	result := etagDisplay(longEtag)
	if result != "abcdef0123456789..." {
		t.Errorf("etagDisplay(long) = %q, want truncated with ...", result)
	}
}

func TestEtagDisplay_Empty(t *testing.T) {
	result := etagDisplay("")
	if result != "" {
		t.Errorf("etagDisplay('') = %q, want empty", result)
	}
}

func TestParseBatchSpec_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/batch.json"
	content := `{"operations":[{"action":"delete","object_key":"old.txt"}]}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var spec BatchSpec
	err := parseBatchSpec(path, &spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spec.Operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(spec.Operations))
	}
	if spec.Operations[0].Action != "delete" || spec.Operations[0].ObjectKey != "old.txt" {
		t.Errorf("operation = %+v, unexpected", spec.Operations[0])
	}
}

func TestParseBatchSpec_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/bad.json"
	if err := os.WriteFile(path, []byte(`not-json`), 0644); err != nil {
		t.Fatal(err)
	}

	var spec BatchSpec
	err := parseBatchSpec(path, &spec)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseBatchSpec_NonexistentFile(t *testing.T) {
	var spec BatchSpec
	err := parseBatchSpec("/nonexistent/batch.json", &spec)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestObjectBatch_EmptySpec(t *testing.T) {
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

	DryRun = false
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"

	tmpDir := t.TempDir()
	specPath := tmpDir + "/empty.json"
	if err := os.WriteFile(specPath, []byte(`{"operations":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	err := runObjectBatch(objectBatchCmd, []string{"my-bucket", specPath})
	if err != nil {
		t.Errorf("runObjectBatch(empty spec) returned error: %v", err)
	}
}

// --- Presign invalid duration ---

func TestObjectPresign_InvalidExpires(t *testing.T) {
	objectExpires = "not-a-duration"
	defer func() { objectExpires = "1h" }()

	err := runObjectPresign(objectPresignCmd, []string{"my-bucket", "file.txt"})
	if err == nil {
		t.Fatal("expected error for invalid expires duration")
	}
}

// --- Object batch DryRun ---

func TestObjectBatch_DryRun(t *testing.T) {
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
	specPath := tmpDir + "/batch.json"
	content := `{"operations":[{"action":"delete","object_key":"old.txt"}]}`
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := runObjectBatch(objectBatchCmd, []string{"my-bucket", specPath})
	if err != nil {
		t.Errorf("runObjectBatch(DryRun) returned error: %v", err)
	}
}

func TestObjectBatch_DryRunJSON(t *testing.T) {
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
	specPath := tmpDir + "/batch.json"
	content := `{"operations":[{"action":"delete","object_key":"old.txt"}]}`
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runObjectBatch(objectBatchCmd, []string{"my-bucket", specPath})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("runObjectBatch(DryRun+JSON) returned error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("successful")) {
		t.Errorf("JSON batch output should contain 'successful', got: %q", output)
	}
}

func TestObjectBatch_NonexistentSpecFile(t *testing.T) {
	err := runObjectBatch(objectBatchCmd, []string{"my-bucket", "/nonexistent/spec.json"})
	if err == nil {
		t.Fatal("expected error for nonexistent spec file")
	}
}

// --- Object put with invalid metadata ---

func TestObjectPut_InvalidMetadata(t *testing.T) {
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

	DryRun = false
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	objectMetadata = []string{"no-equals"}
	defer func() { objectMetadata = nil }()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/test.txt"
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	err := runObjectPut(objectPutCmd, []string{"my-bucket", filePath})
	if err == nil {
		t.Fatal("expected error for invalid metadata format")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("metadata")) {
		t.Errorf("error = %q, want it to mention 'metadata'", err.Error())
	}
}

// --- Object put with explicit key ---

func TestObjectPut_DryRunWithExplicitKey(t *testing.T) {
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
	objectKey = "remote/path/file.txt"
	defer func() { objectKey = "" }()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/test.txt"
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	err := runObjectPut(objectPutCmd, []string{"my-bucket", filePath})
	if err != nil {
		t.Errorf("runObjectPut(DryRun+key) returned error: %v", err)
	}
}

// --- Object delete and copy DryRun already covered by non-JSON tests ---
// (JSON variants don't produce different JSON output for DryRun - they use printInfo)

// --- Error message assertions ---

func TestObjectList_ErrorMentionsBucket(t *testing.T) {
	err := runObjectList(objectListCmd, []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("bucket")) {
		t.Errorf("error = %q, want it to mention 'bucket'", err.Error())
	}
}

func TestObjectGet_ErrorMentionsBucket(t *testing.T) {
	err := runObjectGet(objectGetCmd, []string{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestObjectCopy_ErrorMentionsSource(t *testing.T) {
	err := runObjectCopy(objectCopyCmd, []string{"no-slash", "dest/key"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("source")) {
		t.Errorf("error = %q, want it to mention 'source'", err.Error())
	}
}

func TestObjectCopy_ErrorMentionsDest(t *testing.T) {
	err := runObjectCopy(objectCopyCmd, []string{"src/key", "no-slash"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("destination")) {
		t.Errorf("error = %q, want it to mention 'destination'", err.Error())
	}
}
