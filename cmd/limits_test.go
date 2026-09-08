package cmd

import (
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// The CLI wiring test asserts flag registration and help surface. Service
// behavior is covered by pkg/cosmoflare tests; rendering helpers are pure and
// tested directly below.
func TestLimitsCommandRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "limits" {
			found = true
		}
	}
	if !found {
		t.Fatal("limits command not registered on root")
	}
}

func TestSortRowsByPercent(t *testing.T) {
	rows := []cosmoflare.LimitRow{
		{Resource: "r2.buckets", Used: 1, Limit: 1000000, Percent: 0.1},
		{Resource: "dns.records", Scope: "a.io", Used: 90, Limit: 100, Percent: 90},
		{Resource: "workers.scripts", Used: 10, Limit: 100, Percent: 10},
	}
	got := sortRowsByPercent(rows)
	if got[0].Resource != "dns.records" || got[1].Resource != "workers.scripts" || got[2].Resource != "r2.buckets" {
		t.Fatalf("sort order wrong: %+v", got)
	}
}
