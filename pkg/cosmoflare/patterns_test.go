package cosmoflare

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func scanProjectFixtureRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "project")
}

func findCachedRoute(t *testing.T, inv *ProjectCacheInventory, route string) CachedRoute {
	t.Helper()
	for _, r := range inv.Routes {
		if r.Route == route {
			return r
		}
	}
	t.Fatalf("route %q not found in inventory (have %d routes)", route, len(inv.Routes))
	return CachedRoute{}
}

func TestScanProjectCacheInventory(t *testing.T) {
	inv, err := ScanProjectCache(scanProjectFixtureRoot(t))
	if err != nil {
		t.Fatalf("ScanProjectCache: %v", err)
	}
	if len(inv.Routes) != 4 {
		t.Fatalf("routes count = %d, want 4 (meta 3 + user 1)", len(inv.Routes))
	}
	if inv.TotalGETRoutes != 6 {
		t.Fatalf("TotalGETRoutes = %d, want 6 (meta 3 + user 1 + stats 2)", inv.TotalGETRoutes)
	}
	if inv.FilesScanned != 4 {
		t.Fatalf("FilesScanned = %d, want 4 (spec file skipped)", inv.FilesScanned)
	}
	if inv.Root != filepath.Join("testdata", "project") {
		t.Fatalf("Root = %q, want the scan root as given", inv.Root)
	}

	stats := findCachedRoute(t, inv, "GET /stats")
	if !stats.KeyFn {
		t.Errorf("/stats KeyFn = false, want true")
	}
	if stats.QueryAware {
		t.Errorf("/stats QueryAware = true, want false")
	}
	if stats.TTL != 600 {
		t.Errorf("/stats TTL = %d, want 600", stats.TTL)
	}
	if stats.KeyPrefix != "api:meta:stats" {
		t.Errorf("/stats KeyPrefix = %q, want %q", stats.KeyPrefix, "api:meta:stats")
	}
	if stats.HasAuth {
		t.Errorf("/stats HasAuth = true, want false")
	}
	if stats.File != filepath.ToSlash(filepath.Join("routes", "meta.ts")) {
		t.Errorf("/stats File = %q, want routes/meta.ts ('/'-separated)", stats.File)
	}

	cities := findCachedRoute(t, inv, "GET /cities")
	if !cities.QueryAware {
		t.Errorf("/cities QueryAware = false, want true (options contain .query()")
	}
	if cities.TTL != 21600 {
		t.Errorf("/cities TTL = %d, want 21600", cities.TTL)
	}
	if cities.KeyPrefix != "api:meta:cities:" {
		t.Errorf("/cities KeyPrefix = %q, want %q", cities.KeyPrefix, "api:meta:cities:")
	}
	if !cities.KeyFn {
		t.Errorf("/cities KeyFn = false, want true")
	}

	states := findCachedRoute(t, inv, "GET /states")
	if !states.QueryAware {
		t.Errorf("/states QueryAware = false, want true (options text contains .query()")
	}
	if states.KeyPrefix != "api:meta:states:" {
		t.Errorf("/states KeyPrefix = %q, want %q", states.KeyPrefix, "api:meta:states:")
	}
	if states.HasAuth {
		t.Errorf("/states HasAuth = true, want false")
	}

	me := findCachedRoute(t, inv, "GET /me")
	if !me.HasAuth {
		t.Errorf("/me HasAuth = false, want true (authMiddleware in route window)")
	}
	if me.TTL != 300 {
		t.Errorf("/me TTL = %d, want 300", me.TTL)
	}
	if me.KeyPrefix != "api:user:me:" {
		t.Errorf("/me KeyPrefix = %q, want %q", me.KeyPrefix, "api:user:me:")
	}
}

func TestScanProjectCacheSkipsTestFiles(t *testing.T) {
	inv, err := ScanProjectCache(scanProjectFixtureRoot(t))
	if err != nil {
		t.Fatalf("ScanProjectCache: %v", err)
	}
	for _, r := range inv.Routes {
		if strings.Contains(r.File, ".spec.") {
			t.Errorf("route from spec file leaked into inventory: %+v", r)
		}
	}
	for _, a := range inv.Aggregates {
		if strings.Contains(a.File, ".spec.") {
			t.Errorf("aggregate from spec file leaked into inventory: %+v", a)
		}
	}
}

func TestScanProjectCacheInvalidation(t *testing.T) {
	inv, err := ScanProjectCache(scanProjectFixtureRoot(t))
	if err != nil {
		t.Fatalf("ScanProjectCache: %v", err)
	}
	if len(inv.Invalidation) != 2 {
		t.Fatalf("invalidation count = %d, want 2 (churchCacheKey slug + page:/church template)", len(inv.Invalidation))
	}
	patterns := map[string]bool{}
	for _, ic := range inv.Invalidation {
		if ic.File != filepath.ToSlash(filepath.Join("middleware", "cache.ts")) {
			t.Errorf("invalidation file = %q, want middleware/cache.ts", ic.File)
		}
		patterns[ic.KeyPattern] = true
	}
	if !patterns["churchCacheKey"] {
		t.Errorf("function-call delete arg did not yield bare name churchCacheKey; got %v", patterns)
	}
	if !patterns["page:/church/"] {
		t.Errorf("template delete arg did not yield literal head page:/church/; got %v", patterns)
	}
}

func TestScanProjectCacheAggregates(t *testing.T) {
	uncachedSQLAgg := `SELECT country, COUNT(*) FROM churches GROUP BY country`
	inv, err := ScanProjectCache(scanProjectFixtureRoot(t))
	if err != nil {
		t.Fatalf("ScanProjectCache: %v", err)
	}
	if len(inv.Aggregates) != 1 {
		t.Fatalf("aggregates count = %d, want 1 (WHERE'd /detail excluded)", len(inv.Aggregates))
	}
	agg := inv.Aggregates[0]
	if agg.Route != "GET /summary" {
		t.Errorf("aggregate route = %q, want %q", agg.Route, "GET /summary")
	}
	if !strings.Contains(strings.ToUpper(agg.SQL), "COUNT(") || !strings.Contains(strings.ToUpper(agg.SQL), "GROUP BY") {
		t.Errorf("aggregate SQL = %q, want COUNT( + GROUP BY", agg.SQL)
	}
	if strings.Contains(strings.ToUpper(agg.SQL), "WHERE") {
		t.Errorf("aggregate SQL = %q, want no WHERE", agg.SQL)
	}
	if agg.File != filepath.ToSlash(filepath.Join("routes", "stats.ts")) {
		t.Errorf("aggregate file = %q, want routes/stats.ts", agg.File)
	}
	_ = uncachedSQLAgg
}

func TestScanProjectCacheMiddlewareBehavior(t *testing.T) {
	inv, err := ScanProjectCache(scanProjectFixtureRoot(t))
	if err != nil {
		t.Fatalf("ScanProjectCache: %v", err)
	}
	if len(inv.Middleware) != 1 {
		t.Fatalf("middleware count = %d, want 1", len(inv.Middleware))
	}
	mw := inv.Middleware[0]
	if mw.File != filepath.ToSlash(filepath.Join("middleware", "cache.ts")) {
		t.Errorf("middleware file = %q, want middleware/cache.ts", mw.File)
	}
	if !mw.EmitsCacheControl {
		t.Errorf("EmitsCacheControl = false, want true (Cache-Control header line present)")
	}
	if !mw.OnlyStatus200 {
		t.Errorf("OnlyStatus200 = false, want true (=== 200 guard present)")
	}
}

func TestScanProjectCacheMissingRoot(t *testing.T) {
	_, err := ScanProjectCache(filepath.Join(scanProjectFixtureRoot(t), "does-not-exist"))
	if err == nil {
		t.Fatal("ScanProjectCache on missing root: err = nil, want error")
	}
	if !strings.Contains(err.Error(), "does not exist") || !strings.Contains(err.Error(), "does-not-exist") {
		t.Errorf("error = %q, want it to name the missing directory", err)
	}
}

// TestScanProjectCacheMultiLineShapes pins the corpus-calibrated scanner
// behaviors the fixtures alone did not cover: comment banners above
// multi-line route defs must not double-count, and a block-bodied keyFn with
// ttl declared before the keyFn and incidental literals (query reads,
// early `return null`) must yield the real key prefix and ttl.
func TestScanProjectCacheMultiLineShapes(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("routes/inline.ts", `// -- GET /api/x/inline ----------------
// comment banner above a multi-line def must not double-count.
xRoutes.get(
  '/inline',
  kvCache({
    ttl: 3600,
    // Window is anchored to today, so the date belongs in the key.
    keyFn: (c) => {
      const city = c.req.query('city');
      if (!city) return null;
      const days = c.req.query('days') || '7';
      return `+"`"+`api:x:inline:${city}:${days}`+"`"+`;
    },
  }),
  async (c) => {
    return c.json({ ok: true })
  }
)
`)
	write("routes/func.ts", `// banner
fRoutes.get(
  '/by-slug',
  kvCache({
    keyFn: (ctx: any) => churchCacheKey(ctx.req.param('slug')),
    ttl: 3600,
  }),
  async (c) => {
    const slug = c.req.param('slug');
    return c.json({ slug })
  }
)
`)
	inv, err := ScanProjectCache(dir)
	if err != nil {
		t.Fatalf("ScanProjectCache: %v", err)
	}
	var inline, bySlug *CachedRoute
	for i := range inv.Routes {
		switch inv.Routes[i].Route {
		case "GET /inline":
			inline = &inv.Routes[i]
		case "GET /by-slug":
			bySlug = &inv.Routes[i]
		}
	}
	if inline == nil || bySlug == nil {
		t.Fatalf("routes missing: %+v", inv.Routes)
	}
	if inline.TTL != 3600 {
		t.Errorf("inline TTL = %d, want 3600 (ttl declared before keyFn)", inline.TTL)
	}
	if inline.KeyPrefix != "api:x:inline:" {
		t.Errorf("inline KeyPrefix = %q, want api:x:inline:", inline.KeyPrefix)
	}
	if bySlug.KeyPrefix != "" {
		t.Errorf("bySlug KeyPrefix = %q, want \"\" (function-call key)", bySlug.KeyPrefix)
	}
	if inv.TotalGETRoutes != 2 {
		t.Fatalf("TotalGETRoutes = %d, want 2 (banners must not count)", inv.TotalGETRoutes)
	}
}
