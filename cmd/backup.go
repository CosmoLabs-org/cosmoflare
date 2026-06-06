/*
Package cmd provides the backup command for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/CosmoLabs-org/cosmoflare/internal/interactive"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup profile configurations",
	Long: `Secure profile backup with encryption and format options.

This command creates secure backups of your R2Go2 profiles with support
for multiple formats and encryption. Backups exclude sensitive API tokens
for security and can be restored on any system.

Features:
• Encrypted backups with password protection
• Plain JSON exports
• Environment variable scripts
• Automatic file organization
• Cross-platform compatibility

Examples:
  cosmoflare backup                    # Interactive backup with format selection
  cosmoflare backup --format=enc       # Create encrypted backup
  cosmoflare backup --format=json      # Create plain JSON backup
  cosmoflare backup --format=env       # Create environment variable script`,
	RunE: runBackup,
}

var (
	backupFormat string
)

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVar(&backupFormat, "format", "", "Backup format: enc, json, env")
}

func runBackup(cmd *cobra.Command, args []string) error {
	backupManager, err := interactive.NewBackupManager()
	if err != nil {
		return fmt.Errorf("failed to create backup manager: %w", err)
	}

	if backupFormat != "" {
		switch backupFormat {
		case "enc":
			return backupManager.CreateEncryptedBackup()
		case "json":
			return backupManager.CreatePlainBackup()
		case "env":
			return backupManager.CreateEnvironmentBackup()
		default:
			printError("Invalid format. Use: enc, json, or env")
			return fmt.Errorf("invalid backup format: %s", backupFormat)
		}
	}

	return backupManager.ShowBackupInterface()
}