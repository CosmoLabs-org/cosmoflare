/*
Tests for the opt-in, env-gated update check (FEAT-043).

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package updatecheck

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// unsetenv restores the previous state of the env var after the test.
func unsetenv(t *testing.T, name string) {
	t.Helper()
	old, had := os.LookupEnv(name)
	if had {
		if err := os.Setenv(name, old); err != nil {
			t.Fatalf("restore env: %v", err)
		}
	} else if err := os.Unsetenv(name); err != nil {
		t.Fatalf("unset env: %v", err)
	}
}

func TestShouldRun(t *testing.T) {
	tests := []struct {
		name  string
		value *string // nil means unset
		want  bool
	}{
		{"unset", nil, false},
		{"one", strPtr("1"), true},
		{"true", strPtr("true"), false},
		{"zero", strPtr("0"), false},
		{"empty", strPtr(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == nil {
				if err := os.Unsetenv("COSMOFLARE_UPDATE_CHECK"); err != nil {
					t.Fatalf("unset env: %v", err)
				}
			} else {
				t.Setenv("COSMOFLARE_UPDATE_CHECK", *tt.value)
			}
			defer unsetenv(t, "COSMOFLARE_UPDATE_CHECK")
			if got := ShouldRun(); got != tt.want {
				t.Errorf("ShouldRun() = %v, want %v", got, tt.want)
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest  string
		current string
		want    bool
	}{
		{"v0.28.2", "0.28.2", false}, // equal (v-prefix on latest)
		{"0.28.2", "v0.28.2", false}, // equal (v-prefix on current)
		{"v0.28.3", "v0.28.2", true}, // newer, v-prefix on both
		{"0.28.1", "0.28.2", false},  // older
		{"0.28.10", "0.28.2", true},  // numeric, not lexicographic
		{"0.28.2", "0.28.10", false},
		{"0.28.1", "0.28", true},  // longer version with equal prefix is newer
		{"0.28", "0.28.1", false}, // shorter is older
		{"1.0.0", "0.99.99", true},
		{"", "0.28.2", false}, // malformed latest is never "newer"
		{"not-a-version", "0.28.2", false},
	}
	for _, tt := range tests {
		if got := IsNewer(tt.latest, tt.current); got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

// withTestURL points latestReleaseURL at the given URL and restores it after.
func withTestURL(t *testing.T, url string) {
	t.Helper()
	old := latestReleaseURL
	latestReleaseURL = url
	t.Cleanup(func() { latestReleaseURL = old })
}

func TestFetchLatest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("User-Agent"); !strings.HasPrefix(got, "cosmoflare/") {
				t.Errorf("User-Agent = %q, want prefix %q", got, "cosmoflare/")
			}
			_, _ = w.Write([]byte(`{"tag_name": "v9.9.9", "name": "release"}`))
		}))
		defer srv.Close()
		withTestURL(t, srv.URL)

		got, err := FetchLatest(context.Background())
		if err != nil {
			t.Fatalf("FetchLatest() error = %v", err)
		}
		if got != "v9.9.9" {
			t.Errorf("FetchLatest() = %q, want %q", got, "v9.9.9")
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{not json`))
		}))
		defer srv.Close()
		withTestURL(t, srv.URL)

		if _, err := FetchLatest(context.Background()); err == nil {
			t.Error("FetchLatest() error = nil, want error for malformed JSON")
		}
	})

	t.Run("HTTP 500", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer srv.Close()
		withTestURL(t, srv.URL)

		if _, err := FetchLatest(context.Background()); err == nil {
			t.Error("FetchLatest() error = nil, want error for HTTP 500")
		}
	})

	t.Run("context deadline", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(500 * time.Millisecond)
			_, _ = w.Write([]byte(`{"tag_name": "v9.9.9"}`))
		}))
		defer srv.Close()
		withTestURL(t, srv.URL)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		if _, err := FetchLatest(ctx); err == nil {
			t.Error("FetchLatest() error = nil, want error for expired context")
		}
	})
}

func TestMaybePrintNotice(t *testing.T) {
	newStderr := func() *bytes.Buffer { return &bytes.Buffer{} }

	t.Run("env unset: no output, no request", func(t *testing.T) {
		if err := os.Unsetenv("COSMOFLARE_UPDATE_CHECK"); err != nil {
			t.Fatalf("unset env: %v", err)
		}
		defer unsetenv(t, "COSMOFLARE_UPDATE_CHECK")

		hits := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hits++
			_, _ = w.Write([]byte(`{"tag_name": "v9.9.9"}`))
		}))
		defer srv.Close()
		withTestURL(t, srv.URL)

		stderr := newStderr()
		MaybePrintNotice(context.Background(), "0.28.2", stderr)
		if stderr.Len() != 0 {
			t.Errorf("stderr = %q, want empty", stderr.String())
		}
		if hits != 0 {
			t.Errorf("handler hits = %d, want 0", hits)
		}
	})

	t.Run("newer version: exactly one line", func(t *testing.T) {
		t.Setenv("COSMOFLARE_UPDATE_CHECK", "1")
		defer unsetenv(t, "COSMOFLARE_UPDATE_CHECK")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"tag_name": "v9.9.9"}`))
		}))
		defer srv.Close()
		withTestURL(t, srv.URL)

		stderr := newStderr()
		MaybePrintNotice(context.Background(), "0.28.2", stderr)
		got := stderr.String()
		if !strings.Contains(got, "update available") {
			t.Errorf("stderr = %q, want it to contain %q", got, "update available")
		}
		if strings.Count(got, "\n") != 1 {
			t.Errorf("stderr = %q, want exactly one line", got)
		}
		if !strings.Contains(got, "v9.9.9") || !strings.Contains(got, "0.28.2") {
			t.Errorf("stderr = %q, want latest and current versions present", got)
		}
	})

	t.Run("equal version: silent", func(t *testing.T) {
		t.Setenv("COSMOFLARE_UPDATE_CHECK", "1")
		defer unsetenv(t, "COSMOFLARE_UPDATE_CHECK")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"tag_name": "v0.28.2"}`))
		}))
		defer srv.Close()
		withTestURL(t, srv.URL)

		stderr := newStderr()
		MaybePrintNotice(context.Background(), "0.28.2", stderr)
		if stderr.Len() != 0 {
			t.Errorf("stderr = %q, want empty for equal version", stderr.String())
		}
	})

	t.Run("fetch error: no output", func(t *testing.T) {
		t.Setenv("COSMOFLARE_UPDATE_CHECK", "1")
		defer unsetenv(t, "COSMOFLARE_UPDATE_CHECK")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer srv.Close()
		withTestURL(t, srv.URL)

		stderr := newStderr()
		MaybePrintNotice(context.Background(), "0.28.2", stderr)
		if stderr.Len() != 0 {
			t.Errorf("stderr = %q, want empty on fetch error", stderr.String())
		}
	})
}
