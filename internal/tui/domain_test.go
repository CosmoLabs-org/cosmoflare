package tui

import (
	"strings"
	"testing"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDomainStatusIcon(t *testing.T) {
	cases := map[string]string{
		"cloudflare": "cf✓",
		"external":   "ext✗",
		"mismatch":   "cf⚠",
		"weird":      "?",
	}
	for ns, want := range cases {
		if got := domainStatusIcon(&cosmoflare.DomainStatus{NSStatus: ns}); got != want {
			t.Errorf("ns=%q: got %q want %q", ns, got, want)
		}
	}
}

func TestDomainBrowserModel_EmptyRenders(t *testing.T) {
	m := NewDomainBrowserModel(nil)
	if v := m.View(); v == "" {
		t.Fatal("empty model rendered nothing")
	}
	if m.Selected() != nil {
		t.Error("empty model should have no selection")
	}
}

func TestDomainBrowserModel_Navigation(t *testing.T) {
	domains := []*cosmoflare.DomainStatus{
		{Zone: &cosmoflare.Zone{Name: "a.com"}, NSStatus: "cloudflare"},
		{Zone: &cosmoflare.Zone{Name: "b.com"}, NSStatus: "external"},
	}
	m := NewDomainBrowserModel(domains)
	if m.Selected().Zone.Name != "a.com" {
		t.Fatalf("initial selection = %q, want a.com", m.Selected().Zone.Name)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Selected().Zone.Name != "b.com" {
		t.Fatalf("after down = %q, want b.com", m.Selected().Zone.Name)
	}
	// clamp at the bottom
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.Selected().Zone.Name != "b.com" {
		t.Errorf("should clamp at last entry, got %q", m.Selected().Zone.Name)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.Selected().Zone.Name != "a.com" {
		t.Errorf("after up = %q, want a.com", m.Selected().Zone.Name)
	}
}

func TestDomainBrowserModel_DetailPaneRedirects(t *testing.T) {
	d := &cosmoflare.DomainStatus{Zone: &cosmoflare.Zone{Name: "x.com"}, NSStatus: "cloudflare", SSLStatus: "valid"}
	m := NewDomainBrowserModel([]*cosmoflare.DomainStatus{d})
	m.SetDetail(&cosmoflare.DomainDetail{
		DomainStatus: *d,
		Redirects:    []cosmoflare.RedirectRule{{When: "/old", Destination: "https://new.com", StatusCode: 301}},
	})
	out := m.View()
	if !strings.Contains(out, "https://new.com") {
		t.Fatalf("detail pane missing redirect destination:\n%s", out)
	}
}

func oneDomain() []*cosmoflare.DomainStatus {
	return []*cosmoflare.DomainStatus{{Zone: &cosmoflare.Zone{ID: "z1", Name: "x.com"}, NSStatus: "cloudflare"}}
}

func TestDomainBrowserModel_QuickAddOpens(t *testing.T) {
	m := NewDomainBrowserModel(oneDomain())
	if m.quickAddOpen() {
		t.Fatal("quick-add should start closed")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if !m.quickAddOpen() {
		t.Fatal("'a' should open the quick-add prompt")
	}
}

func TestDomainBrowserModel_QuickAddSubmit(t *testing.T) {
	var gotZone, gotDest string
	m := NewDomainBrowserModel(oneDomain())
	m.SetCreateRedirect(func(zoneID, dest string) error { gotZone, gotDest = zoneID, dest; return nil })
	m.StartQuickAdd()
	for _, r := range "https://new.example" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if gotZone != "z1" || gotDest != "https://new.example" {
		t.Fatalf("createRedirect got (%q, %q), want (z1, https://new.example)", gotZone, gotDest)
	}
	if m.quickAddOpen() {
		t.Error("quick-add should close after submit")
	}
}

func TestDomainBrowserModel_QuickAddCancel(t *testing.T) {
	called := false
	m := NewDomainBrowserModel(oneDomain())
	m.SetCreateRedirect(func(_, _ string) error { called = true; return nil })
	m.StartQuickAdd()
	if !m.quickAddOpen() {
		t.Fatal("quick-add should be open")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.quickAddOpen() {
		t.Error("Esc should cancel the quick-add prompt")
	}
	if called {
		t.Error("cancel must not invoke createRedirect")
	}
}

func TestDomainBrowserModel_QuickAddNoSelectionNoop(t *testing.T) {
	m := NewDomainBrowserModel(nil)
	m.StartQuickAdd()
	if m.quickAddOpen() {
		t.Error("quick-add must not open with no domain selected")
	}
}

func TestDomainBrowserDetailShowsRedirectIssueBadge(t *testing.T) {
	m := NewDomainBrowserModel([]*cosmoflare.DomainStatus{
		{Zone: &cosmoflare.Zone{ID: "z1", Name: "example.com"}, RedirectIssue: "loop"},
	})
	view := m.View()
	if !strings.Contains(view, "redirect: loop") {
		t.Fatal("detail pane must render the redirect issue badge when set")
	}

	m2 := NewDomainBrowserModel([]*cosmoflare.DomainStatus{
		{Zone: &cosmoflare.Zone{ID: "z1", Name: "example.com"}},
	})
	if strings.Contains(m2.View(), "redirect:") {
		t.Fatal("badge must be absent when RedirectIssue is empty")
	}
}
