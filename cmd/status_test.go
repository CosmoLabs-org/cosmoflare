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
