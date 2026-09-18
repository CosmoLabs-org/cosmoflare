package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestGetBucketNotificationService_NonNil verifies the service factory
// returns a usable service for both empty and populated credentials (the
// constructor itself performs no I/O).
func TestGetBucketNotificationService_NonNil(t *testing.T) {
	t.Run("empty creds", func(t *testing.T) {
		if svc := getBucketNotificationService(); svc == nil {
			t.Fatal("getBucketNotificationService() = nil, want non-nil service")
		}
	})
	t.Run("populated creds", func(t *testing.T) {
		oldAcct, oldTok := AccountID, APIToken
		t.Cleanup(func() { AccountID, APIToken = oldAcct, oldTok })
		AccountID, APIToken = "acct-1", "tok-1"

		if svc := getBucketNotificationService(); svc == nil {
			t.Fatal("getBucketNotificationService() = nil, want non-nil service")
		}
	})
}

// buckNotificationsCreateCmd builds a fresh command carrying the same flags
// the real create command registers, so tests never mutate global flag
// state shared with other tests.
func buckNotificationsCreateCmd() *cobra.Command {
	c := &cobra.Command{Use: "create [bucket]"}
	c.Flags().String("queue-id", "", "")
	c.Flags().StringArray("action", nil, "")
	c.Flags().String("event-type", "", "")
	c.Flags().String("prefix", "", "")
	c.Flags().String("suffix", "", "")
	c.Flags().String("description", "", "")
	return c
}

// TestRunBucketNotificationsCreate_InvalidEventType verifies an unknown
// --event-type grouping is rejected before any API call is attempted.
func TestRunBucketNotificationsCreate_InvalidEventType(t *testing.T) {
	c := buckNotificationsCreateCmd()
	if err := c.Flags().Set("queue-id", "queue-1"); err != nil {
		t.Fatal(err)
	}
	if err := c.Flags().Set("event-type", "object-rename"); err != nil {
		t.Fatal(err)
	}

	err := runBucketNotificationsCreate(c, []string{"my-bucket"})
	if err == nil || !strings.Contains(err.Error(), "invalid --event-type") {
		t.Fatalf("expected invalid --event-type error, got %v", err)
	}
}

// TestRunBucketNotificationsCreate_EventTypeGroupings verifies the two
// documented groupings pass validation by reaching the (failing) API call
// with empty credentials — proving the mapping path executes offline until
// the network boundary.
func TestRunBucketNotificationsCreate_EventTypeGroupings(t *testing.T) {
	oldAcct, oldTok := AccountID, APIToken
	t.Cleanup(func() { AccountID, APIToken = oldAcct, oldTok })
	AccountID, APIToken = "", ""

	for _, eventType := range []string{"object-create", "object-delete"} {
		t.Run(eventType, func(t *testing.T) {
			c := buckNotificationsCreateCmd()
			if err := c.Flags().Set("queue-id", "queue-1"); err != nil {
				t.Fatal(err)
			}
			if err := c.Flags().Set("event-type", eventType); err != nil {
				t.Fatal(err)
			}
			err := runBucketNotificationsCreate(c, []string{"my-bucket"})
			if err == nil || !strings.Contains(err.Error(), "failed to set notifications") {
				t.Fatalf("expected set-notifications error (offline), got %v", err)
			}
		})
	}
}

// TestRunBucketNotificationsGet_MissingBucketArg verifies the get runner
// requires a bucket argument.
func TestRunBucketNotificationsGet_MissingBucketArg(t *testing.T) {
	err := runBucketNotificationsGet(bucketNotificationsGetCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "bucket name is required") {
		t.Fatalf("expected bucket-required error, got %v", err)
	}
}

// TestRunBucketNotificationsList_MissingBucketArg verifies the list runner
// requires a bucket argument.
func TestRunBucketNotificationsList_MissingBucketArg(t *testing.T) {
	err := runBucketNotificationsList(bucketNotificationsListCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "bucket name is required") {
		t.Fatalf("expected bucket-required error, got %v", err)
	}
}

// TestRunBucketNotificationsDelete_MissingBucketArg verifies the delete
// runner requires a bucket argument.
func TestRunBucketNotificationsDelete_MissingBucketArg(t *testing.T) {
	err := runBucketNotificationsDelete(bucketNotificationsDeleteCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "bucket name is required") {
		t.Fatalf("expected bucket-required error, got %v", err)
	}
}
