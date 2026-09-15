package cmd

import (
	"context"
	"strings"
	"testing"
)

// statusRunGlobals snapshots the credential and output-mode globals read by
// runStatus and its collectors, and clears ambient Cloudflare credentials so
// every service construction fails offline before any network call.
func statusRunGlobals(t *testing.T) {
	t.Helper()
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")

	oldAccountID, oldAPIToken := AccountID, APIToken
	oldJSON, oldDry, oldVerbose := JSONOutput, DryRun, statusVerbose
	t.Cleanup(func() {
		AccountID, APIToken = oldAccountID, oldAPIToken
		JSONOutput, DryRun, statusVerbose = oldJSON, oldDry, oldVerbose
	})
	AccountID, APIToken = "", ""
	JSONOutput, DryRun, statusVerbose = false, false, false
}

// TestRunStatus_NoCredentials verifies runStatus succeeds offline when no
// credentials are configured: every collector records its construction
// error into the report and the dashboard still renders.
func TestRunStatus_NoCredentials(t *testing.T) {
	statusRunGlobals(t)

	if err := runStatus(statusCmd, nil); err != nil {
		t.Fatalf("runStatus without credentials should still render, got %v", err)
	}
}

// TestRunStatus_NoCredentialsJSON verifies the JSON envelope path of the
// offline dashboard.
func TestRunStatus_NoCredentialsJSON(t *testing.T) {
	statusRunGlobals(t)
	JSONOutput = true

	if err := runStatus(statusCmd, nil); err != nil {
		t.Fatalf("runStatus JSON without credentials should succeed, got %v", err)
	}
}

// TestRunStatus_VerboseOffline verifies the --verbose flag does not change
// the offline behavior.
func TestRunStatus_VerboseOffline(t *testing.T) {
	statusRunGlobals(t)
	statusVerbose = true

	if err := runStatus(statusCmd, nil); err != nil {
		t.Fatalf("runStatus --verbose without credentials should succeed, got %v", err)
	}
}

// TestCollectStatusSections_NoCredentials verifies each collector records
// the credential-validation error in its report section instead of
// attempting a network call.
func TestCollectStatusSections_NoCredentials(t *testing.T) {
	cases := []struct {
		name    string
		collect func(*StatusReport, func(string))
		wantErr string
	}{
		{
			"zones",
			func(r *StatusReport, add func(string)) {
				collectStatusZones(context.Background(), r, add)
			},
			"account ID is required",
		},
		{
			"workers",
			func(r *StatusReport, add func(string)) {
				collectStatusWorkers(context.Background(), r, add)
			},
			"account ID is required",
		},
		{
			"kv",
			func(r *StatusReport, add func(string)) {
				collectStatusKV(context.Background(), r, add)
			},
			"account ID is required",
		},
		{
			"r2",
			func(r *StatusReport, add func(string)) {
				collectStatusR2(context.Background(), r, add)
			},
			"CLOUDFLARE_ACCOUNT_ID is required",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			statusRunGlobals(t)

			report := &StatusReport{}
			var errs []string
			tc.collect(report, func(msg string) { errs = append(errs, msg) })

			if len(errs) != 1 || !strings.Contains(errs[0], tc.name) {
				t.Fatalf("expected one %q collector error, got %v", tc.name, errs)
			}
			sectionErr := statusSectionError(tc.name, report)
			if !strings.Contains(sectionErr, tc.wantErr) {
				t.Fatalf("section error %q missing %q", sectionErr, tc.wantErr)
			}
		})
	}
}

// statusSectionError returns the recorded error for a named section.
func statusSectionError(name string, r *StatusReport) string {
	switch name {
	case "zones":
		return r.Zones.Error
	case "workers":
		return r.Workers.Error
	case "kv":
		return r.KV.Error
	case "r2":
		return r.Buckets.Error
	}
	return ""
}
