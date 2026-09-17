package cmd

import (
	"os"
	"strings"
	"testing"
)

// buckWithStdin replaces os.Stdin with a pipe preloaded with input and
// restores the original on cleanup, so interactive prompt readers can be
// exercised deterministically.
func buckWithStdin(t *testing.T, input string) {
	t.Helper()
	saved := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = saved
		r.Close()
	})
}

// buckClearAuthEnv empties the Cloudflare credential environment variables
// for the duration of a test (an empty value behaves like unset for the
// os.Getenv checks in auth.go).
func buckClearAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_EMAIL", "")
}

// TestSelectAuthMethod verifies the interactive method chooser maps each
// accepted input to the right method and falls back to "token" on invalid
// or empty input.
func TestSelectAuthMethod(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"numeric one selects token", "1\n", "token"},
		{"numeric two selects key", "2\n", "key"},
		{"word token", "token\n", "token"},
		{"word key", "key\n", "key"},
		{"whitespace trimmed", "  2  \n", "key"},
		{"invalid choice defaults to token", "9\n", "token"},
		{"garbage defaults to token", "banana\n", "token"},
		{"EOF defaults to token", "", "token"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buckWithStdin(t, tc.input)
			if got := selectAuthMethod(); got != tc.want {
				t.Errorf("selectAuthMethod() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestPromptForAPIToken verifies the API token prompt trims surrounding
// whitespace and returns an empty string at EOF.
func TestPromptForAPIToken(t *testing.T) {
	t.Run("trims whitespace", func(t *testing.T) {
		buckWithStdin(t, "  tok-abcdef123456  \n")
		if got := promptForAPIToken(); got != "tok-abcdef123456" {
			t.Errorf("promptForAPIToken() = %q, want trimmed token", got)
		}
	})
	t.Run("empty line yields empty string", func(t *testing.T) {
		buckWithStdin(t, "\n")
		if got := promptForAPIToken(); got != "" {
			t.Errorf("promptForAPIToken() = %q, want empty", got)
		}
	})
	t.Run("EOF yields empty string without panic", func(t *testing.T) {
		buckWithStdin(t, "")
		if got := promptForAPIToken(); got != "" {
			t.Errorf("promptForAPIToken() = %q, want empty at EOF", got)
		}
	})
}

// TestPromptForAccountID verifies the account ID prompt returns trimmed
// input and an empty string at EOF.
func TestPromptForAccountID(t *testing.T) {
	t.Run("returns trimmed account id", func(t *testing.T) {
		buckWithStdin(t, "acct-1234567890abcdef\n")
		if got := promptForAccountID(); got != "acct-1234567890abcdef" {
			t.Errorf("promptForAccountID() = %q, want %q", got, "acct-1234567890abcdef")
		}
	})
	t.Run("EOF yields empty string without panic", func(t *testing.T) {
		buckWithStdin(t, "")
		if got := promptForAccountID(); got != "" {
			t.Errorf("promptForAccountID() = %q, want empty at EOF", got)
		}
	})
}

// TestPromptForEmail verifies the email prompt returns trimmed input and an
// empty string at EOF.
func TestPromptForEmail(t *testing.T) {
	t.Run("returns trimmed email", func(t *testing.T) {
		buckWithStdin(t, "  ops@example.com \n")
		if got := promptForEmail(); got != "ops@example.com" {
			t.Errorf("promptForEmail() = %q, want %q", got, "ops@example.com")
		}
	})
	t.Run("EOF yields empty string without panic", func(t *testing.T) {
		buckWithStdin(t, "")
		if got := promptForEmail(); got != "" {
			t.Errorf("promptForEmail() = %q, want empty at EOF", got)
		}
	})
}

// TestRunAuthStatus_PartialEnv verifies runAuthStatus fails with the
// missing-credentials error when only one of the two required environment
// variables is set, instead of attempting a connection.
func TestRunAuthStatus_PartialEnv(t *testing.T) {
	cases := []struct {
		name   string
		token  string
		acctID string
	}{
		{"token without account id", "tok-abc", ""},
		{"account id without token", "", "acct-abc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buckClearAuthEnv(t)
			if tc.token != "" {
				t.Setenv("CLOUDFLARE_API_TOKEN", tc.token)
			}
			if tc.acctID != "" {
				t.Setenv("CLOUDFLARE_ACCOUNT_ID", tc.acctID)
			}
			err := runAuthStatus(authStatusCmd, nil)
			if err == nil {
				t.Fatal("expected error when credentials are incomplete")
			}
			if !strings.Contains(err.Error(), "CLOUDFLARE_API_TOKEN") &&
				!strings.Contains(err.Error(), "CLOUDFLARE_ACCOUNT_ID") {
				t.Errorf("error = %q, want it to mention the missing env vars", err.Error())
			}
		})
	}
}

// TestRunAuthStatus_EmptyEnvIsError verifies the fully-unset case still
// surfaces the client creation error (guard against the error being
// silently swallowed after the profile checks).
func TestRunAuthStatus_EmptyEnvIsError(t *testing.T) {
	buckClearAuthEnv(t)
	if err := runAuthStatus(authStatusCmd, nil); err == nil {
		t.Fatal("expected error with no credentials configured")
	}
}
