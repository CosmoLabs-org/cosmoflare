package cmd

import (
	"strings"
	"testing"
)

// domainsRunSnapshot snapshots the domains list globals and the credential
// pair, restoring everything on cleanup.
func domainsRunSnapshot(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	origFilter, origName := domainsFilter, domainsName
	origPage, origPerPage := domainsPage, domainsPerPage
	origSort, origDetail, origEnrich := domainsSort, domainsDetail, domainsEnrich
	t.Cleanup(func() {
		domainsFilter, domainsName = origFilter, origName
		domainsPage, domainsPerPage = origPage, origPerPage
		domainsSort, domainsDetail, domainsEnrich = origSort, origDetail, origEnrich
	})
	domainsFilter, domainsName = "", ""
	domainsPage, domainsPerPage = 1, 50
	domainsSort = ""
	domainsDetail, domainsEnrich = false, false
}

// TestRunDomains_MissingCreds verifies the list runner fails fast with the
// zone-service construction error when credentials are absent.
func TestRunDomains_MissingCreds(t *testing.T) {
	domainsRunSnapshot(t)

	err := runDomains(domainsCmd, nil)
	if err == nil {
		t.Fatal("runDomains with empty credentials should return an error")
	}
	if !strings.Contains(err.Error(), "failed to create zone service") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to create zone service")
	}
}

// TestRunDomains_DryRunReportsOptions verifies the dry-run branch echoes the
// resolved list options. Dummy credentials pass construction; no request is
// made because dry-run short-circuits before the list call.
func TestRunDomains_DryRunReportsOptions(t *testing.T) {
	domainsRunSnapshot(t)
	AccountID, APIToken = "acct", "tok"
	DryRun = true
	domainsFilter = "active"
	domainsPage, domainsPerPage = 2, 10

	out := capturePrint(t, func() {
		if err := runDomains(domainsCmd, nil); err != nil {
			t.Errorf("dry-run should exit cleanly, got %v", err)
		}
	})

	for _, want := range []string{"DRY RUN: Would list domains", `filter="active"`, "page=2", "per-page=10"} {
		if !strings.Contains(out, want) {
			t.Errorf("dry-run output missing %q: %q", want, out)
		}
	}
}

// TestRunDomainsDetail_EmptyList verifies the detail renderer reports an
// empty result set instead of dereferencing the pagination envelope.
func TestRunDomainsDetail_EmptyList(t *testing.T) {
	domainsRunSnapshot(t)

	out := capturePrint(t, func() {
		if err := runDomainsDetail(nil, nil, nil, nil); err != nil {
			t.Errorf("empty detail list should exit cleanly, got %v", err)
		}
	})

	if !strings.Contains(out, "No domains found") {
		t.Errorf("expected empty-list notice, got %q", out)
	}
}

// The per-domain fallback branch of runDomainsDetail (GetDetail failure →
// minimal detail from list data) needs a stubbed DomainService seam;
// runDomains builds its service inline rather than through the
// newDomainService factory var, so that branch stays uncovered until the
// runner routes through the seam.
