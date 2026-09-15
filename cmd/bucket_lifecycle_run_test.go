package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

// lifecycleRunGlobals snapshots and restores the package globals the
// lifecycle runners read, so tests cannot leak state between each other.
func lifecycleRunGlobals(t *testing.T) {
	t.Helper()
	oldAccount, oldToken := AccountID, APIToken
	oldDry, oldJSON := DryRun, JSONOutput
	t.Cleanup(func() {
		AccountID, APIToken = oldAccount, oldToken
		DryRun, JSONOutput = oldDry, oldJSON
	})
}

// lifecycleResetFlags restores the given flags on a command to their default
// values and clears their "changed" marks, so subtests start pristine.
func lifecycleResetFlags(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		if f := cmd.Flags().Lookup(name); f != nil {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		}
	}
}

var lifecycleSetFlagNames = []string{
	"file", "prefix", "expire-seconds", "abort-multipart-seconds",
	"transition-ia-seconds", "rule-id", "disabled", "force",
}

// lifecycleAgePtr returns a pointer to v for building expected MaxAge values.
func lifecycleAgePtr(v int64) *int64 { return &v }

// lifecycleWriteRules writes content to a rules file in a temp dir and
// returns its path.
func lifecycleWriteRules(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rules.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestRunBucketLifecycle_RequiresBucket verifies the get, set and clear
// runners all reject a missing bucket argument before any service is built.
func TestRunBucketLifecycle_RequiresBucket(t *testing.T) {
	lifecycleRunGlobals(t)
	cases := []struct {
		name string
		cmd  *cobra.Command
		run  func(cmd *cobra.Command, args []string) error
	}{
		{"get", bucketLifecycleGetCmd, runBucketLifecycleGet},
		{"set", bucketLifecycleSetCmd, runBucketLifecycleSet},
		{"clear", bucketLifecycleClearCmd, runBucketLifecycleClear},
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

// TestRunBucketLifecycleSet_InputValidation verifies the set runner rejects
// each malformed input mode before any network call is possible.
func TestRunBucketLifecycleSet_InputValidation(t *testing.T) {
	lifecycleRunGlobals(t)
	cases := []struct {
		name    string
		set     func(t *testing.T)
		wantErr string
	}{
		{"nothing to set", func(*testing.T) {}, "nothing to set"},
		{"file and shorthand", func(t *testing.T) {
			mustSet(t, bucketLifecycleSetCmd, "file", "rules.json")
			mustSet(t, bucketLifecycleSetCmd, "expire-seconds", "60")
		}, "mutually exclusive"},
		{"file not found", func(t *testing.T) {
			mustSet(t, bucketLifecycleSetCmd, "file", filepath.Join(t.TempDir(), "missing.json"))
		}, "failed to read"},
		{"invalid json", func(t *testing.T) {
			mustSet(t, bucketLifecycleSetCmd, "file", lifecycleWriteRules(t, "not-json"))
		}, "invalid rules file"},
		{"missing rules array", func(t *testing.T) {
			mustSet(t, bucketLifecycleSetCmd, "file", lifecycleWriteRules(t, `{"other": []}`))
		}, `missing "rules" array`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lifecycleResetFlags(bucketLifecycleSetCmd, lifecycleSetFlagNames...)
			tc.set(t)
			err := runBucketLifecycleSet(bucketLifecycleSetCmd, []string{"my-bucket"})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
	lifecycleResetFlags(bucketLifecycleSetCmd, lifecycleSetFlagNames...)
}

// TestParseLifecycleShorthandFlags verifies the shorthand flag parser builds
// the expected rule for each flag combination, including generated IDs.
func TestParseLifecycleShorthandFlags(t *testing.T) {
	cases := []struct {
		name string
		set  func(t *testing.T)
		want cosmoflare.LifecycleRule
	}{
		{"expire only", func(t *testing.T) {
			mustSet(t, bucketLifecycleSetCmd, "expire-seconds", "3600")
		}, cosmoflare.LifecycleRule{
			ID: "expire after 3600s", Enabled: true,
			DeleteObjectsTransition: &cosmoflare.LifecycleTransition{
				Condition: cosmoflare.LifecycleCondition{Type: "Age", MaxAge: lifecycleAgePtr(3600)},
			},
		}},
		{"abort only", func(t *testing.T) {
			mustSet(t, bucketLifecycleSetCmd, "abort-multipart-seconds", "604800")
		}, cosmoflare.LifecycleRule{
			ID: "abort multipart uploads after 604800s", Enabled: true,
			AbortMultipartUploadsTransition: &cosmoflare.LifecycleTransition{
				Condition: cosmoflare.LifecycleCondition{Type: "Age", MaxAge: lifecycleAgePtr(604800)},
			},
		}},
		{"transition only", func(t *testing.T) {
			mustSet(t, bucketLifecycleSetCmd, "transition-ia-seconds", "86400")
		}, cosmoflare.LifecycleRule{
			ID: "transition to InfrequentAccess after 86400s", Enabled: true,
			StorageClassTransitions: []cosmoflare.StorageClassTransition{{
				Condition:    cosmoflare.LifecycleCondition{Type: "Age", MaxAge: lifecycleAgePtr(86400)},
				StorageClass: "InfrequentAccess",
			}},
		}},
		{"all flags with prefix", func(t *testing.T) {
			mustSet(t, bucketLifecycleSetCmd, "prefix", "img/")
			mustSet(t, bucketLifecycleSetCmd, "expire-seconds", "10")
			mustSet(t, bucketLifecycleSetCmd, "abort-multipart-seconds", "20")
			mustSet(t, bucketLifecycleSetCmd, "transition-ia-seconds", "30")
		}, cosmoflare.LifecycleRule{
			ID:         `expire after 10s, abort multipart uploads after 20s, transition to InfrequentAccess after 30s under prefix "img/"`,
			Enabled:    true,
			Conditions: cosmoflare.LifecycleConditions{Prefix: "img/"},
			DeleteObjectsTransition: &cosmoflare.LifecycleTransition{
				Condition: cosmoflare.LifecycleCondition{Type: "Age", MaxAge: lifecycleAgePtr(10)},
			},
			AbortMultipartUploadsTransition: &cosmoflare.LifecycleTransition{
				Condition: cosmoflare.LifecycleCondition{Type: "Age", MaxAge: lifecycleAgePtr(20)},
			},
			StorageClassTransitions: []cosmoflare.StorageClassTransition{{
				Condition:    cosmoflare.LifecycleCondition{Type: "Age", MaxAge: lifecycleAgePtr(30)},
				StorageClass: "InfrequentAccess",
			}},
		}},
		{"custom id and disabled", func(t *testing.T) {
			mustSet(t, bucketLifecycleSetCmd, "expire-seconds", "60")
			mustSet(t, bucketLifecycleSetCmd, "rule-id", "custom-rule")
			mustSet(t, bucketLifecycleSetCmd, "disabled", "true")
		}, cosmoflare.LifecycleRule{
			ID: "custom-rule",
			DeleteObjectsTransition: &cosmoflare.LifecycleTransition{
				Condition: cosmoflare.LifecycleCondition{Type: "Age", MaxAge: lifecycleAgePtr(60)},
			},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lifecycleResetFlags(bucketLifecycleSetCmd, lifecycleSetFlagNames...)
			tc.set(t)
			got, err := parseLifecycleShorthandFlags(bucketLifecycleSetCmd)
			if err != nil {
				t.Fatalf("parseLifecycleShorthandFlags failed: %v", err)
			}
			lifecycleAssertRule(t, got, tc.want)
		})
	}
	lifecycleResetFlags(bucketLifecycleSetCmd, lifecycleSetFlagNames...)
}

// lifecycleAssertRule compares a parsed shorthand rule against the expected
// one field by field, including optional transitions.
func lifecycleAssertRule(t *testing.T, got, want cosmoflare.LifecycleRule) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.Enabled != want.Enabled {
		t.Errorf("Enabled = %v, want %v", got.Enabled, want.Enabled)
	}
	if got.Conditions.Prefix != want.Conditions.Prefix {
		t.Errorf("Prefix = %q, want %q", got.Conditions.Prefix, want.Conditions.Prefix)
	}
	lifecycleAssertTransition(t, "delete", got.DeleteObjectsTransition, want.DeleteObjectsTransition)
	lifecycleAssertTransition(t, "abort", got.AbortMultipartUploadsTransition, want.AbortMultipartUploadsTransition)
	if len(got.StorageClassTransitions) != len(want.StorageClassTransitions) {
		t.Fatalf("StorageClassTransitions len = %d, want %d",
			len(got.StorageClassTransitions), len(want.StorageClassTransitions))
	}
	for i := range got.StorageClassTransitions {
		if got.StorageClassTransitions[i].StorageClass != want.StorageClassTransitions[i].StorageClass {
			t.Errorf("StorageClass[%d] = %q, want %q", i,
				got.StorageClassTransitions[i].StorageClass, want.StorageClassTransitions[i].StorageClass)
		}
		gotAge := got.StorageClassTransitions[i].Condition.MaxAge
		wantAge := want.StorageClassTransitions[i].Condition.MaxAge
		if gotAge == nil || wantAge == nil || *gotAge != *wantAge {
			t.Errorf("MaxAge[%d] = %v, want %v", i, gotAge, wantAge)
		}
	}
}

// lifecycleAssertTransition compares two optional transitions by MaxAge.
func lifecycleAssertTransition(t *testing.T, label string, got, want *cosmoflare.LifecycleTransition) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Fatalf("%s transition presence = %v, want %v", label, got != nil, want != nil)
	}
	if got == nil {
		return
	}
	if got.Condition.MaxAge == nil || want.Condition.MaxAge == nil ||
		*got.Condition.MaxAge != *want.Condition.MaxAge {
		t.Errorf("%s MaxAge = %v, want %v", label, got.Condition.MaxAge, want.Condition.MaxAge)
	}
}

// TestLifecycleSeconds verifies the transition-condition renderer handles
// nil transitions, nil MaxAge, and real values.
func TestLifecycleSeconds(t *testing.T) {
	cases := []struct {
		name string
		tr   *cosmoflare.LifecycleTransition
		want string
	}{
		{"nil transition", nil, "-"},
		{"nil max age", &cosmoflare.LifecycleTransition{}, "-"},
		{"with age", &cosmoflare.LifecycleTransition{
			Condition: cosmoflare.LifecycleCondition{MaxAge: lifecycleAgePtr(30)},
		}, "30s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := lifecycleSeconds(tc.tr); got != tc.want {
				t.Errorf("lifecycleSeconds = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestLifecycleTransitionSummary verifies the storage-class summary renders
// empty, plain, and aged transitions.
func TestLifecycleTransitionSummary(t *testing.T) {
	cases := []struct {
		name string
		rule cosmoflare.LifecycleRule
		want string
	}{
		{"empty", cosmoflare.LifecycleRule{}, "-"},
		{"without max age", cosmoflare.LifecycleRule{
			StorageClassTransitions: []cosmoflare.StorageClassTransition{{StorageClass: "Glacier"}},
		}, "Glacier"},
		{"with max age", cosmoflare.LifecycleRule{
			StorageClassTransitions: []cosmoflare.StorageClassTransition{{
				StorageClass: "InfrequentAccess",
				Condition:    cosmoflare.LifecycleCondition{MaxAge: lifecycleAgePtr(90)},
			}},
		}, "InfrequentAccess@90s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := lifecycleTransitionSummary(tc.rule); got != tc.want {
				t.Errorf("lifecycleTransitionSummary = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestConfirmBucketLifecycleReplace verifies --force and dry-run mode both
// skip the interactive confirmation prompt.
func TestConfirmBucketLifecycleReplace(t *testing.T) {
	lifecycleRunGlobals(t)
	cases := []struct {
		name  string
		force bool
		dry   bool
	}{
		{"force", true, false},
		{"dry run", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			DryRun = tc.dry
			if !confirmBucketLifecycleReplace("my-bucket", "replace lifecycle rules", tc.force) {
				t.Fatalf("expected confirmation to be skipped for %s", tc.name)
			}
		})
	}
}

// TestGetBucketLifecycleService verifies the service factory returns a
// non-nil service for the given credentials.
func TestGetBucketLifecycleService(t *testing.T) {
	lifecycleRunGlobals(t)
	if getBucketLifecycleService() == nil {
		t.Fatal("expected non-nil lifecycle service")
	}
}

// TestRunBucketLifecycle_ServiceFailsOffline verifies the get, set and clear
// runners surface the wrapped service error when the API call fails (no
// reachable API with the fake credentials).
func TestRunBucketLifecycle_ServiceFailsOffline(t *testing.T) {
	lifecycleRunGlobals(t)
	AccountID = "acct-123"
	APIToken = "fake-token-1234567890"

	t.Run("get", func(t *testing.T) {
		err := runBucketLifecycleGet(bucketLifecycleGetCmd, []string{"my-bucket"})
		if err == nil || !strings.Contains(err.Error(), "failed to get lifecycle rules") {
			t.Fatalf("expected get service error, got %v", err)
		}
	})
	t.Run("set", func(t *testing.T) {
		lifecycleResetFlags(bucketLifecycleSetCmd, lifecycleSetFlagNames...)
		mustSet(t, bucketLifecycleSetCmd, "force", "true")
		mustSet(t, bucketLifecycleSetCmd, "expire-seconds", "60")
		err := runBucketLifecycleSet(bucketLifecycleSetCmd, []string{"my-bucket"})
		if err == nil || !strings.Contains(err.Error(), "failed to set lifecycle rules") {
			t.Fatalf("expected set service error, got %v", err)
		}
	})
	t.Run("clear", func(t *testing.T) {
		lifecycleResetFlags(bucketLifecycleClearCmd, "force")
		mustSet(t, bucketLifecycleClearCmd, "force", "true")
		err := runBucketLifecycleClear(bucketLifecycleClearCmd, []string{"my-bucket"})
		if err == nil || !strings.Contains(err.Error(), "failed to clear lifecycle rules") {
			t.Fatalf("expected clear service error, got %v", err)
		}
	})
	lifecycleResetFlags(bucketLifecycleSetCmd, lifecycleSetFlagNames...)
	lifecycleResetFlags(bucketLifecycleClearCmd, "force")
}
