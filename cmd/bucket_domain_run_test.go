package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// bucketDomainFlagReset restores the given flags to their default values and
// clears their "changed" marks once the test finishes.
func bucketDomainFlagReset(t *testing.T, cmd *cobra.Command, names ...string) {
	t.Helper()
	for _, name := range names {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			continue
		}
		t.Cleanup(func() {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	}
}

// TestRunBucketDomainAttach_ArgumentValidation verifies the attach runner
// rejects a missing bucket argument and a missing --domain flag.
func TestRunBucketDomainAttach_ArgumentValidation(t *testing.T) {
	runGlobalsSnapshot(t)
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"no bucket", nil, "bucket name is required"},
		{"no domain flag", []string{"my-bucket"}, "--domain is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runBucketDomainAttach(bucketDomainAttachCmd, tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunBucketDomainAttach_ZoneResolutionFailsOffline verifies that without
// --zone-id and with empty credentials, attach aborts while creating the zone
// service — before any network call is possible.
func TestRunBucketDomainAttach_ZoneResolutionFailsOffline(t *testing.T) {
	runGlobalsSnapshot(t)
	bucketDomainFlagReset(t, bucketDomainAttachCmd, "domain", "zone-id")
	AccountID = ""
	APIToken = ""
	if err := bucketDomainAttachCmd.Flags().Set("domain", "cdn.example.com"); err != nil {
		t.Fatal(err)
	}

	err := runBucketDomainAttach(bucketDomainAttachCmd, []string{"my-bucket"})
	if err == nil || !strings.Contains(err.Error(), "failed to create zone service") {
		t.Fatalf("expected zone service error, got %v", err)
	}
}

// TestRunBucketDomainList_RequiresBucket verifies the list runner rejects a
// missing bucket argument before any service is built.
func TestRunBucketDomainList_RequiresBucket(t *testing.T) {
	runGlobalsSnapshot(t)

	err := runBucketDomainList(bucketDomainListCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "bucket name is required") {
		t.Fatalf("expected bucket-required error, got %v", err)
	}
}

// TestRunBucketDomain_TwoArgumentRequirement verifies the get, verify, update
// and detach runners all require both bucket and domain arguments.
func TestRunBucketDomain_TwoArgumentRequirement(t *testing.T) {
	runGlobalsSnapshot(t)
	cases := []struct {
		name string
		cmd  *cobra.Command
		run  func(cmd *cobra.Command, args []string) error
	}{
		{"get", bucketDomainGetCmd, runBucketDomainGet},
		{"verify", bucketDomainVerifyCmd, runBucketDomainVerify},
		{"update", bucketDomainUpdateCmd, runBucketDomainUpdate},
		{"detach", bucketDomainDetachCmd, runBucketDomainDetach},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(tc.cmd, []string{"my-bucket"})
			if err == nil || !strings.Contains(err.Error(), "bucket name and domain are required") {
				t.Fatalf("expected two-argument error, got %v", err)
			}
		})
	}
}

// TestRunBucketDomainVerify_InvalidTimeout verifies the verify runner rejects
// a malformed --timeout duration before any polling starts.
func TestRunBucketDomainVerify_InvalidTimeout(t *testing.T) {
	runGlobalsSnapshot(t)
	bucketDomainFlagReset(t, bucketDomainVerifyCmd, "timeout")
	if err := bucketDomainVerifyCmd.Flags().Set("timeout", "not-a-duration"); err != nil {
		t.Fatal(err)
	}

	err := runBucketDomainVerify(bucketDomainVerifyCmd, []string{"my-bucket", "cdn.example.com"})
	if err == nil || !strings.Contains(err.Error(), "invalid --timeout duration") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

// TestRunBucketDomainUpdate_FlagValidation verifies the update runner rejects
// mutually exclusive flags and a run with nothing to update.
func TestRunBucketDomainUpdate_FlagValidation(t *testing.T) {
	runGlobalsSnapshot(t)
	bucketDomainFlagReset(t, bucketDomainUpdateCmd, "enabled", "disabled", "min-tls", "cipher")
	cases := []struct {
		name    string
		set     func(*testing.T)
		wantErr string
	}{
		{"mutually exclusive", func(t *testing.T) {
			mustSet(t, bucketDomainUpdateCmd, "enabled", "true")
			mustSet(t, bucketDomainUpdateCmd, "disabled", "true")
		}, "mutually exclusive"},
		{"nothing to update", func(t *testing.T) {}, "nothing to update"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bucketDomainClearChanged(bucketDomainUpdateCmd, "enabled", "disabled", "min-tls", "cipher")
			tc.set(t)
			err := runBucketDomainUpdate(bucketDomainUpdateCmd, []string{"my-bucket", "cdn.example.com"})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

// bucketDomainClearChanged resets flags to their default values and clears
// their "changed" marks so subtests cannot inherit each other's flag state.
func bucketDomainClearChanged(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		if f := cmd.Flags().Lookup(name); f != nil {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		}
	}
}

// mustSet sets a command flag, failing the test on error.
func mustSet(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	if err := cmd.Flags().Set(name, value); err != nil {
		t.Fatalf("failed to set flag %s: %v", name, err)
	}
}

// TestBucketDomainStatuses verifies the status helper maps nil domains and
// blank status fields to "unknown" while passing real values through.
func TestBucketDomainStatuses(t *testing.T) {
	cases := []struct {
		name            string
		domain          *cosmoflare.BucketDomain
		wantOwner, want string
	}{
		{"nil domain", nil, "unknown", "unknown"},
		{"nil status", &cosmoflare.BucketDomain{}, "unknown", "unknown"},
		{"blank statuses filled in", &cosmoflare.BucketDomain{
			Status: &cosmoflare.BucketDomainStatus{},
		}, "unknown", "unknown"},
		{"active statuses", &cosmoflare.BucketDomain{
			Status: &cosmoflare.BucketDomainStatus{Ownership: "active", SSL: "active"},
		}, "active", "active"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ownership, ssl := bucketDomainStatuses(tc.domain)
			if ownership != tc.wantOwner || ssl != tc.want {
				t.Fatalf("statuses = (%q, %q), want (%q, %q)",
					ownership, ssl, tc.wantOwner, tc.want)
			}
		})
	}
}
