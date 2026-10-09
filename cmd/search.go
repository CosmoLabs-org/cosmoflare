/*
Command search — fuzzy command discovery over the CLI's own command tree
(FEAT-pEKR6K6). Agents and humans find commands without walking --help
trees: `cosmoflare search "upload file"` ranks matching commands with
their full path and description, top 5, with a --json envelope.

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// searchExit is stubbed in tests so exit paths can be asserted without
// killing the test process.
var searchExit = os.Exit

// searchEntry is one indexed command leaf.
type searchEntry struct {
	Name  string
	Path  string
	Short string
}

// excludedSearchLeaves never appear in results: the search command itself
// and the cobra builtins.
var excludedSearchLeaves = map[string]bool{"search": true, "help": true, "completion": true}

// buildSearchIndex walks the live command tree and returns every leaf
// command with its full invocation path.
func buildSearchIndex() []searchEntry {
	var index []searchEntry
	var walk func(c *cobra.Command, path string)
	walk = func(c *cobra.Command, path string) {
		name := c.Name()
		if name != "" {
			path = strings.TrimSpace(path + " " + name)
		}
		if len(c.Commands()) == 0 {
			if name != "" && !excludedSearchLeaves[name] {
				index = append(index, searchEntry{Name: name, Path: path, Short: c.Short})
			}
			return
		}
		for _, sub := range c.Commands() {
			walk(sub, path)
		}
	}
	for _, sub := range rootCmd.Commands() {
		walk(sub, "cosmoflare")
	}
	return index
}

// atWordBoundary reports whether term occurs in s starting at a word
// boundary (start of string, or after a space, dash or underscore).
func atWordBoundary(s, term string) bool {
	for i := 0; i+len(term) <= len(s); i++ {
		if s[i:i+len(term)] != term {
			continue
		}
		if i == 0 || s[i-1] == ' ' || s[i-1] == '-' || s[i-1] == '_' {
			return true
		}
	}
	return false
}

// isSubsequence reports whether every rune of needle appears in haystack
// in order.
func isSubsequence(needle, haystack string) bool {
	ni, hi := 0, 0
	for ni < len(needle) && hi < len(haystack) {
		if needle[ni] == haystack[hi] {
			ni++
		}
		hi++
	}
	return ni == len(needle)
}

// termScore classifies one query term against an entry: name word-boundary
// beats a path substring beats a description word-boundary; a subsequence
// over the name is the weakest signal. Zero means no match for the term.
func termScore(e searchEntry, term string) int {
	name := strings.ToLower(e.Name)
	path := strings.ToLower(e.Path)
	desc := strings.ToLower(e.Short)
	switch {
	case atWordBoundary(name, term):
		return 30
	case strings.Contains(path, term):
		return 20
	case atWordBoundary(desc, term):
		return 10
	case isSubsequence(term, name):
		return 5
	}
	return 0
}

// rankSearchResults scores every entry against the (case-insensitive)
// query, keeps entries whose every term matched somewhere, and sorts by
// score descending with the invocation path as the deterministic
// tie-break. An exact name match always wins, a name prefix next.
func rankSearchResults(entries []searchEntry, query string) []searchEntry {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	terms := strings.Fields(q)
	type scored struct {
		entry searchEntry
		score int
	}
	var hits []scored
	for _, e := range entries {
		total := 0
		ok := true
		for _, term := range terms {
			s := termScore(e, term)
			if s == 0 {
				ok = false
				break
			}
			total += s
		}
		if !ok {
			continue
		}
		name := strings.ToLower(e.Name)
		if name == q {
			total += 100
		} else if strings.HasPrefix(name, q) {
			total += 60
		}
		hits = append(hits, scored{entry: e, score: total})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].entry.Path < hits[j].entry.Path
	})
	out := make([]searchEntry, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.entry)
	}
	return out
}

// searchJSONEnvelope renders the machine-readable result: query plus every
// result with command name, full path, and description.
func searchJSONEnvelope(query string, entries []searchEntry) []byte {
	type result struct {
		Command     string `json:"command"`
		Path        string `json:"path"`
		Description string `json:"description"`
	}
	env := struct {
		Query   string   `json:"query"`
		Results []result `json:"results"`
	}{Query: query}
	env.Results = make([]result, 0, len(entries)) // non-nil even when empty
	for _, e := range entries {
		env.Results = append(env.Results, result{Command: e.Name, Path: e.Path, Description: e.Short})
	}
	b, err := json.Marshal(env)
	if err != nil {
		return []byte(`{"query":"","results":[]}`)
	}
	return b
}

// runSearch backs the command: validates the query, ranks the live index,
// and prints either the human-aligned list or the JSON envelope. No match
// and an empty query both exit 1.
func runSearch(cmd *cobra.Command, args []string) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "search needs a query — try `cosmoflare search \"upload file\"` or list everything with `cosmoflare --help`")
		searchExit(1)
		return
	}
	query := args[0]
	results := rankSearchResults(buildSearchIndex(), query)
	if len(results) > 5 {
		results = results[:5]
	}
	if JSONOutput {
		cmd.OutOrStdout().Write(searchJSONEnvelope(query, results))
	} else if len(results) == 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "No matching commands for '%s'\n", query)
	} else {
		width := 0
		for _, r := range results {
			if len(r.Path) > width {
				width = len(r.Path)
			}
		}
		for _, r := range results {
			fmt.Fprintf(cmd.OutOrStdout(), "%-*s — %s\n", width, r.Path, r.Short)
		}
	}
	if len(results) == 0 {
		searchExit(1)
	}
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Fuzzy-search commands by name or description",
	Long: `Fuzzy-search commands by name or description.

Ranks the CLI's full command tree against the query — exact name matches
first, then name prefixes, then description hits — and prints the top 5
with their full invocation path. Exit code 1 when nothing matches.

Examples:
  cosmoflare search "upload file"        # find upload commands
  cosmoflare search bucket               # bucket + bucketdoctor
  cosmoflare search "object put" --json  # machine-readable envelope`,
	Args: cobra.MaximumNArgs(1),
	Run:  runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
