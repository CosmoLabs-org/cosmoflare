/*
Package tui provides the interactive terminal dashboard for Cosmoflare.

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package tui

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type browserObjectsMsg struct {
	listing ObjectListing
	err     error
}

type browserHeadMsg struct {
	detail *ObjectDetail
	err    error
}

type browserDeleteMsg struct {
	err error
}

// ---------------------------------------------------------------------------
// BrowserModel — split-pane bucket/object browser
// ---------------------------------------------------------------------------

// BrowserModel provides a split-pane layout for browsing R2 buckets and objects.
// Wide terminals (>=100 cols) show buckets and objects side-by-side; narrow
// terminals show a single pane with Tab to switch.
type BrowserModel struct {
	data   DataSource
	width  int
	height int

	buckets   []Bucket
	bucketIdx int

	listing     *ObjectListing
	objectIdx   int
	tokenStack  []string
	prefixStack []string

	focusLeft bool

	confirmAction string
	confirmTarget string

	showModal   bool
	modalDetail *ObjectDetail
	modalErr    error

	loadingObjects bool
}

// NewBrowserModel creates a BrowserModel wired to the given DataSource.
func NewBrowserModel(ds DataSource) BrowserModel {
	return BrowserModel{
		data:        ds,
		focusLeft:   true,
		prefixStack: []string{""},
		tokenStack:  []string{""},
	}
}

// ---------------------------------------------------------------------------
// Accessors / helpers
// ---------------------------------------------------------------------------

// SetSize updates the available terminal dimensions.
func (b *BrowserModel) SetSize(w, h int) {
	b.width = w
	b.height = h
}

// SetBuckets replaces the bucket list.
func (b *BrowserModel) SetBuckets(buckets []Bucket) {
	b.buckets = buckets
}

// SelectedBucket returns the currently highlighted bucket, or nil.
func (b BrowserModel) SelectedBucket() *Bucket {
	if len(b.buckets) == 0 || b.bucketIdx < 0 || b.bucketIdx >= len(b.buckets) {
		return nil
	}
	return &b.buckets[b.bucketIdx]
}

func (b BrowserModel) isWide() bool { return b.width >= 100 }

func (b *BrowserModel) toggleFocus() { b.focusLeft = !b.focusLeft }

func (b BrowserModel) currentPrefix() string {
	if len(b.prefixStack) == 0 {
		return ""
	}
	return b.prefixStack[len(b.prefixStack)-1]
}

func (b BrowserModel) breadcrumb() string {
	bucket := b.SelectedBucket()
	if bucket == nil {
		return ""
	}
	prefix := b.currentPrefix()
	if prefix == "" {
		return bucket.Name
	}
	return bucket.Name + " > " + prefix
}

func (b BrowserModel) totalRightItems() int {
	if b.listing == nil {
		return 0
	}
	return len(b.listing.Dirs) + len(b.listing.Objects)
}

func (b BrowserModel) selectedIsDir() bool {
	if b.listing == nil {
		return false
	}
	return b.objectIdx < len(b.listing.Dirs)
}

func (b BrowserModel) selectedObject() *ObjectItem {
	if b.listing == nil {
		return nil
	}
	objStart := len(b.listing.Dirs)
	idx := b.objectIdx - objStart
	if idx < 0 || idx >= len(b.listing.Objects) {
		return nil
	}
	return &b.listing.Objects[idx]
}

func (b BrowserModel) selectedDirName() string {
	if b.listing == nil || b.objectIdx < 0 || b.objectIdx >= len(b.listing.Dirs) {
		return ""
	}
	return b.listing.Dirs[b.objectIdx]
}

// ---------------------------------------------------------------------------
// Prefix navigation
// ---------------------------------------------------------------------------

func (b *BrowserModel) pushPrefix(dir string) {
	b.prefixStack = append(b.prefixStack, dir)
	b.objectIdx = 0
	b.listing = nil
	b.tokenStack = []string{""}
}

func (b *BrowserModel) popPrefix() {
	if len(b.prefixStack) <= 1 {
		b.focusLeft = true
		return
	}
	b.prefixStack = b.prefixStack[:len(b.prefixStack)-1]
	b.objectIdx = 0
	b.listing = nil
	b.tokenStack = []string{""}
}

// ---------------------------------------------------------------------------
// Commands (tea.Cmd producers)
// ---------------------------------------------------------------------------

func (b BrowserModel) fetchObjects(bucket, prefix, token string) tea.Cmd {
	return func() tea.Msg {
		if b.data == nil || !b.data.Available() {
			return browserObjectsMsg{err: fmt.Errorf("no data source available")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		listing, err := b.data.FetchObjects(ctx, bucket, prefix, token)
		return browserObjectsMsg{listing: listing, err: err}
	}
}

func (b BrowserModel) headObject(bucket, key string) tea.Cmd {
	return func() tea.Msg {
		if b.data == nil || !b.data.Available() {
			return browserHeadMsg{err: fmt.Errorf("no data source available")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		detail, err := b.data.HeadObject(ctx, bucket, key)
		return browserHeadMsg{detail: detail, err: err}
	}
}

func (b BrowserModel) deleteObject(bucket, key string) tea.Cmd {
	return func() tea.Msg {
		if b.data == nil || !b.data.Available() {
			return browserDeleteMsg{err: fmt.Errorf("no data source available")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := b.data.DeleteObject(ctx, bucket, key)
		return browserDeleteMsg{err: err}
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// Update handles messages for the browser sub-model.
func (b BrowserModel) Update(msg tea.Msg) (BrowserModel, tea.Cmd) {
	switch msg := msg.(type) {
	case browserObjectsMsg:
		b.loadingObjects = false
		if msg.err != nil {
			return b, nil
		}
		b.listing = &msg.listing
		b.objectIdx = 0
		return b, nil

	case browserHeadMsg:
		b.showModal = true
		b.modalDetail = msg.detail
		b.modalErr = msg.err
		return b, nil

	case browserDeleteMsg:
		if msg.err == nil {
			// Refresh listing after successful delete.
			bucket := b.SelectedBucket()
			if bucket != nil {
				token := ""
				if len(b.tokenStack) > 0 {
					token = b.tokenStack[len(b.tokenStack)-1]
				}
				b.loadingObjects = true
				return b, b.fetchObjects(bucket.Name, b.currentPrefix(), token)
			}
		}
		return b, nil

	case tea.KeyMsg:
		// Close modal on any key.
		if b.showModal {
			if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "q" {
				b.showModal = false
				b.modalDetail = nil
				b.modalErr = nil
			}
			return b, nil
		}

		if b.confirmAction != "" {
			return b.handleConfirm(msg)
		}
		return b.handleKey(msg)
	}
	return b, nil
}

// ---------------------------------------------------------------------------
// Key handling
// ---------------------------------------------------------------------------

func (b BrowserModel) handleKey(msg tea.KeyMsg) (BrowserModel, tea.Cmd) {
	key := msg.String()

	switch key {
	case "tab":
		b.toggleFocus()
		return b, nil

	case "up", "k":
		if b.focusLeft {
			if b.bucketIdx > 0 {
				b.bucketIdx--
			}
		} else {
			if b.objectIdx > 0 {
				b.objectIdx--
			}
		}
		return b, nil

	case "down", "j":
		if b.focusLeft {
			if b.bucketIdx < len(b.buckets)-1 {
				b.bucketIdx++
			}
		} else {
			total := b.totalRightItems()
			if b.objectIdx < total-1 {
				b.objectIdx++
			}
		}
		return b, nil

	case "enter":
		if b.focusLeft {
			// Select bucket → switch to right pane and fetch root listing.
			bucket := b.SelectedBucket()
			if bucket == nil {
				return b, nil
			}
			b.focusLeft = false
			b.prefixStack = []string{""}
			b.tokenStack = []string{""}
			b.objectIdx = 0
			b.listing = nil
			b.loadingObjects = true
			return b, b.fetchObjects(bucket.Name, "", "")
		}
		// Right pane enter.
		if b.selectedIsDir() {
			dir := b.selectedDirName()
			if dir != "" {
				bucket := b.SelectedBucket()
				if bucket != nil {
					b.pushPrefix(dir)
					b.loadingObjects = true
					return b, b.fetchObjects(bucket.Name, dir, "")
				}
			}
		} else if obj := b.selectedObject(); obj != nil {
			bucket := b.SelectedBucket()
			if bucket != nil {
				return b, b.headObject(bucket.Name, obj.Key)
			}
		}
		return b, nil

	case "backspace":
		if !b.focusLeft {
			b.popPrefix()
			if !b.focusLeft {
				// Still on right pane — fetch at new prefix.
				bucket := b.SelectedBucket()
				if bucket != nil {
					b.loadingObjects = true
					return b, b.fetchObjects(bucket.Name, b.currentPrefix(), "")
				}
			}
		}
		return b, nil

	case "r":
		bucket := b.SelectedBucket()
		if bucket != nil && !b.focusLeft {
			token := ""
			if len(b.tokenStack) > 0 {
				token = b.tokenStack[len(b.tokenStack)-1]
			}
			b.loadingObjects = true
			return b, b.fetchObjects(bucket.Name, b.currentPrefix(), token)
		}
		return b, nil

	case "n":
		if b.listing != nil && b.listing.HasMore && b.listing.NextToken != "" {
			bucket := b.SelectedBucket()
			if bucket != nil {
				b.tokenStack = append(b.tokenStack, b.listing.NextToken)
				b.loadingObjects = true
				return b, b.fetchObjects(bucket.Name, b.currentPrefix(), b.listing.NextToken)
			}
		}
		return b, nil

	case "p":
		if len(b.tokenStack) > 1 {
			b.tokenStack = b.tokenStack[:len(b.tokenStack)-1]
			token := b.tokenStack[len(b.tokenStack)-1]
			bucket := b.SelectedBucket()
			if bucket != nil {
				b.loadingObjects = true
				return b, b.fetchObjects(bucket.Name, b.currentPrefix(), token)
			}
		}
		return b, nil

	case "d":
		if !b.focusLeft {
			if obj := b.selectedObject(); obj != nil {
				b.confirmAction = "delete"
				b.confirmTarget = obj.Key
			}
		}
		return b, nil
	}

	return b, nil
}

func (b BrowserModel) handleConfirm(msg tea.KeyMsg) (BrowserModel, tea.Cmd) {
	key := msg.String()
	switch key {
	case "y":
		bucket := b.SelectedBucket()
		action := b.confirmAction
		target := b.confirmTarget
		b.confirmAction = ""
		b.confirmTarget = ""
		if action == "delete" && bucket != nil {
			return b, b.deleteObject(bucket.Name, target)
		}
		return b, nil
	case "n", "esc":
		b.confirmAction = ""
		b.confirmTarget = ""
		return b, nil
	}
	return b, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the browser sub-model.
func (b BrowserModel) View() string {
	if b.showModal {
		return b.renderModal()
	}

	if !b.isWide() {
		// Narrow: single pane, Tab switches.
		if b.focusLeft {
			return b.renderLeftPane(b.width, b.height)
		}
		header := lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Render("📁 " + b.breadcrumb())
		body := b.renderRightPane(b.width, b.height-1)
		return lipgloss.JoinVertical(lipgloss.Left, header, body)
	}

	// Wide: side-by-side.
	leftW := b.width*30/100
	if leftW < 20 {
		leftW = 20
	}
	rightW := b.width - leftW - 3 // 3 for border padding

	leftBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(b.paneBorderColor(true)).
		Width(leftW).
		Height(b.height - 2)

	rightBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(b.paneBorderColor(false)).
		Width(rightW).
		Height(b.height - 2)

	left := leftBorder.Render(b.renderLeftPane(leftW-2, b.height-4))
	right := rightBorder.Render(b.renderRightPane(rightW-2, b.height-4))

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (b BrowserModel) paneBorderColor(isLeft bool) lipgloss.Color {
	if b.focusLeft == isLeft {
		return primaryColor
	}
	return mutedColor
}

// ---------------------------------------------------------------------------
// renderLeftPane
// ---------------------------------------------------------------------------

func (b BrowserModel) renderLeftPane(w, h int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Width(w).
		Render("Buckets")

	var lines []string
	for i, bkt := range b.buckets {
		prefix := "  "
		style := lipgloss.NewStyle().Width(w)
		if i == b.bucketIdx {
			prefix = "▸ "
			if b.focusLeft {
				style = style.Background(primaryColor).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
			} else {
				style = style.Foreground(primaryColor).Bold(true)
			}
		}
		name := truncate(prefix+bkt.Name, w)
		lines = append(lines, style.Render(name))
	}

	if len(b.buckets) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(mutedColor).Render("No buckets found."))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return lipgloss.JoinVertical(lipgloss.Left, title, content)
}

// ---------------------------------------------------------------------------
// renderRightPane
// ---------------------------------------------------------------------------

func (b BrowserModel) renderRightPane(w, h int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Width(w).
		Render("📁 " + b.breadcrumb())

	if b.loadingObjects {
		return lipgloss.JoinVertical(lipgloss.Left, title, "Loading...")
	}
	if b.listing == nil {
		return lipgloss.JoinVertical(lipgloss.Left, title, lipgloss.NewStyle().Foreground(mutedColor).Render("Press Enter on a bucket to browse."))
	}

	total := b.totalRightItems()
	if total == 0 {
		msg := "Empty"
		if b.currentPrefix() != "" {
			msg = "No objects at this prefix."
		}
		detail := b.renderDetailPanel(w)
		footer := b.renderBrowserFooter(w)
		return lipgloss.JoinVertical(lipgloss.Left, title,
			lipgloss.NewStyle().Foreground(mutedColor).Render(msg), detail, footer)
	}

	// Compute how much space is available for the list.
	// Reserve lines for title(1) + detail(~4) + footer(1).
	listHeight := h - 6
	if listHeight < 3 {
		listHeight = 3
	}

	var lines []string
	// Directories first.
	for i, dir := range b.listing.Dirs {
		displayName := "📁 " + path.Base(strings.TrimSuffix(dir, "/")) + "/"
		style := lipgloss.NewStyle().Width(w)
		if i == b.objectIdx {
			if !b.focusLeft {
				style = style.Background(primaryColor).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
			} else {
				style = style.Foreground(primaryColor).Bold(true)
			}
		}
		lines = append(lines, style.Render(truncate(displayName, w)))
	}

	// Objects.
	dirCount := len(b.listing.Dirs)
	for i, obj := range b.listing.Objects {
		name := path.Base(obj.Key)
		size := utils.FormatBytes(obj.Size)
		date := obj.LastModified.Format("2006-01-02")
		line := fmt.Sprintf("%-*s  %8s  %s", w-24, truncate(name, w-24), size, date)

		style := lipgloss.NewStyle().Width(w)
		globalIdx := dirCount + i
		if globalIdx == b.objectIdx {
			if !b.focusLeft {
				style = style.Background(primaryColor).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
			} else {
				style = style.Foreground(primaryColor).Bold(true)
			}
		}
		lines = append(lines, style.Render(truncate(line, w)))
	}

	// Scroll if needed: show a window around objectIdx.
	if len(lines) > listHeight {
		start := b.objectIdx - listHeight/2
		if start < 0 {
			start = 0
		}
		end := start + listHeight
		if end > len(lines) {
			end = len(lines)
			start = end - listHeight
			if start < 0 {
				start = 0
			}
		}
		lines = lines[start:end]
	}

	list := lipgloss.JoinVertical(lipgloss.Left, lines...)
	detail := b.renderDetailPanel(w)
	footer := b.renderBrowserFooter(w)

	return lipgloss.JoinVertical(lipgloss.Left, title, list, detail, footer)
}

// ---------------------------------------------------------------------------
// renderDetailPanel
// ---------------------------------------------------------------------------

func (b BrowserModel) renderDetailPanel(w int) string {
	if b.focusLeft || b.totalRightItems() == 0 {
		return ""
	}

	sep := lipgloss.NewStyle().Foreground(mutedColor).Width(w).Render(strings.Repeat("─", w))

	if b.selectedIsDir() {
		dir := b.selectedDirName()
		info := fmt.Sprintf("📁 %s — Enter to browse, Backspace to go up", path.Base(strings.TrimSuffix(dir, "/")))
		return lipgloss.JoinVertical(lipgloss.Left, sep,
			lipgloss.NewStyle().Foreground(textColor).Render(truncate(info, w)))
	}

	obj := b.selectedObject()
	if obj == nil {
		return ""
	}

	keyLine := lipgloss.NewStyle().Bold(true).Foreground(textColor).Render(truncate("Key: "+obj.Key, w))
	metaLine := fmt.Sprintf("Size: %s | Modified: %s | ETag: %s",
		utils.FormatBytes(obj.Size),
		obj.LastModified.Format("2006-01-02 15:04"),
		obj.ETag)
	ct := obj.ContentType
	if ct == "" {
		ct = "—"
	}
	ctLine := "Content-Type: " + ct
	hintLine := lipgloss.NewStyle().Foreground(mutedColor).Render("Enter = metadata | d = delete | Backspace = up")

	return lipgloss.JoinVertical(lipgloss.Left, sep,
		keyLine,
		lipgloss.NewStyle().Foreground(textColor).Render(truncate(metaLine, w)),
		lipgloss.NewStyle().Foreground(textColor).Render(truncate(ctLine, w)),
		hintLine,
	)
}

// ---------------------------------------------------------------------------
// renderModal
// ---------------------------------------------------------------------------

func (b BrowserModel) renderModal() string {
	modalW := b.width - 10
	if modalW < 40 {
		modalW = 40
	}
	if modalW > 80 {
		modalW = 80
	}

	borderColor := primaryColor
	if b.modalErr != nil {
		borderColor = errorColor
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(borderColor).
		Width(modalW).
		Padding(1, 2)

	if b.modalErr != nil {
		content := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Foreground(errorColor).Render("Error"),
			"",
			b.modalErr.Error(),
			"",
			lipgloss.NewStyle().Foreground(mutedColor).Render("Esc = close"),
		)
		return lipgloss.Place(b.width, b.height, lipgloss.Center, lipgloss.Center, style.Render(content))
	}

	if b.modalDetail != nil {
		d := b.modalDetail
		metaLines := ""
		if len(d.Metadata) > 0 {
			var parts []string
			for k, v := range d.Metadata {
				parts = append(parts, fmt.Sprintf("  %s: %s", k, v))
			}
			metaLines = "\nMetadata:\n" + strings.Join(parts, "\n")
		}

		content := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("Object Detail"),
			"",
			"Key:          "+d.Key,
			"Size:         "+utils.FormatBytes(d.Size),
			"Modified:     "+d.LastModified.Format("2006-01-02 15:04:05"),
			"Content-Type: "+d.ContentType,
			"ETag:         "+d.ETag,
			metaLines,
			"",
			lipgloss.NewStyle().Foreground(mutedColor).Render("Esc = close"),
		)
		return lipgloss.Place(b.width, b.height, lipgloss.Center, lipgloss.Center, style.Render(content))
	}

	return ""
}

// ---------------------------------------------------------------------------
// renderBrowserFooter
// ---------------------------------------------------------------------------

func (b BrowserModel) renderBrowserFooter(w int) string {
	if b.confirmAction != "" {
		// Extract just the filename for the prompt.
		name := path.Base(b.confirmTarget)
		prompt := fmt.Sprintf("Delete '%s'? (y/n)", name)
		return lipgloss.NewStyle().Foreground(errorColor).Bold(true).Width(w).Render(prompt)
	}

	hint := "Tab = switch pane | ↑↓ = navigate | Enter = open"
	if b.listing != nil && b.listing.HasMore {
		hint += " | n = next page"
	}
	if len(b.tokenStack) > 1 {
		hint += " | p = prev page"
	}

	return lipgloss.NewStyle().Foreground(mutedColor).Width(w).Render(hint)
}
