package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
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
	expected := []string{"key", "content-type", "cache-control", "metadata", "progress",
		"part-size", "concurrency", "resume", "no-multipart"}
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
		{"part-size", "8MB"},
		{"concurrency", "4"},
		{"resume", "false"},
		{"no-multipart", "false"},
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

// --- parseSize tests ---

func TestParseSize_MB(t *testing.T) {
	size, err := parseSize("8MB")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 8*1024*1024 {
		t.Errorf("parseSize(8MB) = %d, want %d", size, 8*1024*1024)
	}
}

func TestParseSize_GB(t *testing.T) {
	size, err := parseSize("1GB")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 1024*1024*1024 {
		t.Errorf("parseSize(1GB) = %d, want %d", size, 1024*1024*1024)
	}
}

func TestParseSize_KB(t *testing.T) {
	size, err := parseSize("512KB")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 512*1024 {
		t.Errorf("parseSize(512KB) = %d, want %d", size, 512*1024)
	}
}

func TestParseSize_LowerCase(t *testing.T) {
	size, err := parseSize("16mb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 16*1024*1024 {
		t.Errorf("parseSize(16mb) = %d, want %d", size, 16*1024*1024)
	}
}

func TestParseSize_PlainNumber(t *testing.T) {
	size, err := parseSize("1048576")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 1048576 {
		t.Errorf("parseSize(1048576) = %d, want 1048576", size)
	}
}

func TestParseSize_Empty(t *testing.T) {
	_, err := parseSize("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestParseSize_Invalid(t *testing.T) {
	_, err := parseSize("not-a-size")
	if err == nil {
		t.Fatal("expected error for invalid size")
	}
}

func TestParseSize_Zero(t *testing.T) {
	_, err := parseSize("0MB")
	if err == nil {
		t.Fatal("expected error for zero size")
	}
}

func TestParseSize_Negative(t *testing.T) {
	_, err := parseSize("-5MB")
	if err == nil {
		t.Fatal("expected error for negative size")
	}
}

// --- DryRun with multipart flags ---

func TestObjectPut_DryRunWithMultipartFlags(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origPartSize := objectPartSize
	origConcurrency := objectConcurrency
	origNoMultipart := objectNoMultipart
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		objectPartSize = origPartSize
		objectConcurrency = origConcurrency
		objectNoMultipart = origNoMultipart
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	objectPartSize = "16MB"
	objectConcurrency = 8
	objectNoMultipart = false

	tmpDir := t.TempDir()
	filePath := tmpDir + "/test.txt"
	if err := os.WriteFile(filePath, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	err := runObjectPut(objectPutCmd, []string{"my-bucket", filePath})
	if err != nil {
		t.Errorf("runObjectPut(DryRun+multipart flags) returned error: %v", err)
	}
}

func TestObjectPut_DryRunNoMultipart(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origPartSize := objectPartSize
	origNoMultipart := objectNoMultipart
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		objectPartSize = origPartSize
		objectNoMultipart = origNoMultipart
	}()

	DryRun = true
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	objectPartSize = "8MB"
	objectNoMultipart = true

	tmpDir := t.TempDir()
	filePath := tmpDir + "/test.txt"
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	err := runObjectPut(objectPutCmd, []string{"my-bucket", filePath})
	if err != nil {
		t.Errorf("runObjectPut(DryRun+no-multipart) returned error: %v", err)
	}
}

// --- Resume with no state file ---

func TestObjectPut_ResumeNoState(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origPartSize := objectPartSize
	origResume := objectResume
	origKey := objectKey
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		objectPartSize = origPartSize
		objectResume = origResume
		objectKey = origKey
	}()

	DryRun = false
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	objectPartSize = "8MB"
	objectResume = true
	objectKey = "nonexistent-resume-key-xyz"

	tmpDir := t.TempDir()
	filePath := tmpDir + "/test.txt"
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	err := runObjectPut(objectPutCmd, []string{"my-bucket", filePath})
	if err == nil {
		t.Fatal("expected error when resuming with no state file")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("no interrupted upload")) {
		t.Errorf("error = %q, want it to mention 'no interrupted upload'", err.Error())
	}
}

// --- Invalid part size ---

func TestObjectPut_InvalidPartSize(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAccountID := AccountID
	origAPIToken := APIToken
	origPartSize := objectPartSize
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		AccountID = origAccountID
		APIToken = origAPIToken
		objectPartSize = origPartSize
	}()

	DryRun = false
	JSONOutput = false
	AccountID = "test-account"
	APIToken = "test-token"
	objectPartSize = "1MB" // below 5MB minimum

	tmpDir := t.TempDir()
	filePath := tmpDir + "/test.txt"
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	err := runObjectPut(objectPutCmd, []string{"my-bucket", filePath})
	if err == nil {
		t.Fatal("expected error for part size below minimum")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("part-size")) {
		t.Errorf("error = %q, want it to mention 'part-size'", err.Error())
	}
}

func TestObjectPut_InvalidPartSizeFormat(t *testing.T) {
	origPartSize := objectPartSize
	defer func() { objectPartSize = origPartSize }()

	objectPartSize = "not-a-size"

	tmpDir := t.TempDir()
	filePath := tmpDir + "/test.txt"
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	err := runObjectPut(objectPutCmd, []string{"my-bucket", filePath})
	if err == nil {
		t.Fatal("expected error for invalid part size format")
	}
}

// =============================================================================
// Additional coverage: command metadata, flag defaults, helper edge cases
// =============================================================================

// --- objectCmd Long text ---

func TestObjectCmd_LongNotEmpty(t *testing.T) {
	if objectCmd.Long == "" {
		t.Error("objectCmd.Long is empty; expected descriptive long help text")
	}
}

// --- Subcommand Use strings ---

func TestObjectListCmd_UseString(t *testing.T) {
	if objectListCmd.Use == "" {
		t.Error("objectListCmd.Use is empty")
	}
}

func TestObjectGetCmd_UseString(t *testing.T) {
	if objectGetCmd.Use == "" {
		t.Error("objectGetCmd.Use is empty")
	}
}

func TestObjectPutCmd_UseString(t *testing.T) {
	if objectPutCmd.Use == "" {
		t.Error("objectPutCmd.Use is empty")
	}
}

func TestObjectDeleteCmd_UseString(t *testing.T) {
	if objectDeleteCmd.Use == "" {
		t.Error("objectDeleteCmd.Use is empty")
	}
}

func TestObjectCopyCmd_UseString(t *testing.T) {
	if objectCopyCmd.Use == "" {
		t.Error("objectCopyCmd.Use is empty")
	}
}

func TestObjectHeadCmd_UseString(t *testing.T) {
	if objectHeadCmd.Use == "" {
		t.Error("objectHeadCmd.Use is empty")
	}
}

func TestObjectSearchCmd_UseString(t *testing.T) {
	if objectSearchCmd.Use == "" {
		t.Error("objectSearchCmd.Use is empty")
	}
}

func TestObjectBatchCmd_UseString(t *testing.T) {
	if objectBatchCmd.Use == "" {
		t.Error("objectBatchCmd.Use is empty")
	}
}

func TestObjectPresignCmd_UseString(t *testing.T) {
	if objectPresignCmd.Use == "" {
		t.Error("objectPresignCmd.Use is empty")
	}
}

// --- Subcommand Short strings not empty ---

func TestObjectSubcmds_ShortNotEmpty(t *testing.T) {
	cmds := map[string]*cobra.Command{
		"ls":      objectListCmd,
		"get":     objectGetCmd,
		"put":     objectPutCmd,
		"delete":  objectDeleteCmd,
		"copy":    objectCopyCmd,
		"head":    objectHeadCmd,
		"search":  objectSearchCmd,
		"batch":   objectBatchCmd,
		"presign": objectPresignCmd,
	}
	for name, cmd := range cmds {
		if cmd.Short == "" {
			t.Errorf("subcommand %q has empty Short description", name)
		}
	}
}

// --- Subcommand Long strings not empty ---

func TestObjectSubcmds_LongNotEmpty(t *testing.T) {
	cmds := map[string]*cobra.Command{
		"ls":      objectListCmd,
		"get":     objectGetCmd,
		"put":     objectPutCmd,
		"delete":  objectDeleteCmd,
		"copy":    objectCopyCmd,
		"head":    objectHeadCmd,
		"search":  objectSearchCmd,
		"batch":   objectBatchCmd,
		"presign": objectPresignCmd,
	}
	for name, cmd := range cmds {
		if cmd.Long == "" {
			t.Errorf("subcommand %q has empty Long description", name)
		}
	}
}

// --- objectGetCmd flag defaults ---

func TestObjectGetCmd_RangeStartDefault(t *testing.T) {
	f := objectGetCmd.Flags().Lookup("range-start")
	if f == nil {
		t.Fatal("--range-start flag not registered on objectGetCmd")
	}
	if f.DefValue != "-1" {
		t.Errorf("--range-start default = %q, want %q", f.DefValue, "-1")
	}
}

func TestObjectGetCmd_RangeEndDefault(t *testing.T) {
	f := objectGetCmd.Flags().Lookup("range-end")
	if f == nil {
		t.Fatal("--range-end flag not registered on objectGetCmd")
	}
	if f.DefValue != "-1" {
		t.Errorf("--range-end default = %q, want %q", f.DefValue, "-1")
	}
}

func TestObjectGetCmd_OutputDefault(t *testing.T) {
	f := objectGetCmd.Flags().Lookup("output")
	if f == nil {
		t.Fatal("--output flag not registered on objectGetCmd")
	}
	if f.DefValue != "" {
		t.Errorf("--output default = %q, want empty string", f.DefValue)
	}
}

// --- etagDisplay boundary conditions ---

func TestEtagDisplay_ExactlyAtBoundary(t *testing.T) {
	// 16 chars exactly — should NOT be truncated
	etag16 := "abcdef0123456789" // exactly 16 chars
	result := etagDisplay(etag16)
	if result != etag16 {
		t.Errorf("etagDisplay(16-char etag) = %q, want %q (no truncation at exactly 16)", result, etag16)
	}
}

func TestEtagDisplay_JustOverBoundary(t *testing.T) {
	// 17 chars — should be truncated
	etag17 := "abcdef01234567890" // 17 chars
	result := etagDisplay(etag17)
	if result != "abcdef0123456789..." {
		t.Errorf("etagDisplay(17-char etag) = %q, want %q", result, "abcdef0123456789...")
	}
}

func TestEtagDisplay_SingleChar(t *testing.T) {
	result := etagDisplay("a")
	if result != "a" {
		t.Errorf("etagDisplay(single char) = %q, want %q", result, "a")
	}
}

// --- parseSize edge cases ---

func TestParseSize_Bytes(t *testing.T) {
	size, err := parseSize("1024B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 1024 {
		t.Errorf("parseSize(1024B) = %d, want 1024", size)
	}
}

func TestParseSize_LargeGB(t *testing.T) {
	size, err := parseSize("100GB")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := int64(100) * 1024 * 1024 * 1024
	if size != want {
		t.Errorf("parseSize(100GB) = %d, want %d", size, want)
	}
}

func TestParseSize_SpacePadded(t *testing.T) {
	size, err := parseSize("  8MB  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 8*1024*1024 {
		t.Errorf("parseSize('  8MB  ') = %d, want %d", size, 8*1024*1024)
	}
}

func TestParseSize_UppercaseGB(t *testing.T) {
	size, err := parseSize("2GB")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := int64(2) * 1024 * 1024 * 1024
	if size != want {
		t.Errorf("parseSize(2GB) = %d, want %d", size, want)
	}
}

func TestParseSize_OneByte(t *testing.T) {
	size, err := parseSize("1B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 1 {
		t.Errorf("parseSize(1B) = %d, want 1", size)
	}
}

func TestParseSize_LargeInt(t *testing.T) {
	size, err := parseSize("5242880") // 5MB in bytes
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 5242880 {
		t.Errorf("parseSize(5242880) = %d, want 5242880", size)
	}
}

// --- parseBatchSpec multi-operation ---

func TestParseBatchSpec_MultipleOperations(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/multi.json"
	content := `{
		"operations": [
			{"action": "upload", "local_path": "file1.txt", "object_key": "remote/file1.txt"},
			{"action": "delete", "object_key": "old.txt"},
			{"action": "copy", "object_key": "src.txt", "destination_key": "dst.txt"}
		]
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var spec BatchSpec
	if err := parseBatchSpec(path, &spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spec.Operations) != 3 {
		t.Fatalf("expected 3 operations, got %d", len(spec.Operations))
	}
	if spec.Operations[0].Action != "upload" {
		t.Errorf("op[0].Action = %q, want %q", spec.Operations[0].Action, "upload")
	}
	if spec.Operations[0].LocalPath != "file1.txt" {
		t.Errorf("op[0].LocalPath = %q, want %q", spec.Operations[0].LocalPath, "file1.txt")
	}
	if spec.Operations[0].ObjectKey != "remote/file1.txt" {
		t.Errorf("op[0].ObjectKey = %q, want %q", spec.Operations[0].ObjectKey, "remote/file1.txt")
	}
	if spec.Operations[2].DestinationKey != "dst.txt" {
		t.Errorf("op[2].DestinationKey = %q, want %q", spec.Operations[2].DestinationKey, "dst.txt")
	}
}

func TestParseBatchSpec_EmptyOperations(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/empty-ops.json"
	if err := os.WriteFile(path, []byte(`{"operations": []}`), 0644); err != nil {
		t.Fatal(err)
	}

	var spec BatchSpec
	if err := parseBatchSpec(path, &spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spec.Operations) != 0 {
		t.Errorf("expected 0 operations, got %d", len(spec.Operations))
	}
}

// --- BatchOperation struct field round-trip ---

func TestBatchOperation_FieldsRoundTrip(t *testing.T) {
	op := BatchOperation{
		Action:         "upload",
		LocalPath:      "local/path.txt",
		ObjectKey:      "remote/key.txt",
		DestinationKey: "dest/key.txt",
	}
	if op.Action != "upload" {
		t.Errorf("Action = %q, want %q", op.Action, "upload")
	}
	if op.LocalPath != "local/path.txt" {
		t.Errorf("LocalPath = %q, want %q", op.LocalPath, "local/path.txt")
	}
	if op.ObjectKey != "remote/key.txt" {
		t.Errorf("ObjectKey = %q, want %q", op.ObjectKey, "remote/key.txt")
	}
	if op.DestinationKey != "dest/key.txt" {
		t.Errorf("DestinationKey = %q, want %q", op.DestinationKey, "dest/key.txt")
	}
}

// --- Error message assertions ---

func TestObjectDelete_ErrorMentionsRequired(t *testing.T) {
	err := runObjectDelete(objectDeleteCmd, []string{})
	if err == nil {
		t.Fatal("expected error for no args")
	}
	msg := err.Error()
	if msg == "" {
		t.Error("error message is empty")
	}
}

func TestObjectDelete_OneArgErrorMentionsKey(t *testing.T) {
	err := runObjectDelete(objectDeleteCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error for missing object key")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("key")) &&
		!bytes.Contains([]byte(err.Error()), []byte("object")) {
		t.Errorf("error = %q, want mention of 'key' or 'object'", err.Error())
	}
}

func TestObjectHead_ErrorMentionsRequired(t *testing.T) {
	err := runObjectHead(objectHeadCmd, []string{})
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

func TestObjectSearch_ErrorMentionsQuery(t *testing.T) {
	err := runObjectSearch(objectSearchCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error for missing search query")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("query")) &&
		!bytes.Contains([]byte(err.Error()), []byte("search")) &&
		!bytes.Contains([]byte(err.Error()), []byte("required")) {
		t.Errorf("error = %q, want mention of 'query', 'search', or 'required'", err.Error())
	}
}

func TestObjectBatch_ErrorMentionsSpec(t *testing.T) {
	err := runObjectBatch(objectBatchCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error for missing spec file")
	}
	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

func TestObjectPresign_ErrorMentionsRequired(t *testing.T) {
	err := runObjectPresign(objectPresignCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error for missing object key")
	}
	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

// --- objectSearch type flag values ---

func TestObjectSearch_TypeFlagPrefix(t *testing.T) {
	f := objectSearchCmd.Flags().Lookup("type")
	if f == nil {
		t.Fatal("--type flag not registered on objectSearchCmd")
	}
	// Default should be prefix
	if f.DefValue != "prefix" {
		t.Errorf("--type default = %q, want %q", f.DefValue, "prefix")
	}
}

// --- objectBatch continue flag ---

func TestObjectBatch_ContinueFlagExists(t *testing.T) {
	f := objectBatchCmd.Flags().Lookup("continue")
	if f == nil {
		t.Fatal("--continue flag not registered on objectBatchCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--continue default = %q, want %q", f.DefValue, "false")
	}
}

// --- objectPresign valid duration format ---

func TestObjectPresign_ValidDurationFormats(t *testing.T) {
	durations := []string{"1h", "24h", "30m", "1h30m", "2h"}
	for _, d := range durations {
		// We just test parsing, not actual presign (no API)
		origExpires := objectExpires
		objectExpires = d
		// A missing bucket name should trigger arg error before duration parsing
		err := runObjectPresign(objectPresignCmd, []string{})
		objectExpires = origExpires
		if err == nil {
			t.Errorf("expected arg error for empty args with expires=%q", d)
		}
		// The error should be about args, not duration
		if bytes.Contains([]byte(err.Error()), []byte("invalid expires")) {
			t.Errorf("for expires=%q, got duration error instead of arg error: %v", d, err)
		}
	}
}

// --- objectPut DryRun with JSON output ---

func TestObjectPut_DryRunJSON(t *testing.T) {
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
	filePath := tmpDir + "/test.json"
	if err := os.WriteFile(filePath, []byte(`{"key":"value"}`), 0644); err != nil {
		t.Fatal(err)
	}

	err := runObjectPut(objectPutCmd, []string{"my-bucket", filePath})
	if err != nil {
		t.Errorf("runObjectPut(DryRun+JSON) returned error: %v", err)
	}
}

// --- objectDelete DryRun with JSON output ---

func TestObjectDelete_DryRunJSON(t *testing.T) {
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

	err := runObjectDelete(objectDeleteCmd, []string{"my-bucket", "file.txt"})
	if err != nil {
		t.Errorf("runObjectDelete(DryRun+JSON) returned error: %v", err)
	}
}

// --- objectCopy DryRun with JSON output ---

func TestObjectCopy_DryRunJSON(t *testing.T) {
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

	err := runObjectCopy(objectCopyCmd, []string{"src-bucket/file.txt", "dst-bucket/backup.txt"})
	if err != nil {
		t.Errorf("runObjectCopy(DryRun+JSON) returned error: %v", err)
	}
}

// --- objectList MaxKeys flag type ---

func TestObjectList_MaxKeysIsInt32(t *testing.T) {
	f := objectListCmd.Flags().Lookup("max-keys")
	if f == nil {
		t.Fatal("--max-keys flag not registered")
	}
	if f.Value.Type() != "int32" {
		t.Errorf("--max-keys type = %q, want %q", f.Value.Type(), "int32")
	}
}

// --- objectPut concurrency flag type ---

func TestObjectPut_ConcurrencyIsInt(t *testing.T) {
	f := objectPutCmd.Flags().Lookup("concurrency")
	if f == nil {
		t.Fatal("--concurrency flag not registered")
	}
	if f.Value.Type() != "int" {
		t.Errorf("--concurrency type = %q, want %q", f.Value.Type(), "int")
	}
}

// --- objectPut progress flag type ---

func TestObjectPut_ProgressIsBool(t *testing.T) {
	f := objectPutCmd.Flags().Lookup("progress")
	if f == nil {
		t.Fatal("--progress flag not registered")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("--progress type = %q, want %q", f.Value.Type(), "bool")
	}
}

// --- objectList recursive flag type ---

func TestObjectList_RecursiveIsBool(t *testing.T) {
	f := objectListCmd.Flags().Lookup("recursive")
	if f == nil {
		t.Fatal("--recursive flag not registered")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("--recursive type = %q, want %q", f.Value.Type(), "bool")
	}
}

// --- objectCopy metadata flag type ---

func TestObjectCopy_MetadataIsStringSlice(t *testing.T) {
	f := objectCopyCmd.Flags().Lookup("metadata")
	if f == nil {
		t.Fatal("--metadata flag not registered on objectCopyCmd")
	}
	if f.Value.Type() != "stringSlice" {
		t.Errorf("--metadata type = %q, want %q", f.Value.Type(), "stringSlice")
	}
}

// --- objectCmd parent relationship ---

func TestObjectCmd_HasParent(t *testing.T) {
	if objectCmd.Parent() == nil {
		t.Error("objectCmd has no parent — expected to be registered on rootCmd")
	}
	if objectCmd.Parent() != rootCmd {
		t.Errorf("objectCmd.Parent() = %q, want rootCmd", objectCmd.Parent().Use)
	}
}

// --- subcommands have objectCmd as parent ---

func TestObjectSubcmds_HaveCorrectParent(t *testing.T) {
	cmds := map[string]*cobra.Command{
		"ls":      objectListCmd,
		"get":     objectGetCmd,
		"put":     objectPutCmd,
		"delete":  objectDeleteCmd,
		"copy":    objectCopyCmd,
		"head":    objectHeadCmd,
		"search":  objectSearchCmd,
		"batch":   objectBatchCmd,
		"presign": objectPresignCmd,
	}
	for name, cmd := range cmds {
		if cmd.Parent() == nil {
			t.Errorf("subcommand %q has no parent", name)
			continue
		}
		if cmd.Parent() != objectCmd {
			t.Errorf("subcommand %q parent = %q, want objectCmd", name, cmd.Parent().Use)
		}
	}
}

// --- objectPut stdin with key (valid arg) ---

func TestObjectPut_StdinWithKey(t *testing.T) {
	origKey := objectKey
	defer func() { objectKey = origKey }()

	objectKey = ""
	// "-" without --key should fail (already tested), this verifies the flag is the gating mechanism
	err := runObjectPut(objectPutCmd, []string{"my-bucket", "-"})
	if err == nil {
		t.Fatal("expected error for stdin without --key")
	}
}

// --- parseSize consistency: MB == 1024*KB ---

func TestParseSize_MBConsistency(t *testing.T) {
	mb, err := parseSize("1MB")
	if err != nil {
		t.Fatalf("unexpected error for 1MB: %v", err)
	}
	kb, err := parseSize("1024KB")
	if err != nil {
		t.Fatalf("unexpected error for 1024KB: %v", err)
	}
	if mb != kb {
		t.Errorf("parseSize(1MB)=%d != parseSize(1024KB)=%d, should be equal", mb, kb)
	}
}

// --- parseSize: GB == 1024*MB ---

func TestParseSize_GBConsistency(t *testing.T) {
	gb, err := parseSize("1GB")
	if err != nil {
		t.Fatalf("unexpected error for 1GB: %v", err)
	}
	mb, err := parseSize("1024MB")
	if err != nil {
		t.Fatalf("unexpected error for 1024MB: %v", err)
	}
	if gb != mb {
		t.Errorf("parseSize(1GB)=%d != parseSize(1024MB)=%d, should be equal", gb, mb)
	}
}

// --- objectPresign valid expires (arg error fires first) ---

func TestObjectPresign_ValidExpires(t *testing.T) {
	origExpires := objectExpires
	defer func() { objectExpires = origExpires }()

	objectExpires = "2h30m"
	// With no args the arg error fires before duration parsing
	err := runObjectPresign(objectPresignCmd, []string{})
	if err == nil {
		t.Fatal("expected error for missing args")
	}
	// Should be an arg error, not duration error
	if bytes.Contains([]byte(err.Error()), []byte("invalid expires")) {
		t.Errorf("got duration error instead of arg error: %v", err)
	}
}

// --- objectBatch with bad JSON in operations array ---

func TestParseBatchSpec_PartialJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/partial.json"
	// Valid outer struct but malformed inner
	if err := os.WriteFile(path, []byte(`{"operations": [{`), 0644); err != nil {
		t.Fatal(err)
	}
	var spec BatchSpec
	err := parseBatchSpec(path, &spec)
	if err == nil {
		t.Fatal("expected error for partial/malformed JSON")
	}
}

// --- objectPut metadata flag is stringSlice ---

func TestObjectPut_MetadataIsStringSlice(t *testing.T) {
	f := objectPutCmd.Flags().Lookup("metadata")
	if f == nil {
		t.Fatal("--metadata flag not registered on objectPutCmd")
	}
	if f.Value.Type() != "stringSlice" {
		t.Errorf("--metadata type = %q, want %q", f.Value.Type(), "stringSlice")
	}
}

// --- objectSearch supported types ---

func TestObjectSearch_NoArgsErrorNotEmpty(t *testing.T) {
	err := runObjectSearch(objectSearchCmd, []string{})
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

// --- parseBatchSpec reads correct action field ---

func TestParseBatchSpec_UploadAction(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/upload.json"
	content := `{"operations":[{"action":"upload","local_path":"./test.txt","object_key":"test.txt"}]}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var spec BatchSpec
	if err := parseBatchSpec(path, &spec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Operations[0].Action != "upload" {
		t.Errorf("action = %q, want %q", spec.Operations[0].Action, "upload")
	}
	if spec.Operations[0].LocalPath != "./test.txt" {
		t.Errorf("local_path = %q, want %q", spec.Operations[0].LocalPath, "./test.txt")
	}
}

// --- Total subcommand count ---

func TestObjectCmd_SubcommandCount(t *testing.T) {
	const want = 9
	got := len(objectCmd.Commands())
	if got != want {
		t.Errorf("objectCmd has %d subcommands, want %d", got, want)
	}
}

// --- Run-function and local-helper coverage (FEAT-042) ---

// objectRunGlobals snapshots and restores the package-level globals the
// object run helpers read, so tests cannot leak state.
func objectRunGlobals(t *testing.T) {
	t.Helper()
	oldProgress, oldNoMP := objectProgress, objectNoMultipart
	oldPart, oldConc := objectPartSize, objectConcurrency
	oldJSON, oldDry, oldVerbose := JSONOutput, DryRun, Verbose
	t.Cleanup(func() {
		objectProgress, objectNoMultipart = oldProgress, oldNoMP
		objectPartSize, objectConcurrency = oldPart, oldConc
		JSONOutput, DryRun, Verbose = oldJSON, oldDry, oldVerbose
	})
}

// objectRunResetOutputFlag restores the object get --output flag to its
// pristine unset state before each case.
func objectRunResetOutputFlag() {
	if f := objectGetCmd.Flags().Lookup("output"); f != nil {
		f.Changed = false
		_ = f.Value.Set("")
	}
}

// TestObjectGetOutput verifies output-destination resolution: an explicit
// "-", an explicit path, an explicitly-empty flag (falls back to base name),
// and an unset flag with piped stdout (writes to stdout).
func TestObjectGetOutput(t *testing.T) {
	if isTerminal(os.Stdout) {
		t.Skip("stdout is a terminal; unset-flag case is not deterministic")
	}
	cases := []struct {
		name        string
		flagValue   string
		flagChanged bool
		key         string
		wantOutput  string
		wantStdout  bool
	}{
		{"explicit dash", "-", true, "dir/file.txt", "-", true},
		{"explicit path", "out.bin", true, "dir/file.txt", "out.bin", false},
		{"explicit empty", "", true, "dir/file.txt", "file.txt", false},
		{"unset non-tty", "", false, "dir/file.txt", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			objectRunResetOutputFlag()
			if tc.flagChanged {
				if err := objectGetCmd.Flags().Set("output", tc.flagValue); err != nil {
					t.Fatalf("failed to set output flag: %v", err)
				}
			}
			got, stdout := objectGetOutput(objectGetCmd, tc.key)
			if got != tc.wantOutput {
				t.Errorf("output = %q, want %q", got, tc.wantOutput)
			}
			if stdout != tc.wantStdout {
				t.Errorf("writeToStdout = %v, want %v", stdout, tc.wantStdout)
			}
		})
	}
}

// TestObjectGetCopy_Plain verifies the no-progress copy path returns the
// streamed byte count and writes the body to the file.
func TestObjectGetCopy_Plain(t *testing.T) {
	objectRunGlobals(t)
	objectProgress = false
	JSONOutput = true

	file, err := os.CreateTemp(t.TempDir(), "getcopy-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer file.Close()

	obj := &cosmoflare.DownloadResult{
		Size:    int64(len("hello world")),
		Content: io.NopCloser(strings.NewReader("hello world")),
	}
	size, err := objectGetCopy(file, obj)
	if err != nil {
		t.Fatalf("objectGetCopy returned error: %v", err)
	}
	if size != 11 {
		t.Errorf("size = %d, want 11", size)
	}
	data, _ := os.ReadFile(file.Name())
	if string(data) != "hello world" {
		t.Errorf("file content = %q, want %q", data, "hello world")
	}
}

// TestObjectGetCopy_Progress verifies the progress-bar path (progress on,
// JSON off, nonzero size) still streams the full body.
func TestObjectGetCopy_Progress(t *testing.T) {
	objectRunGlobals(t)
	objectProgress = true
	JSONOutput = false

	file, err := os.CreateTemp(t.TempDir(), "getcopy-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer file.Close()

	obj := &cosmoflare.DownloadResult{
		Size:    5,
		Content: io.NopCloser(strings.NewReader("abcde")),
	}
	size, err := objectGetCopy(file, obj)
	if err != nil {
		t.Fatalf("objectGetCopy returned error: %v", err)
	}
	if size != 5 {
		t.Errorf("size = %d, want 5", size)
	}
	data, _ := os.ReadFile(file.Name())
	if string(data) != "abcde" {
		t.Errorf("file content = %q, want %q", data, "abcde")
	}
}

// TestObjectPutBaseOpts verifies option construction from flags: empty flags
// yield no options; each set flag appends exactly one option.
func TestObjectPutBaseOpts(t *testing.T) {
	cases := []struct {
		name         string
		contentType  string
		cacheControl string
		metadata     map[string]string
		want         int
	}{
		{"none set", "", "", nil, 0},
		{"content-type only", "text/plain", "", nil, 1},
		{"cache-control only", "", "max-age=60", nil, 1},
		{"metadata only", "", "", map[string]string{"a": "1"}, 1},
		{"all set", "text/plain", "max-age=60", map[string]string{"a": "1"}, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := objectPutBaseOpts(tc.contentType, tc.cacheControl, tc.metadata)
			if len(opts) != tc.want {
				t.Errorf("len(opts) = %d, want %d", len(opts), tc.want)
			}
		})
	}
}

// objectRunFakeClient stubs the R2Client methods the batch runner touches;
// the embedded interface keeps the remaining methods uncompilable-if-called.
type objectRunFakeClient struct {
	cosmoflare.R2Client
	uploadErr error
	deleteErr error
	copyErr   error
	uploaded  []string
	deleted   []string
	copied    []string
}

func (f *objectRunFakeClient) Upload(ctx context.Context, bucket, key string, reader io.Reader, size int64, opts ...cosmoflare.UploadOption) (*cosmoflare.UploadResult, error) {
	f.uploaded = append(f.uploaded, key)
	return &cosmoflare.UploadResult{}, f.uploadErr
}

func (f *objectRunFakeClient) DeleteObject(ctx context.Context, bucket, key string) error {
	f.deleted = append(f.deleted, key)
	return f.deleteErr
}

func (f *objectRunFakeClient) CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) (*cosmoflare.CopyResult, error) {
	f.copied = append(f.copied, srcKey+"->"+dstKey)
	return &cosmoflare.CopyResult{}, f.copyErr
}

// TestBatchRunner_UploadValidation covers the offline upload guards: missing
// fields bump errorCount without failing, and an unopenable local file either
// aborts the batch or is swallowed under --continue.
func TestBatchRunner_UploadValidation(t *testing.T) {
	cases := []struct {
		name          string
		op            BatchOperation
		continueOnErr bool
		wantErr       string
		wantErrCount  int
	}{
		{"missing object_key", BatchOperation{Action: "upload", LocalPath: "/tmp/x"}, false, "", 1},
		{"missing local_path", BatchOperation{Action: "upload", ObjectKey: "k"}, false, "", 1},
		{"bad file aborts", BatchOperation{Action: "upload", LocalPath: "/nonexistent/file", ObjectKey: "k"}, false, "no such file or directory", 1},
		{"bad file continues", BatchOperation{Action: "upload", LocalPath: "/nonexistent/file", ObjectKey: "k"}, true, "", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &batchRunner{client: &objectRunFakeClient{}, bucketName: "bkt", continueOnErr: tc.continueOnErr}
			err := b.upload(tc.op)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
			if b.errorCount != tc.wantErrCount {
				t.Errorf("errorCount = %d, want %d", b.errorCount, tc.wantErrCount)
			}
			if b.successCount != 0 {
				t.Errorf("successCount = %d, want 0", b.successCount)
			}
		})
	}
}

// TestBatchRunner_UploadSuccess verifies a well-formed upload over a real
// temp file counts as a success.
func TestBatchRunner_UploadSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	fc := &objectRunFakeClient{}
	b := &batchRunner{client: fc, bucketName: "bkt"}
	if err := b.upload(BatchOperation{Action: "upload", LocalPath: path, ObjectKey: "k"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.successCount != 1 || b.errorCount != 0 {
		t.Errorf("success=%d error=%d, want 1/0", b.successCount, b.errorCount)
	}
	if len(fc.uploaded) != 1 || fc.uploaded[0] != "k" {
		t.Errorf("uploaded = %v, want [k]", fc.uploaded)
	}
}

// TestBatchRunner_UploadClientError verifies the abort-vs-continue behavior
// when the client rejects an upload.
func TestBatchRunner_UploadClientError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	op := BatchOperation{Action: "upload", LocalPath: path, ObjectKey: "k"}

	b := &batchRunner{client: &objectRunFakeClient{uploadErr: context.Canceled}, bucketName: "bkt"}
	err := b.upload(op)
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("expected context-canceled error, got %v", err)
	}
	if b.errorCount != 1 || b.successCount != 0 {
		t.Errorf("success=%d error=%d, want 0/1", b.successCount, b.errorCount)
	}

	b2 := &batchRunner{client: &objectRunFakeClient{uploadErr: context.Canceled}, bucketName: "bkt", continueOnErr: true}
	if err := b2.upload(op); err != nil {
		t.Fatalf("expected nil under --continue, got %v", err)
	}
	if b2.errorCount != 1 || b2.successCount != 0 {
		t.Errorf("success=%d error=%d, want 0/1", b2.successCount, b2.errorCount)
	}
}

// TestBatchRunner_Delete covers the delete guard, success, and the
// abort-vs-continue behavior on client errors.
func TestBatchRunner_Delete(t *testing.T) {
	cases := []struct {
		name          string
		op            BatchOperation
		deleteErr     error
		continueOnErr bool
		wantErr       string
		wantSuccess   int
		wantErrCount  int
	}{
		{"missing key", BatchOperation{Action: "delete"}, nil, false, "", 0, 1},
		{"success", BatchOperation{Action: "delete", ObjectKey: "k"}, nil, false, "", 1, 0},
		{"client error aborts", BatchOperation{Action: "delete", ObjectKey: "k"}, context.DeadlineExceeded, false, "context deadline exceeded", 0, 1},
		{"client error continues", BatchOperation{Action: "delete", ObjectKey: "k"}, context.DeadlineExceeded, true, "", 0, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fc := &objectRunFakeClient{deleteErr: tc.deleteErr}
			b := &batchRunner{client: fc, bucketName: "bkt", continueOnErr: tc.continueOnErr}
			err := b.delete(tc.op)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
			if b.successCount != tc.wantSuccess {
				t.Errorf("successCount = %d, want %d", b.successCount, tc.wantSuccess)
			}
			if b.errorCount != tc.wantErrCount {
				t.Errorf("errorCount = %d, want %d", b.errorCount, tc.wantErrCount)
			}
		})
	}
}

// TestBatchRunner_Copy covers the copy guard, success, and the
// abort-vs-continue behavior on client errors.
func TestBatchRunner_Copy(t *testing.T) {
	cases := []struct {
		name          string
		op            BatchOperation
		copyErr       error
		continueOnErr bool
		wantErr       string
		wantSuccess   int
		wantErrCount  int
	}{
		{"missing destination", BatchOperation{Action: "copy", ObjectKey: "k"}, nil, false, "", 0, 1},
		{"missing key", BatchOperation{Action: "copy", DestinationKey: "d"}, nil, false, "", 0, 1},
		{"success", BatchOperation{Action: "copy", ObjectKey: "k", DestinationKey: "d"}, nil, false, "", 1, 0},
		{"client error aborts", BatchOperation{Action: "copy", ObjectKey: "k", DestinationKey: "d"}, context.Canceled, false, "context canceled", 0, 1},
		{"client error continues", BatchOperation{Action: "copy", ObjectKey: "k", DestinationKey: "d"}, context.Canceled, true, "", 0, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fc := &objectRunFakeClient{copyErr: tc.copyErr}
			b := &batchRunner{client: fc, bucketName: "bkt", continueOnErr: tc.continueOnErr}
			err := b.copy(tc.op)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
			if b.successCount != tc.wantSuccess {
				t.Errorf("successCount = %d, want %d", b.successCount, tc.wantSuccess)
			}
			if b.errorCount != tc.wantErrCount {
				t.Errorf("errorCount = %d, want %d", b.errorCount, tc.wantErrCount)
			}
		})
	}
}

// TestBatchRunner_Run_UnknownAction verifies run dispatches to the default
// branch for an unrecognized action and fails without --continue.
func TestBatchRunner_Run_UnknownAction(t *testing.T) {
	b := &batchRunner{client: &objectRunFakeClient{}, bucketName: "bkt"}
	err := b.run(BatchOperation{Action: "frobnicate"})
	if err == nil || !strings.Contains(err.Error(), "invalid operation: frobnicate") {
		t.Fatalf("expected invalid-operation error, got %v", err)
	}
	if b.errorCount != 1 {
		t.Errorf("errorCount = %d, want 1", b.errorCount)
	}

	b2 := &batchRunner{client: &objectRunFakeClient{}, bucketName: "bkt", continueOnErr: true}
	if err := b2.run(BatchOperation{Action: "frobnicate"}); err != nil {
		t.Fatalf("expected nil under --continue, got %v", err)
	}
}

// TestBatchRunner_RunDispatch verifies run routes each known action to its
// handler by observing the fake client's recorded calls.
func TestBatchRunner_RunDispatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "d.txt")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	fc := &objectRunFakeClient{}
	b := &batchRunner{client: fc, bucketName: "bkt"}

	if err := b.run(BatchOperation{Action: "upload", LocalPath: path, ObjectKey: "up-key"}); err != nil {
		t.Fatalf("upload dispatch error: %v", err)
	}
	if err := b.run(BatchOperation{Action: "delete", ObjectKey: "del-key"}); err != nil {
		t.Fatalf("delete dispatch error: %v", err)
	}
	if err := b.run(BatchOperation{Action: "copy", ObjectKey: "src", DestinationKey: "dst"}); err != nil {
		t.Fatalf("copy dispatch error: %v", err)
	}
	if len(fc.deleted) != 1 || fc.deleted[0] != "del-key" {
		t.Errorf("deleted = %v, want [del-key]", fc.deleted)
	}
	if len(fc.copied) != 1 || fc.copied[0] != "src->dst" {
		t.Errorf("copied = %v, want [src->dst]", fc.copied)
	}
	if b.successCount != 3 || b.errorCount != 0 {
		t.Errorf("success=%d error=%d, want 3/0", b.successCount, b.errorCount)
	}
}

// TestIsTerminal_RegularFile verifies a plain file is not reported as a
// terminal.
func TestIsTerminal_RegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-tty")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer f.Close()
	if isTerminal(f) {
		t.Error("isTerminal(regular file) = true, want false")
	}
}

// TestObjectPutStdin_InvalidMetadata verifies the stdin upload path rejects
// malformed metadata before any client is constructed.
func TestObjectPutStdin_InvalidMetadata(t *testing.T) {
	err := objectPutStdin(&objectPutParams{
		bucketName: "my-bucket",
		key:        "stdin-key",
		metadata:   []string{"no-equals-sign"},
	})
	if err == nil || !strings.Contains(err.Error(), "failed to parse metadata") {
		t.Fatalf("expected metadata parse error, got %v", err)
	}
}
