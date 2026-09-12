// Package render turns a planmodel.Module tree (plus its summary.Report)
// into static output: plain text for terminals and logs, or JSON for
// machine consumption. It is the non-interactive counterpart to
// internal/ui — both consume the same planmodel/filter/summary layers,
// so `tfp show plan.json --output text` and the TUI never disagree
// about what the plan contains.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/gusandrioli/tfp/internal/filter"
	"github.com/gusandrioli/tfp/internal/planmodel"
	"github.com/gusandrioli/tfp/internal/summary"
)

// Text writes a terraform-plan-style textual rendering of root to w,
// followed by a summary line. filters may be nil for an unfiltered view.
//
// Filtering only ever hides diff lines from the rendered output — rep
// (built from the unfiltered tree by internal/summary) is never touched,
// so the summary line's counts always reflect the real plan.
func Text(w io.Writer, root *planmodel.Module, rep summary.Report, filters *filter.Set) error {
	if err := writeModule(w, root, filters); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "\nPlan: %d to add, %d to change, %d to destroy, %d to replace, %d to forget.\n",
		rep.Total.Create, rep.Total.Update, rep.Total.Delete, rep.Total.Replace, rep.Total.Forget)
	return err
}

func writeModule(w io.Writer, m *planmodel.Module, filters *filter.Set) error {
	resources := make([]*planmodel.Resource, len(m.Resources))
	copy(resources, m.Resources)
	sort.Slice(resources, func(i, j int) bool { return resources[i].Address < resources[j].Address })

	for _, r := range resources {
		if err := writeResource(w, r, filters); err != nil {
			return err
		}
	}

	children := make([]*planmodel.Module, len(m.Children))
	copy(children, m.Children)
	sort.Slice(children, func(i, j int) bool { return children[i].Address() < children[j].Address() })

	for _, c := range children {
		if err := writeModule(w, c, filters); err != nil {
			return err
		}
	}
	return nil
}

func writeResource(w io.Writer, r *planmodel.Resource, filters *filter.Set) error {
	if _, err := fmt.Fprintf(w, "\n  # %s %s\n", r.Address, ActionPhrase(r.Kind)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %s resource %q %q {\n", r.Kind.Symbol(), r.Type, r.Name); err != nil {
		return err
	}
	hidden := 0
	for _, d := range r.Diffs {
		if filters.Hides(r.Type, d) {
			hidden++
			continue
		}
		if err := writeDiff(w, d); err != nil {
			return err
		}
	}
	if hidden > 0 {
		if _, err := fmt.Fprintf(w, "      # %d change(s) filtered\n", hidden); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, "    }")
	return err
}

func writeDiff(w io.Writer, d planmodel.AttributeDiff) error {
	_, err := fmt.Fprintf(w, "      %s\n", FormatDiff(d))
	return err
}

// FormatDiff renders a single attribute diff the way terraform plan text
// does, e.g. `~ metadata.labels["app.kubernetes.io/version"] = "9.2.2" -> "9.2.4"`,
// with a trailing `# forces replacement` when this attribute is why the
// resource can't just be updated in place. Exported so internal/ui can
// render the same line in the detail pane without duplicating the
// unknown/sensitive-value/forces-replacement handling.
func FormatDiff(d planmodel.AttributeDiff) string {
	symbol, rhs := diffSymbolAndRHS(d)
	line := symbol + " " + d.Path.String() + " = " + rhs
	if d.ForcesReplacement {
		line += " # forces replacement"
	}
	return line
}

// FormatAttributeLine renders one line of a resource's full attribute
// dump (see planmodel.Resource.FullAttributes): a changed leaf renders
// exactly like FormatDiff; an unchanged leaf renders plain — its
// current value, no change symbol — indented to line up with the
// changed lines around it.
func FormatAttributeLine(d planmodel.AttributeDiff) string {
	if d.Changed {
		return FormatDiff(d)
	}
	value := formatValue(d.After)
	if d.Sensitive {
		value = "(sensitive value)"
	}
	return "  " + d.Path.String() + " = " + value
}

func diffSymbolAndRHS(d planmodel.AttributeDiff) (symbol, rhs string) {
	before, after := formatValue(d.Before), formatValue(d.After)
	if d.Sensitive {
		before, after = "(sensitive value)", "(sensitive value)"
	}

	if d.Unknown {
		// After is untrustworthy for unknown leaves (often literally
		// absent from the plan JSON) — never treat this as a removal
		// just because After is nil.
		after = "(known after apply)"
		if d.Before == nil {
			return "+", after
		}
		return "~", before + " -> " + after
	}

	switch {
	case d.Before == nil && d.After != nil:
		return "+", after
	case d.Before != nil && d.After == nil:
		return "-", before
	default:
		return "~", before + " -> " + after
	}
}

// ActionPhrase renders the human-readable phrase terraform plan text
// uses for a change kind, e.g. "will be updated in-place".
func ActionPhrase(k planmodel.ChangeKind) string {
	switch k {
	case planmodel.ChangeCreate:
		return "will be created"
	case planmodel.ChangeUpdate:
		return "will be updated in-place"
	case planmodel.ChangeDelete:
		return "will be destroyed"
	case planmodel.ChangeReplace:
		return "must be replaced"
	case planmodel.ChangeForget:
		return "will be removed from state"
	default:
		return "has an unrecognized change"
	}
}

func formatValue(v any) string {
	switch val := v.(type) {
	case nil:
		return "null"
	case string:
		return fmt.Sprintf("%q", val)
	case bool, float64:
		return fmt.Sprintf("%v", val)
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}
