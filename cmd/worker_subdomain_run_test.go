package cmd

import (
	"strings"
	"testing"
)

// workerSubdomainGlobals snapshots the credential globals the subdomain
// commands read, so tests cannot leak state and cannot dial the API.
func workerSubdomainGlobals(t *testing.T) {
	t.Helper()
	oldAcct, oldToken := AccountID, APIToken
	AccountID, APIToken = "", ""
	t.Cleanup(func() { AccountID, APIToken = oldAcct, oldToken })
}

func TestWorkerSubdomainSetValidation(t *testing.T) {
	workerSubdomainGlobals(t)

	if err := runWorkerSubdomainSet(workerSubdomainSetCmd, nil); err == nil || !strings.Contains(err.Error(), "subdomain is required") {
		t.Fatalf("expected missing-arg error, got %v", err)
	}

	cases := []struct {
		name  string
		input string
	}{
		{"uppercase", "My-Team"},
		{"underscore", "my_team"},
		{"dot", "my.team"},
		{"space", "my team"},
		{"unicode", "tém"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runWorkerSubdomainSet(workerSubdomainSetCmd, []string{tc.input})
			if err == nil || !strings.Contains(err.Error(), "subdomain may only contain lowercase letters, digits, and hyphens") {
				t.Fatalf("expected format error for %q, got %v", tc.input, err)
			}
		})
	}
}

func TestWorkerSubdomainSetValidNameFailsOffline(t *testing.T) {
	workerSubdomainGlobals(t)
	// A well-formed name passes local validation and only then fails on
	// the missing credentials.
	err := runWorkerSubdomainSet(workerSubdomainSetCmd, []string{"my-team"})
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected offline service error, got %v", err)
	}
}

func TestWorkerSubdomainGetFailsOffline(t *testing.T) {
	workerSubdomainGlobals(t)
	err := runWorkerSubdomainGet(workerSubdomainGetCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create worker service") {
		t.Fatalf("expected offline service error, got %v", err)
	}
}

func TestWorkerSubdomainValidateSubdomainName(t *testing.T) {
	for _, ok := range []string{"my-team", "team123", "a", "1-2-3"} {
		if err := validateSubdomainName(ok); err != nil {
			t.Errorf("expected %q to be valid, got %v", ok, err)
		}
	}
	for _, bad := range []string{"", "My-Team", "my_team", "my.team"} {
		if err := validateSubdomainName(bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}

func TestRegisterWorkerSubdomainCmds(t *testing.T) {
	parent := workerCmd
	registerWorkerSubdomainCmds(parent)
	for _, name := range []string{"subdomain"} {
		found := false
		for _, c := range parent.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("worker tree missing %q subcommand", name)
		}
	}
	if found := workerSubdomainCmd.Commands(); len(found) < 2 {
		t.Fatalf("expected get and set subcommands under subdomain, got %d", len(found))
	}
}
