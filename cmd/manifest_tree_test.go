package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/cmdmanifest"
)

// collectCobraPaths walks the live cobra tree and returns the path of every
// runnable leaf command (groups and the built-in help command excluded).
func collectCobraPaths(t *testing.T, groups ...string) map[string]bool {
	t.Helper()
	paths := map[string]bool{}
	var walk func(c *cobra.Command, prefix []string)
	walk = func(c *cobra.Command, prefix []string) {
		if c.Name() == "help" {
			return
		}
		path := append(append([]string{}, prefix...), c.Name())
		subs := c.Commands()
		if len(subs) == 0 {
			paths[strings.Join(path, " ")] = true
			return
		}
		for _, sub := range subs {
			walk(sub, path)
		}
	}
	for _, group := range groups {
		for _, sub := range rootCmd.Commands() {
			if sub.Name() == group {
				walk(sub, nil)
			}
		}
	}
	return paths
}

// TestManifestCLIPathsResolveAgainstCobraTree is the FEAT-020 cross-check:
// every registry entry must name a command that actually exists in the live
// cobra tree. A registered-but-missing path is silent breakage — the audit
// consumer would stamp "unregistered" for a real command, and permission
// consumers would never fire.
func TestManifestCLIPathsResolveAgainstCobraTree(t *testing.T) {
	live := collectCobraPaths(t, "worker", "kv", "d1", "dns", "email", "waf", "ssl", "cache", "hyperdrive", "alerts", "tunnel", "account", "logpush")
	if len(live) == 0 {
		t.Fatal("cobra tree walk found no commands under wave groups")
	}
	for _, c := range cmdmanifest.Load().Commands() {
		key := strings.Join(c.CLIPath, " ")
		switch c.CLIPath[0] {
		case "worker", "kv", "d1", "dns", "email", "waf", "ssl", "cache", "hyperdrive", "alerts", "tunnel", "account", "logpush":
			if !live[key] {
				t.Errorf("%q: registered CLI path %q not found in live cobra tree", c.ID, key)
			}
		case "r2", "sync", "watch":
			// wave-1 groups, verified by their own suite; nothing to do here
		default:
			t.Errorf("%q: CLI path %q belongs to no known wave group", c.ID, key)
		}
	}
}

// TestCobraTreeLeavesAllRegistered is the wave-completeness mirror of the
// cross-check: every live leaf under a wave-2 group must have a registry
// entry, so audit stamping and dry-run defaults cover the whole surface.
func TestCobraTreeLeavesAllRegistered(t *testing.T) {
	live := collectCobraPaths(t, "worker", "kv", "d1", "dns", "email", "waf", "ssl", "cache", "hyperdrive", "alerts", "tunnel", "account", "logpush")
	if len(live) == 0 {
		t.Fatal("cobra tree walk found no commands under wave groups")
	}
	m := cmdmanifest.Load()
	missing := make([]string, 0)
	for path := range live {
		segs := strings.Split(path, " ")
		if _, ok := m.ResolveCLI(segs...); !ok {
			missing = append(missing, path)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("%d live commands lack registry entries: %v", len(missing), missing)
	}
}

// TestDumpCobraTree is a development aid for authoring waves: prints the
// live leaf paths under the wave groups, sorted, one per line.
func TestDumpCobraTree(t *testing.T) {
	if os.Getenv("DUMP_TREE") == "" {
		t.Skip("set DUMP_TREE=1 to dump")
	}
	live := collectCobraPaths(t, "worker", "kv", "d1", "dns", "email", "waf", "ssl", "cache", "hyperdrive", "alerts", "tunnel", "account", "logpush")
	out := make([]string, 0, len(live))
	for p := range live {
		out = append(out, p)
	}
	sort.Strings(out)
	for _, p := range out {
		fmt.Println(p)
	}
}
