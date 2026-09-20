package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

// accountAuditLogsCmd is the Cloudflare ACCOUNT audit log — who changed
// what in your Cloudflare account, as recorded by Cloudflare's API. The
// local cosmoflare mutation log lives under 'cosmoflare audit' instead.
var accountAuditLogsCmd = &cobra.Command{
	Use:   "audit-logs",
	Short: "Query the Cloudflare account audit log",
	Long: `Query the audit log Cloudflare keeps for your account: every change
made through the dashboard or API, with actor, action, resource and
before/after values.

This is Cloudflare's account-level audit trail, NOT cosmoflare's local
mutation log (that one is 'cosmoflare audit'). Requires a token with the
"Audit Logs: Read" account permission.

Commands:
  list     Query audit-log entries with filters
  history  Show one entry's change history

Examples:
  cosmoflare account audit-logs list --since 2026-09-01
  cosmoflare account audit-logs list --actor-email admin@example.com --action "DNS Record Create"
  cosmoflare account audit-logs list --format ndjson > entries.ndjson
  cosmoflare account audit-logs list --format csv > entries.csv
  cosmoflare account audit-logs history LOG_ENTRY_ID --json`,
}

var accountAuditListCmd = &cobra.Command{
	Use:   "list",
	Short: "Query account audit-log entries",
	Long: `Query the account audit log, newest first.

Filters:
  --actor-email   Only entries performed by this actor (email)
  --actor-id      Only entries performed by this actor (user ID)
  --actor-type    Only entries by this actor type (user, cloudflare, ...)
  --action        Only entries of this action type (e.g. "DNS Record Create")
  --zone          Only entries for this zone name
  --since         Only entries after this time (RFC3339 or YYYY-MM-DD)
  --before        Only entries before this time (RFC3339 or YYYY-MM-DD)
  --per-page      Page size, 1-100 (API default 20)

Output formats:
  --format table  Human-readable table (default)
  --format ndjson One JSON object per line (machine export)
  --format csv    Comma-separated values with a header row (machine export)

Examples:
  cosmoflare account audit-logs list
  cosmoflare account audit-logs list --since 2026-09-01T00:00:00Z --per-page 100
  cosmoflare account audit-logs list --format ndjson
  cosmoflare account audit-logs list --format csv > audit.csv`,
	RunE: runAccountAuditList,
}

var accountAuditHistoryCmd = &cobra.Command{
	Use:   "history <log-id>",
	Short: "Show one audit-log entry's change history",
	Long: `Show the change history of a single audit-log entry: the chain of
prior states that led to the entry's current value.

Examples:
  cosmoflare account audit-logs history LOG_ENTRY_ID
  cosmoflare account audit-logs history LOG_ENTRY_ID --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAccountAuditHistory,
}

// Flags for audit-logs list.
var (
	accountAuditActorEmail string
	accountAuditActorID    string
	accountAuditActorType  string
	accountAuditAction     string
	accountAuditZone       string
	accountAuditSince      string
	accountAuditBefore     string
	accountAuditPerPage    int
	accountAuditFormat     string
)

func init() {
	accountCmd.AddCommand(accountAuditLogsCmd)

	accountAuditLogsCmd.AddCommand(accountAuditListCmd)
	accountAuditLogsCmd.AddCommand(accountAuditHistoryCmd)

	f := accountAuditListCmd.Flags()
	f.StringVar(&accountAuditActorEmail, "actor-email", "", "Filter by actor email")
	f.StringVar(&accountAuditActorID, "actor-id", "", "Filter by actor user ID")
	f.StringVar(&accountAuditActorType, "actor-type", "", "Filter by actor type (user, cloudflare, ...)")
	f.StringVar(&accountAuditAction, "action", "", "Filter by action type (e.g. \"DNS Record Create\")")
	f.StringVar(&accountAuditZone, "zone", "", "Filter by zone name")
	f.StringVar(&accountAuditSince, "since", "", "Only entries after this time (RFC3339 or YYYY-MM-DD)")
	f.StringVar(&accountAuditBefore, "before", "", "Only entries before this time (RFC3339 or YYYY-MM-DD)")
	f.IntVar(&accountAuditPerPage, "per-page", 0, "Page size, 1-100 (API default 20)")
	f.StringVar(&accountAuditFormat, "format", "table", "Output format: table, ndjson or csv")
}

// getAccountAuditService creates the account audit-log service from the
// credential globals.
func getAccountAuditService(opts ...cosmoflare.AccountAuditOption) (*cosmoflare.AccountAuditService, error) {
	return cosmoflare.NewAccountAuditServiceFromCreds(AccountID, APIToken, opts...)
}

func runAccountAuditList(cmd *cobra.Command, args []string) error {
	switch accountAuditFormat {
	case "table", "ndjson", "csv":
	default:
		return outErrf("invalid format %q: must be table, ndjson, or csv", accountAuditFormat)
	}

	svc, err := getAccountAuditService()
	if err != nil {
		return outErr("failed to create account audit service", err)
	}
	logs, err := svc.Logs(context.Background(), cosmoflare.AccountAuditFilter{
		ActorID:   accountAuditActorID,
		ActorType: accountAuditActorType,
		Action:    accountAuditAction,
		Zone:      accountAuditZone,
		Since:     accountAuditSince,
		Before:    accountAuditBefore,
		PerPage:   accountAuditPerPage,
	})
	if err != nil {
		return outErr("failed to query account audit logs", err)
	}

	// The export formats write raw records to stdout; they ARE the
	// machine-readable surface, so they bypass the JSON envelope.
	switch accountAuditFormat {
	case "ndjson":
		fmt.Print(accountAuditNDJSON(logs))
		return nil
	case "csv":
		fmt.Print(accountAuditCSV(logs))
		return nil
	}

	return outResult(logs, func() {
		if len(logs) == 0 {
			printInfo("No audit-log entries matched the given filters")
			return
		}
		fmt.Println("WHEN\tACTION\tRESULT\tACTOR\tRESOURCE\tID")
		for _, e := range logs {
			actor := e.Actor.Email
			if actor == "" {
				actor = e.Actor.ID
			}
			if actor == "" {
				actor = e.Actor.Type
			}
			fmt.Printf("%s\t%s\t%t\t%s\t%s/%s\t%s\n",
				e.When, e.Action.Type, e.Action.Result, actor, e.Resource.Type, e.Resource.ID, e.ID)
		}
		printInfo("Total: %d entries — use --per-page to page through more", len(logs))
	})
}

func runAccountAuditHistory(cmd *cobra.Command, args []string) error {
	logID := args[0]

	svc, err := getAccountAuditService()
	if err != nil {
		return outErr("failed to create account audit service", err)
	}
	entries, err := svc.History(context.Background(), logID)
	if err != nil {
		return outErr("failed to fetch audit-log history", err)
	}

	return outPayload("audit-log history fetched", func() any {
		return map[string]any{"id": logID, "entries": entries}
	}, func() {
		printSuccess("Change history for audit-log entry '%s'", logID)
		for i, e := range entries {
			actor := e.Actor.Email
			if actor == "" {
				actor = e.Actor.ID
			}
			fmt.Printf("%d. %s  %s  (actor: %s)\n", i+1, e.When, e.Action.Type, actor)
			if e.OldValue != "" {
				fmt.Printf("   old: %s\n", strings.SplitN(e.OldValue, "\n", 2)[0])
			}
			if e.NewValue != "" {
				fmt.Printf("   new: %s\n", strings.SplitN(e.NewValue, "\n", 2)[0])
			}
		}
		if len(entries) == 0 {
			printInfo("No history recorded for this entry")
		}
	})
}

// accountAuditNDJSON renders audit-log entries as newline-delimited JSON:
// one JSON object per line, no wrapping array. Nil input renders nothing.
func accountAuditNDJSON(logs []cosmoflare.AccountAuditLog) string {
	var b strings.Builder
	for _, e := range logs {
		line, err := json.Marshal(e)
		if err != nil {
			continue
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// accountAuditCSV renders audit-log entries as CSV: a fixed header row
// followed by one row per entry. Nil input renders just the header.
func accountAuditCSV(logs []cosmoflare.AccountAuditLog) string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"id", "when", "action", "result", "actor_id", "actor_email", "actor_type", "resource_type", "resource_id"})
	for _, e := range logs {
		_ = w.Write([]string{
			e.ID,
			e.When,
			e.Action.Type,
			fmt.Sprintf("%t", e.Action.Result),
			e.Actor.ID,
			e.Actor.Email,
			e.Actor.Type,
			e.Resource.Type,
			e.Resource.ID,
		})
	}
	w.Flush()
	return b.String()
}
