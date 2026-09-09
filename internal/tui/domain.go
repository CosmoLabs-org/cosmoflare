/*
Package tui — domain browser.

DomainBrowserModel is a split-pane browser over the account's domains, mirroring
BrowserModel: a left list of domains with nameserver-status badges and a right
detail pane (nameservers, SSL, registrar overlay, redirect rules). It also has a
quick-add prompt for creating a redirect on the selected domain, and a
standalone launcher (RunDomainBrowser) used by `cosmoflare domains tui`.

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package tui

import (
	"fmt"
	"strings"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DomainBrowserModel renders a split-pane browser over a set of domains.
type DomainBrowserModel struct {
	domains  []*cosmoflare.DomainStatus
	detail   *cosmoflare.DomainDetail
	selected int
	width    int
	height   int

	// Quick-add redirect state.
	quickAdd       bool
	input          textinput.Model
	createRedirect func(zoneID, destination string) error
	status         string // transient status/error line
}

// NewDomainBrowserModel builds a browser over the given domain statuses.
func NewDomainBrowserModel(domains []*cosmoflare.DomainStatus) DomainBrowserModel {
	ti := textinput.New()
	ti.Placeholder = "https://destination.example/"
	ti.Prompt = "redirect → "
	ti.CharLimit = 2048
	return DomainBrowserModel{domains: domains, input: ti}
}

// SetSize updates the available terminal dimensions.
func (m *DomainBrowserModel) SetSize(w, h int) { m.width, m.height = w, h }

// SetDetail attaches enriched detail for the selected domain (right pane).
func (m *DomainBrowserModel) SetDetail(d *cosmoflare.DomainDetail) { m.detail = d }

// SetCreateRedirect wires the callback used by the quick-add prompt.
func (m *DomainBrowserModel) SetCreateRedirect(fn func(zoneID, destination string) error) {
	m.createRedirect = fn
}

// Selected returns the highlighted domain, or nil when the list is empty.
func (m DomainBrowserModel) Selected() *cosmoflare.DomainStatus {
	if m.selected < 0 || m.selected >= len(m.domains) {
		return nil
	}
	return m.domains[m.selected]
}

// quickAddOpen reports whether the quick-add prompt is active.
func (m DomainBrowserModel) quickAddOpen() bool { return m.quickAdd }

// StartQuickAdd opens the quick-add redirect prompt for the selected domain.
// It is a no-op when nothing is selected.
func (m *DomainBrowserModel) StartQuickAdd() {
	if m.Selected() == nil {
		return
	}
	m.quickAdd = true
	m.status = ""
	m.input.SetValue("")
	m.input.Focus()
}

// domainStatusIcon maps a domain's nameserver status to a compact list badge.
func domainStatusIcon(d *cosmoflare.DomainStatus) string {
	switch d.NSStatus {
	case "cloudflare":
		return "cf✓"
	case "external":
		return "ext✗"
	case "mismatch":
		return "cf⚠"
	default:
		return "?"
	}
}

// Update handles navigation and the quick-add prompt.
func (m DomainBrowserModel) Update(msg tea.Msg) (DomainBrowserModel, tea.Cmd) {
	if m.quickAdd {
		return m.updateQuickAdd(msg)
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
				m.detail = nil
			}
		case "down", "j":
			if m.selected < len(m.domains)-1 {
				m.selected++
				m.detail = nil
			}
		case "a":
			m.StartQuickAdd()
		}
	}
	return m, nil
}

// updateQuickAdd drives the textinput while the quick-add prompt is open.
func (m DomainBrowserModel) updateQuickAdd(msg tea.Msg) (DomainBrowserModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEnter:
			dest := strings.TrimSpace(m.input.Value())
			sel := m.Selected()
			if dest != "" && sel != nil && sel.Zone != nil && m.createRedirect != nil {
				if err := m.createRedirect(sel.Zone.ID, dest); err != nil {
					m.status = "error: " + err.Error()
				} else {
					m.status = "redirect added → " + dest
				}
			}
			m.closeQuickAdd()
			return m, nil
		case tea.KeyEsc:
			m.status = ""
			m.closeQuickAdd()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *DomainBrowserModel) closeQuickAdd() {
	m.quickAdd = false
	m.input.Blur()
	m.input.SetValue("")
}

// View renders the split-pane browser (with the quick-add line when open).
func (m DomainBrowserModel) View() string {
	width := m.width
	if width <= 0 {
		width = 80
	}
	leftW := width / 2
	if leftW < 28 {
		leftW = 28
	}

	rows := []string{lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("Domains")}
	if len(m.domains) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(mutedColor).Render("No domains found."))
	}
	nameW := leftW - 8
	for i, d := range m.domains {
		line := fmt.Sprintf("%-*s %s", nameW, truncate(domainName(d), nameW), domainStatusIcon(d))
		style := lipgloss.NewStyle()
		if i == m.selected {
			style = style.Background(primaryColor).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
		}
		rows = append(rows, style.Render(line))
	}
	rows = append(rows, lipgloss.NewStyle().Foreground(mutedColor).Render(fmt.Sprintf("%d domain(s)  ·  a: add redirect  ·  q: quit", len(m.domains))))

	left := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Width(leftW).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
	right := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).
		Render(lipgloss.JoinVertical(lipgloss.Left, m.detailPane()...))

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	switch {
	case m.quickAdd:
		prompt := lipgloss.NewStyle().Foreground(primaryColor).Render(
			fmt.Sprintf("Add redirect for %s:", domainName(m.Selected())))
		return lipgloss.JoinVertical(lipgloss.Left, body, prompt, m.input.View())
	case m.status != "":
		return lipgloss.JoinVertical(lipgloss.Left, body, lipgloss.NewStyle().Foreground(mutedColor).Render(m.status))
	default:
		return body
	}
}

// detailPane renders the right-hand detail for the selected domain.
func (m DomainBrowserModel) detailPane() []string {
	sel := m.Selected()
	if sel == nil {
		return []string{lipgloss.NewStyle().Foreground(mutedColor).Render("Select a domain.")}
	}
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render(domainName(sel)),
		"NS:     " + sel.NSStatus,
		"SSL:    " + sel.SSLStatus,
		"Health: " + sel.HealthStatus,
	}
	if sel.RedirectIssue != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(mutedColor).Render("redirect: "+sel.RedirectIssue))
	}
	if m.detail != nil {
		if len(m.detail.NameServers) > 0 {
			lines = append(lines, "Nameservers: "+strings.Join(m.detail.NameServers, ", "))
		}
		if reg := m.detail.Registrar; reg != nil {
			label := reg.Registrar
			if reg.RegistrarName != "" {
				label = reg.RegistrarName + " (" + reg.Registrar + ")"
			}
			lines = append(lines, "Registrar: "+label)
			// Auto-renew is intentionally not shown: the Cloudflare SDK read
			// model does not expose it (see cosmoflare.RegistrarInfo).
			if reg.ExpiresAt != nil {
				lines = append(lines, "Expires:   "+reg.ExpiresAt.Format("2006-01-02"))
			}
		}
		if len(m.detail.Redirects) > 0 {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Redirects:"))
			for _, r := range m.detail.Redirects {
				lines = append(lines, fmt.Sprintf("  %s → %s (%d)", r.When, r.Destination, r.StatusCode))
			}
		}
	}
	return lines
}

// domainName returns the display name for a domain status.
func domainName(d *cosmoflare.DomainStatus) string {
	if d != nil && d.Zone != nil {
		return d.Zone.Name
	}
	return "(unknown)"
}

// ---------------------------------------------------------------------------
// Standalone launcher (used by `cosmoflare domains tui`)
// ---------------------------------------------------------------------------

// domainTUIModel wraps DomainBrowserModel as a top-level bubbletea program.
type domainTUIModel struct {
	browser DomainBrowserModel
}

func (t domainTUIModel) Init() tea.Cmd { return nil }

func (t domainTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.browser.SetSize(msg.Width, msg.Height)
		return t, nil
	case tea.KeyMsg:
		if !t.browser.quickAddOpen() {
			switch msg.String() {
			case "q", "ctrl+c":
				return t, tea.Quit
			}
		}
	}
	var cmd tea.Cmd
	t.browser, cmd = t.browser.Update(msg)
	return t, cmd
}

func (t domainTUIModel) View() string { return t.browser.View() }

// RunDomainBrowser launches the interactive domain browser. createRedirect is
// invoked by the quick-add prompt (may be nil to disable adding).
func RunDomainBrowser(domains []*cosmoflare.DomainStatus, createRedirect func(zoneID, destination string) error) error {
	b := NewDomainBrowserModel(domains)
	b.SetCreateRedirect(createRedirect)
	p := tea.NewProgram(domainTUIModel{browser: b}, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
