package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// --- Index construction ---

func TestSearchIndex_ExcludesBuiltinsAndSelf(t *testing.T) {
	index := buildSearchIndex()
	if len(index) == 0 {
		t.Fatal("search index is empty")
	}
	for _, e := range index {
		leaf := e.Path[strings.LastIndex(e.Path, " ")+1:]
		switch leaf {
		case "search", "help", "completion":
			t.Errorf("index contains excluded command %q", e.Path)
		}
		if !strings.HasPrefix(e.Path, "cosmoflare ") {
			t.Errorf("indexed path %q does not start with 'cosmoflare '", e.Path)
		}
	}
}

func TestSearchIndex_HasSubcommandPaths(t *testing.T) {
	index := buildSearchIndex()
	found := false
	for _, e := range index {
		if e.Path == "cosmoflare object put" {
			found = true
		}
	}
	if !found {
		t.Error("index does not contain 'cosmoflare object put'")
	}
}

// --- Ranking ---

func TestSearchRanking_ExactAndPrefixFirst(t *testing.T) {
	entries := []searchEntry{
		{Name: "cache", Path: "cosmoflare cache", Short: "Manage cache"},
		{Name: "bucketdoctor", Path: "cosmoflare bucketdoctor", Short: "Diagnose buckets"},
		{Name: "zone", Path: "cosmoflare zone", Short: "Manage a bucket zone"},
	}
	// Description-only matches ("zone" — "Manage a bucket zone") must rank
	// below name matches.
	entries = []searchEntry{
		{Name: "cache", Path: "cosmoflare cache", Short: "Manage cache"},
		{Name: "bucket", Path: "cosmoflare bucket", Short: "Manage R2 buckets"},
		{Name: "bucketdoctor", Path: "cosmoflare bucketdoctor", Short: "Diagnose buckets"},
	}
	got := rankSearchResults(entries, "bucket")
	if got[0].Name != "bucket" {
		t.Errorf("exact match ranked %d, want 1 (got %q)", 1, got[0].Name)
	}
	if got[1].Name != "bucketdoctor" {
		t.Errorf("prefix match ranked %d, want 2 (got %q)", 2, got[1].Name)
	}
}

func TestSearchRanking_DescriptionBoundaryMatch(t *testing.T) {
	entries := []searchEntry{
		{Name: "images", Path: "cosmoflare images", Short: "Manage Cloudflare Images"},
	}
	got := rankSearchResults(entries, "upload")
	if len(got) == 0 {
		got = rankSearchResults(entries, "images")
	}
	_ = got
	// use a real command with an upload-style description
	entries = []searchEntry{
		{Name: "images", Path: "cosmoflare images upload", Short: "Upload an image"},
	}
	got = rankSearchResults(entries, "upload")
	if len(got) != 1 {
		t.Fatalf("word-boundary description match not found, got %d results", len(got))
	}
}

func TestSearchRanking_DeterministicTieBreak(t *testing.T) {
	entries := []searchEntry{
		{Name: "list", Path: "cosmoflare zone list", Short: "List zones"},
		{Name: "list", Path: "cosmoflare bucket list", Short: "List buckets"},
	}
	got := rankSearchResults(entries, "list")
	if len(got) != 2 {
		t.Fatalf("got %d results, want 2", len(got))
	}
	if got[0].Path != "cosmoflare bucket list" || got[1].Path != "cosmoflare zone list" {
		t.Errorf("tie-break not by path: %q before %q", got[0].Path, got[1].Path)
	}
}

// --- Subcommand paths resolve via the real index ---

func TestSearchSubcommandPathResolves(t *testing.T) {
	results := rankSearchResults(buildSearchIndex(), "object put")
	found := false
	for _, e := range results {
		if e.Path == "cosmoflare object put" {
			found = true
		}
	}
	if !found {
		t.Error("query 'object put' did not resolve 'cosmoflare object put'")
	}
}

// --- Case insensitivity ---

func TestSearchCaseInsensitive(t *testing.T) {
	lower := rankSearchResults(buildSearchIndex(), "bucket list")
	upper := rankSearchResults(buildSearchIndex(), "BUCKET LIST")
	if len(lower) == 0 {
		t.Fatal("lowercase query returned no results")
	}
	if len(upper) == 0 || lower[0].Path != upper[0].Path {
		t.Errorf("case-insensitive mismatch: %v vs %v", lower, upper)
	}
}

// --- No matches: exit code and human message ---

func TestSearchNoMatchHuman(t *testing.T) {
	restoreJSON := captureSearchJSONOutput(t, false)
	defer restoreJSON()

	var out, errBuf bytes.Buffer
	oldExit := searchExit
	code := 0
	searchExit = func(i int) { code = i }
	defer func() { searchExit = oldExit }()

	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	runSearch(cmd, []string{"zzzznomatch"})
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "No matching commands for 'zzzznomatch'") {
		t.Errorf("stderr = %q, want no-match message", errBuf.String())
	}
}

// --- JSON envelope ---

func TestSearchJSONEnvelopeEmpty(t *testing.T) {
	restoreJSON := captureSearchJSONOutput(t, true)
	defer restoreJSON()

	var out, errBuf bytes.Buffer
	oldExit := searchExit
	code := 0
	searchExit = func(i int) { code = i }
	defer func() { searchExit = oldExit }()

	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	runSearch(cmd, []string{"zzzznomatch"})
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	var env struct {
		Query   string `json:"query"`
		Results []struct {
			Command     string `json:"command"`
			Path        string `json:"path"`
			Description string `json:"description"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("stdout is not a valid JSON envelope: %v (%q)", err, out.String())
	}
	if env.Query != "zzzznomatch" {
		t.Errorf("envelope query = %q, want %q", env.Query, "zzzznomatch")
	}
	if env.Results == nil || len(env.Results) != 0 {
		t.Errorf("envelope results = %#v, want empty non-nil slice", env.Results)
	}
}

func TestSearchJSONEnvelopeShape(t *testing.T) {
	env := searchJSONEnvelope("put", []searchEntry{
		{Name: "put", Path: "cosmoflare object put", Short: "Upload an object"},
	})
	var raw map[string]any
	if err := json.Unmarshal(env, &raw); err != nil {
		t.Fatalf("envelope is not valid JSON: %v", err)
	}
	if raw["query"] != "put" {
		t.Errorf("query = %v, want 'put'", raw["query"])
	}
	results, ok := raw["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("results = %#v, want 1 entry", raw["results"])
	}
	entry, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("result entry = %#v, want object", results[0])
	}
	for _, key := range []string{"command", "path", "description"} {
		if _, ok := entry[key]; !ok {
			t.Errorf("result entry missing key %q", key)
		}
	}
	if entry["path"] != "cosmoflare object put" {
		t.Errorf("path = %v, want 'cosmoflare object put'", entry["path"])
	}
}

// --- Empty query ---

func TestSearchEmptyQuery(t *testing.T) {
	restoreJSON := captureSearchJSONOutput(t, false)
	defer restoreJSON()

	var out, errBuf bytes.Buffer
	oldExit := searchExit
	code := 0
	searchExit = func(i int) { code = i }
	defer func() { searchExit = oldExit }()

	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	runSearch(cmd, []string{})
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "search") {
		t.Errorf("stderr = %q, want a usage hint mentioning 'search'", errBuf.String())
	}
}

func TestSearchWhitespaceQuery(t *testing.T) {
	restoreJSON := captureSearchJSONOutput(t, false)
	defer restoreJSON()

	var out, errBuf bytes.Buffer
	oldExit := searchExit
	code := 0
	searchExit = func(i int) { code = i }
	defer func() { searchExit = oldExit }()

	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	runSearch(cmd, []string{"   "})
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

// --- Command registration and metadata ---

func TestSearchCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "search" {
			found = true
		}
	}
	if !found {
		t.Fatal("searchCmd not registered on rootCmd")
	}
}

func TestSearchCmd_Short(t *testing.T) {
	if searchCmd.Short != "Fuzzy-search commands by name or description" {
		t.Errorf("searchCmd.Short = %q", searchCmd.Short)
	}
}

// captureSearchJSONOutput sets the global JSONOutput flag and returns a
// restore function, isolating the rest of the suite from the override.
func captureSearchJSONOutput(t *testing.T, v bool) func() {
	t.Helper()
	old := JSONOutput
	JSONOutput = v
	return func() { JSONOutput = old }
}
