package cmd

import (
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// notificationsRunGlobals snapshots and restores the package globals the
// notification runners read so tests cannot leak state between each other.
func notificationsRunGlobals(t *testing.T) {
	t.Helper()
	oldAccount, oldToken := AccountID, APIToken
	oldDry, oldJSON := DryRun, JSONOutput
	t.Cleanup(func() {
		AccountID, APIToken = oldAccount, oldToken
		DryRun, JSONOutput = oldDry, oldJSON
	})
}

// notificationsResetFlags restores the given flags to their default values
// and clears their "changed" marks, so subtests start pristine. Repeatable
// string-array flags are cleared via SliceValue.Replace because calling Set
// with their "[]" default text would append a literal "[]" element.
func notificationsResetFlags(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			continue
		}
		if sv, ok := f.Value.(pflag.SliceValue); ok {
			_ = sv.Replace(nil)
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	}
}

var notificationsCreateFlagNames = []string{
	"queue-id", "action", "event-type", "prefix", "suffix", "description",
}

// TestRunBucketNotifications_RequiresBucket verifies all four runners reject
// a missing bucket argument before any service is built.
func TestRunBucketNotifications_RequiresBucket(t *testing.T) {
	notificationsRunGlobals(t)
	cases := []struct {
		name string
		cmd  *cobra.Command
		run  func(cmd *cobra.Command, args []string) error
	}{
		{"list", bucketNotificationsListCmd, runBucketNotificationsList},
		{"create", bucketNotificationsCreateCmd, runBucketNotificationsCreate},
		{"get", bucketNotificationsGetCmd, runBucketNotificationsGet},
		{"delete", bucketNotificationsDeleteCmd, runBucketNotificationsDelete},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(tc.cmd, nil)
			if err == nil || !strings.Contains(err.Error(), "bucket name is required") {
				t.Fatalf("expected bucket-required error, got %v", err)
			}
		})
	}
}

// TestRunBucketNotificationsCreate_Validation verifies every flag-validation
// branch of the create runner, all of which return before any network call.
func TestRunBucketNotificationsCreate_Validation(t *testing.T) {
	notificationsRunGlobals(t)
	cases := []struct {
		name    string
		set     func(t *testing.T)
		wantErr string
	}{
		{
			"missing-queue-id",
			func(t *testing.T) {},
			"--queue-id is required",
		},
		{
			"action-and-event-type",
			func(t *testing.T) {
				_ = bucketNotificationsCreateCmd.Flags().Set("queue-id", "queue-1")
				_ = bucketNotificationsCreateCmd.Flags().Set("action", "PutObject")
				_ = bucketNotificationsCreateCmd.Flags().Set("event-type", "object-create")
			},
			"pass either --action or --event-type, not both",
		},
		{
			"neither-action-nor-event-type",
			func(t *testing.T) {
				_ = bucketNotificationsCreateCmd.Flags().Set("queue-id", "queue-1")
			},
			"pass at least one --action or an --event-type",
		},
		{
			"invalid-event-type",
			func(t *testing.T) {
				_ = bucketNotificationsCreateCmd.Flags().Set("queue-id", "queue-1")
				_ = bucketNotificationsCreateCmd.Flags().Set("event-type", "bogus")
			},
			`invalid --event-type "bogus"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			notificationsResetFlags(bucketNotificationsCreateCmd, notificationsCreateFlagNames...)
			tc.set(t)
			t.Cleanup(func() {
				notificationsResetFlags(bucketNotificationsCreateCmd, notificationsCreateFlagNames...)
			})

			err := runBucketNotificationsCreate(bucketNotificationsCreateCmd, []string{"my-bucket"})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunBucketNotificationsGet_RequiresQueueID verifies the get runner's
// queue-ID guard fires before the service call.
func TestRunBucketNotificationsGet_RequiresQueueID(t *testing.T) {
	notificationsRunGlobals(t)
	notificationsResetFlags(bucketNotificationsGetCmd, "queue-id")

	err := runBucketNotificationsGet(bucketNotificationsGetCmd, []string{"my-bucket"})
	if err == nil || !strings.Contains(err.Error(), "--queue-id is required") {
		t.Fatalf("expected queue-id error, got %v", err)
	}
}

// TestRunBucketNotificationsDelete_Validation verifies the delete runner's
// queue-ID, rule-selection and mutual-exclusion guards.
func TestRunBucketNotificationsDelete_Validation(t *testing.T) {
	notificationsRunGlobals(t)
	flagNames := []string{"queue-id", "rule-id", "all", "force"}
	cases := []struct {
		name    string
		set     func(t *testing.T)
		wantErr string
	}{
		{
			"missing-queue-id",
			func(t *testing.T) {},
			"--queue-id is required",
		},
		{
			"no-rule-ids-and-not-all",
			func(t *testing.T) {
				_ = bucketNotificationsDeleteCmd.Flags().Set("queue-id", "queue-1")
			},
			"pass at least one --rule-id or --all",
		},
		{
			"all-with-rule-ids",
			func(t *testing.T) {
				_ = bucketNotificationsDeleteCmd.Flags().Set("queue-id", "queue-1")
				_ = bucketNotificationsDeleteCmd.Flags().Set("rule-id", "rule-1")
				_ = bucketNotificationsDeleteCmd.Flags().Set("all", "true")
			},
			"--all ignores --rule-id; pass only one of them",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			notificationsResetFlags(bucketNotificationsDeleteCmd, flagNames...)
			tc.set(t)
			t.Cleanup(func() {
				notificationsResetFlags(bucketNotificationsDeleteCmd, flagNames...)
			})

			err := runBucketNotificationsDelete(bucketNotificationsDeleteCmd, []string{"my-bucket"})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestBucketNotificationEventTypes_Map verifies the wrangler-compatible
// event-type groupings expand to their exact API action lists.
func TestBucketNotificationEventTypes_Map(t *testing.T) {
	cases := []struct {
		eventType string
		want      []cosmoflare.NotificationAction
	}{
		{
			"object-create",
			[]cosmoflare.NotificationAction{
				cosmoflare.ActionPutObject,
				cosmoflare.ActionCopyObject,
				cosmoflare.ActionCompleteMultipartUpload,
			},
		},
		{
			"object-delete",
			[]cosmoflare.NotificationAction{
				cosmoflare.ActionDeleteObject,
				cosmoflare.ActionLifecycleDeletion,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.eventType, func(t *testing.T) {
			got, ok := bucketNotificationEventTypes[tc.eventType]
			if !ok {
				t.Fatalf("event type %q missing from map", tc.eventType)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
	// Unknown groupings must be absent so the create runner rejects them.
	if _, ok := bucketNotificationEventTypes["object-replicate"]; ok {
		t.Fatal("unexpected event type present in map")
	}
}
