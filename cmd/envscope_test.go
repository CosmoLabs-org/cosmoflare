package cmd

import (
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
)

func TestApplyResourcePrefix(t *testing.T) {
	tests := []struct {
		name    string
		profile *config.Profile
		in      string
		want    string
	}{
		{
			name:    "no active profile is a no-op",
			profile: nil,
			in:      "assets",
			want:    "assets",
		},
		{
			name:    "profile without prefix is a no-op",
			profile: &config.Profile{Name: "staging"},
			in:      "assets",
			want:    "assets",
		},
		{
			name:    "prefix is prepended",
			profile: &config.Profile{Name: "staging", ResourcePrefix: "stg-"},
			in:      "assets",
			want:    "stg-assets",
		},
		{
			name:    "already-prefixed name passes through",
			profile: &config.Profile{Name: "staging", ResourcePrefix: "stg-"},
			in:      "stg-assets",
			want:    "stg-assets",
		},
		{
			name:    "empty name stays empty",
			profile: &config.Profile{Name: "staging", ResourcePrefix: "stg-"},
			in:      "",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ActiveProfile = tt.profile
			defer func() { ActiveProfile = nil }()

			if got := applyResourcePrefix(tt.in); got != tt.want {
				t.Errorf("applyResourcePrefix(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMatchesResourcePrefix(t *testing.T) {
	tests := []struct {
		name    string
		profile *config.Profile
		in      string
		want    bool
	}{
		{
			name:    "no active profile matches everything",
			profile: nil,
			in:      "anything",
			want:    true,
		},
		{
			name:    "profile without prefix matches everything",
			profile: &config.Profile{Name: "prod"},
			in:      "anything",
			want:    true,
		},
		{
			name:    "prefixed name matches",
			profile: &config.Profile{Name: "staging", ResourcePrefix: "stg-"},
			in:      "stg-assets",
			want:    true,
		},
		{
			name:    "unprefixed name does not match",
			profile: &config.Profile{Name: "staging", ResourcePrefix: "stg-"},
			in:      "prod-assets",
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ActiveProfile = tt.profile
			defer func() { ActiveProfile = nil }()

			if got := matchesResourcePrefix(tt.in); got != tt.want {
				t.Errorf("matchesResourcePrefix(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
