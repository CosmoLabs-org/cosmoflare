package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	d1TimeTravelTimestamp string
	d1TimeTravelForce     bool
)

var d1TimeTravelCmd = &cobra.Command{
	Use:   "time-travel",
	Short: "Restore a D1 database to a point in time",
	Long: `Restore a D1 database using Cloudflare's time-travel feature.

Commands:
  restore   Restore the database to a given timestamp

Examples:
  cosmoflare d1 time-travel restore <database-id> --timestamp 2026-01-01T00:00:00Z`,
}

var d1TimeTravelRestoreCmd = &cobra.Command{
	Use:   "restore [database-id]",
	Short: "Restore a database to a point in time",
	Long: `Restore a D1 database to the state it had at --timestamp.

Time-travel restores are limited to 10 per database within any rolling
10-minute window. This command checks that quota before calling the API
and reports the remaining restores afterward.

Examples:
  cosmoflare d1 time-travel restore 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --timestamp 2026-01-01T00:00:00Z
  cosmoflare d1 time-travel restore 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --timestamp 2026-01-01T00:00:00Z --dry-run
  cosmoflare d1 time-travel restore 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --timestamp 2026-01-01T00:00:00Z --force`,
	Args: cobra.ExactArgs(1),
	RunE: runD1TimeTravelRestore,
}

func init() {
	d1Cmd.AddCommand(d1TimeTravelCmd)
	d1TimeTravelCmd.AddCommand(d1TimeTravelRestoreCmd)

	d1TimeTravelRestoreCmd.Flags().StringVar(&d1TimeTravelTimestamp, "timestamp", "", "Point-in-time to restore to, ISO8601 (e.g. 2026-01-01T00:00:00Z)")
	d1TimeTravelRestoreCmd.Flags().BoolVar(&d1TimeTravelForce, "force", false, "Skip confirmation prompt")
	_ = d1TimeTravelRestoreCmd.MarkFlagRequired("timestamp")
}

func runD1TimeTravelRestore(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	timestamp, err := time.Parse(time.RFC3339, d1TimeTravelTimestamp)
	if err != nil {
		return fmt.Errorf("invalid --timestamp %q: must be ISO8601 (RFC3339), e.g. 2026-01-01T00:00:00Z: %w", d1TimeTravelTimestamp, err)
	}

	svc, err := getD1Service()
	if err != nil {
		return outErr("failed to create D1 service", err)
	}

	quota, err := svc.TimeTravelQuotaCheck(databaseID)
	if err != nil {
		return outErr("failed to check time-travel quota", err)
	}
	if quota.Used >= quota.Limit {
		return outErrf("time-travel restore quota exceeded (%d/%d used); resets at %s", quota.Used, quota.Limit, quota.WindowResetsAt.Format(time.RFC3339))
	}

	if DryRun {
		return outPayload("DRY RUN: Would restore database", func() any {
			return map[string]any{
				"database_id": databaseID,
				"timestamp":   timestamp.UTC().Format(time.RFC3339),
				"quota_used":  quota.Used,
				"quota_limit": quota.Limit,
			}
		}, func() {
			printInfo("DRY RUN: Would restore database '%s' to '%s' (%d/%d restores used this window)", databaseID, timestamp.UTC().Format(time.RFC3339), quota.Used, quota.Limit)
		})
	}

	if !d1TimeTravelForce {
		fmt.Printf("Type the database ID '%s' to confirm restoring to '%s': ", databaseID, timestamp.UTC().Format(time.RFC3339))
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)
		if response != databaseID {
			printInfo("Time-travel restore cancelled")
			return nil
		}
	}

	result, err := svc.TimeTravelRestore(context.Background(), databaseID, timestamp)
	if err != nil {
		return outErr("failed to restore database", err)
	}

	remaining := quota.Limit - (quota.Used + 1)

	return outPayload("Database restored", func() any {
		return map[string]any{
			"result":             result,
			"remaining_restores": remaining,
			"window_resets_at":   quota.WindowResetsAt.Format(time.RFC3339),
		}
	}, func() {
		printSuccess("Database '%s' restored to '%s'", databaseID, timestamp.UTC().Format(time.RFC3339))
		printInfo("%d restore(s) remaining in the current window", remaining)
	})
}
