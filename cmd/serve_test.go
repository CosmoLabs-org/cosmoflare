package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
)

// newTestAdapter builds a serveAdapter against an isolated temp HOME with the
// keychain disabled, seeded with the given profiles (current = first). Keeps
// the Cloudflare-requiring methods out of scope — only the local logic is
// unit-testable here.
func newTestAdapter(t *testing.T, profiles ...*config.Profile) *serveAdapter {
	t.Helper()
	t.Setenv("COSMOFLARE_NO_KEYCHAIN", "1") // never probe the OS keychain
	t.Setenv("HOME", t.TempDir())

	cm, err := config.NewConfigManager()
	require.NoError(t, err)
	for _, p := range profiles {
		require.NoError(t, cm.SetProfile(p))
	}
	if len(profiles) > 0 {
		require.NoError(t, cm.SetCurrent(profiles[0].Name))
	}
	return &serveAdapter{cm: cm}
}

func TestServeAdapter_AccountsListsProfiles(t *testing.T) {
	a := newTestAdapter(t,
		&config.Profile{Name: "work", AccountID: "acc1", APIToken: "tok1"},
		&config.Profile{Name: "personal", AccountID: "acc2", APIToken: "tok2"},
	)

	got, err := a.Accounts(context.Background())
	require.NoError(t, err)

	list, ok := got.([]map[string]string)
	require.True(t, ok, "Accounts should return []map[string]string, got %T", got)
	names := make([]string, 0, len(list))
	for _, m := range list {
		names = append(names, m["name"])
	}
	assert.ElementsMatch(t, []string{"work", "personal"}, names)
}

func TestServeAdapter_AccountsEmptyWhenNoProfiles(t *testing.T) {
	a := newTestAdapter(t)
	got, err := a.Accounts(context.Background())
	require.NoError(t, err)
	list, ok := got.([]map[string]string)
	require.True(t, ok)
	assert.Empty(t, list)
}

func TestServeAdapter_ResolveProfile(t *testing.T) {
	a := newTestAdapter(t,
		&config.Profile{Name: "work", AccountID: "acc-work", APIToken: "tw"},
		&config.Profile{Name: "personal", AccountID: "acc-per", APIToken: "tp"},
	)

	// Named profile resolves to that profile's creds.
	p, err := a.resolveProfile("personal")
	require.NoError(t, err)
	assert.Equal(t, "acc-per", p.AccountID)

	// Empty name falls back to the current profile ("work").
	p, err = a.resolveProfile("")
	require.NoError(t, err)
	assert.Equal(t, "work", p.Name)

	// Unknown name errors (the daemon surfaces this as cloudflare_online=false).
	_, err = a.resolveProfile("does-not-exist")
	assert.Error(t, err)
}

func TestServeAdapter_CFMethodsErrorWithoutCreds(t *testing.T) {
	// No profiles configured → resolveProfile returns the "no current profile"
	// error. The CF methods must propagate it (not panic, not hang) so the REST
	// handler maps it to cloudflare_online=false + 502.
	a := newTestAdapter(t)
	ctx := context.Background()

	_, err := a.Zones(ctx, "")
	assert.Error(t, err)
	_, err = a.Workers(ctx, "")
	assert.Error(t, err)
	_, err = a.KV(ctx, "")
	assert.Error(t, err)
	_, err = a.R2Buckets(ctx, "")
	assert.Error(t, err)
}

func TestRandomToken(t *testing.T) {
	tok, err := randomToken()
	require.NoError(t, err)
	// 16 random bytes → 32 hex chars (128 bits).
	assert.Len(t, tok, 32)
	for _, c := range tok {
		assert.True(t, strings.ContainsRune("0123456789abcdef", c), "non-hex char %q", c)
	}
	// Two calls produce different tokens.
	tok2, err := randomToken()
	require.NoError(t, err)
	assert.NotEqual(t, tok, tok2)
}
