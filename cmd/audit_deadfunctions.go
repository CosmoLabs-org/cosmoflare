package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// FEAT-071: Cloudflare Pages ignores the functions/ directory entirely when
// _worker.js exists in the build output. Handlers that look deployed are
// dead: every request falls through to the worker (or the static asset), and
// nothing errors at build or deploy time. This audit surfaces the mismatch
// per-project: handlers present + _worker.js present ⇒ every functions/
// route is SHADOWED with the fix (port to framework routes or merge into the
// worker).

// deadFunctionsExit is stubbed in tests so the exit-1-when-shadowed path can
// be asserted without terminating the test binary (same pattern as searchExit).
var deadFunctionsExit = os.Exit

// deadFunctionsFixLine is the remediation surfaced for each dead endpoint,
// taken from the FEAT-071 issue (FB-31, MyCarGuide).
const deadFunctionsFixLine = "Port this handler to framework routes (e.g. Astro src/pages/api) or merge it into _worker.js — Cloudflare Pages ignores functions/ entirely when _worker.js exists."

// deadFunctionsWorkerPaths are the documented build-output locations checked
// for _worker.js, in precedence order (root, dist, build).
var deadFunctionsWorkerPaths = []string{"_worker.js", "dist/_worker.js", "build/_worker.js"}

// deadFunctionsShadowed is one dead endpoint: the functions/ route path and
// the handler file that will never execute.
type deadFunctionsShadowed struct {
	Route string `json:"route"`
	File  string `json:"file"`
}

// deadFunctionsReport is the --json payload: the shadowed routes (empty when
// the project is clean or there is nothing to compare) and the located
// _worker.js path, or null.
type deadFunctionsReport struct {
	Shadowed []deadFunctionsShadowed `json:"shadowed"`
	Worker   *string                 `json:"worker"`
}

var auditDeadFunctionsCmd = &cobra.Command{
	Use:   "dead-functions <project-dir>",
	Short: "Detect functions/ handlers shadowed by _worker.js (they never execute)",
	Long: `Audit a Cloudflare Pages project for dead functions/ handlers.

When the build output contains _worker.js, Cloudflare Pages ignores the
functions/ directory entirely — every handler in it silently never runs.
Requests that should hit those endpoints fall through to the worker or the
static asset, usually returning HTML with HTTP 200.

This command flags every functions/ route as SHADOWED when both are true:
functions/ contains .ts/.js handler files AND a _worker.js exists at
<project-dir>/_worker.js, <project-dir>/dist/_worker.js, or
<project-dir>/build/_worker.js.

Exit code is 1 when shadowed endpoints are found, 0 otherwise.

Examples:
  cosmoflare audit dead-functions ./web            # Human-readable report
  cosmoflare audit dead-functions ./web --json     # {"shadowed":[...],"worker":"..."}
  cosmoflare audit dead-functions "$PWD" --json && ./deploy.sh   # Gate a deploy`,
	Args: cobra.ExactArgs(1),
	RunE: runAuditDeadFunctions,
}

func init() {
	auditCmd.AddCommand(auditDeadFunctionsCmd)
}

// auditDeadFunctionsScan inspects the project directory and returns the
// shadowed-endpoint report. It never talks to the network.
func auditDeadFunctionsScan(projectDir string) (deadFunctionsReport, error) {
	report := deadFunctionsReport{Shadowed: []deadFunctionsShadowed{}}

	handlers, err := findFunctionsHandlers(projectDir)
	if err != nil {
		return report, err
	}
	if worker := findFunctionsWorkerOutput(projectDir); worker != "" {
		w := worker
		report.Worker = &w
		for _, f := range handlers {
			report.Shadowed = append(report.Shadowed, deadFunctionsShadowed{
				Route: functionsRoutePath(projectDir, f),
				File:  f,
			})
		}
	}
	return report, nil
}

// findFunctionsHandlers returns the sorted .ts/.js handler files under
// <projectDir>/functions, or an empty slice when the directory is absent.
func findFunctionsHandlers(projectDir string) ([]string, error) {
	root := filepath.Join(projectDir, "functions")
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return []string{}, nil
	}
	var handlers []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".ts" || ext == ".js" {
			handlers = append(handlers, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(handlers)
	return handlers, nil
}

// findFunctionsWorkerOutput returns the first existing _worker.js from the
// documented build-output locations, or "" when none is present.
func findFunctionsWorkerOutput(projectDir string) string {
	for _, rel := range deadFunctionsWorkerPaths {
		p := filepath.Join(projectDir, filepath.FromSlash(rel))
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// functionsRoutePath maps a handler file to the URL route Pages would have
// served from it: functions/api/subscribe.ts → /api/subscribe, and a trailing
// index segment collapses to its directory (functions/index.ts → /).
func functionsRoutePath(projectDir, file string) string {
	root := filepath.Join(projectDir, "functions")
	rel, err := filepath.Rel(root, file)
	if err != nil {
		rel = file
	}
	rel = filepath.ToSlash(rel)
	rel = strings.TrimSuffix(rel, filepath.Ext(rel))
	if base := path2Base(rel); base == "index" {
		rel = strings.TrimSuffix(rel, "index")
		rel = strings.TrimSuffix(rel, "/")
	}
	return "/" + rel
}

// path2Base returns the final segment of a slash-separated path.
func path2Base(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

func runAuditDeadFunctions(cmd *cobra.Command, args []string) error {
	projectDir := args[0]
	if st, err := os.Stat(projectDir); err != nil || !st.IsDir() {
		return outErrf("project directory not found: %s", projectDir)
	}

	report, err := auditDeadFunctionsScan(projectDir)
	if err != nil {
		return outErr("failed to scan functions/ directory", err)
	}

	if JSONOutput {
		return printJSON(report)
	}

	if report.Worker == nil {
		fmt.Println("No _worker.js found (checked _worker.js, dist/_worker.js, build/_worker.js) — functions/ handlers are live, nothing shadowed.")
		return nil
	}
	if len(report.Shadowed) == 0 {
		fmt.Printf("_worker.js at %s, but functions/ has no .ts/.js handler files — nothing shadowed.\n", *report.Worker)
		return nil
	}

	fmt.Printf("Cloudflare Pages ignores functions/ when _worker.js exists (%s) — %d dead endpoint(s):\n\n", *report.Worker, len(report.Shadowed))
	for _, s := range report.Shadowed {
		fmt.Printf("  SHADOWED  %-30s %s\n", s.Route, s.File)
	}
	fmt.Printf("\nFix: %s\n", deadFunctionsFixLine)
	deadFunctionsExit(1)
	return nil
}
