package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/cmdmanifest"
)

// TestCliPathOf verifies path derivation from the live cobra tree and the
// nil-command contract (direct runner invocations bypass cobra wiring).
func TestCliPathOf(t *testing.T) {
	bucketDelete := cliPathOfCmdOr(bucketDeleteCmd, "bucket delete")
	if strings.Join(bucketDelete, " ") != "bucket delete" {
		t.Errorf("cliPathOfCmdOr(bucketDeleteCmd) = %v", bucketDelete)
	}
	if _, ok := cmdmanifest.Load().ResolveCLI(bucketDelete...); !ok {
		t.Errorf("derived path %v does not resolve in the registry", bucketDelete)
	}
	if got := cliPathOf(nil); got != nil {
		t.Errorf("cliPathOf(nil) = %v, want nil", got)
	}
	if got := cliPathOfCmdOr(nil, "d1 delete"); strings.Join(got, " ") != "d1 delete" {
		t.Errorf("nil-cmd fallback = %v, want [d1 delete]", got)
	}
}

// TestWithDestructiveDefaults pins the wrapper contract: a registry-
// destructive runner observes DryRun=true unless --force; an explicit
// --dry-run is never downgraded; non-destructive commands pass the
// operator's flag through verbatim.
func TestWithDestructiveDefaults(t *testing.T) {
	orig := DryRun
	defer func() { DryRun = orig }()

	observed := false
	runner := func(c *cobra.Command, args []string) error {
		observed = DryRun
		return nil
	}

	cases := []struct {
		name       string
		cmd        *cobra.Command
		dryRun     bool
		setForce   bool
		wantDryRun bool
	}{
		{"destructive defaults dry", bucketDeleteCmd, false, false, true},
		{"destructive force executes", bucketDeleteCmd, false, true, false},
		{"explicit dry-run wins", bucketDeleteCmd, true, true, true},
		{"non-destructive passthrough", dnsDeleteCmd, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			DryRun = tc.dryRun
			if err := tc.cmd.Flags().Set("force", boolStr(tc.setForce)); err != nil {
				t.Fatalf("force flag: %v", err)
			}
			observed = false
			if err := withDestructiveDefaults(tc.cmd, runner)(tc.cmd, []string{"x"}); err != nil {
				t.Fatalf("wrapped runner: %v", err)
			}
			if observed != tc.wantDryRun {
				t.Errorf("%s: runner observed DryRun=%v, want %v", tc.name, observed, tc.wantDryRun)
			}
			// reset for the next case
			if err := tc.cmd.Flags().Set("force", "false"); err != nil {
				t.Fatalf("force reset: %v", err)
			}
		})
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// TestDestructiveCommandsWrappedInDryDefault sweeps the registry: every
// command flagged Destructive, invoked through cobra wiring with empty
// credentials, must come back as a dry-run preview — never a service-
// construction error. That proves the registration wrapper covers every
// destructive command with zero runner-side wiring.
func TestDestructiveCommandsWrappedInDryDefault(t *testing.T) {
	runGlobalsSnapshot(t)
	JSONOutput = false

	m := cmdmanifest.Load()
	// One dummy argument satisfies every delete runner's arity check.
	dummy := map[string]string{
		"bucket delete": "bucket", "worker delete": "worker",
		"kv namespace delete": "ns", "d1 delete": "db",
		"waf list delete": "list", "hyperdrive delete": "config",
		"account member remove": "member", "logpush job delete": "123",
		"loadbalancer pool delete": "pool", "loadbalancer monitor delete": "mon",
		"tunnel delete": "tun", "tunnel cleanup": "tun",
	}
	for _, c := range m.Commands() {
		if !c.Destructive {
			continue
		}
		key := strings.Join(c.CLIPath, " ")
		t.Run(key, func(t *testing.T) {
			target, ok := lookupCommand(c.CLIPath)
			if !ok {
				t.Fatalf("command %q not found in the cobra tree", key)
			}
			if target.RunE == nil {
				t.Fatalf("command %q has no RunE", key)
			}
			out := capturePrint(t, func() {
				if err := target.RunE(target, []string{dummy[key]}); err != nil {
					t.Errorf("dry default should preview, got error: %v", err)
				}
			})
			if !strings.Contains(out, "DRY RUN") {
				t.Errorf("destructive %q did not run dry by default; output: %q", key, out)
			}
		})
	}
}

// lookupCommand resolves a registry CLI path against the live cobra tree.
func lookupCommand(path []string) (*cobra.Command, bool) {
	var current *cobra.Command = rootCmd
	for _, seg := range path {
		found := false
		for _, sub := range current.Commands() {
			if sub.Name() == seg {
				current, found = sub, true
				break
			}
		}
		if !found {
			return nil, false
		}
	}
	return current, true
}
