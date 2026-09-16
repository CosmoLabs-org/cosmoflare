/*
Package updatecheck implements the opt-in, env-gated update check (FEAT-043).

The check is completely inert unless the environment variable
COSMOFLARE_UPDATE_CHECK is set to exactly "1". Even when enabled it never
fails a command, never writes to stdout (which would corrupt --json output)
and never exits the process; the notice, if any, is a single line on stderr.

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// envVar gates the whole feature; anything other than "1" keeps it inert.
const envVar = "COSMOFLARE_UPDATE_CHECK"

// defaultReleaseURL is the GitHub API endpoint for the latest release.
const defaultReleaseURL = "https://api.github.com/repos/CosmoLabs-org/cosmoflare/releases/latest"

// latestReleaseURL is a var (not a const) so tests can point FetchLatest at
// an httptest server. Production code always reads the default.
var latestReleaseURL = defaultReleaseURL

// fetchTimeout bounds every request regardless of the caller's context.
const fetchTimeout = 2 * time.Second

// httpClient has a hard timeout so the check can never block a command
// beyond fetchTimeout even if the context has no deadline.
var httpClient = &http.Client{Timeout: fetchTimeout}

// versionForUserAgent is interpolated into the User-Agent header. It defaults
// to "dev" and is set by MaybePrintNotice to the running AppVersion.
var versionForUserAgent = "dev"

// release mirrors the subset of the GitHub releases/latest payload we need.
type release struct {
	TagName string `json:"tag_name"`
}

// ShouldRun reports whether the update check is enabled via the environment.
// It defaults to off: unset or any value other than "1" disables the check
// and guarantees zero network activity.
func ShouldRun() bool {
	return os.Getenv(envVar) == "1"
}

// LatestReleaseURL returns the URL the latest release is fetched from.
func LatestReleaseURL() string {
	return latestReleaseURL
}

// IsNewer does a pure semver-ish comparison: leading "v" is stripped, the
// rest is split on "." and compared numerically left-to-right. A longer
// version with an equal prefix is newer (0.28.2 < 0.28.10, 0.28 < 0.28.1).
// Malformed input is never considered newer.
func IsNewer(latest, current string) bool {
	l := parseSemver(latest)
	c := parseSemver(current)
	if l == nil {
		return false
	}
	for i := 0; i < len(l) || i < len(c); i++ {
		var lv, cv int
		if i < len(l) {
			lv = l[i]
		}
		if i < len(c) {
			cv = c[i]
		}
		if lv != cv {
			return lv > cv
		}
	}
	return false
}

// parseSemver converts "v1.2.3" into [1 2 3], or nil if the string is not a
// plain dotted-numeric version.
func parseSemver(v string) []int {
	if len(v) > 0 && (v[0] == 'v' || v[0] == 'V') {
		v = v[1:]
	}
	var out []int
	n := 0
	hasDigit := false
	for i := 0; i < len(v); i++ {
		switch ch := v[i]; {
		case ch == '.':
			if !hasDigit {
				return nil
			}
			out = append(out, n)
			n = 0
			hasDigit = false
		case ch >= '0' && ch <= '9':
			n = n*10 + int(ch-'0')
			hasDigit = true
		default:
			return nil
		}
	}
	if !hasDigit {
		return nil
	}
	return append(out, n)
}

// FetchLatest fetches the latest release tag from the GitHub API. It returns
// an error on any transport failure, non-200 status or malformed JSON; the
// caller is expected to stay silent on error.
func FetchLatest(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LatestReleaseURL(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "cosmoflare/"+versionForUserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("update check: unexpected status %s", resp.Status)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("update check: missing tag_name")
	}
	return rel.TagName, nil
}

// MaybePrintNotice composes the update check: it is silent unless the env
// gate is on AND the latest release is newer than current, in which case it
// writes exactly one line to w (which must be stderr in production so --json
// output on stdout is never polluted). Every failure mode is silent.
func MaybePrintNotice(ctx context.Context, current string, w io.Writer) {
	if !ShouldRun() {
		return
	}
	versionForUserAgent = current
	latest, err := FetchLatest(ctx)
	if err != nil {
		return
	}
	if !IsNewer(latest, current) {
		return
	}
	fmt.Fprintf(w, "cosmoflare: update available: %s (current %s) — https://github.com/CosmoLabs-org/cosmoflare/releases\n", latest, current)
}
