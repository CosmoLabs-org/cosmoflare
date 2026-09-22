package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// FEAT-018 P0: `pages deploy DIR --verify` — deploy, poll until the
// deployment answers, then GET each verify path and report a pass/fail
// table. Non-zero exit on any failed check so agents can gate on it.

var (
	pagesDeployProject string
	pagesDeployBranch  string
	pagesDeployVerify  []string
	pagesDeployBuildID string
)

const (
	pagesVerifyMaxAttempts = 10
	pagesVerifyInterval    = 5 * time.Second
	pagesVerifyHTTPTimeout = 15 * time.Second
)

// pagesVerifyGet is the HTTP probe seam (stubbed in tests).
var pagesVerifyGet = func(url string) (int, error) {
	client := &http.Client{Timeout: pagesVerifyHTTPTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

// pagesVerifySleep is the poll-interval seam (stubbed in tests).
var pagesVerifySleep = time.Sleep

// pagesEnvLookup is the environment seam for the build-id fallback chain.
var pagesEnvLookup = os.Getenv

// pagesGitShortRev runs `git rev-parse --short HEAD`, returning "" on any
// error (per contract: exec git, ignore error).
var pagesGitShortRev = func() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

var pagesDeployCmd = &cobra.Command{
	Use:   "deploy [directory]",
	Short: "Deploy to Cloudflare Pages and verify routes",
	Long: `Deploy a directory to a Cloudflare Pages project, then verify the deployment.

With --verify, after the deploy the command polls the deployment URL until it
is live (up to 10 attempts, 5s apart), then GETs each verify path and checks
for an HTTP 200. The exit code is non-zero when any check fails, so scripts
and agents can gate on it.

The build identifier is resolved as: --build-id flag > PUBLIC_BUILD_ID env
var > ` + "`git rev-parse --short HEAD`" + `.

Examples:
  cosmoflare pages deploy ./dist --project my-site --verify "/,/about,/health"
  cosmoflare pages deploy ./dist --project my-site --verify / --build-id abc1234 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runPagesDeploy,
}

func init() {
	pagesCmd.AddCommand(pagesDeployCmd)

	pagesDeployCmd.Flags().StringVar(&pagesDeployProject, "project", "", "Pages project name (defaults to the directory base name)")
	pagesDeployCmd.Flags().StringVar(&pagesDeployBranch, "branch", "", "Branch to deploy (defaults to the project's production branch)")
	pagesDeployCmd.Flags().StringSliceVar(&pagesDeployVerify, "verify", nil, "Comma-separated (or repeatable) paths to GET after deploy; any non-2xx fails")
	pagesDeployCmd.Flags().StringVar(&pagesDeployBuildID, "build-id", "", "Build identifier (flag > PUBLIC_BUILD_ID env > git short rev)")
}

// pagesVerifyCheck is one route probe result.
type pagesVerifyCheck struct {
	Path   string `json:"path"`
	Status int    `json:"status"`
	OK     bool   `json:"ok"`
	Ms     int64  `json:"ms"`
}

// pagesVerifyResult is the --json envelope.
type pagesVerifyResult struct {
	DeployedURL string             `json:"deployed_url"`
	BuildID     string             `json:"build_id"`
	Checks      []pagesVerifyCheck `json:"checks"`
}

func runPagesDeploy(cmd *cobra.Command, args []string) error {
	dir := args[0]

	project := pagesDeployProject
	if project == "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			abs = dir
		}
		project = filepath.Base(abs)
	}

	paths := parseVerifyPaths(pagesDeployVerify)
	buildID := resolvePagesBuildID(pagesDeployBuildID)

	if DryRun {
		return outPayload("DRY RUN: Would deploy and verify", func() any {
			return map[string]any{
				"directory":    dir,
				"project":      project,
				"branch":       pagesDeployBranch,
				"build_id":     buildID,
				"verify_paths": paths,
			}
		}, func() {
			printInfo("DRY RUN: Would deploy '%s' to Pages project '%s' (build-id %q)", dir, project, buildID)
			printInfo("DRY RUN: Would poll the deployment URL until live (up to %d attempts, %s apart)", pagesVerifyMaxAttempts, pagesVerifyInterval)
			for _, p := range paths {
				printInfo("DRY RUN: Would GET '%s' and require HTTP 200", p)
			}
		})
	}

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return outErrf("deploy directory %q does not exist or is not a directory", dir)
	}

	svc, err := getPagesService()
	if err != nil {
		return outErr("failed to create Pages service", err)
	}

	deployment, err := svc.CreateDeployment(context.Background(), project, pagesDeployBranch)
	if err != nil {
		return outErr("failed to create deployment", err)
	}
	deployedURL := deployment.URL

	printInfo("Deployment created (ID: %s)", deployment.ID)
	printInfo("Polling %s until live (build-id %q)...", deployedURL, buildID)

	live := pollPagesDeployment(deployedURL, paths)
	if !live {
		res := pagesVerifyResult{DeployedURL: deployedURL, BuildID: buildID, Checks: []pagesVerifyCheck{}}
		emitVerifyOutput(res, false)
		return fmt.Errorf("deployment %s did not become live within %d attempts (%s apart)", deployedURL, pagesVerifyMaxAttempts, pagesVerifyInterval)
	}

	res := pagesVerifyResult{
		DeployedURL: deployedURL,
		BuildID:     buildID,
		Checks:      checkVerifyPaths(deployedURL, paths),
	}
	allOK := true
	for _, c := range res.Checks {
		if !c.OK {
			allOK = false
		}
	}
	emitVerifyOutput(res, allOK)
	if !allOK {
		return fmt.Errorf("verify failed: %d of %d check(s) did not return HTTP 200", countFailedChecks(res.Checks), len(res.Checks))
	}
	return nil
}

// emitVerifyOutput renders the result as the JSON envelope or the human
// pass/fail table.
func emitVerifyOutput(res pagesVerifyResult, allOK bool) {
	if NewPresenter().IsJSON() {
		_ = printJSON(res)
		return
	}
	renderVerifyTable(os.Stdout, res)
	if allOK {
		printSuccess("All %d check(s) passed", len(res.Checks))
	} else {
		printError("%d of %d check(s) failed", countFailedChecks(res.Checks), len(res.Checks))
	}
}

// renderVerifyTable writes the path/status/ms table for a fixed check set.
func renderVerifyTable(w io.Writer, res pagesVerifyResult) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PATH\tSTATUS\tMS\tRESULT")
	for _, c := range res.Checks {
		result := "FAIL"
		if c.OK {
			result = "PASS"
		}
		fmt.Fprintf(tw, "%s\t%d\t%d\t%s\n", c.Path, c.Status, c.Ms, result)
	}
	tw.Flush()
	fmt.Fprintf(w, "Deployed URL: %s\n", res.DeployedURL)
	fmt.Fprintf(w, "Build ID:     %s\n", res.BuildID)
}

// countFailedChecks returns how many checks did not pass.
func countFailedChecks(checks []pagesVerifyCheck) int {
	n := 0
	for _, c := range checks {
		if !c.OK {
			n++
		}
	}
	return n
}

// parseVerifyPaths splits comma-separated specs, trims whitespace, drops
// empties, and dedupes while preserving order. Relative paths are kept
// as-is (they are joined onto the deployment URL later).
func parseVerifyPaths(specs []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(specs))
	for _, spec := range specs {
		for _, p := range strings.Split(spec, ",") {
			p = strings.TrimSpace(p)
			if p == "" || seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// pagesPollOutcome is the pure poll decision result.
type pagesPollOutcome int

const (
	pagesPollKeep   pagesPollOutcome = iota // keep polling
	pagesPollLive                           // deployment answered 2xx
	pagesPollFailed                         // attempts exhausted
)

// pagesPollDecision decides what to do after poll attempt `attempt`
// (1-based) which succeeded (`ok`, i.e. a 2xx on any target) or not.
func pagesPollDecision(attempt int, ok bool, maxAttempts int) pagesPollOutcome {
	if ok {
		return pagesPollLive
	}
	if attempt >= maxAttempts {
		return pagesPollFailed
	}
	return pagesPollKeep
}

// pollPagesDeployment polls the deployment URL (or any verify path when
// paths are given) until a 2xx is seen or attempts run out. Returns whether
// the deployment went live.
func pollPagesDeployment(deployedURL string, paths []string) bool {
	targets := paths
	if len(targets) == 0 {
		targets = []string{"/"}
	}
	for attempt := 1; attempt <= pagesVerifyMaxAttempts; attempt++ {
		ok := false
		for _, p := range targets {
			status, err := pagesVerifyGet(joinVerifyURL(deployedURL, p))
			if err == nil && status >= 200 && status < 300 {
				ok = true
				break
			}
		}
		switch pagesPollDecision(attempt, ok, pagesVerifyMaxAttempts) {
		case pagesPollLive:
			return true
		case pagesPollFailed:
			return false
		}
		pagesVerifySleep(pagesVerifyInterval)
	}
	return false
}

// checkVerifyPaths GETs each path once and records status/ok/ms.
func checkVerifyPaths(deployedURL string, paths []string) []pagesVerifyCheck {
	checks := make([]pagesVerifyCheck, 0, len(paths))
	for _, p := range paths {
		url := joinVerifyURL(deployedURL, p)
		start := time.Now()
		status, err := pagesVerifyGet(url)
		ms := time.Since(start).Milliseconds()
		checks = append(checks, pagesVerifyCheck{
			Path:   p,
			Status: status,
			OK:     err == nil && status >= 200 && status < 300,
			Ms:     ms,
		})
	}
	return checks
}

// joinVerifyURL joins a relative path onto the deployment base URL;
// absolute http(s) URLs pass through unchanged.
func joinVerifyURL(base, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(path, "/")
}

// resolvePagesBuildID implements the fallback chain:
// flag > PUBLIC_BUILD_ID env > git rev-parse --short HEAD.
func resolvePagesBuildID(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := pagesEnvLookup("PUBLIC_BUILD_ID"); v != "" {
		return v
	}
	return pagesGitShortRev()
}
