package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var bucketNotificationsCmd = &cobra.Command{
	Use:   "notifications",
	Short: "Manage R2 bucket event notifications to Queues",
	Long: `Manage R2 event notification rules that send per-object events
(created, deleted, copied, multipart-completed, lifecycle-expired) to
Cloudflare Queues.

Prerequisites: the queue must already exist in your account, and a consumer
should pull from it. The target is Cloudflare Queues ONLY — there is no
webhook target.

Limits: a bucket can have at most 100 rules; overlapping rules that would
double-notify the same event are rejected by the API.

Actions: PutObject, CopyObject, DeleteObject, CompleteMultipartUpload,
LifecycleDeletion. Convenience event types: object-create (PutObject,
CopyObject, CompleteMultipartUpload) and object-delete (DeleteObject,
LifecycleDeletion).

Commands:
  list     List queues wired to a bucket and their rules
  create   Create/replace a queue's rule set on a bucket
  get      Show one queue's rules (with rule IDs)
  delete   Remove rules or a whole queue configuration

Examples:
  cosmoflare bucket notifications list my-bucket
  cosmoflare bucket notifications create my-bucket --queue-id QUEUE_ID --event-type object-create --prefix img/ --suffix .jpeg
  cosmoflare bucket notifications get my-bucket --queue-id QUEUE_ID
  cosmoflare bucket notifications delete my-bucket --queue-id QUEUE_ID --all --force`,
}

var bucketNotificationsCreateCmd = &cobra.Command{
	Use:   "create [bucket]",
	Short: "Create or replace event notification rules for a queue",
	Long: `Create or replace the rule set a queue receives from a bucket (PUT,
so existing rules for this queue are replaced).

Pick actions with --action (repeatable: PutObject, CopyObject, DeleteObject,
CompleteMultipartUpload, LifecycleDeletion) or a grouping with --event-type
(object-create | object-delete) — not both. Optionally narrow the rule with
--prefix and/or --suffix filters and describe it with --description.

The queue must already exist; up to 100 rules per bucket are allowed.

Examples:
  cosmoflare bucket notifications create my-bucket --queue-id QUEUE_ID --event-type object-create --prefix img/ --suffix .jpeg
  cosmoflare bucket notifications create my-bucket --queue-id QUEUE_ID --action PutObject --action DeleteObject`,
	RunE: runBucketNotificationsCreate,
}

var bucketNotificationsListCmd = &cobra.Command{
	Use:   "list [bucket]",
	Short: "List queues wired to a bucket and their rules",
	Long: `List every queue that receives event notifications from an R2 bucket,
with rule counts and actions. Use --json for the full shape including rule
IDs, prefixes and suffixes.

Examples:
  cosmoflare bucket notifications list my-bucket --json`,
	RunE: runBucketNotificationsList,
}

var bucketNotificationsGetCmd = &cobra.Command{
	Use:   "get [bucket]",
	Short: "Show one queue's notification rules",
	Long: `Show the rules one queue receives from a bucket, including rule IDs
and creation timestamps (needed by 'notifications delete').

Examples:
  cosmoflare bucket notifications get my-bucket --queue-id QUEUE_ID`,
	RunE: runBucketNotificationsGet,
}

var bucketNotificationsDeleteCmd = &cobra.Command{
	Use:   "delete [bucket]",
	Short: "Remove notification rules or a whole queue configuration",
	Long: `Remove specific rules from a queue (--rule-id, repeatable, IDs from
'notifications get') or the queue's entire configuration (--all, no ids sent).

Without --all at least one --rule-id is required. Asks for confirmation
unless --force.

Examples:
  cosmoflare bucket notifications delete my-bucket --queue-id QUEUE_ID --rule-id RULE_ID
  cosmoflare bucket notifications delete my-bucket --queue-id QUEUE_ID --all --force`,
	RunE: runBucketNotificationsDelete,
}

func init() {
	bucketCmd.AddCommand(bucketNotificationsCmd)

	bucketNotificationsCmd.AddCommand(bucketNotificationsListCmd)
	bucketNotificationsCmd.AddCommand(bucketNotificationsCreateCmd)
	bucketNotificationsCmd.AddCommand(bucketNotificationsGetCmd)
	bucketNotificationsCmd.AddCommand(bucketNotificationsDeleteCmd)

	bucketNotificationsCreateCmd.Flags().String("queue-id", "", "Target queue ID (required; the queue must already exist)")
	bucketNotificationsCreateCmd.Flags().StringArray("action", nil, "Action to notify (repeatable): PutObject, CopyObject, DeleteObject, CompleteMultipartUpload, LifecycleDeletion")
	bucketNotificationsCreateCmd.Flags().String("event-type", "", "Action grouping: object-create or object-delete (mutually exclusive with --action)")
	bucketNotificationsCreateCmd.Flags().String("prefix", "", "Only notify objects whose key starts with this prefix")
	bucketNotificationsCreateCmd.Flags().String("suffix", "", "Only notify objects whose key ends with this suffix")
	bucketNotificationsCreateCmd.Flags().String("description", "", "Optional rule description")

	bucketNotificationsGetCmd.Flags().String("queue-id", "", "Queue ID to inspect (required)")

	bucketNotificationsDeleteCmd.Flags().String("queue-id", "", "Queue ID (required)")
	bucketNotificationsDeleteCmd.Flags().StringArray("rule-id", nil, "Rule ID to remove (repeatable; see 'notifications get')")
	bucketNotificationsDeleteCmd.Flags().Bool("all", false, "Remove the queue's entire configuration from the bucket")
	bucketNotificationsDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

// getBucketNotificationService creates the notification service using the
// same global credentials the bucket commands rely on.
func getBucketNotificationService() *cosmoflare.BucketNotificationService {
	return cosmoflare.NewBucketNotificationService(AccountID, APIToken)
}

// bucketNotificationEventTypes maps the wrangler-compatible groupings to
// their exact API action lists.
var bucketNotificationEventTypes = map[string][]cosmoflare.NotificationAction{
	"object-create": {cosmoflare.ActionPutObject, cosmoflare.ActionCopyObject, cosmoflare.ActionCompleteMultipartUpload},
	"object-delete": {cosmoflare.ActionDeleteObject, cosmoflare.ActionLifecycleDeletion},
}

func runBucketNotificationsList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucket := args[0]

	svc := getBucketNotificationService()
	queues, err := svc.List(context.Background(), bucket)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list notifications: %v", err))
		}
		return fmt.Errorf("failed to list notifications: %w", err)
	}

	if JSONOutput {
		return printJSON(queues)
	}

	if len(queues) == 0 {
		printInfo("No event notification queues wired to bucket '%s'", bucket)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "QUEUE ID\tQUEUE NAME\tRULES\tACTIONS")
	for _, q := range queues {
		actions := map[string]bool{}
		for _, r := range q.Rules {
			for _, a := range r.Actions {
				actions[string(a)] = true
			}
		}
		names := make([]string, 0, len(actions))
		for a := range actions {
			names = append(names, a)
		}
		sort.Strings(names)
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", q.QueueID, q.QueueName, len(q.Rules), strings.Join(names, ", "))
	}
	w.Flush()
	printInfo("Total: %d queue(s)", len(queues))
	return nil
}

func runBucketNotificationsCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucket := args[0]
	queueID, _ := cmd.Flags().GetString("queue-id")
	actionsRaw, _ := cmd.Flags().GetStringArray("action")
	eventType, _ := cmd.Flags().GetString("event-type")
	prefix, _ := cmd.Flags().GetString("prefix")
	suffix, _ := cmd.Flags().GetString("suffix")
	description, _ := cmd.Flags().GetString("description")

	if queueID == "" {
		return fmt.Errorf("--queue-id is required")
	}
	if len(actionsRaw) > 0 && eventType != "" {
		return fmt.Errorf("pass either --action or --event-type, not both")
	}
	if len(actionsRaw) == 0 && eventType == "" {
		return fmt.Errorf("pass at least one --action or an --event-type (object-create or object-delete)")
	}

	var actions []cosmoflare.NotificationAction
	if eventType != "" {
		group, ok := bucketNotificationEventTypes[eventType]
		if !ok {
			return fmt.Errorf("invalid --event-type %q: must be object-create or object-delete", eventType)
		}
		actions = group
	} else {
		actions = make([]cosmoflare.NotificationAction, 0, len(actionsRaw))
		for _, a := range actionsRaw {
			actions = append(actions, cosmoflare.NotificationAction(a))
		}
	}

	rule := cosmoflare.NotificationRule{
		Actions:     actions,
		Description: description,
		Prefix:      prefix,
		Suffix:      suffix,
	}

	svc := getBucketNotificationService()
	if err := svc.Set(context.Background(), bucket, queueID, []cosmoflare.NotificationRule{rule}); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to set notifications: %v", err))
		}
		return fmt.Errorf("failed to set notifications: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON(fmt.Sprintf("notification rules set on bucket %s for queue %s", bucket, queueID), rule)
	}
	names := make([]string, 0, len(actions))
	for _, a := range actions {
		names = append(names, string(a))
	}
	printSuccess("Queue '%s' now receives %s events from bucket '%s'", queueID, strings.Join(names, ", "), bucket)
	printInfo("This replaced any previous rules for this queue; run 'cosmoflare bucket notifications get %s --queue-id %s' to verify", bucket, queueID)
	return nil
}

func runBucketNotificationsGet(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucket := args[0]
	queueID, _ := cmd.Flags().GetString("queue-id")
	if queueID == "" {
		return fmt.Errorf("--queue-id is required")
	}

	svc := getBucketNotificationService()
	q, err := svc.Get(context.Background(), bucket, queueID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get notifications: %v", err))
		}
		return fmt.Errorf("failed to get notifications: %w", err)
	}

	if JSONOutput {
		return printJSON(q)
	}

	fmt.Printf("Queue ID:   %s\n", q.QueueID)
	fmt.Printf("Queue Name: %s\n", q.QueueName)
	if len(q.Rules) == 0 {
		printInfo("No rules configured for this queue")
		return nil
	}
	fmt.Printf("Rules:      %d\n", len(q.Rules))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RULE ID\tACTIONS\tPREFIX\tSUFFIX\tCREATED")
	for _, r := range q.Rules {
		names := make([]string, 0, len(r.Actions))
		for _, a := range r.Actions {
			names = append(names, string(a))
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.RuleID, strings.Join(names, ","), r.Prefix, r.Suffix, r.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	w.Flush()
	return nil
}

func runBucketNotificationsDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucket := args[0]
	queueID, _ := cmd.Flags().GetString("queue-id")
	ruleIDs, _ := cmd.Flags().GetStringArray("rule-id")
	all, _ := cmd.Flags().GetBool("all")
	force, _ := cmd.Flags().GetBool("force")

	if queueID == "" {
		return fmt.Errorf("--queue-id is required")
	}
	if !all && len(ruleIDs) == 0 {
		return fmt.Errorf("pass at least one --rule-id or --all to remove the queue's entire configuration")
	}
	if all && len(ruleIDs) > 0 {
		return fmt.Errorf("--all ignores --rule-id; pass only one of them")
	}

	target := fmt.Sprintf("%d rule(s)", len(ruleIDs))
	if all {
		target = "the entire queue configuration"
	}

	if !force && !DryRun {
		fmt.Printf("Are you sure you want to delete %s for queue '%s' on bucket '%s'? [y/N]: ", target, queueID, bucket)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Cancelled")
			return nil
		}
	}

	ids := ruleIDs
	if all {
		ids = nil
	}

	svc := getBucketNotificationService()
	if err := svc.Delete(context.Background(), bucket, queueID, ids); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete notifications: %v", err))
		}
		return fmt.Errorf("failed to delete notifications: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON(fmt.Sprintf("deleted %s for queue %s on bucket %s", target, queueID, bucket), nil)
	}
	printSuccess("Deleted %s for queue '%s' on bucket '%s'", target, queueID, bucket)
	return nil
}
