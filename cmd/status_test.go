package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "status" {
			found = true
			break
		}
	}
	assert.True(t, found, "status command should be registered on root")
}

func TestStatusCmd_HasRunE(t *testing.T) {
	assert.NotNil(t, statusCmd.RunE)
}

func TestStatusCmd_Flags(t *testing.T) {
	f := statusCmd.Flags()
	verbose := f.Lookup("verbose")
	assert.NotNil(t, verbose)
	assert.Equal(t, "false", verbose.DefValue)
}

func TestStatusReport_JSONFields(t *testing.T) {
	r := &StatusReport{
		Timestamp: "2026-01-01T00:00:00Z",
		AccountID: "test-account",
		Zones:     StatusSection{Count: 3},
		Workers:   StatusSection{Count: 5},
		KV:        StatusSection{Count: 2},
		Buckets:   StatusSection{Count: 10},
		SSL:       SSLStatusSection{Active: 3},
	}
	assert.Equal(t, 3, r.Zones.Count)
	assert.Equal(t, 5, r.Workers.Count)
	assert.Equal(t, 10, r.Buckets.Count)
	assert.Equal(t, 3, r.SSL.Active)
}

func TestPrintStatusLine_WithError(t *testing.T) {
	assert.NotPanics(t, func() {
		printStatusLine("Test", 0, "connection refused")
	})
}

func TestPrintStatusLine_Success(t *testing.T) {
	assert.NotPanics(t, func() {
		printStatusLine("Zones", 5, "")
	})
}

func TestPrintStatusDashboard_NoPanic(t *testing.T) {
	r := &StatusReport{
		Zones:   StatusSection{Count: 2},
		Workers: StatusSection{Count: 3, Error: "timeout"},
		SSL:     SSLStatusSection{Active: 1, Pending: 1},
		Errors:  []string{"workers: timeout"},
	}
	assert.NotPanics(t, func() {
		printStatusDashboard(r)
	})
}

// --- Long description content ---

func TestStatusCmd_LongDescription(t *testing.T) {
	assert.NotEmpty(t, statusCmd.Long, "statusCmd.Long should not be empty")
	assert.Contains(t, statusCmd.Long, "dashboard", "Long description should mention 'dashboard'")
	assert.Contains(t, statusCmd.Long, "status", "Long description should mention 'status'")
}

// --- No subcommands ---

func TestStatusCmd_NoSubcommands(t *testing.T) {
	assert.Equal(t, 0, len(statusCmd.Commands()), "statusCmd should have no subcommands")
}

// --- Verbose flag type ---

func TestStatusCmd_VerboseFlagType(t *testing.T) {
	f := statusCmd.Flags().Lookup("verbose")
	assert.NotNil(t, f)
	assert.Equal(t, "bool", f.Value.Type())
}

// --- Verbose flag usage string ---

func TestStatusCmd_VerboseFlagUsage(t *testing.T) {
	f := statusCmd.Flags().Lookup("verbose")
	assert.NotNil(t, f)
	assert.NotEmpty(t, f.Usage, "verbose flag should have usage text")
}

// --- StatusReport with all errors ---

func TestStatusReport_AllErrors(t *testing.T) {
	r := &StatusReport{
		Timestamp:   "2026-01-01T00:00:00Z",
		AccountID:   "test",
		Zones:       StatusSection{Count: 0, Error: "zones error"},
		Workers:     StatusSection{Count: 0, Error: "workers error"},
		KV:          StatusSection{Count: 0, Error: "kv error"},
		Buckets:     StatusSection{Count: 0, Error: "buckets error"},
		DNS:         StatusSection{Count: 0, Error: "dns error"},
		SSL:         SSLStatusSection{Error: "ssl error"},
		QueryTimeMs: 100,
		Errors:      []string{"zones error", "workers error", "kv error", "buckets error"},
	}
	assert.Equal(t, 4, len(r.Errors))
	assert.Equal(t, "zones error", r.Zones.Error)
	assert.Equal(t, "workers error", r.Workers.Error)
	assert.Equal(t, "kv error", r.KV.Error)
	assert.Equal(t, "buckets error", r.Buckets.Error)
}

// --- SSLStatusSection fields ---

func TestSSLStatusSection_Fields(t *testing.T) {
	ssl := SSLStatusSection{
		Active:   5,
		Pending:  2,
		Inactive: 1,
	}
	assert.Equal(t, 5, ssl.Active)
	assert.Equal(t, 2, ssl.Pending)
	assert.Equal(t, 1, ssl.Inactive)
	assert.Empty(t, ssl.Error)
}

// --- StatusReport zero-value ---

func TestStatusReport_ZeroValue(t *testing.T) {
	r := &StatusReport{}
	assert.Equal(t, 0, r.Zones.Count)
	assert.Equal(t, 0, r.Workers.Count)
	assert.Equal(t, 0, r.KV.Count)
	assert.Equal(t, 0, r.Buckets.Count)
	assert.Empty(t, r.Errors)
	assert.NotPanics(t, func() {
		printStatusDashboard(r)
	})
}
