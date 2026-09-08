package cosmoflare

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestLimitFor(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		plan     string
		want     uint64
		wantOK   bool
	}{
		{"workers scripts free", "workers.scripts", "free", 100, true},
		{"workers scripts paid", "workers.scripts", "paid", 500, true},
		{"workers daily requests free", "workers.daily_requests", "free", 100000, true},
		{"workers daily requests paid is unlimited", "workers.daily_requests", "paid", 0, true},
		{"r2 buckets plan-independent", "r2.buckets", "", 1000000, true},
		{"r2 custom domains plan-independent", "r2.custom_domains_per_bucket", "paid", 100, true},
		{"workers resource with unknown plan", "workers.scripts", "unknown", 0, false},
		{"unknown resource", "nope", "free", 0, false},
		{"workers resource with empty plan", "workers.scripts", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := limitFor(tt.resource, tt.plan)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("limitFor(%q, %q) = (%d, %v), want (%d, %v)",
					tt.resource, tt.plan, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestDNSRecordsStaticLimit(t *testing.T) {
	tests := []struct {
		name    string
		plan    string
		created string // ISO date; empty = zero time
		want    uint64
		wantOK  bool
	}{
		{"pro", "pro", "", 3500, true},
		{"business", "business", "", 3500, true},
		{"free before cutoff", "free", "2024-01-15", 1000, true},
		{"free on cutoff", "free", "2024-09-01", 200, true},
		{"free after cutoff", "free", "2025-06-01", 200, true},
		{"enterprise has no per-zone limit", "enterprise", "", 0, false},
		{"unknown plan", "", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created := time.Time{}
			if tt.created != "" {
				parsed, err := time.Parse("2006-01-02", tt.created)
				if err != nil {
					t.Fatalf("bad fixture date %q: %v", tt.created, err)
				}
				created = parsed
			}
			got, ok := dnsRecordsStaticLimit(tt.plan, created)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("dnsRecordsStaticLimit(%q, %v) = (%d, %v), want (%d, %v)",
					tt.plan, created, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestNewLimitsServiceDefaults(t *testing.T) {
	s := NewLimitsService("acct", "tok")
	if s.accountID != "acct" || s.apiToken != "tok" {
		t.Fatal("credentials not stored")
	}
	if s.httpClient == nil {
		t.Fatal("default HTTP client missing")
	}
	if s.baseURL != "https://api.cloudflare.com/client/v4" {
		t.Fatalf("baseURL = %q", s.baseURL)
	}
}

func TestNewLimitsServiceValidation(t *testing.T) {
	if _, err := NewLimitsService("", "tok").Snapshot(context.Background(), ""); err == nil {
		t.Fatal("empty account ID must fail validation")
	} else if !strings.Contains(err.Error(), "account ID is required") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := NewLimitsService("acct", "").Snapshot(context.Background(), ""); err == nil {
		t.Fatal("empty token must fail validation")
	} else if !strings.Contains(err.Error(), "API token is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
