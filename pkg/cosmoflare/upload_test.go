package cosmoflare

import (
	"strings"
	"testing"
)

func TestWithContentType(t *testing.T) {
	cfg := &uploadConfig{}
	opt := WithContentType("application/json")
	opt(cfg)
	if cfg.contentType != "application/json" {
		t.Errorf("expected contentType=application/json, got %s", cfg.contentType)
	}
}

func TestWithUploadCacheControl(t *testing.T) {
	cfg := &uploadConfig{}
	opt := WithUploadCacheControl("public, max-age=3600")
	opt(cfg)
	if cfg.cacheControl != "public, max-age=3600" {
		t.Errorf("expected cacheControl=public, max-age=3600, got %s", cfg.cacheControl)
	}
}

func TestWithMetadata(t *testing.T) {
	cfg := &uploadConfig{}
	m := map[string]string{"author": "test", "version": "1.0"}
	opt := WithMetadata(m)
	opt(cfg)
	if len(cfg.metadata) != 2 {
		t.Fatalf("expected 2 metadata entries, got %d", len(cfg.metadata))
	}
	if cfg.metadata["author"] != "test" {
		t.Errorf("expected metadata[author]=test, got %s", cfg.metadata["author"])
	}
	if cfg.metadata["version"] != "1.0" {
		t.Errorf("expected metadata[version]=1.0, got %s", cfg.metadata["version"])
	}
}

func TestUploadWithPartSize(t *testing.T) {
	cfg := &uploadConfig{}
	opt := WithPartSize(16 * 1024 * 1024)
	opt(cfg)
	if cfg.partSize != 16*1024*1024 {
		t.Errorf("expected partSize=16MB, got %d", cfg.partSize)
	}
}

func TestUploadWithConcurrency(t *testing.T) {
	cfg := &uploadConfig{}
	opt := WithConcurrency(5)
	opt(cfg)
	if cfg.concurrency != 5 {
		t.Errorf("expected concurrency=5, got %d", cfg.concurrency)
	}
}

func TestUploadWithProgressCallback(t *testing.T) {
	cfg := &uploadConfig{}
	called := false
	opt := WithProgressCallback(func(uploaded, total int64) {
		called = true
	})
	opt(cfg)
	if cfg.progressCallback == nil {
		t.Fatal("expected progressCallback to be set")
	}
	cfg.progressCallback(100, 200)
	if !called {
		t.Error("expected progressCallback to be invoked")
	}
}

func TestUploadOptionChaining(t *testing.T) {
	cfg := &uploadConfig{}
	opts := []UploadOption{
		WithContentType("image/png"),
		WithUploadCacheControl("no-cache"),
		WithMetadata(map[string]string{"env": "prod"}),
		WithPartSize(32 * 1024 * 1024),
		WithConcurrency(10),
	}
	for _, o := range opts {
		o(cfg)
	}
	if cfg.contentType != "image/png" {
		t.Errorf("contentType: got %s", cfg.contentType)
	}
	if cfg.cacheControl != "no-cache" {
		t.Errorf("cacheControl: got %s", cfg.cacheControl)
	}
	if cfg.metadata["env"] != "prod" {
		t.Errorf("metadata[env]: got %s", cfg.metadata["env"])
	}
	if cfg.partSize != 32*1024*1024 {
		t.Errorf("partSize: got %d", cfg.partSize)
	}
	if cfg.concurrency != 10 {
		t.Errorf("concurrency: got %d", cfg.concurrency)
	}
}

func TestMultipartThresholdConstant(t *testing.T) {
	// multipartThreshold should be 100MB
	expected := int64(100 * 1024 * 1024)
	if multipartThreshold != expected {
		t.Errorf("expected multipartThreshold=%d, got %d", expected, multipartThreshold)
	}
}

func TestDefaultPartSizeConstant(t *testing.T) {
	// defaultPartSize should be 8MB
	expected := int64(8 * 1024 * 1024)
	if defaultPartSize != expected {
		t.Errorf("expected defaultPartSize=%d, got %d", expected, defaultPartSize)
	}
}

func TestDefaultConcurrencyConstant(t *testing.T) {
	if defaultConcurrency != 3 {
		t.Errorf("expected defaultConcurrency=3, got %d", defaultConcurrency)
	}
}

func TestUploadDefaultCachePolicy(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"style.css", "public, max-age=31536000, immutable"},
		{"app.js", "public, max-age=31536000, immutable"},
		{"photo.jpg", "public, max-age=86400"},
		{"photo.jpeg", "public, max-age=86400"},
		{"image.png", "public, max-age=86400"},
		{"anim.gif", "public, max-age=86400"},
		{"pic.webp", "public, max-age=86400"},
		{"logo.svg", "public, max-age=86400"},
		{"favicon.ico", "public, max-age=86400"},
		{"font.woff", "public, max-age=31536000, immutable"},
		{"font.woff2", "public, max-age=31536000, immutable"},
		{"font.ttf", "public, max-age=31536000, immutable"},
		{"font.otf", "public, max-age=31536000, immutable"},
		{"font.eot", "public, max-age=31536000, immutable"},
		{"index.html", "public, max-age=0, must-revalidate"},
		{"page.htm", "public, max-age=0, must-revalidate"},
		{"data.json", ""},
		{"readme.txt", ""},
		{"noext", ""},
		{"path/to/file.css", "public, max-age=31536000, immutable"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := defaultCachePolicy(tt.key)
			if got != tt.want {
				t.Errorf("defaultCachePolicy(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestUploadExtension(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"file.txt", ".txt"},
		{"archive.tar.gz", ".gz"},
		{"noext", ""},
		{"path/to/file.json", ".json"},
		{".hidden", ".hidden"},
		{"dir/noext", ""},
		{"a.b.c.d", ".d"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := extension(tt.key)
			if got != tt.want {
				t.Errorf("extension(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestUploadDetectContentType(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"page.html", "text/html; charset=utf-8"},
		{"style.css", "text/css; charset=utf-8"},
		{"app.js", "application/javascript"},
		{"data.json", "application/json"},
		{"image.png", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"doc.pdf", "application/pdf"},
		{"archive.zip", "application/zip"},
		{"font.woff2", "font/woff2"},
		{"unknown.xyz", ""},
		{"noext", ""},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := detectContentType(tt.key)
			if got != tt.want {
				t.Errorf("detectContentType(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestUploadValidation_EmptyBucket(t *testing.T) {
	// validateBucketName("") returns an error; Upload wraps it as R2ValidationError
	err := validateBucketName("")
	if err == nil {
		t.Fatal("expected error for empty bucket name")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error about empty bucket, got: %s", err.Error())
	}
}

func TestUploadValidation_ShortBucket(t *testing.T) {
	err := validateBucketName("ab")
	if err == nil {
		t.Fatal("expected error for short bucket name")
	}
	if !strings.Contains(err.Error(), "between 3 and 63") {
		t.Errorf("expected length error, got: %s", err.Error())
	}
}

func TestUploadValidation_InvalidBucketChars(t *testing.T) {
	err := validateBucketName("My-Bucket")
	if err == nil {
		t.Fatal("expected error for uppercase bucket name")
	}
	if !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("expected invalid character error, got: %s", err.Error())
	}
}

func TestUploadValidation_BucketStartsWithHyphen(t *testing.T) {
	err := validateBucketName("-my-bucket")
	if err == nil {
		t.Fatal("expected error for bucket starting with hyphen")
	}
	if !strings.Contains(err.Error(), "cannot start or end") {
		t.Errorf("expected start/end error, got: %s", err.Error())
	}
}

func TestUploadValidation_ValidBucket(t *testing.T) {
	err := validateBucketName("my-valid-bucket")
	if err != nil {
		t.Errorf("expected no error for valid bucket, got: %s", err.Error())
	}
}

func TestUploadValidation_EmptyKeyError(t *testing.T) {
	// Verify the validation error constructor produces correct output
	ve := validationError("Upload", "object key is required")
	if ve.Op != "Upload" {
		t.Errorf("expected Op=Upload, got %s", ve.Op)
	}
	if !strings.Contains(ve.Error(), "object key is required") {
		t.Errorf("expected 'object key is required' in error, got: %s", ve.Error())
	}
}

func TestUploadValidation_NilReaderError(t *testing.T) {
	ve := validationError("Upload", "reader is required")
	if ve.Op != "Upload" {
		t.Errorf("expected Op=Upload, got %s", ve.Op)
	}
	if !strings.Contains(ve.Error(), "reader is required") {
		t.Errorf("expected 'reader is required' in error, got: %s", ve.Error())
	}
}

func TestProgressCallbackValues(t *testing.T) {
	cfg := &uploadConfig{}
	var capturedUploaded, capturedTotal int64
	WithProgressCallback(func(uploaded, total int64) {
		capturedUploaded = uploaded
		capturedTotal = total
	})(cfg)

	cfg.progressCallback(500, 1000)
	if capturedUploaded != 500 {
		t.Errorf("expected uploaded=500, got %d", capturedUploaded)
	}
	if capturedTotal != 1000 {
		t.Errorf("expected total=1000, got %d", capturedTotal)
	}

	// Simulate progress update
	cfg.progressCallback(1000, 1000)
	if capturedUploaded != 1000 {
		t.Errorf("expected uploaded=1000, got %d", capturedUploaded)
	}
}

func TestUploadConfigDefaults(t *testing.T) {
	// A fresh uploadConfig should have zero values (defaults applied by caller)
	cfg := &uploadConfig{}
	if cfg.contentType != "" {
		t.Errorf("expected empty contentType, got %s", cfg.contentType)
	}
	if cfg.cacheControl != "" {
		t.Errorf("expected empty cacheControl, got %s", cfg.cacheControl)
	}
	if cfg.metadata != nil {
		t.Errorf("expected nil metadata, got %v", cfg.metadata)
	}
	if cfg.partSize != 0 {
		t.Errorf("expected partSize=0, got %d", cfg.partSize)
	}
	if cfg.concurrency != 0 {
		t.Errorf("expected concurrency=0, got %d", cfg.concurrency)
	}
	if cfg.progressCallback != nil {
		t.Error("expected nil progressCallback")
	}
}

func TestWithMetadataNilMap(t *testing.T) {
	cfg := &uploadConfig{}
	WithMetadata(nil)(cfg)
	if cfg.metadata != nil {
		t.Errorf("expected nil metadata when nil map passed, got %v", cfg.metadata)
	}
}

func TestWithMetadataOverwrite(t *testing.T) {
	cfg := &uploadConfig{}
	WithMetadata(map[string]string{"a": "1"})(cfg)
	WithMetadata(map[string]string{"b": "2"})(cfg)
	// Second call should overwrite, not merge
	if _, ok := cfg.metadata["a"]; ok {
		t.Error("expected first metadata to be overwritten")
	}
	if cfg.metadata["b"] != "2" {
		t.Errorf("expected metadata[b]=2, got %s", cfg.metadata["b"])
	}
}

func TestSizeExceedsMultipartThreshold(t *testing.T) {
	// Verify the threshold logic: sizes > 100MB should trigger multipart
	smallSize := int64(50 * 1024 * 1024)  // 50MB
	largeSize := int64(150 * 1024 * 1024) // 150MB
	exactSize := int64(multipartThreshold) // exactly 100MB

	if smallSize > multipartThreshold {
		t.Error("50MB should not exceed multipart threshold")
	}
	if largeSize <= multipartThreshold {
		t.Error("150MB should exceed multipart threshold")
	}
	if exactSize > multipartThreshold {
		t.Error("exactly 100MB should not exceed threshold (strict >)")
	}
}
