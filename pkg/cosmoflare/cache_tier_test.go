package cosmoflare

import "testing"

func TestCacheClassifyFonts(t *testing.T) {
	for _, ext := range []string{".woff", ".woff2", ".ttf", ".otf", ".eot"} {
		if got := CacheClassify("font" + ext); got != CacheImmutable {
			t.Errorf("CacheClassify(font%s) = %d, want CacheImmutable", ext, got)
		}
	}
}

func TestCacheClassifyHashedJS(t *testing.T) {
	if got := CacheClassify("app.abc123.js"); got != CacheImmutable {
		t.Errorf("hashed JS should be CacheImmutable, got %d", got)
	}
	if got := CacheClassify("main-deadbeef.css"); got != CacheImmutable {
		t.Errorf("hashed CSS should be CacheImmutable, got %d", got)
	}
}

func TestCacheClassifyUnhashedStatic(t *testing.T) {
	for _, ext := range []string{".css", ".js", ".mjs"} {
		if got := CacheClassify("style" + ext); got != CacheStatic {
			t.Errorf("CacheClassify(style%s) = %d, want CacheStatic", ext, got)
		}
	}
}

func TestCacheClassifyImages(t *testing.T) {
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico", ".avif"} {
		if got := CacheClassify("img" + ext); got != CacheLongStatic {
			t.Errorf("CacheClassify(img%s) = %d, want CacheLongStatic", ext, got)
		}
	}
}

func TestCacheClassifyMedia(t *testing.T) {
	for _, ext := range []string{".mp4", ".webm", ".mp3", ".wav", ".ogg"} {
		if got := CacheClassify("media" + ext); got != CacheLongStatic {
			t.Errorf("CacheClassify(media%s) = %d, want CacheLongStatic", ext, got)
		}
	}
}

func TestCacheClassifyDynamic(t *testing.T) {
	for _, ext := range []string{".html", ".htm"} {
		if got := CacheClassify("page" + ext); got != CacheDynamic {
			t.Errorf("CacheClassify(page%s) = %d, want CacheDynamic", ext, got)
		}
	}
	for _, ext := range []string{".json", ".xml", ".yaml", ".yml"} {
		if got := CacheClassify("data" + ext); got != CacheDynamic {
			t.Errorf("CacheClassify(data%s) = %d, want CacheDynamic", ext, got)
		}
	}
}

func TestCacheClassifyArchives(t *testing.T) {
	for _, ext := range []string{".pdf", ".zip", ".tar", ".gz", ".bz2"} {
		if got := CacheClassify("archive" + ext); got != CacheStatic {
			t.Errorf("CacheClassify(archive%s) = %d, want CacheStatic", ext, got)
		}
	}
}

func TestCacheClassifyUnknown(t *testing.T) {
	if got := CacheClassify("file.xyz"); got != CacheStatic {
		t.Errorf("unknown extension should default to CacheStatic, got %d", got)
	}
}

func TestCacheControlHeader(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"font.woff2", "public, max-age=31536000, immutable"},
		{"style.css", "public, max-age=86400"},
		{"photo.jpg", "public, max-age=604800"},
		{"index.html", "public, max-age=0, must-revalidate"},
	}
	for _, tt := range tests {
		if got := CacheControlHeader(tt.key); got != tt.want {
			t.Errorf("CacheControlHeader(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestIsHashedFilenamePositive(t *testing.T) {
	hashed := []string{"app.abc123.js", "main-deadbeef.css", "bundle.a1b2c3d4e5f6.js", "chunk.ff00ff.mjs"}
	for _, name := range hashed {
		if !isHashedFilename(name) {
			t.Errorf("isHashedFilename(%q) = false, want true", name)
		}
	}
}

func TestIsHashedFilenameNegative(t *testing.T) {
	unhashed := []string{"style.css", "script.js", "index.html", "app.js"}
	for _, name := range unhashed {
		if isHashedFilename(name) {
			t.Errorf("isHashedFilename(%q) = true, want false", name)
		}
	}
}

func TestIsHexString(t *testing.T) {
	if !isHexString("abc123") {
		t.Error("abc123 should be hex")
	}
	if !isHexString("DEADBEEF") {
		t.Error("DEADBEEF should be hex")
	}
	if isHexString("") {
		t.Error("empty string should not be hex")
	}
	if isHexString("xyz123") {
		t.Error("xyz123 should not be hex")
	}
	if isHexString("hello") {
		t.Error("hello should not be hex")
	}
}
