package utils

import "testing"

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"zero", 0, "0 B"},
		{"one byte", 1, "1 B"},
		{"below KB", 1023, "1023 B"},
		{"exactly 1 KB", 1024, "1.0 KB"},
		{"1.5 KB", 1536, "1.5 KB"},
		{"exactly 1 MB", 1048576, "1.0 MB"},
		{"exactly 1 GB", 1073741824, "1.0 GB"},
		{"large value", 1099511627776, "1.0 TB"},
		{"negative", -1, "-1 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMaskAccountID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"short 4 chars", "abcd", "****"},
		{"exactly 8 chars", "abcdefgh", "********"},
		{"9 chars", "abcdefghi", "abcd*fghi"},
		{"32 char ID", "a1b2c3d4e5f67890abcdef1234567890", "a1b2************************7890"},
		{"12 chars", "abcdefghijkl", "abcd****ijkl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAccountID(tt.input)
			if result != tt.expected {
				t.Errorf("MaskAccountID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
