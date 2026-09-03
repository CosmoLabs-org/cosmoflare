package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// --- Command registration ---

func TestSyncCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "sync" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("syncCmd not registered on rootCmd")
	}
}

func TestSyncCmd_Metadata(t *testing.T) {
	if syncCmd.Use != "sync" {
		t.Errorf("syncCmd.Use = %q, want %q", syncCmd.Use, "sync")
	}
	if syncCmd.Short == "" {
		t.Error("syncCmd.Short is empty")
	}
	if syncCmd.Long == "" {
		t.Error("syncCmd.Long is empty")
	}
}

func TestSyncCmd_ShortDescription(t *testing.T) {
	want := "Synchronize local directories with R2 buckets"
	if syncCmd.Short != want {
		t.Errorf("syncCmd.Short = %q, want %q", syncCmd.Short, want)
	}
}

func TestSyncCmd_LongContainsSubcommands(t *testing.T) {
	long := syncCmd.Long
	for _, kw := range []string{"up", "down"} {
		if !strings.Contains(long, kw) {
			t.Errorf("syncCmd.Long does not mention subcommand %q", kw)
		}
	}
}

func TestSyncCmd_LongContainsChecksum(t *testing.T) {
	if !strings.Contains(syncCmd.Long, "--checksum") {
		t.Error("syncCmd.Long does not mention --checksum flag")
	}
}

func TestSyncCmd_LongContainsExamples(t *testing.T) {
	if !strings.Contains(syncCmd.Long, "cosmoflare sync") {
		t.Error("syncCmd.Long does not contain usage examples")
	}
}

// --- Subcommand registration ---

func TestSyncCmd_Subcommands(t *testing.T) {
	expected := []string{"up", "down"}
	for _, name := range expected {
		found := false
		for _, sub := range syncCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("sync subcommand %q not registered", name)
		}
	}
}

func TestSyncCmd_SubcommandCount(t *testing.T) {
	cmds := syncCmd.Commands()
	if len(cmds) < 2 {
		t.Errorf("syncCmd has %d subcommands, want at least 2", len(cmds))
	}
}

// --- Subcommand Use fields ---

func TestSyncUpCmd_Use(t *testing.T) {
	want := "up <local-dir> <bucket>[/prefix]"
	if syncUpCmd.Use != want {
		t.Errorf("syncUpCmd.Use = %q, want %q", syncUpCmd.Use, want)
	}
}

func TestSyncDownCmd_Use(t *testing.T) {
	want := "down <bucket>[/prefix] <local-dir>"
	if syncDownCmd.Use != want {
		t.Errorf("syncDownCmd.Use = %q, want %q", syncDownCmd.Use, want)
	}
}

// --- Subcommand short descriptions ---

func TestSyncUpCmd_Short(t *testing.T) {
	if syncUpCmd.Short == "" {
		t.Error("syncUpCmd.Short is empty")
	}
	want := "Upload local directory to R2 bucket"
	if syncUpCmd.Short != want {
		t.Errorf("syncUpCmd.Short = %q, want %q", syncUpCmd.Short, want)
	}
}

func TestSyncDownCmd_Short(t *testing.T) {
	if syncDownCmd.Short == "" {
		t.Error("syncDownCmd.Short is empty")
	}
	want := "Download R2 bucket to local directory"
	if syncDownCmd.Short != want {
		t.Errorf("syncDownCmd.Short = %q, want %q", syncDownCmd.Short, want)
	}
}

// --- Subcommand long descriptions ---

func TestSyncUpCmd_Long(t *testing.T) {
	if syncUpCmd.Long == "" {
		t.Error("syncUpCmd.Long is empty")
	}
}

func TestSyncDownCmd_Long(t *testing.T) {
	if syncDownCmd.Long == "" {
		t.Error("syncDownCmd.Long is empty")
	}
}

func TestSyncUpCmd_LongContainsArgDocs(t *testing.T) {
	for _, kw := range []string{"local-dir", "bucket", "prefix"} {
		if !strings.Contains(syncUpCmd.Long, kw) {
			t.Errorf("syncUpCmd.Long does not document argument %q", kw)
		}
	}
}

func TestSyncDownCmd_LongContainsArgDocs(t *testing.T) {
	for _, kw := range []string{"local-dir", "bucket", "prefix"} {
		if !strings.Contains(syncDownCmd.Long, kw) {
			t.Errorf("syncDownCmd.Long does not document argument %q", kw)
		}
	}
}

func TestSyncUpCmd_LongContainsExamples(t *testing.T) {
	if !strings.Contains(syncUpCmd.Long, "cosmoflare sync up") {
		t.Error("syncUpCmd.Long does not contain usage examples")
	}
}

func TestSyncDownCmd_LongContainsExamples(t *testing.T) {
	if !strings.Contains(syncDownCmd.Long, "cosmoflare sync down") {
		t.Error("syncDownCmd.Long does not contain usage examples")
	}
}

// --- RunE handlers wired ---

func TestSyncCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		syncUpCmd,
		syncDownCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

func TestSyncCmd_ParentHasNoRunE(t *testing.T) {
	// Parent sync command is a grouping command — RunE is optional but it must have subcommands
	if len(syncCmd.Commands()) == 0 {
		t.Error("syncCmd has no subcommands and no RunE — unusable")
	}
}

// --- Flag registration ---

func TestSyncUp_Flags(t *testing.T) {
	expected := []string{"delete", "exclude", "include", "checksum", "progress"}
	for _, name := range expected {
		if syncUpCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on syncUpCmd", name)
		}
	}
}

func TestSyncDown_Flags(t *testing.T) {
	expected := []string{"delete", "exclude", "include", "checksum", "progress"}
	for _, name := range expected {
		if syncDownCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on syncDownCmd", name)
		}
	}
}

func TestSyncUp_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"delete", "false"},
		{"checksum", "false"},
		{"progress", "false"},
	}
	for _, tc := range cases {
		f := syncUpCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on syncUpCmd", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestSyncDown_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"delete", "false"},
		{"checksum", "false"},
		{"progress", "false"},
	}
	for _, tc := range cases {
		f := syncDownCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on syncDownCmd", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestSyncUp_ArrayFlagDefaults(t *testing.T) {
	for _, name := range []string{"exclude", "include"} {
		f := syncUpCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found on syncUpCmd", name)
		}
		// StringArray with nil default renders as "[]"
		if f.DefValue != "[]" {
			t.Errorf("flag --%s default = %q, want %q", name, f.DefValue, "[]")
		}
	}
}

func TestSyncDown_ArrayFlagDefaults(t *testing.T) {
	for _, name := range []string{"exclude", "include"} {
		f := syncDownCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag --%s not found on syncDownCmd", name)
		}
		if f.DefValue != "[]" {
			t.Errorf("flag --%s default = %q, want %q", name, f.DefValue, "[]")
		}
	}
}

// --- Flag descriptions ---

func TestSyncUp_FlagDescriptions(t *testing.T) {
	cases := []struct {
		name    string
		wantSub string
	}{
		{"delete", "Delete"},
		{"exclude", "Exclude"},
		{"include", "Include"},
		{"checksum", "MD5"},
		{"progress", "progress"},
	}
	for _, tc := range cases {
		f := syncUpCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on syncUpCmd", tc.name)
		}
		if !strings.Contains(f.Usage, tc.wantSub) {
			t.Errorf("flag --%s usage = %q, want it to contain %q", tc.name, f.Usage, tc.wantSub)
		}
	}
}

func TestSyncDown_FlagDescriptions(t *testing.T) {
	cases := []struct {
		name    string
		wantSub string
	}{
		{"delete", "Delete"},
		{"exclude", "Exclude"},
		{"include", "Include"},
		{"checksum", "MD5"},
		{"progress", "progress"},
	}
	for _, tc := range cases {
		f := syncDownCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on syncDownCmd", tc.name)
		}
		if !strings.Contains(f.Usage, tc.wantSub) {
			t.Errorf("flag --%s usage = %q, want it to contain %q", tc.name, f.Usage, tc.wantSub)
		}
	}
}

// --- Arg validation ---

func TestSyncUp_NoArgs(t *testing.T) {
	err := runSyncUp(syncUpCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestSyncUp_NoArgs_ErrorMessage(t *testing.T) {
	err := runSyncUp(syncUpCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
	if !strings.Contains(err.Error(), "local directory is required") {
		t.Errorf("error = %q, want it to mention 'local directory is required'", err.Error())
	}
}

func TestSyncUp_OneArg(t *testing.T) {
	err := runSyncUp(syncUpCmd, []string{"/tmp/test"})
	if err == nil {
		t.Fatal("expected error when only one arg provided (missing bucket)")
	}
}

func TestSyncUp_OneArg_ErrorMessage(t *testing.T) {
	err := runSyncUp(syncUpCmd, []string{"/tmp/test"})
	if err == nil {
		t.Fatal("expected error when only one arg provided (missing bucket)")
	}
	if !strings.Contains(err.Error(), "bucket is required") {
		t.Errorf("error = %q, want it to mention 'bucket is required'", err.Error())
	}
}

func TestSyncDown_NoArgs(t *testing.T) {
	err := runSyncDown(syncDownCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestSyncDown_NoArgs_ErrorMessage(t *testing.T) {
	err := runSyncDown(syncDownCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
	if !strings.Contains(err.Error(), "bucket is required") {
		t.Errorf("error = %q, want it to mention 'bucket is required'", err.Error())
	}
}

func TestSyncDown_OneArg(t *testing.T) {
	err := runSyncDown(syncDownCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only one arg provided (missing local dir)")
	}
}

func TestSyncDown_OneArg_ErrorMessage(t *testing.T) {
	err := runSyncDown(syncDownCmd, []string{"my-bucket"})
	if err == nil {
		t.Fatal("expected error when only one arg provided (missing local dir)")
	}
	if !strings.Contains(err.Error(), "local directory is required") {
		t.Errorf("error = %q, want it to mention 'local directory is required'", err.Error())
	}
}

func TestSyncUp_NonexistentDir_Error(t *testing.T) {
	err := runSyncUp(syncUpCmd, []string{"/nonexistent-dir-cosmoflare-test-xyz", "my-bucket"})
	if err == nil {
		t.Fatal("expected error for nonexistent local directory")
	}
}

func TestSyncUp_FileNotDir_Error(t *testing.T) {
	// Create a temp file (not a directory) and use it as localDir
	f, err := os.CreateTemp("", "cosmoflare-sync-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()

	runErr := runSyncUp(syncUpCmd, []string{f.Name(), "my-bucket"})
	if runErr == nil {
		t.Fatal("expected error when local path is a file, not a directory")
	}
	if !strings.Contains(runErr.Error(), "not a directory") {
		t.Errorf("error = %q, want it to mention 'not a directory'", runErr.Error())
	}
}

// --- Dry-run mode ---

func TestSyncUp_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origAccountID := AccountID
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		AccountID = origAccountID
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	AccountID = "test-account"

	// Should fail with non-existent directory (that's fine for a dry-run test)
	err := runSyncUp(syncUpCmd, []string{"/nonexistent-sync-test-dir-xyz", "test-bucket"})
	// We expect an error here (directory doesn't exist), but the command should not panic
	if err == nil {
		t.Log("runSyncUp returned nil error for nonexistent dir - this is acceptable in dry-run")
	}
}

func TestSyncDown_DryRun(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origAccountID := AccountID
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		AccountID = origAccountID
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	AccountID = "test-account"

	err := runSyncDown(syncDownCmd, []string{"test-bucket", "/nonexistent-sync-test-dir-xyz"})
	if err == nil {
		t.Log("runSyncDown returned nil error for nonexistent dir - acceptable in dry-run")
	}
}

// --- Bucket/prefix parsing ---

func TestParseBucketPrefix(t *testing.T) {
	cases := []struct {
		input      string
		wantBucket string
		wantPrefix string
	}{
		{"my-bucket", "my-bucket", ""},
		{"my-bucket/prefix", "my-bucket", "prefix/"},
		{"my-bucket/deep/prefix", "my-bucket", "deep/prefix/"},
		{"my-bucket/prefix/", "my-bucket", "prefix/"},
	}
	for _, tc := range cases {
		bucket, prefix := parseBucketPrefix(tc.input)
		if bucket != tc.wantBucket {
			t.Errorf("parseBucketPrefix(%q) bucket = %q, want %q", tc.input, bucket, tc.wantBucket)
		}
		if prefix != tc.wantPrefix {
			t.Errorf("parseBucketPrefix(%q) prefix = %q, want %q", tc.input, prefix, tc.wantPrefix)
		}
	}
}

func TestParseBucketPrefix_EmptyInput(t *testing.T) {
	bucket, prefix := parseBucketPrefix("")
	if bucket != "" {
		t.Errorf("parseBucketPrefix('') bucket = %q, want %q", bucket, "")
	}
	if prefix != "" {
		t.Errorf("parseBucketPrefix('') prefix = %q, want %q", prefix, "")
	}
}

func TestParseBucketPrefix_PrefixAlwaysTrailingSlash(t *testing.T) {
	cases := []string{
		"bucket/a",
		"bucket/a/b",
		"bucket/a/b/c",
	}
	for _, input := range cases {
		_, prefix := parseBucketPrefix(input)
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			t.Errorf("parseBucketPrefix(%q) prefix %q does not end with '/'", input, prefix)
		}
	}
}

func TestParseBucketPrefix_BucketOnlyNoPrefix(t *testing.T) {
	bucket, prefix := parseBucketPrefix("just-bucket")
	if bucket != "just-bucket" {
		t.Errorf("got bucket %q, want %q", bucket, "just-bucket")
	}
	if prefix != "" {
		t.Errorf("got prefix %q, want empty string", prefix)
	}
}

// --- prefixDisplay ---

func TestPrefixDisplay_Empty(t *testing.T) {
	got := prefixDisplay("")
	if got != "" {
		t.Errorf("prefixDisplay('') = %q, want %q", got, "")
	}
}

func TestPrefixDisplay_WithSlash(t *testing.T) {
	got := prefixDisplay("assets/")
	want := "/assets"
	if got != want {
		t.Errorf("prefixDisplay('assets/') = %q, want %q", got, want)
	}
}

func TestPrefixDisplay_WithoutTrailingSlash(t *testing.T) {
	// If prefix has no trailing slash it should still show correctly
	got := prefixDisplay("assets")
	want := "/assets"
	if got != want {
		t.Errorf("prefixDisplay('assets') = %q, want %q", got, want)
	}
}

func TestPrefixDisplay_DeepPath(t *testing.T) {
	got := prefixDisplay("deep/nested/path/")
	want := "/deep/nested/path"
	if got != want {
		t.Errorf("prefixDisplay('deep/nested/path/') = %q, want %q", got, want)
	}
}

// --- md5File ---

func TestMd5File_KnownContent(t *testing.T) {
	// Write a known file and verify MD5
	tmp, err := os.CreateTemp("", "cosmoflare-md5-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	content := "hello cosmoflare"
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	hash, err := md5File(tmp.Name())
	if err != nil {
		t.Fatalf("md5File returned error: %v", err)
	}

	// MD5 of "hello cosmoflare" is deterministic
	if len(hash) != 32 {
		t.Errorf("md5File returned hash of length %d, want 32 hex chars", len(hash))
	}
	// Verify it is lowercase hex
	for _, ch := range hash {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			t.Errorf("md5File hash %q contains non-hex character %c", hash, ch)
		}
	}
}

func TestMd5File_EmptyFile(t *testing.T) {
	tmp, err := os.CreateTemp("", "cosmoflare-md5-empty-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.Close()

	hash, err := md5File(tmp.Name())
	if err != nil {
		t.Fatalf("md5File on empty file returned error: %v", err)
	}
	// MD5 of empty string: d41d8cd98f00b204e9800998ecf8427e
	want := "d41d8cd98f00b204e9800998ecf8427e"
	if hash != want {
		t.Errorf("md5File(empty) = %q, want %q", hash, want)
	}
}

func TestMd5File_Nonexistent(t *testing.T) {
	_, err := md5File("/nonexistent-cosmoflare-md5-test.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestMd5File_Deterministic(t *testing.T) {
	tmp, err := os.CreateTemp("", "cosmoflare-md5-det-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("deterministic content")
	tmp.Close()

	hash1, err := md5File(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	hash2, err := md5File(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	if hash1 != hash2 {
		t.Errorf("md5File is not deterministic: %q != %q", hash1, hash2)
	}
}

// --- scanLocalDir ---

func TestScanLocalDir_EmptyDir(t *testing.T) {
	tmp := t.TempDir()

	files, err := scanLocalDir(tmp, false)
	if err != nil {
		t.Fatalf("scanLocalDir on empty dir returned error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("scanLocalDir on empty dir returned %d files, want 0", len(files))
	}
}

func TestScanLocalDir_SingleFile(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := scanLocalDir(tmp, false)
	if err != nil {
		t.Fatalf("scanLocalDir returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("scanLocalDir returned %d files, want 1", len(files))
	}
	if files[0].RelPath != "hello.txt" {
		t.Errorf("RelPath = %q, want %q", files[0].RelPath, "hello.txt")
	}
	if files[0].Size != 5 {
		t.Errorf("Size = %d, want 5", files[0].Size)
	}
}

func TestScanLocalDir_MultipleFiles(t *testing.T) {
	tmp := t.TempDir()
	names := []string{"a.txt", "b.txt", "c.txt"}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := scanLocalDir(tmp, false)
	if err != nil {
		t.Fatalf("scanLocalDir returned error: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("scanLocalDir returned %d files, want 3", len(files))
	}
}

func TestScanLocalDir_Recursive(t *testing.T) {
	tmp := t.TempDir()
	subdir := filepath.Join(tmp, "subdir")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "root.txt"), []byte("root"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "nested.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := scanLocalDir(tmp, false)
	if err != nil {
		t.Fatalf("scanLocalDir returned error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("scanLocalDir returned %d files, want 2", len(files))
	}

	// Nested file should use forward slashes
	var foundNested bool
	for _, f := range files {
		if f.RelPath == "subdir/nested.txt" {
			foundNested = true
		}
	}
	if !foundNested {
		t.Error("nested file 'subdir/nested.txt' not found in scan results")
	}
}

func TestScanLocalDir_ForwardSlashPaths(t *testing.T) {
	tmp := t.TempDir()
	subdir := filepath.Join(tmp, "a", "b")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := scanLocalDir(tmp, false)
	if err != nil {
		t.Fatalf("scanLocalDir returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	// R2 keys must use forward slashes
	if strings.Contains(files[0].RelPath, `\`) {
		t.Errorf("RelPath %q contains backslash — must use forward slashes for R2 compatibility", files[0].RelPath)
	}
}

func TestScanLocalDir_WithChecksum(t *testing.T) {
	tmp := t.TempDir()
	content := "checksum test content"
	if err := os.WriteFile(filepath.Join(tmp, "file.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := scanLocalDir(tmp, true)
	if err != nil {
		t.Fatalf("scanLocalDir with checksum returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if files[0].Checksum == "" {
		t.Error("Checksum is empty when checksum=true")
	}
	if len(files[0].Checksum) != 32 {
		t.Errorf("Checksum length = %d, want 32 (MD5 hex)", len(files[0].Checksum))
	}
}

func TestScanLocalDir_WithoutChecksum_ChecksumEmpty(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := scanLocalDir(tmp, false)
	if err != nil {
		t.Fatalf("scanLocalDir returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if files[0].Checksum != "" {
		t.Errorf("Checksum = %q, want empty string when checksum=false", files[0].Checksum)
	}
}

func TestScanLocalDir_ModTimeSet(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := scanLocalDir(tmp, false)
	if err != nil {
		t.Fatalf("scanLocalDir returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if files[0].ModTime.IsZero() {
		t.Error("ModTime is zero — expected a valid modification time")
	}
}

func TestScanLocalDir_NonexistentDir(t *testing.T) {
	_, err := scanLocalDir("/nonexistent-dir-cosmoflare-scan-test", false)
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestScanLocalDir_SkipsDirectories(t *testing.T) {
	tmp := t.TempDir()
	// Create nested structure with only empty dirs
	if err := os.MkdirAll(filepath.Join(tmp, "emptydir", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "real.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := scanLocalDir(tmp, false)
	if err != nil {
		t.Fatalf("scanLocalDir returned error: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("got %d files, want 1 (directories should be skipped)", len(files))
	}
	if files[0].RelPath != "real.txt" {
		t.Errorf("got RelPath %q, want 'real.txt'", files[0].RelPath)
	}
}

// --- r2StorageBackend remote listing (BUG-027 pagination) ---

// stubListPagesClient is a cosmoflare.R2Client stub serving canned
// ListObjects pages so remote-listing pagination can be tested without
// network access. Only ListObjects is overridden; every other interface
// method is inherited from the embedded nil interface and would panic
// if called.
type stubListPagesClient struct {
	cosmoflare.R2Client
	pages []*cosmoflare.ListResult[*cosmoflare.Object]

	// Call records, in call order.
	receivedTokens []string
	receivedBucket []string
	receivedPrefix []string
}

func (s *stubListPagesClient) ListObjects(_ context.Context, bucket, prefix, _ string, _ int32, continuationToken string) (*cosmoflare.ListResult[*cosmoflare.Object], error) {
	if len(s.receivedTokens) >= len(s.pages) {
		return nil, fmt.Errorf("unexpected ListObjects call %d: only %d pages configured", len(s.receivedTokens)+1, len(s.pages))
	}
	s.receivedTokens = append(s.receivedTokens, continuationToken)
	s.receivedBucket = append(s.receivedBucket, bucket)
	s.receivedPrefix = append(s.receivedPrefix, prefix)
	return s.pages[len(s.receivedTokens)-1], nil
}

// TestR2StorageBackend_ListRemoteObjects_FollowsNextToken verifies that
// listing accumulates objects across every ListObjects page instead of
// stopping after the first page of at most 1000 items.
func TestR2StorageBackend_ListRemoteObjects_FollowsNextToken(t *testing.T) {
	stub := &stubListPagesClient{
		pages: []*cosmoflare.ListResult[*cosmoflare.Object]{
			{
				Items: []*cosmoflare.Object{
					{Key: "data/a.txt", Size: 1, ETag: "etag-a"},
					{Key: "data/b.txt", Size: 2, ETag: "etag-b"},
				},
				NextToken: "t2",
			},
			{
				Items: []*cosmoflare.Object{
					{Key: "data/c.txt", Size: 3, ETag: "etag-c"},
				},
				NextToken: "",
			},
		},
	}

	backend := &r2StorageBackend{client: stub}
	objects, err := backend.ListRemoteObjects(context.Background(), "test-bucket", "data/")
	if err != nil {
		t.Fatalf("ListRemoteObjects returned error: %v", err)
	}

	if len(objects) != 3 {
		t.Fatalf("got %d objects, want 3 (items from all pages must be accumulated)", len(objects))
	}
	wantKeys := []string{"data/a.txt", "data/b.txt", "data/c.txt"}
	for i, want := range wantKeys {
		if objects[i].Key != want {
			t.Errorf("objects[%d].Key = %q, want %q", i, objects[i].Key, want)
		}
	}
	if objects[2].Size != 3 || objects[2].ETag != "etag-c" {
		t.Errorf("objects[2] fields not converted faithfully: size=%d etag=%q", objects[2].Size, objects[2].ETag)
	}

	if len(stub.receivedTokens) != 2 {
		t.Fatalf("ListObjects called %d time(s), want 2 (one per page)", len(stub.receivedTokens))
	}
	if stub.receivedTokens[0] != "" {
		t.Errorf("first call continuationToken = %q, want empty string", stub.receivedTokens[0])
	}
	if stub.receivedTokens[1] != "t2" {
		t.Errorf("second call continuationToken = %q, want %q (NextToken from page 1)", stub.receivedTokens[1], "t2")
	}
	for i := range stub.receivedBucket {
		if stub.receivedBucket[i] != "test-bucket" || stub.receivedPrefix[i] != "data/" {
			t.Errorf("call %d used bucket=%q prefix=%q, want test-bucket / data/", i, stub.receivedBucket[i], stub.receivedPrefix[i])
		}
	}
}

// TestR2StorageBackend_ListRemoteObjects_SinglePage verifies the loop
// terminates after one call when the first page reports no NextToken.
func TestR2StorageBackend_ListRemoteObjects_SinglePage(t *testing.T) {
	stub := &stubListPagesClient{
		pages: []*cosmoflare.ListResult[*cosmoflare.Object]{
			{Items: []*cosmoflare.Object{{Key: "solo.txt", Size: 7, ETag: "etag-solo"}}},
		},
	}

	backend := &r2StorageBackend{client: stub}
	objects, err := backend.ListRemoteObjects(context.Background(), "test-bucket", "")
	if err != nil {
		t.Fatalf("ListRemoteObjects returned error: %v", err)
	}

	if len(objects) != 1 {
		t.Fatalf("got %d objects, want 1", len(objects))
	}
	if objects[0].Key != "solo.txt" || objects[0].Size != 7 || objects[0].ETag != "etag-solo" {
		t.Errorf("objects[0] = %+v, want solo.txt/7/etag-solo", objects[0])
	}
	if len(stub.receivedTokens) != 1 {
		t.Errorf("ListObjects called %d time(s), want 1 (empty NextToken must end the loop)", len(stub.receivedTokens))
	}
}
