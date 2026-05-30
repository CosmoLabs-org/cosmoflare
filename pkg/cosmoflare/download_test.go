package cosmoflare

import (
	"errors"
	"testing"
)

func TestWithOutputPath(t *testing.T) {
	cfg := &downloadConfig{}
	opt := WithOutputPath("/tmp/test-file.txt")
	opt(cfg)

	if cfg.outputPath != "/tmp/test-file.txt" {
		t.Errorf("WithOutputPath: got %q, want %q", cfg.outputPath, "/tmp/test-file.txt")
	}
}

func TestWithOutputPathEmpty(t *testing.T) {
	cfg := &downloadConfig{}
	opt := WithOutputPath("")
	opt(cfg)

	if cfg.outputPath != "" {
		t.Errorf("WithOutputPath empty: got %q, want empty", cfg.outputPath)
	}
}

func TestWithRange(t *testing.T) {
	cfg := &downloadConfig{}
	opt := WithRange(100, 500)
	opt(cfg)

	if cfg.rangeStart != 100 {
		t.Errorf("WithRange start: got %d, want 100", cfg.rangeStart)
	}
	if cfg.rangeEnd != 500 {
		t.Errorf("WithRange end: got %d, want 500", cfg.rangeEnd)
	}
}

func TestWithRangeZeroValues(t *testing.T) {
	cfg := &downloadConfig{}
	opt := WithRange(0, 0)
	opt(cfg)

	if cfg.rangeStart != 0 {
		t.Errorf("WithRange zero start: got %d, want 0", cfg.rangeStart)
	}
	if cfg.rangeEnd != 0 {
		t.Errorf("WithRange zero end: got %d, want 0", cfg.rangeEnd)
	}
}

func TestDownloadMultipleOptions(t *testing.T) {
	cfg := &downloadConfig{}
	opts := []DownloadOption{
		WithOutputPath("/tmp/out.bin"),
		WithRange(0, 1024),
	}
	for _, o := range opts {
		o(cfg)
	}

	if cfg.outputPath != "/tmp/out.bin" {
		t.Errorf("combined outputPath: got %q, want %q", cfg.outputPath, "/tmp/out.bin")
	}
	if cfg.rangeStart != 0 || cfg.rangeEnd != 1024 {
		t.Errorf("combined range: got %d-%d, want 0-1024", cfg.rangeStart, cfg.rangeEnd)
	}
}

func TestDownloadOptionOverride(t *testing.T) {
	cfg := &downloadConfig{}
	opts := []DownloadOption{
		WithOutputPath("/first/path.txt"),
		WithRange(10, 20),
		WithOutputPath("/second/path.txt"),
		WithRange(100, 200),
	}
	for _, o := range opts {
		o(cfg)
	}

	if cfg.outputPath != "/second/path.txt" {
		t.Errorf("override outputPath: got %q, want %q", cfg.outputPath, "/second/path.txt")
	}
	if cfg.rangeStart != 100 || cfg.rangeEnd != 200 {
		t.Errorf("override range: got %d-%d, want 100-200", cfg.rangeStart, cfg.rangeEnd)
	}
}

func TestDownloadValidation_EmptyBucket(t *testing.T) {
	c := &client{}
	_, err := c.Download(t.Context(), "", "some-key")
	if err == nil {
		t.Fatal("expected error for empty bucket, got nil")
	}

	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError, got %T: %v", err, err)
	}
}

func TestDownloadValidation_ShortBucket(t *testing.T) {
	c := &client{}
	_, err := c.Download(t.Context(), "ab", "some-key")
	if err == nil {
		t.Fatal("expected error for short bucket name, got nil")
	}

	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError, got %T: %v", err, err)
	}
}

func TestDownloadValidation_EmptyKey(t *testing.T) {
	c := &client{}
	_, err := c.Download(t.Context(), "valid-bucket", "")
	if err == nil {
		t.Fatal("expected error for empty key, got nil")
	}

	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError, got %T: %v", err, err)
	}
}

func TestDownloadValidation_InvalidBucketChars(t *testing.T) {
	c := &client{}
	_, err := c.Download(t.Context(), "INVALID_BUCKET!", "key.txt")
	if err == nil {
		t.Fatal("expected error for invalid bucket characters, got nil")
	}

	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError, got %T: %v", err, err)
	}
}

func TestDownloadValidation_BucketStartsWithHyphen(t *testing.T) {
	c := &client{}
	_, err := c.Download(t.Context(), "-starts-bad", "key.txt")
	if err == nil {
		t.Fatal("expected error for bucket starting with hyphen, got nil")
	}

	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected R2ValidationError, got %T: %v", err, err)
	}
}

func TestDownloadConfigDefaults(t *testing.T) {
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
