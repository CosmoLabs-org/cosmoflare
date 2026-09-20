package cmd

import (
	"strings"
	"testing"
)

// cacheRunGlobals snapshots and restores the package-level flag variables that
// runCachePurge and runCacheSettings read, so tests cannot leak state.
func cacheRunGlobals(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	oldAll, oldForce := cachePurgeAll, cachePurgeForce
	oldURLs, oldTags, oldHosts := cachePurgeURLs, cachePurgeTags, cachePurgeHosts
	oldTTL, oldDev, oldLevel := cacheBrowserTTL, cacheDevMode, cacheCacheLevel
	t.Cleanup(func() {
		cachePurgeAll, cachePurgeForce = oldAll, oldForce
		cachePurgeURLs, cachePurgeTags, cachePurgeHosts = oldURLs, oldTags, oldHosts
		cacheBrowserTTL, cacheDevMode, cacheCacheLevel = oldTTL, oldDev, oldLevel
	})
}

// cacheRunResetFlags clears slice/bool purge flags and flag "changed" marks so
// each subtest starts from a pristine command state.
func cacheRunResetFlags() {
	cachePurgeAll = false
	cachePurgeForce = false
	cachePurgeURLs = nil
	cachePurgeTags = nil
	cachePurgeHosts = nil
	for _, name := range []string{"browser-ttl", "dev-mode", "cache-level"} {
		if f := cacheSettingsCmd.Flags().Lookup(name); f != nil {
			f.Changed = false
		}
	}
}

// TestRunCachePurge_RequiresZoneID verifies that calling the purge runner
// without arguments fails fast with the zone-ID validation error.
func TestRunCachePurge_RequiresZoneID(t *testing.T) {
	cacheRunGlobals(t)
	cacheRunResetFlags()

	err := runCachePurge(cachePurgeCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "zone ID is required") {
		t.Fatalf("expected zone ID error, got %v", err)
	}
}

// TestRunCachePurge_RequiresPurgeMethod verifies that purging with no
// --all/--url/--tag/--host selection is rejected before any service is built.
func TestRunCachePurge_RequiresPurgeMethod(t *testing.T) {
	cacheRunGlobals(t)
	cacheRunResetFlags()

	err := runCachePurge(cachePurgeCmd, []string{"zone123"})
	if err == nil || !strings.Contains(err.Error(), "at least one purge method") {
		t.Fatalf("expected purge-method error, got %v", err)
	}
}

// TestRunCachePurge_AllWithoutForce verifies the destructive-purge guard:
// --all without --force (and not in dry-run) must be refused.
func TestRunCachePurge_AllWithoutForce(t *testing.T) {
	cacheRunGlobals(t)
	cacheRunResetFlags()
	cachePurgeAll = true
	cachePurgeForce = false
	DryRun = false

	err := runCachePurge(cachePurgeCmd, []string{"zone123"})
	if err == nil || !strings.Contains(err.Error(), "requires --force") {
		t.Fatalf("expected --force guard error, got %v", err)
	}
}

// TestRunCachePurge_AllWithoutForceInDryRun verifies that dry-run mode lifts
// the --force requirement for a purge-all and reports the would-be action.
func TestRunCachePurge_AllWithoutForceInDryRun(t *testing.T) {
	cacheRunGlobals(t)
	cacheRunResetFlags()
	cachePurgeAll = true
	cachePurgeForce = false
	DryRun = true
	APIToken = "fake-token-1234567890"

	if err := runCachePurge(cachePurgeCmd, []string{"zone123"}); err != nil {
		t.Fatalf("dry-run purge-all should succeed without --force: %v", err)
	}
}

// TestRunCachePurge_MissingToken verifies the service-construction error path
// surfaces when no API token is configured.
func TestRunCachePurge_MissingToken(t *testing.T) {
	cacheRunGlobals(t)
	cacheRunResetFlags()
	cachePurgeURLs = []string{"https://example.com/a.css"}
	DryRun = true
	APIToken = ""

	err := runCachePurge(cachePurgeCmd, []string{"zone123"})
	if err == nil || !strings.Contains(err.Error(), "failed to create cache service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
}

// TestRunCachePurge_DryRunSelectors verifies each non-all purge selector
// (URLs, tags, hosts) short-circuits in dry-run mode without network access.
func TestRunCachePurge_DryRunSelectors(t *testing.T) {
	cases := []struct {
		name string
		set  func()
	}{
		{"urls", func() { cachePurgeURLs = []string{"https://a.example.com/1", "https://a.example.com/2"} }},
		{"tags", func() { cachePurgeTags = []string{"static", "images"} }},
		{"hosts", func() { cachePurgeHosts = []string{"assets.example.com"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cacheRunGlobals(t)
			cacheRunResetFlags()
			tc.set()
			DryRun = true
			APIToken = "fake-token-1234567890"

			if err := runCachePurge(cachePurgeCmd, []string{"zone123"}); err != nil {
				t.Fatalf("dry-run purge by %s should succeed: %v", tc.name, err)
			}
		})
	}
}

// TestRunCachePurge_DryRunSelectorsJSON verifies the JSON envelopes for the
// dry-run selector paths still exit successfully.
func TestRunCachePurge_DryRunSelectorsJSON(t *testing.T) {
	cases := []struct {
		name string
		set  func()
	}{
		{"all", func() { cachePurgeAll = true }},
		{"urls", func() { cachePurgeURLs = []string{"https://a.example.com/1"} }},
		{"tags", func() { cachePurgeTags = []string{"static"} }},
		{"hosts", func() { cachePurgeHosts = []string{"cdn.example.com"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cacheRunGlobals(t)
			cacheRunResetFlags()
			tc.set()
			DryRun = true
			JSONOutput = true
			APIToken = "fake-token-1234567890"

			if err := runCachePurge(cachePurgeCmd, []string{"zone123"}); err != nil {
				t.Fatalf("dry-run JSON purge by %s should succeed: %v", tc.name, err)
			}
		})
	}
}

// TestRunCacheSettings_RequiresZoneID verifies the argument guard.
func TestRunCacheSettings_RequiresZoneID(t *testing.T) {
	cacheRunGlobals(t)

	err := runCacheSettings(cacheSettingsCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "zone ID is required") {
		t.Fatalf("expected zone ID error, got %v", err)
	}
}

// TestRunCacheSettings_MissingToken verifies the service-construction error
// path when no API token is configured.
func TestRunCacheSettings_MissingToken(t *testing.T) {
	cacheRunGlobals(t)
	APIToken = ""

	err := runCacheSettings(cacheSettingsCmd, []string{"zone123"})
	if err == nil || !strings.Contains(err.Error(), "failed to create cache service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
}

// TestRunCacheSettings_DryRunUpdate verifies that providing an update flag
// with --dry-run reports the would-be settings update and returns nil.
func TestRunCacheSettings_DryRunUpdate(t *testing.T) {
	cacheRunGlobals(t)
	cacheRunResetFlags()
	DryRun = true
	APIToken = "fake-token-1234567890"
	if err := cacheSettingsCmd.Flags().Set("browser-ttl", "3600"); err != nil {
		t.Fatalf("failed to set browser-ttl flag: %v", err)
	}

	if err := runCacheSettings(cacheSettingsCmd, []string{"zone123"}); err != nil {
		t.Fatalf("dry-run settings update should succeed: %v", err)
	}
}

// TestRunCacheSettings_DryRunUpdateJSON verifies the JSON dry-run envelope
// for settings updates.
func TestRunCacheSettings_DryRunUpdateJSON(t *testing.T) {
	cacheRunGlobals(t)
	cacheRunResetFlags()
	DryRun = true
	JSONOutput = true
	APIToken = "fake-token-1234567890"
	if err := cacheSettingsCmd.Flags().Set("cache-level", "aggressive"); err != nil {
		t.Fatalf("failed to set cache-level flag: %v", err)
	}

	if err := runCacheSettings(cacheSettingsCmd, []string{"zone123"}); err != nil {
		t.Fatalf("dry-run JSON settings update should succeed: %v", err)
	}
}
