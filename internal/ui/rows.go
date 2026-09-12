// Package ui is tfp's interactive Bubble Tea program: a tree pane
// (modules/resources, collapsible), a detail pane (attribute diffs for
// the selected resource), a summary/status bar, and a filter panel — all
// built on internal/planmodel, internal/filter, and internal/summary.
package ui

import (
	"sort"

	"github.com/gusandrioli/tfp/internal/planmodel"
	"github.com/gusandrioli/tfp/internal/summary"
)

// RowKind distinguishes the two kinds of line the tree pane can show.
type RowKind int

const (
	// RowModule is a collapsible module header row.
	RowModule RowKind = iota
	// RowResource is a leaf resource row.
	RowResource
)

// Row is one visible line in the tree pane. Exactly one of Module or
// Resource is set, matching Kind.
type Row struct {
	Kind      RowKind
	Depth     int
	Module    *planmodel.Module
	Resource  *planmodel.Resource
	Counts    summary.Counts // rollup badge, set when Kind == RowModule
	Collapsed bool           // set when Kind == RowModule
}

// Flatten builds the visible row list from root, respecting which
// module addresses are collapsed. The root module contributes its own
// resources and children directly, at depth 0 — there is no header row
// for it (nothing to collapse: it has no address of its own).
func Flatten(root *planmodel.Module, collapsed map[string]bool, rep summary.Report) []Row {
	var rows []Row
	appendResources(&rows, root, 0)
	for _, child := range sortedChildren(root) {
		appendModule(&rows, child, 0, collapsed, rep)
	}
	return rows
}

func appendModule(rows *[]Row, m *planmodel.Module, depth int, collapsed map[string]bool, rep summary.Report) {
	isCollapsed := collapsed[m.Address()]
	*rows = append(*rows, Row{Kind: RowModule, Depth: depth, Module: m, Counts: rep.ByModule[m.Address()], Collapsed: isCollapsed})
	if isCollapsed {
		return
	}
	appendResources(rows, m, depth+1)
	for _, child := range sortedChildren(m) {
		appendModule(rows, child, depth+1, collapsed, rep)
	}
}

func appendResources(rows *[]Row, m *planmodel.Module, depth int) {
	for _, r := range sortedResources(m) {
		*rows = append(*rows, Row{Kind: RowResource, Depth: depth, Resource: r})
	}
}

func sortedResources(m *planmodel.Module) []*planmodel.Resource {
	out := make([]*planmodel.Resource, len(m.Resources))
	copy(out, m.Resources)
	sort.Slice(out, func(i, j int) bool { return out[i].Address < out[j].Address })
	return out
}

func sortedChildren(m *planmodel.Module) []*planmodel.Module {
	out := make([]*planmodel.Module, len(m.Children))
	copy(out, m.Children)
	sort.Slice(out, func(i, j int) bool { return out[i].Address() < out[j].Address() })
	return out
}

// DefaultCollapsed returns the initial collapse state: top-level modules
// open, anything nested deeper collapsed — so a large plan doesn't open
// as a wall of text. See docs/TUI.md#collapse-semantics.
func DefaultCollapsed(root *planmodel.Module) map[string]bool {
	collapsed := make(map[string]bool)
	var walk func(m *planmodel.Module)
	walk = func(m *planmodel.Module) {
		for _, child := range m.Children {
			if len(child.Path) > 1 {
				collapsed[child.Address()] = true
			}
			walk(child)
		}
	}
	walk(root)
	return collapsed
}
