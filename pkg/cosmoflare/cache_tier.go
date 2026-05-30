package cosmoflare

import (
	"path/filepath"
	"strings"
)

// CacheTier represents a cache policy tier.
type CacheTier int

const (
	// CacheImmutable — long-lived static assets that never change (fonts, hashed JS/CSS).
	CacheImmutable CacheTier = iota
	// CacheLongStatic — static assets that rarely change (images, media).
	CacheLongStatic
	// CacheStatic — general static content (CSS, JS without hashes).
	CacheStatic
	// CacheDynamic — content that may change (HTML, API responses).
	CacheDynamic
	// CacheNoCache — content that must not be cached (API tokens, user data).
	CacheNoCache
)

// CacheTierHeader maps each tier to its Cache-Control header value.
var CacheTierHeader = map[CacheTier]string{
	CacheImmutable:  "public, max-age=31536000, immutable",
	CacheLongStatic: "public, max-age=604800",
	CacheStatic:     "public, max-age=86400",
	CacheDynamic:    "public, max-age=0, must-revalidate",
	CacheNoCache:    "no-store, no-cache, must-revalidate",
}

// CacheClassify returns the appropriate cache tier for a file based on its extension.
func CacheClassify(key string) CacheTier {
	ext := strings.ToLower(filepath.Ext(key))
	switch ext {
	case ".woff", ".woff2", ".ttf", ".otf", ".eot":
		return CacheImmutable
	case ".css", ".js", ".mjs":
		// Hashed filenames (e.g., app.abc123.js) get immutable
		if isHashedFilename(key) {
			return CacheImmutable
		}
		return CacheStatic
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico", ".avif":
		return CacheLongStatic
	case ".mp4", ".webm", ".mp3", ".wav", ".ogg":
		return CacheLongStatic
	case ".pdf", ".zip", ".tar", ".gz", ".bz2":
		return CacheStatic
	case ".html", ".htm":
		return CacheDynamic
	case ".json", ".xml", ".yaml", ".yml":
		return CacheDynamic
	default:
		return CacheStatic
	}
}

// CacheControlHeader returns the Cache-Control header value for a file.
func CacheControlHeader(key string) string {
	tier := CacheClassify(key)
	return CacheTierHeader[tier]
}

// isHashedFilename detects content-hashed filenames like "app.abc123.js" or "main-deadbeef.css".
func isHashedFilename(key string) bool {
	base := filepath.Base(key)
	// Remove extension
	name := base
	if idx := strings.LastIndex(base, "."); idx > 0 {
		name = base[:idx]
	}
	// Check for hash pattern: contains a dot or dash followed by 6+ hex chars
	for _, sep := range []string{".", "-"} {
		parts := strings.Split(name, sep)
		if len(parts) >= 2 {
			last := parts[len(parts)-1]
			if isHexString(last) && len(last) >= 6 {
				return true
			}
		}
	}
	return false
}

func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return len(s) > 0
}
