package r2go2

import (
	"bytes"
	"strings"
	"testing"
)

func TestMultipartThreshold(t *testing.T) {
	if multipartThreshold != 100*1024*1024 {
		t.Errorf("multipartThreshold = %d, want 100MB", multipartThreshold)
	}
}

func TestDefaultPartSize(t *testing.T) {
	if defaultPartSize != 8*1024*1024 {
		t.Errorf("defaultPartSize = %d, want 8MB", defaultPartSize)
	}
}

func TestDefaultConcurrency(t *testing.T) {
	if defaultConcurrency != 3 {
		t.Errorf("defaultConcurrency = %d, want 3", defaultConcurrency)
	}
}

func TestWithPartSize(t *testing.T) {
	cfg := &uploadConfig{}
	WithPartSize(16 * 1024 * 1024)(cfg)
	if cfg.partSize != 16*1024*1024 {
		t.Errorf("partSize = %d, want 16MB", cfg.partSize)
	}
}

func TestWithConcurrency(t *testing.T) {
	cfg := &uploadConfig{}
	WithConcurrency(5)(cfg)
	if cfg.concurrency != 5 {
		t.Errorf("concurrency = %d, want 5", cfg.concurrency)
	}
}

func TestWithProgressCallback(t *testing.T) {
	var captured int64
	cfg := &uploadConfig{}
	WithProgressCallback(func(uploaded, total int64) {
		captured = uploaded
	})(cfg)
	if cfg.progressCallback == nil {
		t.Fatal("progressCallback is nil")
	}
	cfg.progressCallback(42, 100)
	if captured != 42 {
		t.Errorf("captured = %d, want 42", captured)
	}
}

func TestUploadOptionsComposed(t *testing.T) {
	cfg := &uploadConfig{
		partSize:    defaultPartSize,
		concurrency: defaultConcurrency,
	}
	WithContentType("text/plain")(cfg)
	WithUploadCacheControl("max-age=3600")(cfg)
	WithMetadata(map[string]string{"key": "value"})(cfg)
	WithPartSize(16 * 1024 * 1024)(cfg)
	WithConcurrency(7)(cfg)

	if cfg.contentType != "text/plain" {
		t.Errorf("contentType = %q, want text/plain", cfg.contentType)
	}
	if cfg.cacheControl != "max-age=3600" {
		t.Errorf("cacheControl = %q, want max-age=3600", cfg.cacheControl)
	}
	if cfg.metadata["key"] != "value" {
		t.Error("metadata not set correctly")
	}
	if cfg.partSize != 16*1024*1024 {
		t.Errorf("partSize = %d, want 16MB", cfg.partSize)
	}
	if cfg.concurrency != 7 {
		t.Errorf("concurrency = %d, want 7", cfg.concurrency)
	}
}

func TestUploadResultPartsField(t *testing.T) {
	result := &UploadResult{
		Key:    "test.txt",
		Bucket: "test-bucket",
		Size:   1024,
		ETag:   "abc123",
		Parts:  5,
	}
	if result.Parts != 5 {
		t.Errorf("Parts = %d, want 5", result.Parts)
	}
}

func TestDefaultCachePolicy(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"style.css", "public, max-age=31536000, immutable"},
		{"app.js", "public, max-age=31536000, immutable"},
		{"image.png", "public, max-age=86400"},
		{"photo.jpg", "public, max-age=86400"},
		{"font.woff2", "public, max-age=31536000, immutable"},
		{"index.html", "public, max-age=0, must-revalidate"},
		{"data.json", ""},
		{"unknown.bin", ""},
	}
	for _, tt := range tests {
		got := defaultCachePolicy(tt.key)
		if got != tt.expected {
			t.Errorf("defaultCachePolicy(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"page.html", "text/html; charset=utf-8"},
		{"style.css", "text/css; charset=utf-8"},
		{"app.js", "application/javascript"},
		{"data.json", "application/json"},
		{"image.png", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"font.woff2", "font/woff2"},
		{"unknown.xyz", ""},
	}
	for _, tt := range tests {
		got := detectContentType(tt.key)
		if got != tt.expected {
			t.Errorf("detectContentType(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}

func TestExtension(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"file.txt", ".txt"},
		{"path/to/file.css", ".css"},
		{"noext", ""},
		{"path/to/noext", ""},
		{"a.b.c", ".c"},
	}
	for _, tt := range tests {
		got := extension(tt.key)
		if got != tt.expected {
			t.Errorf("extension(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}

func TestMultipartNumPartsCalculation(t *testing.T) {
	tests := []struct {
		size     int64
		partSize int64
		expected int64
	}{
		{16 * 1024 * 1024, 8 * 1024 * 1024, 2},           // 16MB / 8MB = 2
		{20 * 1024 * 1024, 8 * 1024 * 1024, 3},           // 20MB / 8MB = 2.5 → 3
		{8 * 1024 * 1024, 8 * 1024 * 1024, 1},            // exactly 1 part
		{1, 8 * 1024 * 1024, 1},                           // tiny file = 1 part
		{100 * 1024 * 1024, 8 * 1024 * 1024, 13},         // 100MB / 8MB = 12.5 → 13
	}
	for _, tt := range tests {
		numParts := (tt.size + tt.partSize - 1) / tt.partSize
		if numParts == 0 {
			numParts = 1
		}
		if numParts != tt.expected {
			t.Errorf("numParts(%d, %d) = %d, want %d", tt.size, tt.partSize, numParts, tt.expected)
		}
	}
}

func TestReaderSplitting(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 20*1024*1024) // 20MB
	partSize := int64(8 * 1024 * 1024)
	numParts := (int64(len(data)) + partSize - 1) / partSize

	var totalRead int64
	for i := int64(0); i < numParts; i++ {
		thisPart := partSize
		remaining := int64(len(data)) - totalRead
		if remaining < thisPart {
			thisPart = remaining
		}
		totalRead += thisPart
	}
	if totalRead != int64(len(data)) {
		t.Errorf("totalRead = %d, want %d", totalRead, len(data))
	}
	if numParts != 3 {
		t.Errorf("numParts = %d, want 3", numParts)
	}
}

func TestUploadValidation(t *testing.T) {
	_, err := NewClient(
		WithAccountID("test-account"),
		WithAPIToken("test-token-12345"),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
}

func TestIsHashedFilename(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"app.abc123.js", true},
		{"main-deadbeef.css", true},
		{"vendor.1a2b3c4d5e.js", true},
		{"regular.css", false},
		{"nohash.js", false},
		{"short.abc.js", false}, // only 3 hex chars
	}
	for _, tt := range tests {
		got := isHashedFilename(tt.key)
		if got != tt.expected {
			t.Errorf("isHashedFilename(%q) = %v, want %v", tt.key, got, tt.expected)
		}
	}
}

func TestMimeTypes(t *testing.T) {
	if len(mimeTypes) == 0 {
		t.Error("mimeTypes map is empty")
	}
	// Verify critical types exist
	required := []string{".html", ".css", ".js", ".json", ".png", ".jpg", ".pdf"}
	for _, ext := range required {
		if _, ok := mimeTypes[ext]; !ok {
			t.Errorf("mimeTypes missing entry for %q", ext)
		}
	}
}

func TestUploadCachePolicyIntegration(t *testing.T) {
	// Verify cache policy is applied for known extensions
	key := "assets/style.css"
	policy := defaultCachePolicy(key)
	if !strings.Contains(policy, "immutable") {
		t.Errorf("cache policy for CSS should be immutable, got: %q", policy)
	}

	// Verify unknown extensions get no cache policy
	key2 := "data/file.xyz"
	policy2 := defaultCachePolicy(key2)
	if policy2 != "" {
		t.Errorf("cache policy for unknown extension should be empty, got: %q", policy2)
	}
}
