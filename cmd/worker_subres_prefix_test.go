package cmd

import (
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
)

// flagPrefixTestEnv sets the active profile plus the worker-subcommand flag
// sources under test, restoring everything afterwards.
func flagPrefixTestEnv(t *testing.T, profile *config.Profile) {
	t.Helper()
	restore := bucketPrefixTestEnv(profile)
	oldPattern, oldScript := workerRoutePattern, workerRouteScript
	oldService := workerDomainService
	t.Cleanup(func() {
		restore()
		workerRoutePattern, workerRouteScript = oldPattern, oldScript
		workerDomainService = oldService
	})
}

// TestWorkerRouteCreate_PrefixScoping: the --script flag references a worker
// by name and must carry the active profile's resource prefix (FEAT-026
// part 3 wave B, flag-sourced name).
func TestWorkerRouteCreate_PrefixScoping(t *testing.T) {
	tests := []struct {
		name    string
		profile *config.Profile
		script  string
		want    string
	}{
		{"prefix applied", &config.Profile{Name: "staging", ResourcePrefix: "stg-"}, "my-worker", "stg-my-worker"},
		{"already prefixed passes through", &config.Profile{Name: "staging", ResourcePrefix: "stg-"}, "stg-my-worker", "stg-my-worker"},
		{"nil profile is a no-op", nil, "my-worker", "my-worker"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagPrefixTestEnv(t, tt.profile)
			workerRoutePattern, workerRouteScript = "api.example.com/*", tt.script

			out := capturePrint(t, func() {
				if err := runWorkerRouteCreate(workerRouteCreateCmd, []string{"zone-123"}); err != nil {
					t.Errorf("runWorkerRouteCreate returned error: %v", err)
				}
			})
			if !strings.Contains(out, tt.want) {
				t.Errorf("dry-run output should mention %q, got: %q", tt.want, out)
			}
		})
	}
}

// TestWorkerDomainAttach_PrefixScoping: the --service flag references the
// worker by name; the hostname and zone stay untouched.
func TestWorkerDomainAttach_PrefixScoping(t *testing.T) {
	flagPrefixTestEnv(t, &config.Profile{Name: "staging", ResourcePrefix: "stg-"})
	workerDomainService, workerDomainZone = "my-worker", "zone-123"

	out := capturePrint(t, func() {
		if err := runWorkerDomainAttach(workerDomainAttachCmd, []string{"api.example.com"}); err != nil {
			t.Errorf("runWorkerDomainAttach returned error: %v", err)
		}
	})
	if !strings.Contains(out, "stg-my-worker") {
		t.Errorf("dry-run output should mention worker %q, got: %q", "stg-my-worker", out)
	}
	if !strings.Contains(out, "api.example.com") {
		t.Errorf("hostname must pass through unchanged, got: %q", out)
	}
}
