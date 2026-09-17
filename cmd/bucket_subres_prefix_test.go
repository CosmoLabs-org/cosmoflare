package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
)

// TestBucketPolicySet_PrefixScoping is the behavioral proof for FEAT-026
// part 3 wave A: bucket sub-resource runners apply the active profile's
// resource prefix to the bucket argument. policy set is the representative
// runner (it has a DryRun branch that echoes the bucket); the remaining
// domain/lifecycle/notifications/policy read sites carry the identical
// applyResourcePrefix wrap at their args[0] reads.
func TestBucketPolicySet_PrefixScoping(t *testing.T) {
	dir := t.TempDir()
	policyFile := filepath.Join(dir, "policy.json")
	if err := os.WriteFile(policyFile, []byte(`{"Version":"2012-10-17"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		profile *config.Profile
		arg     string
		want    string
	}{
		{"prefix applied", &config.Profile{Name: "staging", ResourcePrefix: "stg-"}, "my-bucket", "stg-my-bucket"},
		{"already prefixed passes through", &config.Profile{Name: "staging", ResourcePrefix: "stg-"}, "stg-my-bucket", "stg-my-bucket"},
		{"nil profile is a no-op", nil, "my-bucket", "my-bucket"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := bucketPrefixTestEnv(tt.profile)
			defer restore()

			oldFile := bucketPolicyFile
			bucketPolicyFile = policyFile
			defer func() { bucketPolicyFile = oldFile }()

			out := capturePrint(t, func() {
				if err := runBucketPolicySet(bucketPolicySetCmd, []string{tt.arg}); err != nil {
					t.Errorf("runBucketPolicySet returned error: %v", err)
				}
			})
			if !strings.Contains(out, tt.want) {
				t.Errorf("dry-run output should mention %q, got: %q", tt.want, out)
			}
		})
	}
}
