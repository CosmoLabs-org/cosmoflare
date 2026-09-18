package cmd

import (
	"strings"
	"testing"
)

// buckDiffPopulatedConfig is a .cosmoflare.yaml that configures every
// service the diff subcommands compare, so the runners get past their
// empty-config short circuits.
const buckDiffPopulatedConfig = `name: diff-caps-test
workers:
  api:
    script: api.js
dns:
  zone_id: zone-abc123
  records:
    - type: A
      name: www.example.com
      content: 203.0.113.10
kv:
  namespaces:
    - title: CACHE
r2:
  buckets:
    - name: assets
`

// TestRunDiffAll_PopulatedConfigNoCreds verifies the all-services diff
// with configured resources stops at service creation when credentials
// are missing (offline).
func TestRunDiffAll_PopulatedConfigNoCreds(t *testing.T) {
	diffRunGlobals(t)
	diffRunChdir(t, buckDiffPopulatedConfig)
	AccountID, APIToken = "", ""

	err := runDiffAll(diffCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create diff service") {
		t.Fatalf("expected diff service error, got %v", err)
	}
}

// TestRunDiffWorkers_PopulatedConfigNoCreds verifies the workers diff with
// at least one configured worker reaches the service boundary before
// failing on missing credentials.
func TestRunDiffWorkers_PopulatedConfigNoCreds(t *testing.T) {
	diffRunGlobals(t)
	diffRunChdir(t, buckDiffPopulatedConfig)
	AccountID, APIToken = "", ""

	err := runDiffWorkers(diffWorkersCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create diff service") {
		t.Fatalf("expected diff service error, got %v", err)
	}
}

// TestRunDiffDNS_ZoneIDNoCreds verifies the DNS diff with dns.zone_id set
// reaches the service boundary before failing on missing credentials.
func TestRunDiffDNS_ZoneIDNoCreds(t *testing.T) {
	diffRunGlobals(t)
	diffRunChdir(t, buckDiffPopulatedConfig)
	AccountID, APIToken = "", ""

	err := runDiffDNS(diffDNSCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create diff service") {
		t.Fatalf("expected diff service error, got %v", err)
	}
}

// TestRunDiffKV_NamespacesNoCreds verifies the KV diff with configured
// namespaces reaches the service boundary before failing on credentials.
func TestRunDiffKV_NamespacesNoCreds(t *testing.T) {
	diffRunGlobals(t)
	diffRunChdir(t, buckDiffPopulatedConfig)
	AccountID, APIToken = "", ""

	err := runDiffKV(diffKVCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create diff service") {
		t.Fatalf("expected diff service error, got %v", err)
	}
}

// TestRunDiffR2_BucketsNoCreds verifies the R2 diff with configured
// buckets reaches the service boundary before failing on credentials.
func TestRunDiffR2_BucketsNoCreds(t *testing.T) {
	diffRunGlobals(t)
	diffRunChdir(t, buckDiffPopulatedConfig)
	AccountID, APIToken = "", ""

	err := runDiffR2(diffR2Cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create diff service") {
		t.Fatalf("expected diff service error, got %v", err)
	}
}
