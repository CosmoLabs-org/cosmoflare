package cmd

import (
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestWatchCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "watch [bucket] [directory]" || sub.Name() == "watch" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("watchCmd not registered on rootCmd")
	}
}

func TestWatchCmd_Metadata(t *testing.T) {
	if watchCmd.Name() != "watch" {
		t.Errorf("watchCmd.Name() = %q, want %q", watchCmd.Name(), "watch")
	}
	if watchCmd.Short == "" {
		t.Error("watchCmd.Short is empty")
	}
	if watchCmd.Long == "" {
		t.Error("watchCmd.Long is empty")
	}
}

// --- RunE handler wired ---

func TestWatchCmd_RunE(t *testing.T) {
	if watchCmd.RunE == nil {
		t.Error("watchCmd.RunE is nil")
	}
}

// --- Flag registration ---

func TestWatchCmd_Flags(t *testing.T) {
	expected := []string{"prefix", "exclude", "interval", "delete"}
	for _, name := range expected {
		if watchCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on watchCmd", name)
		}
	}
}

func TestWatchCmd_FlagDefaults(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"prefix", ""},
		{"interval", "1s"},
		{"delete", "false"},
	}
	for _, tc := range cases {
		f := watchCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Fatalf("flag --%s not found on watchCmd", tc.name)
		}
		if f.DefValue != tc.want {
			t.Errorf("flag --%s default = %q, want %q", tc.name, f.DefValue, tc.want)
		}
	}
}

func TestWatchCmd_ExcludeIsSlice(t *testing.T) {
	f := watchCmd.Flags().Lookup("exclude")
	if f == nil {
		t.Fatal("flag --exclude not found")
	}
	if f.Value.Type() != "stringSlice" {
		t.Errorf("--exclude type = %q, want 'stringSlice'", f.Value.Type())
	}
}

// --- Argument validation ---

func TestWatchCmd_RequiresBucketArg(t *testing.T) {
	cmd := &cobra.Command{}
	*cmd = *watchCmd
	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Error("watchCmd with no args should return error")
	}
}

func TestWatchCmd_LongIncludesExamples(t *testing.T) {
	if watchCmd.Long == "" {
		t.Fatal("Long is empty")
	}
	// Long description should include usage examples
	if len(watchCmd.Long) < 100 {
		t.Error("Long description is too short, should include examples")
	}
}

// --- Profile prefix scoping (FEAT-026) ---

func TestScopedWatchBucket(t *testing.T) {
	tests := []struct {
		name    string
		profile *config.Profile
		in      string
		want    string
	}{
		{"prefix applied", &config.Profile{Name: "staging", ResourcePrefix: "stg-"}, "my-bucket", "stg-my-bucket"},
		{"already prefixed passes through", &config.Profile{Name: "staging", ResourcePrefix: "stg-"}, "stg-my-bucket", "stg-my-bucket"},
		{"nil profile is a no-op", nil, "my-bucket", "my-bucket"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ActiveProfile = tt.profile
			defer func() { ActiveProfile = nil }()

			if got := scopedWatchBucket(tt.in); got != tt.want {
				t.Errorf("scopedWatchBucket(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
