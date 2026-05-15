package interactive

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// NewProfileManager
// ---------------------------------------------------------------------------

func TestNewProfileManager(t *testing.T) {
	pm, err := NewProfileManager()
	if err != nil {
		t.Skipf("Skipping profile manager tests: %v", err)
	}
	assert.NotNil(t, pm)
	assert.NotNil(t, pm.configMgr)
}

// ---------------------------------------------------------------------------
// FormatProfileName
// ---------------------------------------------------------------------------

func TestFormatProfileName_VariousInputs(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        string
	}{
		{"prod", "Production", "prod Production"},
		{"dev", "", "dev"},
		{"test", "Test", "test Test"},
		{"staging", "Staging Environment", "staging Staging Environment"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_"+tt.description, func(t *testing.T) {
			result := FormatProfileName(tt.name, tt.description)
			if tt.description == "" {
				assert.Equal(t, tt.want, result)
			} else {
				assert.Contains(t, result, tt.name)
				assert.Contains(t, result, tt.description)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// formatRegion
// ---------------------------------------------------------------------------

func TestFormatRegion_VariousInputs(t *testing.T) {
	tests := []struct {
		region string
		want   string
	}{
		{"auto", "Auto-detect"},
		{"", "Auto-detect"},
		{"us-east-1", "US-EAST-1"},
		{"eu-west-1", "EU-WEST-1"},
		{"ap-southeast-1", "AP-SOUTHEAST-1"},
		{"weur", "WEUR"},
		{"enam", "ENAM"},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			result := formatRegion(tt.region)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ---------------------------------------------------------------------------
// formatTokenStatus
// ---------------------------------------------------------------------------

func TestFormatTokenStatus_VariousInputs(t *testing.T) {
	tests := []struct {
		name  string
		token string
		check func(t *testing.T, result string)
	}{
		{
			"empty token",
			"",
			func(t *testing.T, result string) {
				assert.Contains(t, result, "Not set")
			},
		},
		{
			"too short token",
			"short",
			func(t *testing.T, result string) {
				assert.Contains(t, result, "Invalid")
			},
		},
		{
			"exactly 10 chars",
			"0123456789",
			func(t *testing.T, result string) {
				assert.Contains(t, result, "Configured")
			},
		},
		{
			"long token",
			"this-is-a-valid-token-12345678",
			func(t *testing.T, result string) {
				assert.Contains(t, result, "Configured")
			},
		},
		{
			"exactly 20 chars",
			"12345678901234567890",
			func(t *testing.T, result string) {
				assert.Contains(t, result, "Configured")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTokenStatus(tt.token)
			tt.check(t, result)
		})
	}
}

// ---------------------------------------------------------------------------
// formatAccountIDStatus
// ---------------------------------------------------------------------------

func TestFormatAccountIDStatus_VariousInputs(t *testing.T) {
	tests := []struct {
		name      string
		accountID string
		check     func(t *testing.T, result string)
	}{
		{
			"empty account ID",
			"",
			func(t *testing.T, result string) {
				assert.Contains(t, result, "Not set")
			},
		},
		{
			"too short",
			"short",
			func(t *testing.T, result string) {
				assert.Contains(t, result, "Invalid format")
			},
		},
		{
			"too long",
			"abcdef0123456789abcdef0123456789extra",
			func(t *testing.T, result string) {
				assert.Contains(t, result, "Invalid format")
			},
		},
		{
			"exactly 32 chars",
			strings.Repeat("a", 32),
			func(t *testing.T, result string) {
				assert.Contains(t, result, "Valid format")
			},
		},
		{
			"valid hex",
			"abcdef0123456789abcdef0123456789",
			func(t *testing.T, result string) {
				assert.Contains(t, result, "Valid format")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAccountIDStatus(tt.accountID)
			tt.check(t, result)
		})
	}
}

// ---------------------------------------------------------------------------
// getProfileIcon
// ---------------------------------------------------------------------------

func TestGetProfileIcon_VariousDescriptions(t *testing.T) {
	tests := []struct {
		description string
		want        string
	}{
		{"Production server", "🏭 "},
		{"prod environment", "🏭 "},
		{"Production account", "🏭 "},
		{"staging server", "⚡ "},
		{"dev environment", "⚡ "},
		{"development box", "⚡ "},
		{"test profile", "🧪 "},
		{"testing account", "🧪 "},
		{"personal bucket", "💻 "},
		{"my personal stuff", "💻 "},
		{"work account", "🏢 "},
		{"company bucket", "🏢 "},
		{"random description", "📁 "},
		{"", "📁 "},
		{"backup storage", "📁 "},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := getProfileIcon(tt.description)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ---------------------------------------------------------------------------
// getProfileDescription
// ---------------------------------------------------------------------------

func TestGetProfileDescription_VariousInputs(t *testing.T) {
	tests := []struct {
		description string
		want        string
	}{
		{"", "(no description)"},
		{"short", "(short)"},
		{"a description under 30 chars", "(a description under 30 chars)"},
		{"this is a very long description that exceeds thirty", "(this is a very long descrip...)"},
		{"x", "(x)"},
		{"a", "(a)"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := getProfileDescription(tt.description)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ---------------------------------------------------------------------------
// getProfileDescription edge cases
// ---------------------------------------------------------------------------

func TestGetProfileDescription_EdgeCases(t *testing.T) {
	t.Run("exactly 30 chars", func(t *testing.T) {
		input := strings.Repeat("a", 30)
		result := getProfileDescription(input)
		assert.Equal(t, "("+input+")", result)
		assert.Equal(t, 32, len(result)) // 30 + 2 parens
	})

	t.Run("exactly 31 chars", func(t *testing.T) {
		input := strings.Repeat("b", 31)
		result := getProfileDescription(input)
		assert.True(t, strings.HasPrefix(result, "("))
		assert.True(t, strings.HasSuffix(result, ")"))
		assert.Contains(t, result, "...")
	})

	t.Run("very long description", func(t *testing.T) {
		input := strings.Repeat("x", 100)
		result := getProfileDescription(input)
		assert.True(t, strings.HasPrefix(result, "("))
		assert.True(t, strings.HasSuffix(result, ")"))
		assert.Contains(t, result, "...")
		// Total should be 30 chars
		assert.Equal(t, 32, len(result))
	})
}

// ---------------------------------------------------------------------------
// ProfileManager with mock operations (if available)
// ---------------------------------------------------------------------------

func TestProfileManager_ShowProfileDetails(t *testing.T) {
	pm, err := NewProfileManager()
	if err != nil {
		t.Skip("Skipping: config manager not available")
	}

	t.Run("nonexistent profile", func(t *testing.T) {
		err := pm.ShowProfileDetails("nonexistent-profile-xyz")
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// ProfileManager edge cases
// ---------------------------------------------------------------------------

func TestProfileManager_EdgeCases(t *testing.T) {
	pm, err := NewProfileManager()
	if err != nil {
		t.Skip("Skipping: config manager not available")
	}

	t.Run("empty profile name", func(t *testing.T) {
		err := pm.ShowProfileDetails("")
		assert.Error(t, err)
	})

	t.Run("profile name with special chars", func(t *testing.T) {
		err := pm.ShowProfileDetails("profile/with/slashes")
		assert.Error(t, err)
	})

	t.Run("very long profile name", func(t *testing.T) {
		longName := strings.Repeat("a", 1000)
		err := pm.ShowProfileDetails(longName)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// FormatProfileName with unicode
// ---------------------------------------------------------------------------

func TestFormatProfileName_Unicode(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{"prod", "产品环境"},
		{"dev", "開発環境"},
		{"test", "환경"},
		{"staging", "окружение"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatProfileName(tt.name, tt.description)
			assert.Contains(t, result, tt.name)
			assert.Contains(t, result, tt.description)
		})
	}
}

// ---------------------------------------------------------------------------
// FormatProfileName with very long description
// ---------------------------------------------------------------------------

func TestFormatProfileName_LongDescription(t *testing.T) {
	longDesc := strings.Repeat("very long description ", 100)
	result := FormatProfileName("profile", longDesc)
	assert.Contains(t, result, "profile")
	assert.Contains(t, result, longDesc)
}

// ---------------------------------------------------------------------------
// formatRegion edge cases
// ---------------------------------------------------------------------------

func TestFormatRegion_EdgeCases(t *testing.T) {
	tests := []struct {
		region string
		want   string
	}{
		{"Auto", "AUTO"},
		{"AUTO", "AUTO"},
		{"Us-East-1", "US-EAST-1"},
		{"US_EAST_1", "US_EAST_1"},
		{"ap-northeast-1", "AP-NORTHEAST-1"},
		{"sa-southeast-1", "SA-SOUTHEAST-1"},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			result := formatRegion(tt.region)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ---------------------------------------------------------------------------
// formatTokenStatus security edge cases
// ---------------------------------------------------------------------------

func TestFormatTokenStatus_Security(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"token with spaces", "token with spaces"},
		{"token with newlines", "token\nwith\nnewlines"},
		{"token with tabs", "token\twith\ttabs"},
		{"token with special chars", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"token with unicode", "token世界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTokenStatus(tt.token)
			assert.NotEmpty(t, result)
		})
	}
}

// ---------------------------------------------------------------------------
// formatAccountIDStatus edge cases
// ---------------------------------------------------------------------------

func TestFormatAccountIDStatus_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		accountID string
		wantLen   int
	}{
		{"31 chars", strings.Repeat("a", 31), 0},
		{"33 chars", strings.Repeat("b", 33), 0},
		{"1 char", "a", 0},
		{"100 chars", strings.Repeat("c", 100), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAccountIDStatus(tt.accountID)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, "Invalid")
		})
	}
}

// ---------------------------------------------------------------------------
// getProfileIcon case sensitivity
// ---------------------------------------------------------------------------

func TestGetProfileIcon_CaseSensitivity(t *testing.T) {
	tests := []struct {
		description string
		want        string
	}{
		{"PRODUCTION", "🏭 "},
		{"Production", "🏭 "},
		{"production", "🏭 "},
		{"PROD", "🏭 "},
		{"Prod", "🏭 "},
		{"prod", "🏭 "},
		{"STAGING", "⚡ "},
		{"Staging", "⚡ "},
		{"staging", "⚡ "},
		{"DEV", "⚡ "},
		{"Dev", "⚡ "},
		{"dev", "⚡ "},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := getProfileIcon(tt.description)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ---------------------------------------------------------------------------
// getProfileIcon multiple keywords
// ---------------------------------------------------------------------------

func TestGetProfileIcon_MultipleKeywords(t *testing.T) {
	// Should match first matching keyword in the switch statement order
	// Order: prod/production, stag/dev, test, personal, work/company
	tests := []struct {
		description string
		want        string
	}{
		{"prod test", "🏭 "},     // prod matches before test
		{"test prod", "🏭 "},     // prod matches before test
		{"dev work", "⚡ "},      // dev matches before work
		{"personal test", "🧪 "}, // test matches before personal
		{"test personal", "🧪 "}, // test matches before personal
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := getProfileIcon(tt.description)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ---------------------------------------------------------------------------
// getProfileIcon substring matches
// ---------------------------------------------------------------------------

func TestGetProfileIcon_SubstringMatches(t *testing.T) {
	tests := []struct {
		description string
		want        string
	}{
		{"myprodserver", "🏭 "},      // contains "prod"
		{"thestagingenv", "⚡ "},     // contains "stag"
		{"development", "⚡ "},       // contains "dev"
		{"mytestbucket", "🧪 "},     // contains "test"
		{"mypersonalstuff", "💻 "},  // contains "personal"
		{"theworkserver", "🏢 "},    // contains "work"
		{"mycompanybucket", "🏢 "},  // contains "company"
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := getProfileIcon(tt.description)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ---------------------------------------------------------------------------
// ProfileManager with temporary config directory
// ---------------------------------------------------------------------------

func TestProfileManager_WithTempDir(t *testing.T) {
	// Create a temp directory for testing
	tempDir := t.TempDir()

	// Set config path to temp dir
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	pm, err := NewProfileManager()
	if err != nil {
		t.Skip("Skipping: config manager not available")
	}
	assert.NotNil(t, pm)
}
