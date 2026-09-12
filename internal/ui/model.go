package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gusandrioli/tfp/internal/filter"
	"github.com/gusandrioli/tfp/internal/planmodel"
	"github.com/gusandrioli/tfp/internal/render"
	"github.com/gusandrioli/tfp/internal/summary"
)

type focus int

const (
	focusTree focus = iota
	focusDetail
	focusFilters
)

const (
	defaultWidth    = 100
	defaultHeight   = 30
	statusBarHeight = 1
	minTreeWidth    = 20
)

// Model is tfp's top-level Bubble Tea model. It wraps the already-parsed
// plan (root, rep — both computed once, unfiltered) with everything that
// is purely a view concern: collapse state, cursor position, focus, and
// the active filter.Set.
type Model struct {
	root *planmodel.Module
	rep  summary.Report

	filters   filter.Set
	collapsed map[string]bool

	rows   []Row
	cursor int

	focus        focus
	detailCursor int

	showFilters  bool
	filterCursor int

	showHelp bool

	// showFullResource toggles the detail pane between diffs-only (the
	// default) and every attribute, changed or not — "expand" mode, for
	// reviewing a change with full context. A global mode rather than
	// per-resource, so it stays on as you move between resources.
	showFullResource bool

	width, height int
	quitting      bool
}

// New builds the initial model for root/rep, with default collapse state
// (top-level modules open, deeper ones collapsed — see rows.go) and no
// active filters.
func New(root *planmodel.Module, rep summary.Report) Model {
	m := Model{
		root:      root,
		rep:       rep,
		collapsed: DefaultCollapsed(root),
	}
	m.rebuildRows()
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

func (m *Model) rebuildRows() {
	m.rows = Flatten(m.root, m.collapsed, m.rep)
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m Model) selectedRow() *Row {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	return &m.rows[m.cursor]
}

// currentDetailLines returns the lines the detail pane shows for r,
// filtered by the active rules — every attribute (changed or not) in
// "full resource" mode, changed leaves only otherwise — plus the
// pre-filter count so callers can report how many were hidden.
func (m Model) currentDetailLines(r *planmodel.Resource) (lines []planmodel.AttributeDiff, total int) {
	all := r.Diffs
	if m.showFullResource {
		all = r.FullAttributes()
	}
	lines = make([]planmodel.AttributeDiff, 0, len(all))
	for _, d := range all {
		if !m.filters.Hides(r.Type, d) {
			lines = append(lines, d)
		}
	}
	return lines, len(all)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, defaultKeyMap.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, defaultKeyMap.Help):
		m.showHelp = !m.showHelp
		return m, nil
	}
	if m.showHelp {
		// Any other key is inert while the help overlay is open.
		return m, nil
	}

	switch {
	case key.Matches(msg, defaultKeyMap.SwitchFocus):
		if m.focus == focusTree {
			m.focus = focusDetail
		} else {
			m.focus = focusTree
		}
		return m, nil
	case key.Matches(msg, defaultKeyMap.TogglePanel):
		m.showFilters = !m.showFilters
		switch {
		case m.showFilters:
			m.focus = focusFilters
			m.filterCursor = 0
		case m.focus == focusFilters:
			m.focus = focusTree
		}
		return m, nil
	case key.Matches(msg, defaultKeyMap.ClearFilters):
		m.filters.Clear()
		return m, nil
	case key.Matches(msg, defaultKeyMap.Expand):
		m.showFullResource = !m.showFullResource
		m.detailCursor = 0
		return m, nil
	}

	switch m.focus {
	case focusTree:
		return m.handleTreeKey(msg)
	case focusDetail:
		return m.handleDetailKey(msg)
	case focusFilters:
		return m.handleFiltersKey(msg)
	default:
		return m, nil
	}
}

func (m Model) handleTreeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, defaultKeyMap.Up):
		if m.cursor > 0 {
			m.cursor--
			m.detailCursor = 0
		}
	case key.Matches(msg, defaultKeyMap.Down):
		if m.cursor < len(m.rows)-1 {
			m.cursor++
			m.detailCursor = 0
		}
	case key.Matches(msg, defaultKeyMap.GotoTop):
		m.cursor = 0
		m.detailCursor = 0
	case key.Matches(msg, defaultKeyMap.GotoBottom):
		m.cursor = max(len(m.rows)-1, 0)
		m.detailCursor = 0
	case key.Matches(msg, defaultKeyMap.Toggle):
		row := m.selectedRow()
		if row == nil {
			break
		}
		switch row.Kind {
		case RowModule:
			addr := row.Module.Address()
			m.collapsed[addr] = !m.collapsed[addr]
			m.rebuildRows()
		case RowResource:
			m.focus = focusDetail
			m.detailCursor = 0
		}
	}
	return m, nil
}

func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	row := m.selectedRow()
	if row == nil || row.Kind != RowResource {
		return m, nil
	}
	lines, _ := m.currentDetailLines(row.Resource)

	switch {
	case key.Matches(msg, defaultKeyMap.Up):
		if m.detailCursor > 0 {
			m.detailCursor--
		}
	case key.Matches(msg, defaultKeyMap.Down):
		if m.detailCursor < len(lines)-1 {
			m.detailCursor++
		}
	case key.Matches(msg, defaultKeyMap.FilterGlobal):
		m.addFilter(row.Resource, lines, filter.Rule{Scope: filter.ScopeGlobal})
	case key.Matches(msg, defaultKeyMap.FilterByType):
		m.addFilter(row.Resource, lines, filter.Rule{Scope: filter.ScopeResourceType, ResourceType: row.Resource.Type})
	}
	return m, nil
}

// addFilter adds a rule (with PathSuffix filled in from the line under
// the cursor) and re-clamps detailCursor, since applying it will usually
// shrink the visible line list out from under the current position. A
// no-op on an unchanged line (only reachable in "full resource" mode) —
// there's nothing there to filter.
func (m *Model) addFilter(r *planmodel.Resource, visible []planmodel.AttributeDiff, rule filter.Rule) {
	if m.detailCursor >= len(visible) || !visible[m.detailCursor].Changed {
		return
	}
	rule.PathSuffix = visible[m.detailCursor].Path
	m.filters.Add(rule)

	remaining, _ := m.currentDetailLines(r)
	if m.detailCursor >= len(remaining) {
		m.detailCursor = max(0, len(remaining)-1)
	}
}

func (m Model) handleFiltersKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rules := m.filters.Rules()
	switch {
	case key.Matches(msg, defaultKeyMap.Up):
		if m.filterCursor > 0 {
			m.filterCursor--
		}
	case key.Matches(msg, defaultKeyMap.Down):
		if m.filterCursor < len(rules)-1 {
			m.filterCursor++
		}
	case key.Matches(msg, defaultKeyMap.Remove):
		if m.filterCursor < len(rules) {
			m.filters.Remove(m.filterCursor)
			if remaining := len(m.filters.Rules()); m.filterCursor >= remaining {
				m.filterCursor = max(0, remaining-1)
			}
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.showHelp {
		return m.renderHelp()
	}

	top := lipgloss.JoinHorizontal(lipgloss.Top, m.renderTree(), m.renderDetail())
	sections := []string{top}
	if m.showFilters {
		sections = append(sections, m.renderFilters())
	}
	sections = append(sections, m.renderStatusBar())
	return strings.Join(sections, "\n")
}

func (m Model) effectiveWidth() int {
	if m.width > 0 {
		return m.width
	}
	return defaultWidth
}

func (m Model) effectiveHeight() int {
	if m.height > 0 {
		return m.height
	}
	return defaultHeight
}

func (m Model) treeWidth() int {
	tw := m.effectiveWidth() * 2 / 5
	return max(tw, minTreeWidth)
}

func (m Model) detailWidth() int {
	return max(m.effectiveWidth()-m.treeWidth(), minTreeWidth)
}

func (m Model) paneHeight() int {
	h := m.effectiveHeight() - statusBarHeight - 2 // border top+bottom
	if m.showFilters {
		h -= len(m.filters.Rules()) + 2
	}
	return max(h, 3)
}

// paneContentWidth returns how much of a pane's width is left for text
// once its border (2 cols) and horizontal padding (2 cols) are spent.
func paneContentWidth(paneWidth int) int {
	return max(paneWidth-4, 1)
}

func (m Model) renderTree() string {
	height := m.paneHeight()
	contentWidth := paneContentWidth(m.treeWidth())
	start, end := visibleWindow(len(m.rows), height, m.cursor)

	var b strings.Builder
	for i := start; i < end; i++ {
		line := renderRow(m.rows[i], contentWidth)
		if i == m.cursor {
			line = styleSelected.Render(line)
		}
		b.WriteString(line)
		if i != end-1 {
			b.WriteByte('\n')
		}
	}

	style := stylePane
	if m.focus == focusTree {
		style = stylePaneFocused
	}
	return style.Width(m.treeWidth()).Height(height).Render(b.String())
}

func renderRow(row Row, width int) string {
	indent := strings.Repeat("  ", row.Depth)
	if row.Kind == RowModule {
		arrow := "▾"
		if row.Collapsed {
			arrow = "▸"
		}
		counts := formatCounts(row.Counts)
		// Its own name is enough — ancestor modules are already implied
		// by indentation and the header rows above it, so there's no
		// need to repeat the full dotted address here.
		name := "module." + row.Module.Name()
		nameWidth := width - len(indent) - len(arrow) - 1 - len(counts) - 2
		label := fmt.Sprintf("%s%s %s  %s", indent, arrow, truncate(name, nameWidth), counts)
		return styleModule.Render(strings.TrimRight(label, " "))
	}
	// Likewise, type.name is enough in the tree — the full address is
	// available in the detail pane once selected.
	r := row.Resource
	prefix := indent + r.Kind.Symbol() + " "
	label := prefix + truncate(r.Type+"."+r.Name, width-len(prefix))
	return kindStyle(r.Kind).Render(label)
}

// truncate shortens s to at most width runes, replacing the tail with an
// ellipsis. Long module/resource addresses (the norm in nested-module
// plans) would otherwise wrap inside the bordered pane and break the
// one-row-per-line assumption cursor highlighting and scrolling rely on.
func truncate(s string, width int) string {
	if width <= 1 {
		return s
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width-1]) + "…"
}

func formatCounts(c summary.Counts) string {
	var parts []string
	if c.Create > 0 {
		parts = append(parts, fmt.Sprintf("+%d", c.Create))
	}
	if c.Update > 0 {
		parts = append(parts, fmt.Sprintf("~%d", c.Update))
	}
	if c.Delete > 0 {
		parts = append(parts, fmt.Sprintf("-%d", c.Delete))
	}
	if c.Replace > 0 {
		parts = append(parts, fmt.Sprintf("±%d", c.Replace))
	}
	if c.Forget > 0 {
		parts = append(parts, fmt.Sprintf(".%d", c.Forget))
	}
	return strings.Join(parts, " ")
}

func (m Model) renderDetail() string {
	height := m.paneHeight()
	row := m.selectedRow()

	var b strings.Builder
	switch {
	case row == nil:
		b.WriteString(styleDimmed.Render("No resources to show."))
	case row.Kind != RowResource:
		b.WriteString(styleDimmed.Render("Select a resource to see its changes."))
	default:
		r := row.Resource
		header := r.Address + " " + render.ActionPhrase(r.Kind)
		if m.showFullResource {
			header += styleDimmed.Render(" — full resource (s to collapse)")
		}
		fmt.Fprintf(&b, "%s\n\n", header)

		lines, total := m.currentDetailLines(r)
		hidden := total - len(lines)
		reserved := 2 // the header line + the blank line after it
		if hidden > 0 {
			reserved++ // the "N change(s) filtered" footer line
		}
		linesHeight := max(height-reserved, 1)
		start, end := visibleWindow(len(lines), linesHeight, m.detailCursor)

		for i := start; i < end; i++ {
			line := render.FormatAttributeLine(lines[i])
			if m.focus == focusDetail && i == m.detailCursor {
				line = styleSelected.Render(line)
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}
		if hidden > 0 {
			fmt.Fprintf(&b, "%s\n", styleDimmed.Render(fmt.Sprintf("# %d change(s) filtered", hidden)))
		}
	}

	style := stylePane
	if m.focus == focusDetail {
		style = stylePaneFocused
	}
	return style.Width(m.detailWidth()).Height(height).Render(strings.TrimRight(b.String(), "\n"))
}

func (m Model) renderFilters() string {
	rules := m.filters.Rules()
	var b strings.Builder
	b.WriteString(styleTitle.Render("Active filters"))
	b.WriteByte('\n')
	if len(rules) == 0 {
		b.WriteString(styleDimmed.Render("(none — press f on a diff to add one)"))
	}
	for i, r := range rules {
		line := formatRule(r)
		if m.focus == focusFilters && i == m.filterCursor {
			line = styleSelected.Render(line)
		}
		b.WriteString(line)
		if i != len(rules)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func formatRule(r filter.Rule) string {
	scope := "all resources"
	if r.Scope == filter.ScopeResourceType {
		scope = r.ResourceType
	}
	return fmt.Sprintf("%s (%s)", r.PathSuffix.String(), scope)
}

func (m Model) renderStatusBar() string {
	left := fmt.Sprintf("%d filter(s) active · %d diff(s) hidden", len(m.filters.Rules()), m.hiddenCount())
	right := "j/k move · g/G top/bottom · tab switch · enter select/toggle · f/F filter · s expand · p filters · ? help · q quit"
	return styleStatusBar.Render(left + "   " + right)
}

// helpBindings lists every binding worth documenting, in display order —
// the single source of truth for the "?" overlay, so it can never list a
// binding that Update doesn't actually handle (or vice versa).
func helpBindings() []key.Binding {
	k := defaultKeyMap
	return []key.Binding{
		k.Up, k.Down, k.GotoTop, k.GotoBottom, k.SwitchFocus, k.Toggle,
		k.Expand, k.FilterGlobal, k.FilterByType, k.TogglePanel, k.Remove, k.ClearFilters,
		k.Help, k.Quit,
	}
}

func (m Model) renderHelp() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("tfp — keybindings"))
	b.WriteString("\n\n")
	for _, binding := range helpBindings() {
		h := binding.Help()
		fmt.Fprintf(&b, "  %-10s %s\n", h.Key, h.Desc)
	}
	b.WriteString("\npress ? to close")

	style := stylePane
	return style.Width(m.effectiveWidth() - 4).Render(strings.TrimRight(b.String(), "\n"))
}

func (m Model) hiddenCount() int {
	total := 0
	var walk func(mod *planmodel.Module)
	walk = func(mod *planmodel.Module) {
		for _, r := range mod.Resources {
			for _, d := range r.Diffs {
				if m.filters.Hides(r.Type, d) {
					total++
				}
			}
		}
		for _, c := range mod.Children {
			walk(c)
		}
	}
	walk(m.root)
	return total
}

// visibleWindow computes the [start, end) slice of rows to render so
// that cursor stays in view within a pane of the given height.
func visibleWindow(total, height, cursor int) (start, end int) {
	if height <= 0 || total <= height {
		return 0, total
	}
	start = max(cursor-height/2, 0)
	end = start + height
	if end > total {
		end = total
		start = max(end-height, 0)
	}
	return start, end
}
