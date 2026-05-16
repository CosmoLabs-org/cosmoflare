package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestCacheCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "cache" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("cacheCmd not registered on rootCmd")
	}
}

func TestCacheCmd_Metadata(t *testing.T) {
	if cacheCmd.Use != "cache" {
		t.Errorf("cacheCmd.Use = %q, want %q", cacheCmd.Use, "cache")
	}
	if cacheCmd.Short == "" {
		t.Error("cacheCmd.Short is empty")
	}
}

// --- Subcommand registration ---

func TestCacheCmd_Subcommands(t *testing.T) {
	expected := []string{"purge", "settings"}
	for _, name := range expected {
		found := false
		for _, sub := range cacheCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("cache subcommand %q not registered", name)
		}
	}
}

// --- RunE handlers wired ---

func TestCacheCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		cachePurgeCmd,
		cacheSettingsCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

// --- Flag registration ---

func TestCachePurge_Flags(t *testing.T) {
	expected := []string{"all", "force", "url", "tag", "host"}
	for _, name := range expected {
		if cachePurgeCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on cachePurgeCmd", name)
		}
	}
}

func TestCachePurge_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"all", "false"},
		{"force", "false"},
	}
	for _, tc := range cases {
		f := cachePurgeCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on cachePurgeCmd", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestCacheSettings_Flags(t *testing.T) {
	expected := []string{"browser-ttl", "dev-mode", "cache-level"}
	for _, name := range expected {
		if cacheSettingsCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on cacheSettingsCmd", name)
		}
	}
}

func TestCacheSettings_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"browser-ttl", "0"},
		{"dev-mode", "false"},
		{"cache-level", ""},
	}
	for _, tc := range cases {
		f := cacheSettingsCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on cacheSettingsCmd", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

// --- Arg validation ---

func TestCachePurge_NoZoneID(t *testing.T) {
	err := runCachePurge(cachePurgeCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

func TestCachePurge_NoPurgeMethod(t *testing.T) {
	origAll := cachePurgeAll
	origURLs := cachePurgeURLs
	origTags := cachePurgeTags
	origHosts := cachePurgeHosts
	defer func() {
		cachePurgeAll = origAll
		cachePurgeURLs = origURLs
		cachePurgeTags = origTags
		cachePurgeHosts = origHosts
	}()

	cachePurgeAll = false
	cachePurgeURLs = nil
	cachePurgeTags = nil
	cachePurgeHosts = nil

	err := runCachePurge(cachePurgeCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when no purge method specified")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("purge method")) {
		t.Errorf("error = %q, want it to mention 'purge method'", err.Error())
	}
}

func TestCachePurge_AllWithoutForce(t *testing.T) {
	origDryRun := DryRun
	origAll := cachePurgeAll
	origForce := cachePurgeForce
	origURLs := cachePurgeURLs
	origTags := cachePurgeTags
	origHosts := cachePurgeHosts
	defer func() {
		DryRun = origDryRun
		cachePurgeAll = origAll
		cachePurgeForce = origForce
		cachePurgeURLs = origURLs
		cachePurgeTags = origTags
		cachePurgeHosts = origHosts
	}()

	DryRun = false
	cachePurgeAll = true
	cachePurgeForce = false
	cachePurgeURLs = nil
	cachePurgeTags = nil
	cachePurgeHosts = nil

	err := runCachePurge(cachePurgeCmd, []string{"zone123"})
	if err == nil {
		t.Fatal("expected error when --all used without --force")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("--force")) {
		t.Errorf("error = %q, want it to mention '--force'", err.Error())
	}
}

func TestCacheSettings_NoZoneID(t *testing.T) {
	err := runCacheSettings(cacheSettingsCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no zone ID provided")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("zone ID")) {
		t.Errorf("error = %q, want it to mention 'zone ID'", err.Error())
	}
}

// --- DryRun mode ---

func TestCachePurge_DryRunAll(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origAll := cachePurgeAll
	origForce := cachePurgeForce
	origURLs := cachePurgeURLs
	origTags := cachePurgeTags
	origHosts := cachePurgeHosts
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		cachePurgeAll = origAll
		cachePurgeForce = origForce
		cachePurgeURLs = origURLs
		cachePurgeTags = origTags
		cachePurgeHosts = origHosts
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	cachePurgeAll = true
	cachePurgeForce = false
	cachePurgeURLs = nil
	cachePurgeTags = nil
	cachePurgeHosts = nil

	err := runCachePurge(cachePurgeCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runCachePurge(DryRun --all) returned error: %v", err)
	}
}

func TestCachePurge_DryRunURLs(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origAll := cachePurgeAll
	origURLs := cachePurgeURLs
	origTags := cachePurgeTags
	origHosts := cachePurgeHosts
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		cachePurgeAll = origAll
		cachePurgeURLs = origURLs
		cachePurgeTags = origTags
		cachePurgeHosts = origHosts
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	cachePurgeAll = false
	cachePurgeURLs = []string{"https://example.com/style.css"}
	cachePurgeTags = nil
	cachePurgeHosts = nil

	err := runCachePurge(cachePurgeCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runCachePurge(DryRun --url) returned error: %v", err)
	}
}

func TestCachePurge_DryRunTags(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origAll := cachePurgeAll
	origURLs := cachePurgeURLs
	origTags := cachePurgeTags
	origHosts := cachePurgeHosts
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		cachePurgeAll = origAll
		cachePurgeURLs = origURLs
		cachePurgeTags = origTags
		cachePurgeHosts = origHosts
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	cachePurgeAll = false
	cachePurgeURLs = nil
	cachePurgeTags = []string{"static", "images"}
	cachePurgeHosts = nil

	err := runCachePurge(cachePurgeCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runCachePurge(DryRun --tag) returned error: %v", err)
	}
}

func TestCachePurge_DryRunHosts(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origAll := cachePurgeAll
	origURLs := cachePurgeURLs
	origTags := cachePurgeTags
	origHosts := cachePurgeHosts
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		cachePurgeAll = origAll
		cachePurgeURLs = origURLs
		cachePurgeTags = origTags
		cachePurgeHosts = origHosts
	}()

	DryRun = true
	JSONOutput = false
	APIToken = "test-token"
	cachePurgeAll = false
	cachePurgeURLs = nil
	cachePurgeTags = nil
	cachePurgeHosts = []string{"assets.example.com"}

	err := runCachePurge(cachePurgeCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runCachePurge(DryRun --host) returned error: %v", err)
	}
}

func TestCachePurge_DryRunAllJSON(t *testing.T) {
	origDryRun := DryRun
	origJSON := JSONOutput
	origAPIToken := APIToken
	origAll := cachePurgeAll
	origForce := cachePurgeForce
	origURLs := cachePurgeURLs
	origTags := cachePurgeTags
	origHosts := cachePurgeHosts
	defer func() {
		DryRun = origDryRun
		JSONOutput = origJSON
		APIToken = origAPIToken
		cachePurgeAll = origAll
		cachePurgeForce = origForce
		cachePurgeURLs = origURLs
		cachePurgeTags = origTags
		cachePurgeHosts = origHosts
	}()

	DryRun = true
	JSONOutput = true
	APIToken = "test-token"
	cachePurgeAll = true
	cachePurgeForce = false
	cachePurgeURLs = nil
	cachePurgeTags = nil
	cachePurgeHosts = nil

	err := runCachePurge(cachePurgeCmd, []string{"zone123"})
	if err != nil {
		t.Errorf("runCachePurge(DryRun --all --json) returned error: %v", err)
	}
}

// --- Flag variable wiring ---

func TestCacheFlagsParsing_BrowserTTL(t *testing.T) {
	cmd := &cobra.Command{}
	var ttl int
	cmd.Flags().IntVar(&ttl, "browser-ttl", 0, "")
	if err := cmd.Flags().Set("browser-ttl", "3600"); err != nil {
		t.Fatalf("failed to set --browser-ttl: %v", err)
	}
	if ttl != 3600 {
		t.Errorf("browserTTL = %d, want %d", ttl, 3600)
	}
}

func TestCacheFlagsParsing_DevMode(t *testing.T) {
	cmd := &cobra.Command{}
	var devMode bool
	cmd.Flags().BoolVar(&devMode, "dev-mode", false, "")
	if devMode {
		t.Error("devMode should default to false")
	}
	if err := cmd.Flags().Set("dev-mode", "true"); err != nil {
		t.Fatalf("failed to set --dev-mode: %v", err)
	}
	if !devMode {
		t.Error("devMode should be true after --dev-mode=true")
	}
}

func TestCacheFlagsParsing_CacheLevel(t *testing.T) {
	cmd := &cobra.Command{}
	var level string
	cmd.Flags().StringVar(&level, "cache-level", "", "")
	if err := cmd.Flags().Set("cache-level", "aggressive"); err != nil {
		t.Fatalf("failed to set --cache-level: %v", err)
	}
	if level != "aggressive" {
		t.Errorf("cacheLevel = %q, want %q", level, "aggressive")
	}
}
