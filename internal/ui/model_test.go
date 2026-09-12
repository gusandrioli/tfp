package ui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gusandrioli/tfp/internal/summary"
	"github.com/gusandrioli/tfp/internal/ui"
)

func key(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func update(t *testing.T, m ui.Model, msg tea.Msg) ui.Model {
	t.Helper()
	next, _ := m.Update(msg)
	got, ok := next.(ui.Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", next)
	}
	return got
}

func newTestModel(t *testing.T, fixture string) ui.Model {
	t.Helper()
	root := loadFixture(t, fixture)
	rep := summary.Build(root)
	m := ui.New(root, rep)
	// Give it a size so View() doesn't fall back to defaults, matching
	// what a real terminal session would send on startup.
	return update(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
}

func TestModel_NavigateAndFilter(t *testing.T) {
	m := newTestModel(t, "update.json")

	// update.json's tree: module.eks_sp_foundation (row 0, open by
	// default) -> module.elastic_agent (row 1, collapsed by default
	// since it's nested two levels deep) -> kubernetes_role_binding.
	m = update(t, m, key('j'))                       // -> module.elastic_agent header
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // expand it
	m = update(t, m, key('j'))                       // -> resource row
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // enter detail focus

	before := m.View()
	if !strings.Contains(before, `app.kubernetes.io/version`) {
		t.Fatalf("expected the version diff to be visible before filtering:\n%s", before)
	}

	// Filter it out globally.
	m = update(t, m, key('f'))
	after := m.View()
	if strings.Contains(after, `9.2.2" -> "9.2.4"`) {
		t.Fatalf("expected the diff line to be hidden after filtering:\n%s", after)
	}
	if !strings.Contains(after, "1 filter(s) active") {
		t.Fatalf("expected the status bar to report 1 active filter:\n%s", after)
	}
	if !strings.Contains(after, "1 diff(s) hidden") {
		t.Fatalf("expected the status bar to report 1 hidden diff:\n%s", after)
	}
}

func TestModel_CollapseExpand(t *testing.T) {
	m := newTestModel(t, "create.json")

	// create.json: 2 root resources, then the module.child header
	// (open by default — top-level module), then its 1 resource.
	if !strings.Contains(m.View(), "null_resource.labeled") {
		t.Fatalf("expected module.child's resource visible by default:\n%s", m.View())
	}

	// Move to the module.child header (index 2) and collapse it.
	m = update(t, m, key('j'))
	m = update(t, m, key('j'))
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	collapsedView := m.View()
	if strings.Contains(collapsedView, "null_resource.labeled") {
		t.Fatalf("expected module.child's resource hidden once collapsed:\n%s", collapsedView)
	}
	if !strings.Contains(collapsedView, "module.child") {
		t.Fatalf("expected the module.child header to remain visible when collapsed:\n%s", collapsedView)
	}

	// Toggle it back open.
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(m.View(), "null_resource.labeled") {
		t.Fatalf("expected module.child's resource visible again after re-expanding:\n%s", m.View())
	}
}

func TestModel_FilterPanelRemove(t *testing.T) {
	m := newTestModel(t, "update.json")

	// Drill into the resource's detail pane and add a filter.
	m = update(t, m, key('j'))
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = update(t, m, key('j'))
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = update(t, m, key('f'))

	// Open the filters panel; it should take focus and show the rule.
	m = update(t, m, key('p'))
	view := m.View()
	if !strings.Contains(view, "Active filters") {
		t.Fatalf("expected the filters panel to be visible:\n%s", view)
	}
	if !strings.Contains(view, `app.kubernetes.io/version`) {
		t.Fatalf("expected the active rule to be listed:\n%s", view)
	}

	// Remove it.
	m = update(t, m, key('d'))
	view = m.View()
	if !strings.Contains(view, "none") {
		t.Fatalf("expected no active filters after removal:\n%s", view)
	}
}

func TestModel_ClearFilters(t *testing.T) {
	m := newTestModel(t, "update.json")
	m = update(t, m, key('j'))
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = update(t, m, key('j'))
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = update(t, m, key('f'))

	if !strings.Contains(m.View(), "1 filter(s) active") {
		t.Fatal("expected a filter to be active before clearing")
	}

	m = update(t, m, tea.KeyMsg{Type: tea.KeyCtrlR})
	if !strings.Contains(m.View(), "0 filter(s) active") {
		t.Fatalf("expected ctrl+r to clear all filters:\n%s", m.View())
	}
}

func TestModel_Quit(t *testing.T) {
	m := newTestModel(t, "create.json")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected q to issue a command (tea.Quit)")
	}
	got, ok := next.(ui.Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", next)
	}
	if got.View() != "" {
		t.Error("expected an empty view once quitting")
	}
}

func TestModel_HelpOverlay(t *testing.T) {
	m := newTestModel(t, "create.json")

	m = update(t, m, key('?'))
	view := m.View()
	if !strings.Contains(view, "tfp — keybindings") {
		t.Fatalf("expected the help overlay to be shown:\n%s", view)
	}

	// Other keys are inert while help is open.
	m = update(t, m, key('j'))
	if !strings.Contains(m.View(), "tfp — keybindings") {
		t.Fatal("expected an unrelated key to leave the help overlay open")
	}

	m = update(t, m, key('?'))
	if strings.Contains(m.View(), "tfp — keybindings") {
		t.Fatal("expected ? to close the help overlay")
	}
}

func TestModel_GotoTopBottom(t *testing.T) {
	m := newTestModel(t, "create.json")

	m = update(t, m, key('G'))
	atBottom := m.View()
	if !strings.Contains(atBottom, "module.child") {
		t.Fatalf("expected the cursor at the last row (module.child):\n%s", atBottom)
	}

	m = update(t, m, key('g'))
	// After jumping to top, the first root resource's detail should be
	// reachable — a cheap way to confirm the cursor actually moved back
	// without inspecting unexported state.
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(m.View(), "will be created") {
		t.Fatalf("expected the top row's resource detail after g:\n%s", m.View())
	}
}

func TestModel_ExpandShowsUnchangedAttributes(t *testing.T) {
	m := newTestModel(t, "update.json")

	m = update(t, m, key('j'))
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // expand module.elastic_agent
	m = update(t, m, key('j'))                       // -> resource row
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // enter detail focus

	before := m.View()
	if strings.Contains(before, `metadata.name`) {
		t.Fatalf("did not expect an unchanged attribute before expanding:\n%s", before)
	}

	m = update(t, m, key('s'))
	after := m.View()
	if !strings.Contains(after, `metadata.name = "quoter-elastic-agent"`) {
		t.Fatalf("expected an unchanged attribute after pressing s:\n%s", after)
	}
	if !strings.Contains(after, `app.kubernetes.io/version`) {
		t.Fatalf("expected the changed attribute to still show its diff:\n%s", after)
	}
	if !strings.Contains(after, "(s to collapse)") {
		t.Fatalf("expected a mode indicator in the header:\n%s", after)
	}

	// Toggle back off.
	m = update(t, m, key('s'))
	if strings.Contains(m.View(), `metadata.name`) {
		t.Fatal("expected unchanged attributes to disappear again after collapsing")
	}
}

func TestModel_ExpandPersistsAcrossResources(t *testing.T) {
	m := newTestModel(t, "create.json")
	m = update(t, m, key('s')) // toggle on while a root resource is selected (row 0)

	// Rows sort alphabetically: row 0 is random_password.this, which
	// has "keepers" unchanged (null before and after) — only visible in
	// full mode.
	if !strings.Contains(m.View(), "keepers = null") {
		t.Fatalf("expected full mode to already apply to the first resource:\n%s", m.View())
	}

	// Move to random_pet.this (row 1); expand mode should still be on.
	// It has "prefix" unchanged (also null before and after).
	m = update(t, m, key('j'))
	if !strings.Contains(m.View(), "prefix = null") {
		t.Fatalf("expected expand mode to persist across resources:\n%s", m.View())
	}
}

func TestModel_ExpandUnchangedLineIgnoresFilter(t *testing.T) {
	m := newTestModel(t, "update.json")
	m = update(t, m, key('j'))
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = update(t, m, key('j'))
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = update(t, m, key('s')) // expand

	// Cursor starts at 0, which in full mode is the first attribute in
	// path order — "id", unchanged. Pressing f on it must not create a
	// filter rule (nothing to hide).
	m = update(t, m, key('f'))
	if strings.Contains(m.View(), "1 filter(s) active") {
		t.Fatal("expected f on an unchanged line to be a no-op")
	}
}

func TestModel_ExpandScrollsWithinDetailPane(t *testing.T) {
	root := loadFixture(t, "create.json")
	rep := summary.Build(root)
	m := ui.New(root, rep)
	// A short window: random_password.this has 13 attributes, more than
	// fit in a height-8 pane once the header/footer lines are reserved.
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 8})
	m = update(t, m, key('s'))                       // expand, cursor still on row 0 (random_password.this)
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // enter detail focus

	top := m.View()
	if !strings.Contains(top, "bcrypt_hash") {
		t.Fatalf("expected the first attribute visible at the top:\n%s", top)
	}

	// Walk the cursor down past the visible window.
	for range 10 {
		m = update(t, m, key('j'))
	}
	scrolled := m.View()
	if strings.Contains(scrolled, "bcrypt_hash") {
		t.Errorf("expected the top attribute to have scrolled out of view:\n%s", scrolled)
	}
	if !strings.Contains(scrolled, "override_special") {
		t.Errorf("expected attributes further down the list to be visible after scrolling:\n%s", scrolled)
	}
}
