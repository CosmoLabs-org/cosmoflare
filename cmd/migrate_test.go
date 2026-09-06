/*
Package cmd provides migration and bulk operations for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"strings"
	"testing"
)

// --- migrateCmd registration ---

func TestMigrateCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "migrate" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("migrateCmd not registered on rootCmd")
	}
}

func TestMigrateCmd_HasParentCommand(t *testing.T) {
	if migrateCmd.Parent() == nil {
		t.Fatal("migrateCmd has no parent — must be a subcommand of rootCmd")
	}
	if migrateCmd.Parent().Name() != rootCmd.Name() {
		t.Errorf("migrateCmd parent = %q, want %q", migrateCmd.Parent().Name(), rootCmd.Name())
	}
}

// --- migrateCmd metadata ---

func TestMigrateCmd_Use(t *testing.T) {
	if migrateCmd.Use != "migrate" {
		t.Errorf("migrateCmd.Use = %q, want %q", migrateCmd.Use, "migrate")
	}
}

func TestMigrateCmd_ShortNotEmpty(t *testing.T) {
	if migrateCmd.Short == "" {
		t.Error("migrateCmd.Short is empty")
	}
}

func TestMigrateCmd_ShortDescribesMigration(t *testing.T) {
	short := strings.ToLower(migrateCmd.Short)
	if !strings.Contains(short, "migrat") && !strings.Contains(short, "bulk") {
		t.Errorf("migrateCmd.Short %q does not describe migration", migrateCmd.Short)
	}
}

func TestMigrateCmd_LongNotEmpty(t *testing.T) {
	if migrateCmd.Long == "" {
		t.Error("migrateCmd.Long is empty")
	}
}

func TestMigrateCmd_LongMentionsSubcommands(t *testing.T) {
	for _, sub := range []string{"from-s3", "sync", "backup", "restore", "batch"} {
		if !strings.Contains(migrateCmd.Long, sub) {
			t.Errorf("migrateCmd.Long does not mention subcommand %q", sub)
		}
	}
}

func TestMigrateCmd_LongHasExamples(t *testing.T) {
	if !strings.Contains(migrateCmd.Long, "cosmoflare") {
		t.Error("migrateCmd.Long should contain usage examples with 'cosmoflare'")
	}
}

func TestMigrateCmd_LongNoBinaryAlias(t *testing.T) {
	if strings.Contains(migrateCmd.Long, "r2go2") {
		t.Error("migrateCmd.Long contains legacy 'r2go2' binary name — should use 'cosmoflare'")
	}
}

func TestMigrateCmd_NotHidden(t *testing.T) {
	if migrateCmd.Hidden {
		t.Error("migrateCmd.Hidden is true — migrate should be visible in help")
	}
}

// --- Subcommand registration (all 5) ---

func TestMigrateCmd_AllSubcommandsRegistered(t *testing.T) {
	expected := []string{"from-s3", "sync", "backup", "restore", "batch"}
	for _, name := range expected {
		found := false
		for _, sub := range migrateCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("migrate subcommand %q not registered", name)
		}
	}
}

func TestMigrateCmd_HasExactlyFiveSubcommands(t *testing.T) {
	count := len(migrateCmd.Commands())
	if count != 5 {
		t.Errorf("migrateCmd has %d subcommands, want 5", count)
	}
}

// --- migrateFromS3Cmd metadata ---

func TestMigrateFromS3Cmd_UseContainsKeywords(t *testing.T) {
	use := migrateFromS3Cmd.Use
	if !strings.Contains(use, "from-s3") {
		t.Errorf("migrateFromS3Cmd.Use = %q, should contain 'from-s3'", use)
	}
	if !strings.Contains(use, "to-r2") {
		t.Errorf("migrateFromS3Cmd.Use = %q, should contain 'to-r2'", use)
	}
}

func TestMigrateFromS3Cmd_ShortDescribesS3(t *testing.T) {
	short := strings.ToLower(migrateFromS3Cmd.Short)
	if !strings.Contains(short, "s3") {
		t.Errorf("migrateFromS3Cmd.Short %q does not mention S3", migrateFromS3Cmd.Short)
	}
}

func TestMigrateFromS3Cmd_LongNotEmpty(t *testing.T) {
	if migrateFromS3Cmd.Long == "" {
		t.Error("migrateFromS3Cmd.Long is empty")
	}
}

func TestMigrateFromS3Cmd_LongHasExamples(t *testing.T) {
	if !strings.Contains(migrateFromS3Cmd.Long, "cosmoflare migrate from-s3") {
		t.Error("migrateFromS3Cmd.Long should contain usage example with 'cosmoflare migrate from-s3'")
	}
}

func TestMigrateFromS3Cmd_LongNoBinaryAlias(t *testing.T) {
	if strings.Contains(migrateFromS3Cmd.Long, "r2go2") {
		t.Error("migrateFromS3Cmd.Long contains legacy 'r2go2' — should use 'cosmoflare'")
	}
}

func TestMigrateFromS3Cmd_HasRunE(t *testing.T) {
	if migrateFromS3Cmd.RunE == nil {
		t.Error("migrateFromS3Cmd.RunE is nil — command has no handler")
	}
}

// --- migrateSyncCmd metadata ---

func TestMigrateSyncCmd_UseContainsSyncKeyword(t *testing.T) {
	if !strings.Contains(migrateCmd.Use, "migrate") {
		t.Errorf("migrateSyncCmd.Use = %q", migrateSyncCmd.Use)
	}
	if !strings.Contains(migrateSyncCmd.Use, "sync") {
		t.Errorf("migrateSyncCmd.Use = %q, should contain 'sync'", migrateSyncCmd.Use)
	}
}

func TestMigrateSyncCmd_ShortDescribesSync(t *testing.T) {
	short := strings.ToLower(migrateSyncCmd.Short)
	if !strings.Contains(short, "sync") {
		t.Errorf("migrateSyncCmd.Short %q does not describe sync", migrateSyncCmd.Short)
	}
}

func TestMigrateSyncCmd_LongNotEmpty(t *testing.T) {
	if migrateSyncCmd.Long == "" {
		t.Error("migrateSyncCmd.Long is empty")
	}
}

func TestMigrateSyncCmd_HasRunE(t *testing.T) {
	if migrateSyncCmd.RunE == nil {
		t.Error("migrateSyncCmd.RunE is nil — command has no handler")
	}
}

// --- migrateBackupCmd metadata ---

func TestMigrateBackupCmd_UseContainsBackup(t *testing.T) {
	if !strings.Contains(migrateBackupCmd.Use, "backup") {
		t.Errorf("migrateBackupCmd.Use = %q, should contain 'backup'", migrateBackupCmd.Use)
	}
}

func TestMigrateBackupCmd_ShortDescribesBackup(t *testing.T) {
	short := strings.ToLower(migrateBackupCmd.Short)
	if !strings.Contains(short, "backup") {
		t.Errorf("migrateBackupCmd.Short %q does not describe backup", migrateBackupCmd.Short)
	}
}

func TestMigrateBackupCmd_LongNotEmpty(t *testing.T) {
	if migrateBackupCmd.Long == "" {
		t.Error("migrateBackupCmd.Long is empty")
	}
}

func TestMigrateBackupCmd_HasRunE(t *testing.T) {
	if migrateBackupCmd.RunE == nil {
		t.Error("migrateBackupCmd.RunE is nil — command has no handler")
	}
}

// --- migrateRestoreCmd metadata ---

func TestMigrateRestoreCmd_UseContainsRestoreAndTo(t *testing.T) {
	if !strings.Contains(migrateRestoreCmd.Use, "restore") {
		t.Errorf("migrateRestoreCmd.Use = %q, should contain 'restore'", migrateRestoreCmd.Use)
	}
	if !strings.Contains(migrateRestoreCmd.Use, "to") {
		t.Errorf("migrateRestoreCmd.Use = %q, should contain 'to'", migrateRestoreCmd.Use)
	}
}

func TestMigrateRestoreCmd_ShortDescribesRestore(t *testing.T) {
	short := strings.ToLower(migrateRestoreCmd.Short)
	if !strings.Contains(short, "restore") {
		t.Errorf("migrateRestoreCmd.Short %q does not describe restore", migrateRestoreCmd.Short)
	}
}

func TestMigrateRestoreCmd_LongNotEmpty(t *testing.T) {
	if migrateRestoreCmd.Long == "" {
		t.Error("migrateRestoreCmd.Long is empty")
	}
}

func TestMigrateRestoreCmd_HasRunE(t *testing.T) {
	if migrateRestoreCmd.RunE == nil {
		t.Error("migrateRestoreCmd.RunE is nil — command has no handler")
	}
}

// --- migrateBatchCmd metadata ---

func TestMigrateBatchCmd_UseContainsBatch(t *testing.T) {
	if !strings.Contains(migrateBatchCmd.Use, "batch") {
		t.Errorf("migrateBatchCmd.Use = %q, should contain 'batch'", migrateBatchCmd.Use)
	}
}

func TestMigrateBatchCmd_ShortDescribesBatch(t *testing.T) {
	short := strings.ToLower(migrateBatchCmd.Short)
	if !strings.Contains(short, "batch") {
		t.Errorf("migrateBatchCmd.Short %q does not describe batch operations", migrateBatchCmd.Short)
	}
}

func TestMigrateBatchCmd_LongNotEmpty(t *testing.T) {
	if migrateBatchCmd.Long == "" {
		t.Error("migrateBatchCmd.Long is empty")
	}
}

func TestMigrateBatchCmd_HasRunE(t *testing.T) {
	if migrateBatchCmd.RunE == nil {
		t.Error("migrateBatchCmd.RunE is nil — command has no handler")
	}
}

// --- Flag registration on from-s3 ---

func TestMigrateFromS3Cmd_FlagFilter(t *testing.T) {
	f := migrateFromS3Cmd.Flags().Lookup("filter")
	if f == nil {
		t.Fatal("from-s3 flag --filter not registered")
	}
	if f.DefValue != "" {
		t.Errorf("--filter default = %q, want %q", f.DefValue, "")
	}
}

func TestMigrateFromS3Cmd_FlagConcurrency(t *testing.T) {
	f := migrateFromS3Cmd.Flags().Lookup("concurrency")
	if f == nil {
		t.Fatal("from-s3 flag --concurrency not registered")
	}
	if f.DefValue != "10" {
		t.Errorf("--concurrency default = %q, want %q", f.DefValue, "10")
	}
}

func TestMigrateFromS3Cmd_FlagResume(t *testing.T) {
	f := migrateFromS3Cmd.Flags().Lookup("resume")
	if f == nil {
		t.Fatal("from-s3 flag --resume not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--resume default = %q, want %q", f.DefValue, "false")
	}
}

func TestMigrateFromS3Cmd_FlagVerify(t *testing.T) {
	f := migrateFromS3Cmd.Flags().Lookup("verify")
	if f == nil {
		t.Fatal("from-s3 flag --verify not registered")
	}
	if f.DefValue != "true" {
		t.Errorf("--verify default = %q, want %q", f.DefValue, "true")
	}
}

func TestMigrateFromS3Cmd_FlagManifest(t *testing.T) {
	f := migrateFromS3Cmd.Flags().Lookup("manifest")
	if f == nil {
		t.Fatal("from-s3 flag --manifest not registered")
	}
	if f.DefValue != "" {
		t.Errorf("--manifest default = %q, want %q", f.DefValue, "")
	}
}

func TestMigrateFromS3Cmd_FlagAWSRegion(t *testing.T) {
	f := migrateFromS3Cmd.Flags().Lookup("aws-region")
	if f == nil {
		t.Fatal("from-s3 flag --aws-region not registered")
	}
	if f.DefValue != "us-east-1" {
		t.Errorf("--aws-region default = %q, want %q", f.DefValue, "us-east-1")
	}
}

func TestMigrateFromS3Cmd_FlagAWSProfile(t *testing.T) {
	f := migrateFromS3Cmd.Flags().Lookup("aws-profile")
	if f == nil {
		t.Fatal("from-s3 flag --aws-profile not registered")
	}
	if f.DefValue != "default" {
		t.Errorf("--aws-profile default = %q, want %q", f.DefValue, "default")
	}
}

func TestMigrateFromS3Cmd_FlagDeleteSource(t *testing.T) {
	f := migrateFromS3Cmd.Flags().Lookup("delete-source")
	if f == nil {
		t.Fatal("from-s3 flag --delete-source not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--delete-source default = %q, want %q", f.DefValue, "false")
	}
}

func TestMigrateFromS3Cmd_FlagCompress(t *testing.T) {
	f := migrateFromS3Cmd.Flags().Lookup("compress")
	if f == nil {
		t.Fatal("from-s3 flag --compress not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--compress default = %q, want %q", f.DefValue, "false")
	}
}

// --- Flag registration on sync ---

func TestMigrateSyncCmd_FlagDeleteExtras(t *testing.T) {
	f := migrateSyncCmd.Flags().Lookup("delete-extras")
	if f == nil {
		t.Fatal("sync flag --delete-extras not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--delete-extras default = %q, want %q", f.DefValue, "false")
	}
}

func TestMigrateSyncCmd_FlagFilter(t *testing.T) {
	f := migrateSyncCmd.Flags().Lookup("filter")
	if f == nil {
		t.Fatal("sync flag --filter not registered")
	}
}

func TestMigrateSyncCmd_FlagVerify(t *testing.T) {
	f := migrateSyncCmd.Flags().Lookup("verify")
	if f == nil {
		t.Fatal("sync flag --verify not registered")
	}
}

func TestMigrateSyncCmd_FlagDryRun(t *testing.T) {
	f := migrateSyncCmd.Flags().Lookup("dry-run")
	if f == nil {
		t.Fatal("sync flag --dry-run not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--dry-run default = %q, want %q", f.DefValue, "false")
	}
}

// --- Flag registration on backup ---

func TestMigrateBackupCmd_FlagToLocal(t *testing.T) {
	f := migrateBackupCmd.Flags().Lookup("to-local")
	if f == nil {
		t.Fatal("backup flag --to-local not registered")
	}
	if f.DefValue != "" {
		t.Errorf("--to-local default = %q, want %q", f.DefValue, "")
	}
}

func TestMigrateBackupCmd_FlagCompress(t *testing.T) {
	f := migrateBackupCmd.Flags().Lookup("compress")
	if f == nil {
		t.Fatal("backup flag --compress not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--compress default = %q, want %q", f.DefValue, "false")
	}
}

func TestMigrateBackupCmd_FlagIncludeVersions(t *testing.T) {
	f := migrateBackupCmd.Flags().Lookup("include-versions")
	if f == nil {
		t.Fatal("backup flag --include-versions not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--include-versions default = %q, want %q", f.DefValue, "false")
	}
}

func TestMigrateBackupCmd_FlagIncremental(t *testing.T) {
	f := migrateBackupCmd.Flags().Lookup("incremental")
	if f == nil {
		t.Fatal("backup flag --incremental not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--incremental default = %q, want %q", f.DefValue, "false")
	}
}

// --- Flag registration on restore ---

func TestMigrateRestoreCmd_FlagPath(t *testing.T) {
	f := migrateRestoreCmd.Flags().Lookup("path")
	if f == nil {
		t.Fatal("restore flag --path not registered")
	}
	if f.DefValue != "" {
		t.Errorf("--path default = %q, want %q", f.DefValue, "")
	}
}

func TestMigrateRestoreCmd_FlagVerify(t *testing.T) {
	f := migrateRestoreCmd.Flags().Lookup("verify")
	if f == nil {
		t.Fatal("restore flag --verify not registered")
	}
	if f.DefValue != "true" {
		t.Errorf("--verify default = %q, want %q", f.DefValue, "true")
	}
}

func TestMigrateRestoreCmd_FlagResume(t *testing.T) {
	f := migrateRestoreCmd.Flags().Lookup("resume")
	if f == nil {
		t.Fatal("restore flag --resume not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--resume default = %q, want %q", f.DefValue, "false")
	}
}

// --- Flag registration on batch ---

func TestMigrateBatchCmd_FlagContinue(t *testing.T) {
	f := migrateBatchCmd.Flags().Lookup("continue")
	if f == nil {
		t.Fatal("batch flag --continue not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--continue default = %q, want %q", f.DefValue, "false")
	}
}

func TestMigrateBatchCmd_FlagConcurrency(t *testing.T) {
	f := migrateBatchCmd.Flags().Lookup("concurrency")
	if f == nil {
		t.Fatal("batch flag --concurrency not registered")
	}
	if f.DefValue != "5" {
		t.Errorf("--concurrency default = %q, want %q", f.DefValue, "5")
	}
}

// --- parseMigrateFromS3Args ---

func TestParseMigrateFromS3Args_ValidInput(t *testing.T) {
	s3, r2, err := parseMigrateFromS3Args([]string{"my-s3-bucket", "to-r2", "ignored", "my-r2-bucket"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s3 != "my-s3-bucket" {
		t.Errorf("s3Bucket = %q, want %q", s3, "my-s3-bucket")
	}
	if r2 != "my-r2-bucket" {
		t.Errorf("r2Bucket = %q, want %q", r2, "my-r2-bucket")
	}
}

func TestParseMigrateFromS3Args_MissingArgs(t *testing.T) {
	_, _, err := parseMigrateFromS3Args([]string{"my-s3-bucket"})
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "usage") && !strings.Contains(err.Error(), "from-s3") {
		t.Errorf("error %q should mention usage or from-s3", err.Error())
	}
}

func TestParseMigrateFromS3Args_EmptyArgs(t *testing.T) {
	_, _, err := parseMigrateFromS3Args([]string{})
	if err == nil {
		t.Fatal("expected error for empty args, got nil")
	}
}

func TestParseMigrateFromS3Args_WrongKeyword(t *testing.T) {
	_, _, err := parseMigrateFromS3Args([]string{"my-s3-bucket", "into", "r2", "my-r2-bucket"})
	if err == nil {
		t.Fatal("expected error for wrong keyword, got nil")
	}
	if !strings.Contains(err.Error(), "to-r2") {
		t.Errorf("error %q should mention 'to-r2'", err.Error())
	}
}

func TestParseMigrateFromS3Args_ThreeArgs(t *testing.T) {
	_, _, err := parseMigrateFromS3Args([]string{"bucket", "to-r2", "dest"})
	if err == nil {
		t.Fatal("expected error for 3 args (need 4), got nil")
	}
}

// --- parseRestoreArgs ---

func TestParseRestoreArgs_ValidInput(t *testing.T) {
	backup, bucket, err := parseRestoreArgs([]string{"backup.tar.gz", "to", "ignored", "my-bucket"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backup != "backup.tar.gz" {
		t.Errorf("backupFile = %q, want %q", backup, "backup.tar.gz")
	}
	if bucket != "my-bucket" {
		t.Errorf("bucketName = %q, want %q", bucket, "my-bucket")
	}
}

func TestParseRestoreArgs_MissingArgs(t *testing.T) {
	_, _, err := parseRestoreArgs([]string{"backup.tar.gz"})
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
	if !strings.Contains(err.Error(), "usage") && !strings.Contains(err.Error(), "restore") {
		t.Errorf("error %q should mention usage or restore", err.Error())
	}
}

func TestParseRestoreArgs_EmptyArgs(t *testing.T) {
	_, _, err := parseRestoreArgs([]string{})
	if err == nil {
		t.Fatal("expected error for empty args, got nil")
	}
}

func TestParseRestoreArgs_WrongKeyword(t *testing.T) {
	_, _, err := parseRestoreArgs([]string{"backup.tar.gz", "into", "bucket", "my-bucket"})
	if err == nil {
		t.Fatal("expected error for wrong keyword 'into', got nil")
	}
	if !strings.Contains(err.Error(), "to") {
		t.Errorf("error %q should mention 'to'", err.Error())
	}
}

func TestParseRestoreArgs_ThreeArgs(t *testing.T) {
	_, _, err := parseRestoreArgs([]string{"backup.tar.gz", "to", "my-bucket"})
	if err == nil {
		t.Fatal("expected error for 3 args (need 4), got nil")
	}
}

// --- Stub subcommands return "not yet implemented" errors ---

func TestRunMigrateSync_ReturnsNotImplemented(t *testing.T) {
	err := runMigrateSync(migrateSyncCmd, []string{})
	if err == nil {
		t.Fatal("runMigrateSync should return an error (not yet implemented)")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "not yet implemented") && !strings.Contains(msg, "implemented") {
		t.Errorf("error %q should mention 'not yet implemented'", err.Error())
	}
}

func TestRunMigrateBackup_ReturnsNotImplemented(t *testing.T) {
	err := runMigrateBackup(migrateBackupCmd, []string{})
	if err == nil {
		t.Fatal("runMigrateBackup should return an error (not yet implemented)")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "not yet implemented") && !strings.Contains(msg, "implemented") {
		t.Errorf("error %q should mention 'not yet implemented'", err.Error())
	}
}

func TestRunMigrateRestore_ReturnsNotImplemented(t *testing.T) {
	err := runMigrateRestore(migrateRestoreCmd, []string{})
	if err == nil {
		t.Fatal("runMigrateRestore should return an error (not yet implemented)")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "not yet implemented") && !strings.Contains(msg, "implemented") {
		t.Errorf("error %q should mention 'not yet implemented'", err.Error())
	}
}

func TestRunMigrateBatch_ReturnsNotImplemented(t *testing.T) {
	err := runMigrateBatch(migrateBatchCmd, []string{})
	if err == nil {
		t.Fatal("runMigrateBatch should return an error (not yet implemented)")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "not yet implemented") && !strings.Contains(msg, "implemented") {
		t.Errorf("error %q should mention 'not yet implemented'", err.Error())
	}
}

func TestRunMigrateSync_ErrorMentionsROAD007(t *testing.T) {
	err := runMigrateSync(migrateSyncCmd, []string{})
	if err == nil {
		t.Fatal("runMigrateSync should return an error")
	}
	if !strings.Contains(err.Error(), "ROAD-007") {
		t.Errorf("runMigrateSync error %q should reference ROAD-007", err.Error())
	}
}

func TestRunMigrateBackup_ErrorMentionsROAD007(t *testing.T) {
	err := runMigrateBackup(migrateBackupCmd, []string{})
	if err == nil {
		t.Fatal("runMigrateBackup should return an error")
	}
	if !strings.Contains(err.Error(), "ROAD-007") {
		t.Errorf("runMigrateBackup error %q should reference ROAD-007", err.Error())
	}
}

func TestRunMigrateRestore_ErrorMentionsROAD007(t *testing.T) {
	err := runMigrateRestore(migrateRestoreCmd, []string{})
	if err == nil {
		t.Fatal("runMigrateRestore should return an error")
	}
	if !strings.Contains(err.Error(), "ROAD-007") {
		t.Errorf("runMigrateRestore error %q should reference ROAD-007", err.Error())
	}
}

func TestRunMigrateBatch_ErrorMentionsROAD007(t *testing.T) {
	err := runMigrateBatch(migrateBatchCmd, []string{})
	if err == nil {
		t.Fatal("runMigrateBatch should return an error")
	}
	if !strings.Contains(err.Error(), "ROAD-007") {
		t.Errorf("runMigrateBatch error %q should reference ROAD-007", err.Error())
	}
}

// --- runMigrateFromS3 arg validation (via RunE path without API) ---

func TestRunMigrateFromS3_MissingArgsFails(t *testing.T) {
	err := migrateFromS3Cmd.RunE(migrateFromS3Cmd, []string{})
	if err == nil {
		t.Fatal("runMigrateFromS3 with no args should return an error")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "from-s3") && !strings.Contains(errMsg, "usage") {
		t.Errorf("error %q should describe usage", errMsg)
	}
}

func TestRunMigrateFromS3_WrongKeywordFails(t *testing.T) {
	err := migrateFromS3Cmd.RunE(migrateFromS3Cmd, []string{"src", "into", "r2", "dst"})
	if err == nil {
		t.Fatal("runMigrateFromS3 with wrong keyword should return an error")
	}
	if !strings.Contains(err.Error(), "to-r2") {
		t.Errorf("error %q should mention 'to-r2'", err.Error())
	}
}

// --- Help text uses "cosmoflare" not "r2go2" ---

func TestMigrateCmd_HelpNoBinaryAlias(t *testing.T) {
	help := migrateCmd.Long + migrateCmd.Short + migrateCmd.Use
	if strings.Contains(help, "r2go2") {
		t.Error("migrateCmd help text contains legacy 'r2go2' — should use 'cosmoflare'")
	}
}

func TestMigrateFromS3Cmd_HelpNoBinaryAlias(t *testing.T) {
	help := migrateFromS3Cmd.Long + migrateFromS3Cmd.Short + migrateFromS3Cmd.Use
	if strings.Contains(help, "r2go2") {
		t.Error("migrateFromS3Cmd help text contains legacy 'r2go2' — should use 'cosmoflare'")
	}
}

func TestMigrateSyncCmd_HelpNoBinaryAlias(t *testing.T) {
	help := migrateSyncCmd.Long + migrateSyncCmd.Short + migrateSyncCmd.Use
	if strings.Contains(help, "r2go2") {
		t.Error("migrateSyncCmd help text contains legacy 'r2go2' — should use 'cosmoflare'")
	}
}

func TestMigrateBackupCmd_HelpNoBinaryAlias(t *testing.T) {
	help := migrateBackupCmd.Long + migrateBackupCmd.Short + migrateBackupCmd.Use
	if strings.Contains(help, "r2go2") {
		t.Error("migrateBackupCmd help text contains legacy 'r2go2' — should use 'cosmoflare'")
	}
}

func TestMigrateRestoreCmd_HelpNoBinaryAlias(t *testing.T) {
	help := migrateRestoreCmd.Long + migrateRestoreCmd.Short + migrateRestoreCmd.Use
	if strings.Contains(help, "r2go2") {
		t.Error("migrateRestoreCmd help text contains legacy 'r2go2' — should use 'cosmoflare'")
	}
}

func TestMigrateBatchCmd_HelpNoBinaryAlias(t *testing.T) {
	help := migrateBatchCmd.Long + migrateBatchCmd.Short + migrateBatchCmd.Use
	if strings.Contains(help, "r2go2") {
		t.Error("migrateBatchCmd help text contains legacy 'r2go2' — should use 'cosmoflare'")
	}
}

// --- Subcommand parent relationships ---

func TestMigrateFromS3Cmd_ParentIsMigrateCmd(t *testing.T) {
	if migrateFromS3Cmd.Parent() == nil {
		t.Fatal("migrateFromS3Cmd has no parent")
	}
	if migrateFromS3Cmd.Parent().Name() != "migrate" {
		t.Errorf("migrateFromS3Cmd parent = %q, want %q", migrateFromS3Cmd.Parent().Name(), "migrate")
	}
}

func TestMigrateSyncCmd_ParentIsMigrateCmd(t *testing.T) {
	if migrateSyncCmd.Parent() == nil {
		t.Fatal("migrateSyncCmd has no parent")
	}
	if migrateSyncCmd.Parent().Name() != "migrate" {
		t.Errorf("migrateSyncCmd parent = %q, want %q", migrateSyncCmd.Parent().Name(), "migrate")
	}
}

func TestMigrateBackupCmd_ParentIsMigrateCmd(t *testing.T) {
	if migrateBackupCmd.Parent() == nil {
		t.Fatal("migrateBackupCmd has no parent")
	}
	if migrateBackupCmd.Parent().Name() != "migrate" {
		t.Errorf("migrateBackupCmd parent = %q, want %q", migrateBackupCmd.Parent().Name(), "migrate")
	}
}

func TestMigrateRestoreCmd_ParentIsMigrateCmd(t *testing.T) {
	if migrateRestoreCmd.Parent() == nil {
		t.Fatal("migrateRestoreCmd has no parent")
	}
	if migrateRestoreCmd.Parent().Name() != "migrate" {
		t.Errorf("migrateRestoreCmd parent = %q, want %q", migrateRestoreCmd.Parent().Name(), "migrate")
	}
}

func TestMigrateBatchCmd_ParentIsMigrateCmd(t *testing.T) {
	if migrateBatchCmd.Parent() == nil {
		t.Fatal("migrateBatchCmd has no parent")
	}
	if migrateBatchCmd.Parent().Name() != "migrate" {
		t.Errorf("migrateBatchCmd parent = %q, want %q", migrateBatchCmd.Parent().Name(), "migrate")
	}
}
