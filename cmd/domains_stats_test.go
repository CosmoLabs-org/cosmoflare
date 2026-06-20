package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// TestDomainsSubcommandsRegistered asserts the four domains subcommands exist
// and are wired under domainsCmd. This is pure registration wiring — no live
// Cloudflare access is performed.
func TestDomainsSubcommandsRegistered(t *testing.T) {
	subs := map[string]*cobra.Command{
		"get":       domainsGetCmd,
		"stats":     domainsStatsCmd,
		"ns":        domainsNSCmd,
		"redirects": domainsRedirectsCmd,
	}

	for name, cmd := range subs {
		if cmd == nil {
			t.Fatalf("domains subcommand %q var is nil", name)
		}
	}

	registered := make(map[string]bool)
	for _, c := range domainsCmd.Commands() {
		registered[c.Name()] = true
	}

	for name := range subs {
		if !registered[name] {
			t.Errorf("subcommand %q is not registered under domainsCmd", name)
		}
	}
}

// TestDomainsSubcommandsHaveRunE asserts each subcommand has a RunE handler and
// returns errors rather than calling os.Exit (we can't assert the latter
// directly, but a non-nil RunE is the contract the goal requires).
func TestDomainsSubcommandsHaveRunE(t *testing.T) {
	cmds := []*cobra.Command{domainsGetCmd, domainsStatsCmd, domainsNSCmd, domainsRedirectsCmd}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("subcommand %q has nil RunE", c.Name())
		}
	}
}

// TestDomainsServiceFactoryOverridable confirms newDomainService is a package
// var that tests can swap, and that the swapped stub is the one invoked. It
// restores the original factory afterward so other tests are unaffected.
func TestDomainsServiceFactoryOverridable(t *testing.T) {
	orig := newDomainService
	defer func() { newDomainService = orig }()

	var calledWith []bool
	newDomainService = func(enrich bool) (*cosmoflare.DomainService, error) {
		calledWith = append(calledWith, enrich)
		// Return a sentinel error so the caller short-circuits before any
		// network access. We only care that the stub was reached.
		return nil, errStubFactory
	}

	svc, err := newDomainService(true)
	if err != errStubFactory {
		t.Fatalf("expected stub factory error, got svc=%v err=%v", svc, err)
	}
	if len(calledWith) != 1 || calledWith[0] != true {
		t.Fatalf("stub factory not called as expected: %v", calledWith)
	}

	svc, err = newDomainService(false)
	if err != errStubFactory {
		t.Fatalf("expected stub factory error on second call, got svc=%v err=%v", svc, err)
	}
	if len(calledWith) != 2 || calledWith[1] != false {
		t.Fatalf("stub factory enrich arg not threaded through: %v", calledWith)
	}
}

// errStubFactory is a sentinel returned by the stubbed factory in tests.
var errStubFactory = stubErr("stub factory invoked")

type stubErr string

func (e stubErr) Error() string { return string(e) }
