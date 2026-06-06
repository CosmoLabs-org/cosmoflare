package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View and manage the CLI audit log",
	Long: `Audit log management for Cosmoflare CLI mutations.

Every mutation (create, update, delete, deploy) performed through cosmoflare
is recorded in a local audit log at ~/.cosmoflare/audit.log as newline-delimited JSON.

Each entry captures: timestamp, operation, service, resource, action, user, details, and success status.

Commands:
  log       View recent audit log entries
  search    Search audit log entries
  export    Export audit log to JSON or CSV
  clear     Clear the audit log

Examples:
  cosmoflare audit log                          # Show last 50 entries
  cosmoflare audit log --limit 100              # Show last 100 entries
  cosmoflare audit log --since 2026-01-01       # Entries since date
  cosmoflare audit log --json                   # JSON output
  cosmoflare audit search "dns"                 # Search for DNS operations
  cosmoflare audit search "bucket" --type create # Search creates only
  cosmoflare audit export audit-backup.json     # Export to JSON
  cosmoflare audit export audit-backup.csv      # Export to CSV (by extension)
  cosmoflare audit clear --force                # Clear entire log
  cosmoflare audit clear --before 2026-01-01    # Clear old entries`,
}

var (
	auditLimit  int
	auditSince  string
	auditType   string
	auditForce  bool
	auditBefore string
)

// --- audit log ---

var auditLogCmd = &cobra.Command{
	Use:   "log",
	Short: "View recent audit log entries",
	Long: `Display recent entries from the cosmoflare audit log.

The audit log is stored at ~/.cosmoflare/audit.log as newline-delimited JSON.

By default shows the last 50 entries. Use --limit to change the count,
or --since to filter by date.

Examples:
  cosmoflare audit log                    # Last 50 entries
  cosmoflare audit log --limit 10         # Last 10 entries
  cosmoflare audit log --since 2026-06-01 # Since June 1st
  cosmoflare audit log --json             # JSON output`,
	RunE: runAuditLog,
}

// --- audit search ---

var auditSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search audit log entries",
	Long: `Search the audit log for entries matching a query string.

The query is matched case-insensitively against all fields: operation, service,
resource, action, user, timestamp, and detail key/value pairs.

Optionally filter by action type (create, delete, update, deploy, etc.).

Examples:
  cosmoflare audit search "dns"                    # All DNS operations
  cosmoflare audit search "my-bucket"              # Entries mentioning a bucket
  cosmoflare audit search "worker" --type deploy   # Worker deploys only
  cosmoflare audit search "example.com" --json     # JSON output`,
	Args: cobra.ExactArgs(1),
	RunE: runAuditSearch,
}

// --- audit export ---

var auditExportCmd = &cobra.Command{
	Use:   "export [file]",
	Short: "Export audit log to JSON or CSV",
	Long: `Export the audit log to a file in JSON or CSV format.

The format is determined by the file extension (.json or .csv).
If no file is given, JSON is written to stdout.

Examples:
  cosmoflare audit export                      # JSON to stdout
  cosmoflare audit export audit-backup.json    # Export as JSON
  cosmoflare audit export audit-report.csv     # Export as CSV`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAuditExport,
}

// --- audit clear ---

var auditClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the audit log",
	Long: `Clear entries from the cosmoflare audit log.

Without --before, clears the entire log. With --before, only entries
older than the given date are removed.

Requires --force to confirm the operation.

Examples:
  cosmoflare audit clear --force                    # Clear entire log
  cosmoflare audit clear --before 2026-01-01 --force # Clear old entries`,
	RunE: runAuditClear,
}

func init() {
	rootCmd.AddCommand(auditCmd)

	auditCmd.AddCommand(auditLogCmd)
	auditCmd.AddCommand(auditSearchCmd)
	auditCmd.AddCommand(auditExportCmd)
	auditCmd.AddCommand(auditClearCmd)

	// log flags
	auditLogCmd.Flags().IntVar(&auditLimit, "limit", 50, "Maximum number of entries to show")
	auditLogCmd.Flags().StringVar(&auditSince, "since", "", "Show entries since date (YYYY-MM-DD or RFC3339)")

	// search flags
	auditSearchCmd.Flags().StringVar(&auditType, "type", "", "Filter by action type (create, delete, update, deploy)")

	// clear flags
	auditClearCmd.Flags().BoolVar(&auditForce, "force", false, "Confirm clearing the audit log")
	auditClearCmd.Flags().StringVar(&auditBefore, "before", "", "Clear entries before date (YYYY-MM-DD or RFC3339)")
}

func getAuditLogPath() string {
	return cosmoflare.DefaultAuditLogPath()
}

func runAuditLog(cmd *cobra.Command, args []string) error {
	logPath := getAuditLogPath()
	if logPath == "" {
		return fmt.Errorf("could not determine audit log path (home directory not found)")
	}

	var entries []cosmoflare.AuditEntry
	var err error

	if auditSince != "" {
		sinceTime, parseErr := parseDate(auditSince)
		if parseErr != nil {
			return fmt.Errorf("invalid --since value %q: %w (use YYYY-MM-DD or RFC3339)", auditSince, parseErr)
		}
		entries, err = cosmoflare.ReadAuditLogSince(logPath, sinceTime)
	} else {
		entries, err = cosmoflare.ReadAuditLogWithLimit(logPath, auditLimit)
	}

	if err != nil {
		return fmt.Errorf("failed to read audit log: %w", err)
	}

	if len(entries) == 0 {
		if JSONOutput {
			return printSuccessJSON("No audit log entries found", []cosmoflare.AuditEntry{})
		}
		printInfo("No audit log entries found")
		return nil
	}

	if JSONOutput {
		return printSuccessJSON(fmt.Sprintf("Found %d audit log entries", len(entries)), entries)
	}

	printAuditTable(entries)
	return nil
}

func runAuditSearch(cmd *cobra.Command, args []string) error {
	logPath := getAuditLogPath()
	if logPath == "" {
		return fmt.Errorf("could not determine audit log path (home directory not found)")
	}

	query := args[0]
	entries, err := cosmoflare.SearchAuditLog(logPath, query, auditType)
	if err != nil {
		return fmt.Errorf("failed to search audit log: %w", err)
	}

	if len(entries) == 0 {
		if JSONOutput {
			return printSuccessJSON(fmt.Sprintf("No entries matching %q", query), []cosmoflare.AuditEntry{})
		}
		printInfo("No entries matching %q", query)
		return nil
	}

	if JSONOutput {
		return printSuccessJSON(fmt.Sprintf("Found %d entries matching %q", len(entries), query), entries)
	}

	printAuditTable(entries)
	return nil
}

func runAuditExport(cmd *cobra.Command, args []string) error {
	logPath := getAuditLogPath()
	if logPath == "" {
		return fmt.Errorf("could not determine audit log path (home directory not found)")
	}

	entries, err := cosmoflare.ReadAuditLog(logPath)
	if err != nil {
		return fmt.Errorf("failed to read audit log: %w", err)
	}

	if len(entries) == 0 {
		if JSONOutput {
			return printSuccessJSON("No audit log entries to export", []cosmoflare.AuditEntry{})
		}
		printInfo("No audit log entries to export")
		return nil
	}

	// No file argument: JSON to stdout
	if len(args) == 0 {
		return cosmoflare.ExportAuditLogJSON(entries, os.Stdout)
	}

	filePath := args[0]
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create export file %q: %w", filePath, err)
	}
	defer f.Close()

	if strings.HasSuffix(strings.ToLower(filePath), ".csv") {
		if err := cosmoflare.ExportAuditLogCSV(entries, f); err != nil {
			return fmt.Errorf("failed to export CSV: %w", err)
		}
	} else {
		if err := cosmoflare.ExportAuditLogJSON(entries, f); err != nil {
			return fmt.Errorf("failed to export JSON: %w", err)
		}
	}

	if JSONOutput {
		return printSuccessJSON(fmt.Sprintf("Exported %d entries to %s", len(entries), filePath), map[string]interface{}{
			"file":    filePath,
			"entries": len(entries),
		})
	}
	printSuccess("Exported %d entries to %s", len(entries), filePath)
	return nil
}

func runAuditClear(cmd *cobra.Command, args []string) error {
	logPath := getAuditLogPath()
	if logPath == "" {
		return fmt.Errorf("could not determine audit log path (home directory not found)")
	}

	if !auditForce {
		return fmt.Errorf("audit clear requires --force to confirm. This operation cannot be undone")
	}

	var beforeTime time.Time
	if auditBefore != "" {
		var parseErr error
		beforeTime, parseErr = parseDate(auditBefore)
		if parseErr != nil {
			return fmt.Errorf("invalid --before value %q: %w (use YYYY-MM-DD or RFC3339)", auditBefore, parseErr)
		}
	}

	removed, err := cosmoflare.ClearAuditLog(logPath, beforeTime)
	if err != nil {
		return fmt.Errorf("failed to clear audit log: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON(fmt.Sprintf("Cleared %d audit log entries", removed), map[string]interface{}{
			"removed": removed,
		})
	}

	if removed == 0 {
		printInfo("No entries to clear")
	} else {
		printSuccess("Cleared %d audit log entries", removed)
	}
	return nil
}

// printAuditTable formats audit entries as a table.
func printAuditTable(entries []cosmoflare.AuditEntry) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIMESTAMP\tOPERATION\tSERVICE\tRESOURCE\tACTION\tSUCCESS")
	fmt.Fprintln(w, "---------\t---------\t-------\t--------\t------\t-------")
	for _, e := range entries {
		success := "yes"
		if !e.Success {
			success = "FAILED"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			e.Timestamp.Format("2006-01-02 15:04:05"),
			e.Operation,
			e.Service,
			truncate(e.Resource, 30),
			e.Action,
			success,
		)
	}
	w.Flush()
	fmt.Printf("\nShowing %d entries\n", len(entries))
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// parseDate parses a date string in YYYY-MM-DD or RFC3339 format.
func parseDate(s string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try YYYY-MM-DD
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognized date format")
}
