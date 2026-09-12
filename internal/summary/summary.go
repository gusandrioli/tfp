// Package summary computes +/- change counts from a planmodel.Module
// tree: a grand total, one entry per module, and one per resource. It is
// a pure function of the tree and has no dependency on internal/filter —
// the numbers it reports must never drift based on what a filtered view
// happens to be hiding.
package summary

import "github.com/gusandrioli/tfp/internal/planmodel"

// Counts tallies resources by the kind of change they represent.
type Counts struct {
	Create  int
	Update  int
	Delete  int
	Replace int
	Forget  int
}

// Add tallies one resource's Kind into c.
func (c *Counts) Add(kind planmodel.ChangeKind) {
	switch kind {
	case planmodel.ChangeCreate:
		c.Create++
	case planmodel.ChangeUpdate:
		c.Update++
	case planmodel.ChangeDelete:
		c.Delete++
	case planmodel.ChangeReplace:
		c.Replace++
	case planmodel.ChangeForget:
		c.Forget++
	}
}

// Merge folds other into c, returning the sum.
func (c Counts) Merge(other Counts) Counts {
	return Counts{
		Create:  c.Create + other.Create,
		Update:  c.Update + other.Update,
		Delete:  c.Delete + other.Delete,
		Replace: c.Replace + other.Replace,
		Forget:  c.Forget + other.Forget,
	}
}

// Total is the number of resources tallied across all kinds.
func (c Counts) Total() int {
	return c.Create + c.Update + c.Delete + c.Replace + c.Forget
}

// ResourceEntry is one resource's contribution to the report: which kind
// of change it is, and how many attribute-level diffs it carries (the
// "+N ~M -K"-style figure from docs/PLAN.md, at the single-resource
// granularity).
type ResourceEntry struct {
	Kind      planmodel.ChangeKind
	DiffCount int
}

// Report is the aggregated +/- view of an entire plan.
type Report struct {
	Total      Counts
	ByModule   map[string]Counts        // keyed by dotted module address ("" for root)
	ByResource map[string]ResourceEntry // keyed by resource address
}

// Build walks the module tree once and computes the full report.
func Build(root *planmodel.Module) Report {
	r := Report{
		ByModule:   make(map[string]Counts),
		ByResource: make(map[string]ResourceEntry),
	}
	r.Total = buildModule(root, &r)
	return r
}

func buildModule(m *planmodel.Module, r *Report) Counts {
	var c Counts
	for _, res := range m.Resources {
		c.Add(res.Kind)
		r.ByResource[res.Address] = ResourceEntry{
			Kind:      res.Kind,
			DiffCount: len(res.Diffs),
		}
	}
	for _, child := range m.Children {
		c = c.Merge(buildModule(child, r))
	}
	r.ByModule[m.Address()] = c
	return c
}
