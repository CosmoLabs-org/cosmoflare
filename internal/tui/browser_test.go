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

func newTestBrowser() BrowserModel {
	ds := &nullDataSource{}
	return NewBrowserModel(ds)
}

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

func TestBrowserModelInitialState(t *testing.T) {
	b := newTestBrowser()

	if !b.focusLeft {
		t.Error("expected focusLeft=true on init")
	}
	if len(b.prefixStack) != 1 || b.prefixStack[0] != "" {
		t.Errorf("expected prefixStack=[\"\"], got %v", b.prefixStack)
	}
	if len(b.tokenStack) != 1 || b.tokenStack[0] != "" {
		t.Errorf("expected tokenStack=[\"\"], got %v", b.tokenStack)
	}
	if b.listing != nil {
		t.Error("expected listing=nil on init")
	}
	if b.objectIdx != 0 {
		t.Error("expected objectIdx=0 on init")
	}
	if b.bucketIdx != 0 {
		t.Error("expected bucketIdx=0 on init")
	}
	if b.showModal {
		t.Error("expected showModal=false on init")
	}
	if b.confirmAction != "" {
		t.Error("expected confirmAction=\"\" on init")
	}
}

func TestBrowserModelSetBuckets(t *testing.T) {
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

func TestBrowserModelPaneFocus(t *testing.T) {
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

func TestBrowserModelSetSize(t *testing.T) {
	b := newTestBrowser()
	b.SetSize(120, 40)

	if b.width != 120 {
		t.Errorf("expected width=120, got %d", b.width)
	}
	if b.height != 40 {
		t.Errorf("expected height=40, got %d", b.height)
	}
}

func TestBrowserModelIsWide(t *testing.T) {
	b := newTestBrowser()

	b.SetSize(120, 40)
	if !b.isWide() {
		t.Error("expected isWide()=true for width=120")
	}

	b.SetSize(100, 40)
	if !b.isWide() {
		t.Error("expected isWide()=true for width=100")
	}

	b.SetSize(80, 40)
	if b.isWide() {
		t.Error("expected isWide()=false for width=80")
	}

	b.SetSize(99, 40)
	if b.isWide() {
		t.Error("expected isWide()=false for width=99")
	}
}

func TestBrowserModelSelectedBucket(t *testing.T) {
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

func TestBrowserPrefixPush(t *testing.T) {
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

func TestBrowserPrefixPop(t *testing.T) {
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

func TestBrowserPrefixPopAtRoot(t *testing.T) {
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

func TestBrowserTokenStack(t *testing.T) {
	b := newTestBrowser()

	if len(b.tokenStack) != 1 {
		t.Fatalf("expected tokenStack length 1, got %d", len(b.tokenStack))
	}
	if b.tokenStack[0] != "" {
		t.Errorf("expected tokenStack=[\"\"], got %v", b.tokenStack)
	}
}

func TestBrowserBreadcrumb(t *testing.T) {
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

func TestBrowserTotalRightItems(t *testing.T) {
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

func TestBrowserSelectedIsDir(t *testing.T) {
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

func TestBrowserSelectedObject(t *testing.T) {
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

func TestBrowserSelectedDirName(t *testing.T) {
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

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
