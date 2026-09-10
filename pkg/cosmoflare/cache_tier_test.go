package cosmoflare

import "testing"

// TestCacheClassifyFonts verifies that every font extension is classified as
// immutable, since font files never change content for a given URL.
func TestCacheClassifyFonts(t *testing.T) {
	t.Parallel()
	for _, ext := range []string{".woff", ".woff2", ".ttf", ".otf", ".eot"} {
		t.Run("font"+ext, func(t *testing.T) {
			t.Parallel()
			if got := CacheClassify("font" + ext); got != CacheImmutable {
				t.Errorf("CacheClassify(font%s) = %d, want CacheImmutable", ext, got)
			}
		})
	}
}

// TestCacheClassifyHashedJS verifies that JS/CSS filenames carrying a content
// hash are treated as immutable regardless of which separator style is used.
func TestCacheClassifyHashedJS(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"app.abc123.js", "main-deadbeef.css"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := CacheClassify(name); got != CacheImmutable {
				t.Errorf("hashed asset %q = %d, want CacheImmutable", name, got)
			}
		})
	}
}

// TestCacheClassifyUnhashedStatic verifies that unhashed code assets fall back
// to the short-lived static tier so deploys propagate quickly.
func TestCacheClassifyUnhashedStatic(t *testing.T) {
	t.Parallel()
	for _, ext := range []string{".css", ".js", ".mjs"} {
		t.Run("style"+ext, func(t *testing.T) {
			t.Parallel()
			if got := CacheClassify("style" + ext); got != CacheStatic {
				t.Errorf("CacheClassify(style%s) = %d, want CacheStatic", ext, got)
			}
		})
	}
}

// TestCacheClassifyImages verifies that image formats land in the long-static
// tier (7-day max-age), long enough for CDN hits but still revalidated.
func TestCacheClassifyImages(t *testing.T) {
	t.Parallel()
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico", ".avif"} {
		t.Run("img"+ext, func(t *testing.T) {
			t.Parallel()
			if got := CacheClassify("img" + ext); got != CacheLongStatic {
				t.Errorf("CacheClassify(img%s) = %d, want CacheLongStatic", ext, got)
			}
		})
	}
}

// TestCacheClassifyMedia verifies that audio/video containers land in the
// long-static tier alongside images.
func TestCacheClassifyMedia(t *testing.T) {
	t.Parallel()
	for _, ext := range []string{".mp4", ".webm", ".mp3", ".wav", ".ogg"} {
		t.Run("media"+ext, func(t *testing.T) {
			t.Parallel()
			if got := CacheClassify("media" + ext); got != CacheLongStatic {
				t.Errorf("CacheClassify(media%s) = %d, want CacheLongStatic", ext, got)
			}
		})
	}
}

// TestCacheClassifyDynamic verifies that markup and data files are treated as
// dynamic and must always be revalidated with the origin.
func TestCacheClassifyDynamic(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		prefix, ext string
	}{
		{"page", ".html"}, {"page", ".htm"},
		{"data", ".json"}, {"data", ".xml"}, {"data", ".yaml"}, {"data", ".yml"},
	} {
		t.Run(tt.prefix+tt.ext, func(t *testing.T) {
			t.Parallel()
			if got := CacheClassify(tt.prefix + tt.ext); got != CacheDynamic {
				t.Errorf("CacheClassify(%s%s) = %d, want CacheDynamic", tt.prefix, tt.ext, got)
			}
		})
	}
}

// TestCacheClassifyArchives verifies that downloadable documents and archives
// use the standard static tier rather than the immutable or dynamic tiers.
func TestCacheClassifyArchives(t *testing.T) {
	t.Parallel()
	for _, ext := range []string{".pdf", ".zip", ".tar", ".gz", ".bz2"} {
		t.Run("archive"+ext, func(t *testing.T) {
			t.Parallel()
			if got := CacheClassify("archive" + ext); got != CacheStatic {
				t.Errorf("CacheClassify(archive%s) = %d, want CacheStatic", ext, got)
			}
		})
	}
}

// TestCacheClassifyUnknown verifies that unrecognized extensions default to
// the static tier instead of being cached aggressively or never cached.
func TestCacheClassifyUnknown(t *testing.T) {
	t.Parallel()
	if got := CacheClassify("file.xyz"); got != CacheStatic {
		t.Errorf("unknown extension should default to CacheStatic, got %d", got)
	}
}

// TestCacheControlHeader verifies the full Cache-Control header emitted for
// representative files from each tier, including the must-revalidate
// directive required for dynamic content.
func TestCacheControlHeader(t *testing.T) {
	t.Parallel()
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
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()
			if got := CacheControlHeader(tt.key); got != tt.want {
				t.Errorf("CacheControlHeader(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

// TestIsHashedFilenamePositive verifies that filenames containing hex content
// hashes in the common dot- and dash-separated layouts are detected as hashed.
func TestIsHashedFilenamePositive(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"app.abc123.js", "main-deadbeef.css", "bundle.a1b2c3d4e5f6.js", "chunk.ff00ff.mjs"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !isHashedFilename(name) {
				t.Errorf("isHashedFilename(%q) = false, want true", name)
			}
		})
	}
}

// TestIsHashedFilenameNegative verifies that ordinary asset names without a
// hash segment are not misclassified as immutable.
func TestIsHashedFilenameNegative(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"style.css", "script.js", "index.html", "app.js"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if isHashedFilename(name) {
				t.Errorf("isHashedFilename(%q) = true, want false", name)
			}
		})
	}
}

// TestIsHexString verifies hex detection for mixed case, and rejects empty and
// non-hex inputs.
func TestIsHexString(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		in   string
		want bool
	}{
		{"abc123", true},
		{"DEADBEEF", true},
		{"", false}, // empty string is not a hash
		{"xyz123", false},
		{"hello", false},
	} {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := isHexString(tt.in); got != tt.want {
				t.Errorf("isHexString(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
