package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
)

// TASK-011 coverage for the cobra.Args-level resource-prefix mechanism.

func TestPrefixedResourceArgs(t *testing.T) {
	staging := &config.Profile{Name: "staging", ResourcePrefix: "stg-"}

	tests := []struct {
		name    string
		profile *config.Profile
		in      []string
		want    []string
	}{
		{"prefix applied to args[0]", staging, []string{"zone123", "rec-1"}, []string{"stg-zone123", "rec-1"}},
		{"already-prefixed args[0] passes through", staging, []string{"stg-zone123"}, []string{"stg-zone123"}},
		{"later args are untouched", staging, []string{"zone123", "zone123"}, []string{"stg-zone123", "zone123"}},
		{"no profile is a no-op", nil, []string{"zone123"}, []string{"zone123"}},
		{"profile without prefix is a no-op", &config.Profile{Name: "prod"}, []string{"zone123"}, []string{"zone123"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withEnvProfile(t, tt.profile)

			cmd := &cobra.Command{}
			args := append([]string(nil), tt.in...)
			if err := prefixedResourceArgs(cobra.ArbitraryArgs)(cmd, args); err != nil {
				t.Fatalf("wrapped validator returned error: %v", err)
			}
			if len(args) != len(tt.want) {
				t.Fatalf("args length changed: got %v, want %v", args, tt.want)
			}
			for i := range tt.want {
				if args[i] != tt.want[i] {
					t.Errorf("args[%d] = %q, want %q", i, args[i], tt.want[i])
				}
			}
		})
	}

	t.Run("empty args do not panic", func(t *testing.T) {
		withEnvProfile(t, staging)
		if err := prefixedResourceArgs(cobra.MinimumNArgs(0))(&cobra.Command{}, nil); err != nil {
			t.Fatalf("unexpected error on nil args: %v", err)
		}
	})

	t.Run("wrapped validator still enforces arity", func(t *testing.T) {
		withEnvProfile(t, staging)
		if err := prefixedResourceArgs(cobra.MinimumNArgs(1))(&cobra.Command{}, nil); err == nil {
			t.Fatal("expected arity error from wrapped MinimumNArgs(1)")
		}
		if err := prefixedResourceArgs(cobra.MinimumNArgs(2))(&cobra.Command{}, []string{"zone123"}); err == nil {
			t.Fatal("expected arity error from wrapped MinimumNArgs(2)")
		}
	})
}

func TestPrefixedFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("script", "", "worker script name")

	t.Run("prefix applied to flag value", func(t *testing.T) {
		withEnvProfile(t, &config.Profile{Name: "staging", ResourcePrefix: "stg-"})
		if err := cmd.Flags().Set("script", "my-worker"); err != nil {
			t.Fatal(err)
		}
		if got := prefixedFlag(cmd, "script"); got != "stg-my-worker" {
			t.Errorf("prefixedFlag = %q, want %q", got, "stg-my-worker")
		}
	})

	t.Run("already-prefixed flag value is idempotent", func(t *testing.T) {
		withEnvProfile(t, &config.Profile{Name: "staging", ResourcePrefix: "stg-"})
		if err := cmd.Flags().Set("script", "stg-my-worker"); err != nil {
			t.Fatal(err)
		}
		if got := prefixedFlag(cmd, "script"); got != "stg-my-worker" {
			t.Errorf("prefixedFlag = %q, want %q", got, "stg-my-worker")
		}
	})

	t.Run("no profile leaves flag value bare", func(t *testing.T) {
		withEnvProfile(t, nil)
		if err := cmd.Flags().Set("script", "my-worker"); err != nil {
			t.Fatal(err)
		}
		if got := prefixedFlag(cmd, "script"); got != "my-worker" {
			t.Errorf("prefixedFlag = %q, want %q", got, "my-worker")
		}
	})

	t.Run("unknown flag yields empty string", func(t *testing.T) {
		withEnvProfile(t, &config.Profile{Name: "staging", ResourcePrefix: "stg-"})
		if got := prefixedFlag(cmd, "no-such-flag"); got != "" {
			t.Errorf("prefixedFlag on unknown flag = %q, want empty", got)
		}
	})
}

// TestDNSPrefixedArgs_Invocation drives a dns command through the same
// sequence cobra runs — Args validator first, then RunE on the (rewritten)
// args slice — and asserts the runner observes the prefixed args[0] bare.
// DryRun keeps the run offline; rootCmd.Execute is deliberately avoided
// because its --env resolution would reset ActiveProfile.
func TestDNSPrefixedArgs_Invocation(t *testing.T) {
	t.Run("active profile rewrites args[0] before the runner", func(t *testing.T) {
		restore := bucketPrefixTestEnv(&config.Profile{Name: "staging", ResourcePrefix: "stg-"})
		defer restore()

		if dnsCreateCmd.Args == nil {
			t.Fatal("dnsCreateCmd must declare Args (prefixedResourceArgs)")
		}
		args := []string{"zone123"}
		if err := dnsCreateCmd.Args(dnsCreateCmd, args); err != nil {
			t.Fatalf("dns create Args validation failed: %v", err)
		}
		if args[0] != "stg-zone123" {
			t.Fatalf("args[0] after Args validation = %q, want %q", args[0], "stg-zone123")
		}

		out := capturePrint(t, func() {
			if err := runDNSCreate(dnsCreateCmd, args); err != nil {
				t.Errorf("runDNSCreate returned error: %v", err)
			}
		})
		if !strings.Contains(out, "stg-zone123") {
			t.Errorf("dry-run output should mention %q, got: %q", "stg-zone123", out)
		}
	})

	t.Run("without a profile args pass through unchanged", func(t *testing.T) {
		restore := bucketPrefixTestEnv(nil)
		defer restore()

		args := []string{"zone123"}
		if err := dnsCreateCmd.Args(dnsCreateCmd, args); err != nil {
			t.Fatalf("dns create Args validation failed: %v", err)
		}
		if args[0] != "zone123" {
			t.Fatalf("args[0] after Args validation = %q, want %q", args[0], "zone123")
		}

		out := capturePrint(t, func() {
			if err := runDNSCreate(dnsCreateCmd, args); err != nil {
				t.Errorf("runDNSCreate returned error: %v", err)
			}
		})
		if !strings.Contains(out, "zone123") {
			t.Errorf("dry-run output should mention %q, got: %q", "zone123", out)
		}
	})

	t.Run("arity is enforced by the wrapped validator", func(t *testing.T) {
		withEnvProfile(t, nil)
		if err := dnsGetCmd.Args(dnsGetCmd, []string{"zone123"}); err == nil {
			t.Fatal("dns get with one arg should fail Args validation")
		}
		if err := dnsListCmd.Args(dnsListCmd, nil); err == nil {
			t.Fatal("dns list with no args should fail Args validation")
		}
	})
}
