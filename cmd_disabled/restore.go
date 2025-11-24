/*
Package cmd provides the restore command for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/interactive"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore [backup-file]",
	Short: "Restore profile configurations from backup",
	Long: `Restore profile configurations from encrypted or plain backups.

This command restores R2Go2 profiles from previously created backups.
Supports encrypted backups (with password), plain JSON files, and
environment variable scripts.

Features:
• Password-protected encrypted backups
• Plain JSON restoration
• Selective profile restoration
• Profile conflict handling
• Secure token re-entry

Security:
• API tokens are never stored in plain backups
• Encrypted backups use AES-256 encryption
• Password verification for protected backups
• Tokens must be re-entered during restore for security

Examples:
  r2go2 restore                          # Interactive restore with file selection
  r2go2 restore backup-2025-01-15.enc    # Restore specific encrypted backup
  r2go2 restore ~/.r2go2-backup.json     # Restore plain JSON backup`,
	RunE: runRestore,
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}

func runRestore(cmd *cobra.Command, args []string) error {
	backupManager, err := interactive.NewBackupManager()
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}

	if len(args) > 0 {
		// Restore from specific file
		backupFile := args[0]
		return backupManager.ShowRestoreInterface()
	}

	// Interactive restore with file selection
	return backupManager.ShowRestoreInterface()
}