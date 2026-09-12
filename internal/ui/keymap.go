package ui

import "github.com/charmbracelet/bubbles/key"

// keyMap is the single source of truth for tfp's keybindings, used both
// to dispatch key.Matches in Update and to render the help line — so
// the two can never drift apart.
type keyMap struct {
	Up           key.Binding
	Down         key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding
	Toggle       key.Binding // collapse/expand a module row, or enter a resource's detail
	Expand       key.Binding // toggle showing a resource's full attribute set, not just its diffs
	SwitchFocus  key.Binding
	FilterGlobal key.Binding
	FilterByType key.Binding
	TogglePanel  key.Binding
	ClearFilters key.Binding
	Remove       key.Binding // remove the selected rule, when the filter panel is focused
	Help         key.Binding
	Quit         key.Binding
}

var defaultKeyMap = keyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "down"),
	),
	GotoTop: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "jump to top"),
	),
	GotoBottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "jump to bottom"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter", "toggle/select"),
	),
	Expand: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "expand resource"),
	),
	SwitchFocus: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch pane"),
	),
	FilterGlobal: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter (global)"),
	),
	FilterByType: key.NewBinding(
		key.WithKeys("F"),
		key.WithHelp("F", "filter (this type)"),
	),
	TogglePanel: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "filters panel"),
	),
	ClearFilters: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "clear filters"),
	),
	Remove: key.NewBinding(
		key.WithKeys("d", "backspace"),
		key.WithHelp("d", "remove rule"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
