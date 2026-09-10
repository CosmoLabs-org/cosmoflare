package cosmoflare

import (
	"errors"
	"testing"
)

// TestDownloadOptions verifies that each DownloadOption mutates the
// download config as expected, that options compose when applied together,
// and that later options override earlier ones for the same field.
func TestDownloadOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		opts      []DownloadOption
		wantPath  string
		wantStart int64
		wantEnd   int64
	}{
		{
			name:     "output path",
			opts:     []DownloadOption{WithOutputPath("/tmp/test-file.txt")},
			wantPath: "/tmp/test-file.txt",
		},
		{
			name:     "empty output path",
			opts:     []DownloadOption{WithOutputPath("")},
			wantPath: "",
		},
		{
			name:      "range",
			opts:      []DownloadOption{WithRange(100, 500)},
			wantStart: 100,
			wantEnd:   500,
		},
		{
			name:      "zero range",
			opts:      []DownloadOption{WithRange(0, 0)},
			wantStart: 0,
			wantEnd:   0,
		},
		{
			name:      "combined options",
			opts:      []DownloadOption{WithOutputPath("/tmp/out.bin"), WithRange(0, 1024)},
			wantPath:  "/tmp/out.bin",
			wantStart: 0,
			wantEnd:   1024,
		},
		{
			// Last option wins: config is mutated sequentially.
			name:      "later options override earlier ones",
			opts:      []DownloadOption{WithOutputPath("/first/path.txt"), WithRange(10, 20), WithOutputPath("/second/path.txt"), WithRange(100, 200)},
			wantPath:  "/second/path.txt",
			wantStart: 100,
			wantEnd:   200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &downloadConfig{}
			for _, o := range tt.opts {
				o(cfg)
			}
			if cfg.outputPath != tt.wantPath {
				t.Errorf("outputPath = %q, want %q", cfg.outputPath, tt.wantPath)
			}
			if cfg.rangeStart != tt.wantStart {
				t.Errorf("rangeStart = %d, want %d", cfg.rangeStart, tt.wantStart)
			}
			if cfg.rangeEnd != tt.wantEnd {
				t.Errorf("rangeEnd = %d, want %d", cfg.rangeEnd, tt.wantEnd)
			}
		})
	}
}

// TestDownloadValidationScenarios verifies that Download rejects malformed bucket names
// and empty object keys client-side with an *R2ValidationError, before any
// HTTP request is attempted. Each case is a distinct invalid-input scenario.
func TestDownloadValidationScenarios(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		bucket string
		key    string
	}{
		{name: "empty bucket", bucket: "", key: "some-key"},
		{name: "short bucket", bucket: "ab", key: "some-key"},
		{name: "empty key", bucket: "valid-bucket", key: ""},
		{name: "invalid bucket characters", bucket: "INVALID_BUCKET!", key: "key.txt"},
		{name: "bucket starts with hyphen", bucket: "-starts-bad", key: "key.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &client{}
			_, err := c.Download(t.Context(), tt.bucket, tt.key)
			if err == nil {
				t.Fatalf("Download(bucket=%q, key=%q) = nil error, want validation error", tt.bucket, tt.key)
			}
			var valErr *R2ValidationError
			if !errors.As(err, &valErr) {
				t.Errorf("expected R2ValidationError, got %T: %v", err, err)
			}
		})
	}
}

// TestDownloadConfigDefaults verifies that a zero-value download config means
// "no output path override, full-object range", since range 0-0 is treated as
// absent by the downloader.
func TestDownloadConfigDefaults(t *testing.T) {
	t.Parallel()
	cfg := &downloadConfig{}

	if cfg.outputPath != "" {
		t.Errorf("default outputPath should be empty, got %q", cfg.outputPath)
	}
	if cfg.rangeStart != 0 {
		t.Errorf("default rangeStart should be 0, got %d", cfg.rangeStart)
	}
	if cfg.rangeEnd != 0 {
		t.Errorf("default rangeEnd should be 0, got %d", cfg.rangeEnd)
	}
}
