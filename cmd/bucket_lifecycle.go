package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/ux"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var bucketLifecycleCmd = &cobra.Command{
	Use:   "lifecycle",
	Short: "Manage R2 bucket object lifecycle rules",
	Long: `Manage object lifecycle rules on Cloudflare R2 buckets.

Lifecycle rules expire objects, abort stale multipart uploads, and transition
objects to the InfrequentAccess storage class, scoped by key prefix. Durations
are expressed in SECONDS. New buckets abort multipart uploads 7 days after
initiation by default; explicit rules replace that behavior.

IMPORTANT: set and clear REPLACE the whole rule set — the API has no per-rule
delete endpoint. Run get first if you mean to add a rule, then set with the
existing rules plus your change.

Commands:
  get     Show the bucket's lifecycle rules
  set     Replace the bucket's lifecycle rules
  clear   Remove all lifecycle rules from the bucket

Examples:
  cosmoflare bucket lifecycle get my-bucket
  cosmoflare bucket lifecycle set my-bucket --prefix img/ --expire-seconds 7776000
  cosmoflare bucket lifecycle set my-bucket --file rules.json
  cosmoflare bucket lifecycle clear my-bucket --force`,
}

var bucketLifecycleGetCmd = &cobra.Command{
	Use:   "get [bucket]",
	Short: "Show the bucket's lifecycle rules",
	Long: `Show the lifecycle rules configured on an R2 bucket.

Columns: ID, ENABLED, PREFIX, DELETE (expire objects after N seconds),
ABORT MPU (abort stale multipart uploads after N seconds), and TRANSITION
(move to InfrequentAccess after N seconds). Use --json for the raw rules.

Examples:
  cosmoflare bucket lifecycle get my-bucket --json`,
	Args: prefixedResourceArgs(nil),
	RunE: runBucketLifecycleGet,
}

var bucketLifecycleSetCmd = &cobra.Command{
	Use:   "set [bucket]",
	Short: "Replace the bucket's lifecycle rules",
	Long: `Replace ALL lifecycle rules on an R2 bucket.

This is a full replacement: rules not included in this call are deleted.
Run 'cosmoflare bucket lifecycle get' first and merge if you mean to add.

Two input modes (exactly one required):
  --file    path to a JSON file of the shape {"rules":[...]} matching the API body
  shorthand flags build a single rule:
    --prefix P                    key prefix the rule applies to ("" = all objects)
    --expire-seconds N            expire objects after N seconds
    --abort-multipart-seconds N   abort multipart uploads after N seconds
    --transition-ia-seconds N     move objects to InfrequentAccess after N seconds
    --rule-id ID                  rule id (default: a generated description)
    --disabled                    store the rule disabled

Examples:
  cosmoflare bucket lifecycle set my-bucket --prefix img/ --expire-seconds 7776000
  cosmoflare bucket lifecycle set my-bucket --abort-multipart-seconds 604800 --force`,
	Args: prefixedResourceArgs(nil),
	RunE: runBucketLifecycleSet,
}

var bucketLifecycleClearCmd = &cobra.Command{
	Use:   "clear [bucket]",
	Short: "Remove all lifecycle rules from the bucket",
	Long: `Remove every lifecycle rule from an R2 bucket.

The API has no delete endpoint, so clear sends an empty rule set — a full
replacement. The bucket's default behavior (abort multipart uploads 7 days
after initiation) applies once no rules remain.

Examples:
  cosmoflare bucket lifecycle clear my-bucket --force`,
	Args: prefixedResourceArgs(nil),
	RunE: runBucketLifecycleClear,
}

func init() {
	bucketCmd.AddCommand(bucketLifecycleCmd)

	bucketLifecycleCmd.AddCommand(bucketLifecycleGetCmd)
	bucketLifecycleCmd.AddCommand(bucketLifecycleSetCmd)
	bucketLifecycleCmd.AddCommand(bucketLifecycleClearCmd)

	bucketLifecycleSetCmd.Flags().String("file", "", "Path to a {\"rules\":[...]} JSON file (full replacement)")
	bucketLifecycleSetCmd.Flags().String("prefix", "", "Key prefix the shorthand rule applies to (empty = all objects)")
	bucketLifecycleSetCmd.Flags().Int64("expire-seconds", 0, "Expire objects after N seconds (shorthand)")
	bucketLifecycleSetCmd.Flags().Int64("abort-multipart-seconds", 0, "Abort stale multipart uploads after N seconds (shorthand)")
	bucketLifecycleSetCmd.Flags().Int64("transition-ia-seconds", 0, "Move objects to InfrequentAccess after N seconds (shorthand)")
	bucketLifecycleSetCmd.Flags().String("rule-id", "", "Rule id for the shorthand rule (default: generated description)")
	bucketLifecycleSetCmd.Flags().Bool("disabled", false, "Store the shorthand rule disabled")

	bucketLifecycleSetCmd.Flags().Bool("force", false, "Skip confirmation prompt")
	bucketLifecycleClearCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

// getBucketLifecycleService creates the lifecycle service using the same
// global credentials the bucket commands rely on.
func getBucketLifecycleService(opts ...cosmoflare.BucketLifecycleOption) *cosmoflare.BucketLifecycleService {
	return cosmoflare.NewBucketLifecycleService(AccountID, APIToken, opts...)
}

// lifecycleSeconds renders a transition's condition as "Ns" or "-".
func lifecycleSeconds(tr *cosmoflare.LifecycleTransition) string {
	if tr == nil || tr.Condition.MaxAge == nil {
		return "-"
	}
	return fmt.Sprintf("%ds", *tr.Condition.MaxAge)
}

func lifecycleTransitionSummary(rule cosmoflare.LifecycleRule) string {
	parts := make([]string, 0, len(rule.StorageClassTransitions))
	for _, tr := range rule.StorageClassTransitions {
		if tr.Condition.MaxAge != nil {
			parts = append(parts, fmt.Sprintf("%s@%ds", tr.StorageClass, *tr.Condition.MaxAge))
		} else {
			parts = append(parts, tr.StorageClass)
		}
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ",")
}

// confirmBucketLifecycleReplace asks before overwriting every rule at once.
func confirmBucketLifecycleReplace(bucket, action string, force bool) bool {
	if force || DryRun {
		return true
	}
	if !ux.Confirm(fmt.Sprintf("%s on bucket '%s'? This REPLACES all lifecycle rules", action, bucket)) {
		printInfo("Lifecycle %s cancelled", action)
		return false
	}
	return true
}

func runBucketLifecycleGet(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucket := args[0]

	svc := getBucketLifecycleService()
	rules, err := svc.Get(cmd.Context(), bucket)
	if err != nil {
		return outErr("failed to get lifecycle rules", err)
	}

	return outResult(rules, func() {
		if len(rules) == 0 {
			printInfo("No lifecycle rules on bucket '%s' (default: multipart uploads abort after 7 days)", bucket)
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tENABLED\tPREFIX\tDELETE\tABORT MPU\tTRANSITION")
		for _, r := range rules {
			enabled := "false"
			if r.Enabled {
				enabled = "true"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				r.ID, enabled, r.Conditions.Prefix,
				lifecycleSeconds(r.DeleteObjectsTransition),
				lifecycleSeconds(r.AbortMultipartUploadsTransition),
				lifecycleTransitionSummary(r))
		}
		w.Flush()
		printInfo("Total: %d rule(s) — set/clear REPLACE the whole rule set; get first if you mean to add", len(rules))
	})
}

// parseLifecycleShorthandFlags builds one rule from the shorthand flags.
func parseLifecycleShorthandFlags(cmd *cobra.Command) (cosmoflare.LifecycleRule, error) {
	rule := cosmoflare.LifecycleRule{}
	prefix, _ := cmd.Flags().GetString("prefix")
	rule.Conditions = cosmoflare.LifecycleConditions{Prefix: prefix}

	expire, _ := cmd.Flags().GetInt64("expire-seconds")
	abort, _ := cmd.Flags().GetInt64("abort-multipart-seconds")
	transition, _ := cmd.Flags().GetInt64("transition-ia-seconds")
	ruleID, _ := cmd.Flags().GetString("rule-id")
	disabled, _ := cmd.Flags().GetBool("disabled")
	rule.Enabled = !disabled

	if expire > 0 {
		v := expire
		rule.DeleteObjectsTransition = &cosmoflare.LifecycleTransition{
			Condition: cosmoflare.LifecycleCondition{Type: "Age", MaxAge: &v},
		}
	}
	if abort > 0 {
		v := abort
		rule.AbortMultipartUploadsTransition = &cosmoflare.LifecycleTransition{
			Condition: cosmoflare.LifecycleCondition{Type: "Age", MaxAge: &v},
		}
	}
	if transition > 0 {
		v := transition
		rule.StorageClassTransitions = []cosmoflare.StorageClassTransition{{
			Condition:    cosmoflare.LifecycleCondition{Type: "Age", MaxAge: &v},
			StorageClass: "InfrequentAccess",
		}}
	}

	if ruleID != "" {
		rule.ID = ruleID
	} else {
		var parts []string
		if expire > 0 {
			parts = append(parts, fmt.Sprintf("expire after %ds", expire))
		}
		if abort > 0 {
			parts = append(parts, fmt.Sprintf("abort multipart uploads after %ds", abort))
		}
		if transition > 0 {
			parts = append(parts, fmt.Sprintf("transition to InfrequentAccess after %ds", transition))
		}
		id := strings.Join(parts, ", ")
		if prefix != "" {
			id = fmt.Sprintf("%s under prefix %q", id, prefix)
		}
		rule.ID = id
	}
	return rule, nil
}

func runBucketLifecycleSet(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucket := args[0]

	file, _ := cmd.Flags().GetString("file")
	force, _ := cmd.Flags().GetBool("force")
	shorthandSet := cmd.Flags().Changed("prefix") || cmd.Flags().Changed("expire-seconds") ||
		cmd.Flags().Changed("abort-multipart-seconds") || cmd.Flags().Changed("transition-ia-seconds")

	var rules []cosmoflare.LifecycleRule
	switch {
	case file != "" && shorthandSet:
		return fmt.Errorf("--file and the shorthand flags are mutually exclusive: pass exactly one")
	case file != "":
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}
		var body struct {
			Rules []cosmoflare.LifecycleRule `json:"rules"`
		}
		if err := json.Unmarshal(data, &body); err != nil {
			return fmt.Errorf("invalid rules file %s: expected {\"rules\":[...]} JSON: %w", file, err)
		}
		if body.Rules == nil {
			return fmt.Errorf("invalid rules file %s: missing \"rules\" array (use 'bucket lifecycle clear' to remove all rules)", file)
		}
		rules = body.Rules
	case shorthandSet:
		rule, err := parseLifecycleShorthandFlags(cmd)
		if err != nil {
			return err
		}
		rules = []cosmoflare.LifecycleRule{rule}
	default:
		return fmt.Errorf("nothing to set: pass --file or at least one shorthand flag (--prefix, --expire-seconds, --abort-multipart-seconds, --transition-ia-seconds)")
	}

	if !confirmBucketLifecycleReplace(bucket, "replace lifecycle rules", force) {
		return nil
	}

	svc := getBucketLifecycleService()
	if err := svc.Set(cmd.Context(), bucket, rules); err != nil {
		return outErr("failed to set lifecycle rules", err)
	}

	return outPayload("Lifecycle rules replaced", func() any {
		return map[string]interface{}{"bucket": bucket, "rules": rules}
	}, func() {
		printSuccess("Replaced lifecycle rules on bucket '%s' (%d rule(s))", bucket, len(rules))
		printInfo("This replaced the whole rule set; run 'cosmoflare bucket lifecycle get %s' to verify", bucket)
	})
}

func runBucketLifecycleClear(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucket := args[0]
	force, _ := cmd.Flags().GetBool("force")

	if !confirmBucketLifecycleReplace(bucket, "clear all lifecycle rules", force) {
		return nil
	}

	svc := getBucketLifecycleService()
	if err := svc.Set(cmd.Context(), bucket, nil); err != nil {
		return outErr("failed to clear lifecycle rules", err)
	}

	return outPayload("Lifecycle rules cleared", func() any {
		return map[string]string{"bucket": bucket}
	}, func() {
		printSuccess("Cleared all lifecycle rules on bucket '%s'", bucket)
	})
}
