/*
Package tui provides the interactive terminal dashboard for Cosmoflare.

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package tui

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestBrowser returns a BrowserModel backed by the null (no-credentials)
// data source, so tests never touch the network.
func newTestBrowser() BrowserModel {
	ds := &nullDataSource{}
	return NewBrowserModel(ds)
}

// sampleBuckets returns a deterministic three-bucket fixture used by tests
// that need a populated left pane.
func sampleBuckets() []Bucket {
	return []Bucket{
		{Name: "assets", Size: 1024, ObjectCount: 10, Status: "active", CreatedAt: time.Now()},
		{Name: "backups", Size: 2048, ObjectCount: 20, Status: "active", CreatedAt: time.Now()},
		{Name: "logs", Size: 512, ObjectCount: 5, Status: "active", CreatedAt: time.Now()},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestBrowserModelInitialState verifies every field of a freshly constructed
// BrowserModel: focus starts on the bucket pane, both navigation stacks hold
// only the root entry, no listing is loaded, and the modal is closed.
func TestBrowserModelInitialState(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()

	checks := []struct {
		name string
		ok   bool
		msg  string
	}{
		{"focus starts on left pane", b.focusLeft, "expected focusLeft=true on init"},
		{"prefix stack holds only the root", len(b.prefixStack) == 1 && b.prefixStack[0] == "", `expected prefixStack=[""]`},
		{"token stack holds only the root", len(b.tokenStack) == 1 && b.tokenStack[0] == "", `expected tokenStack=[""]`},
		{"no listing loaded", b.listing == nil, "expected listing=nil on init"},
		{"object cursor at zero", b.objectIdx == 0, "expected objectIdx=0 on init"},
		{"bucket cursor at zero", b.bucketIdx == 0, "expected bucketIdx=0 on init"},
		{"modal hidden", !b.showModal, "expected showModal=false on init"},
		{"no pending confirmation", b.confirmAction == "", `expected confirmAction="" on init`},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !c.ok {
				t.Error(c.msg)
			}
		})
	}
}

// TestBrowserModelSetBuckets verifies that SetBuckets stores the provided
// buckets verbatim, preserving both order and names.
func TestBrowserModelSetBuckets(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()
	buckets := sampleBuckets()
	b.SetBuckets(buckets)

	if len(b.buckets) != 3 {
		t.Fatalf("expected 3 buckets, got %d", len(b.buckets))
	}
	if b.buckets[0].Name != "assets" {
		t.Errorf("expected first bucket 'assets', got %q", b.buckets[0].Name)
	}
	if b.buckets[1].Name != "backups" {
		t.Errorf("expected second bucket 'backups', got %q", b.buckets[1].Name)
	}
	if b.buckets[2].Name != "logs" {
		t.Errorf("expected third bucket 'logs', got %q", b.buckets[2].Name)
	}
}

// TestBrowserModelPaneFocus verifies that toggleFocus alternates keyboard
// focus between the left (buckets) and right (objects) panes.
func TestBrowserModelPaneFocus(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()

	if !b.focusLeft {
		t.Error("expected focus on left pane initially")
	}

	b.toggleFocus()
	if b.focusLeft {
		t.Error("expected focus on right pane after first toggle")
	}

	b.toggleFocus()
	if !b.focusLeft {
		t.Error("expected focus on left pane after second toggle")
	}
}

// TestBrowserModelSetSize verifies that SetSize records the terminal
// dimensions used for layout decisions.
func TestBrowserModelSetSize(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()
	b.SetSize(120, 40)

	if b.width != 120 {
		t.Errorf("expected width=120, got %d", b.width)
	}
	if b.height != 40 {
		t.Errorf("expected height=40, got %d", b.height)
	}
}

// TestBrowserModelIsWide verifies the wide-layout threshold: terminal widths
// of 100 columns or more render both panes side by side, narrower widths
// stack them vertically.
func TestBrowserModelIsWide(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		width int
		want  bool
	}{
		{"width 120 is wide", 120, true},
		{"width 100 is the inclusive threshold", 100, true},
		{"width 99 is the largest narrow width", 99, false},
		{"width 80 is narrow", 80, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newTestBrowser()
			b.SetSize(tt.width, 40)
			if got := b.isWide(); got != tt.want {
				t.Errorf("isWide() with width=%d = %v, want %v", tt.width, got, tt.want)
			}
		})
	}
}

// TestBrowserModelSelectedBucket verifies cursor-to-bucket resolution across
// the empty list, the default cursor, an explicit index, and an out-of-range
// index (which must yield nil rather than panic).
func TestBrowserModelSelectedBucket(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()

	// Empty buckets → nil.
	if got := b.SelectedBucket(); got != nil {
		t.Error("expected nil when no buckets")
	}

	b.SetBuckets(sampleBuckets())

	// Default idx=0.
	got := b.SelectedBucket()
	if got == nil {
		t.Fatal("expected non-nil bucket at idx 0")
	}
	if got.Name != "assets" {
		t.Errorf("expected 'assets', got %q", got.Name)
	}

	// Navigate to idx 2.
	b.bucketIdx = 2
	got = b.SelectedBucket()
	if got == nil {
		t.Fatal("expected non-nil bucket at idx 2")
	}
	if got.Name != "logs" {
		t.Errorf("expected 'logs', got %q", got.Name)
	}

	// Out of range → nil.
	b.bucketIdx = 10
	if got := b.SelectedBucket(); got != nil {
		t.Error("expected nil for out-of-range idx")
	}
}

// TestBrowserPrefixPush verifies that entering a directory pushes its prefix
// onto the navigation stack, updates the current prefix, and resets the
// object cursor and cached listing so stale rows cannot leak through.
func TestBrowserPrefixPush(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()

	if b.currentPrefix() != "" {
		t.Errorf("expected empty root prefix, got %q", b.currentPrefix())
	}

	b.pushPrefix("images/")
	if b.currentPrefix() != "images/" {
		t.Errorf("expected 'images/', got %q", b.currentPrefix())
	}
	if len(b.prefixStack) != 2 {
		t.Errorf("expected prefixStack length 2, got %d", len(b.prefixStack))
	}

	b.pushPrefix("images/thumbnails/")
	if b.currentPrefix() != "images/thumbnails/" {
		t.Errorf("expected 'images/thumbnails/', got %q", b.currentPrefix())
	}
	if len(b.prefixStack) != 3 {
		t.Errorf("expected prefixStack length 3, got %d", len(b.prefixStack))
	}

	// Push resets objectIdx and listing.
	if b.objectIdx != 0 {
		t.Error("expected objectIdx=0 after push")
	}
	if b.listing != nil {
		t.Error("expected listing=nil after push")
	}
}

// TestBrowserPrefixPop verifies that leaving a directory unwinds the prefix
// stack one level at a time back to the root.
func TestBrowserPrefixPop(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()
	b.pushPrefix("images/")
	b.pushPrefix("images/thumbnails/")

	b.popPrefix()
	if b.currentPrefix() != "images/" {
		t.Errorf("expected 'images/' after pop, got %q", b.currentPrefix())
	}
	if len(b.prefixStack) != 2 {
		t.Errorf("expected prefixStack length 2, got %d", len(b.prefixStack))
	}

	b.popPrefix()
	if b.currentPrefix() != "" {
		t.Errorf("expected empty root after second pop, got %q", b.currentPrefix())
	}
}

// TestBrowserPrefixPopAtRoot verifies the special case of popping at the
// root: instead of underflowing the stack, focus switches to the bucket pane
// and the stack keeps its single root entry.
func TestBrowserPrefixPopAtRoot(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()
	b.focusLeft = false // start on right pane

	b.popPrefix()

	// Should switch focus to left pane when already at root.
	if !b.focusLeft {
		t.Error("expected focusLeft=true after pop at root")
	}
	// Stack should remain at 1 element.
	if len(b.prefixStack) != 1 {
		t.Errorf("expected prefixStack length 1, got %d", len(b.prefixStack))
	}
}

// TestBrowserTokenStack verifies the pagination-token stack starts with a
// single empty token representing the first page.
func TestBrowserTokenStack(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()

	if len(b.tokenStack) != 1 {
		t.Fatalf("expected tokenStack length 1, got %d", len(b.tokenStack))
	}
	if b.tokenStack[0] != "" {
		t.Errorf("expected tokenStack=[\"\"], got %v", b.tokenStack)
	}
}

// TestBrowserBreadcrumb verifies the breadcrumb trail: at the root it shows
// only the bucket name, and after entering a prefix the path is appended
// after the bucket joined with " > ".
func TestBrowserBreadcrumb(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()
	b.SetBuckets(sampleBuckets())

	// At root.
	bc := b.breadcrumb()
	if bc != "assets" {
		t.Errorf("expected 'assets', got %q", bc)
	}

	// With prefix.
	b.pushPrefix("images/")
	bc = b.breadcrumb()
	if bc != "assets > images/" {
		t.Errorf("expected 'assets > images/', got %q", bc)
	}
}

// TestBrowserTotalRightItems verifies the right-pane item count sums the
// directories and objects of the current listing and is zero when no listing
// is loaded.
func TestBrowserTotalRightItems(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()

	// No listing.
	if b.totalRightItems() != 0 {
		t.Errorf("expected 0 items with nil listing, got %d", b.totalRightItems())
	}

	// With listing.
	b.listing = &ObjectListing{
		Dirs: []string{"dir1/", "dir2/"},
		Objects: []ObjectItem{
			{Key: "file1.txt"},
			{Key: "file2.txt"},
			{Key: "file3.txt"},
		},
	}
	if b.totalRightItems() != 5 {
		t.Errorf("expected 5 items, got %d", b.totalRightItems())
	}
}

// TestBrowserSelectedIsDir verifies that directory entries sort ahead of
// objects in the right pane, so cursor indices 0..len(Dirs)-1 resolve to
// directories and later indices to objects.
func TestBrowserSelectedIsDir(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()
	b.listing = &ObjectListing{
		Dirs:    []string{"images/", "docs/"},
		Objects: []ObjectItem{{Key: "readme.md"}},
	}

	b.objectIdx = 0
	if !b.selectedIsDir() {
		t.Error("expected selectedIsDir=true at idx 0 (first dir)")
	}

	b.objectIdx = 1
	if !b.selectedIsDir() {
		t.Error("expected selectedIsDir=true at idx 1 (second dir)")
	}

	b.objectIdx = 2
	if b.selectedIsDir() {
		t.Error("expected selectedIsDir=false at idx 2 (object)")
	}
}

// TestBrowserSelectedObject verifies that selectedObject returns nil while
// the cursor is on a directory and the actual ObjectItem when it is on an
// object row.
func TestBrowserSelectedObject(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()
	b.listing = &ObjectListing{
		Dirs:    []string{"images/"},
		Objects: []ObjectItem{{Key: "readme.md", Size: 1024}},
	}

	// On a dir → nil.
	b.objectIdx = 0
	if b.selectedObject() != nil {
		t.Error("expected nil when on a directory")
	}

	// On an object → non-nil.
	b.objectIdx = 1
	obj := b.selectedObject()
	if obj == nil {
		t.Fatal("expected non-nil object at idx 1")
	}
	if obj.Key != "readme.md" {
		t.Errorf("expected 'readme.md', got %q", obj.Key)
	}
}

// TestBrowserSelectedDirName verifies directory-name resolution by cursor
// index, including the empty-string result when the cursor points at an
// object row instead of a directory.
func TestBrowserSelectedDirName(t *testing.T) {
	t.Parallel()
	b := newTestBrowser()
	b.listing = &ObjectListing{
		Dirs:    []string{"images/", "docs/"},
		Objects: []ObjectItem{{Key: "file.txt"}},
	}

	b.objectIdx = 0
	if got := b.selectedDirName(); got != "images/" {
		t.Errorf("expected 'images/', got %q", got)
	}

	b.objectIdx = 1
	if got := b.selectedDirName(); got != "docs/" {
		t.Errorf("expected 'docs/', got %q", got)
	}

	// Out of dir range → empty.
	b.objectIdx = 2
	if got := b.selectedDirName(); got != "" {
		t.Errorf("expected empty string for object idx, got %q", got)
	}
}

// TestBrowserViewNarrowLeftPane verifies the stacked (narrow) layout still
// renders the bucket pane with its title and bucket names. Kept sequential:
// InitializeStyles mutates package-level style state.
func TestBrowserViewNarrowLeftPane(t *testing.T) {
	b := newTestBrowser()
	b.SetSize(80, 24)
	b.SetBuckets(sampleBuckets())

	// Ensure styles are initialized.
	InitializeStyles(darkTheme)

	view := b.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
	// Should contain bucket names.
	if !containsStr(view, "assets") {
		t.Error("expected view to contain 'assets'")
	}
	if !containsStr(view, "Buckets") {
		t.Error("expected view to contain 'Buckets' title")
	}
}

// TestBrowserViewWide verifies the side-by-side (wide) layout renders the
// bucket list. Kept sequential: InitializeStyles mutates package-level
// style state.
func TestBrowserViewWide(t *testing.T) {
	b := newTestBrowser()
	b.SetSize(120, 40)
	b.SetBuckets(sampleBuckets())

	InitializeStyles(darkTheme)

	view := b.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
	// Wide mode should show bucket list.
	if !containsStr(view, "assets") {
		t.Error("expected view to contain 'assets'")
	}
}

// containsStr is a test helper for substring checks.
func containsStr(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && contains(haystack, needle)
}

// contains reports whether sub appears anywhere in s without using the
// strings package, keeping browser view assertions dependency-free.
func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
