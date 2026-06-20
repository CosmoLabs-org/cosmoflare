package cmd

import "testing"

func TestDomainsTUICmd_Registered(t *testing.T) {
	if domainsTUICmd == nil {
		t.Fatal("domainsTUICmd is nil")
	}
	if domainsTUICmd.RunE == nil {
		t.Error("domainsTUICmd has no RunE")
	}
	found := false
	for _, c := range domainsCmd.Commands() {
		if c.Name() == "tui" {
			found = true
			break
		}
	}
	if !found {
		t.Error("tui subcommand not registered under domains")
	}
}
