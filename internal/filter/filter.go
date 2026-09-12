// Package filter hides noisy, repeated attribute-level diffs from view
// without touching the underlying plan data. A Set never mutates a
// planmodel.Module tree — it only answers "should this diff be shown?" —
// so summary.Report, computed once from the unfiltered tree, always
// stays truthful regardless of which filters are active.
package filter

import "github.com/gusandrioli/tfp/internal/planmodel"

// Scope controls how broadly a Rule applies.
type Scope int

const (
	// ScopeUnknown is the zero value: an invalid/unset scope.
	ScopeUnknown Scope = iota
	// ScopeGlobal hides a matching path across every resource, of any type.
	ScopeGlobal
	// ScopeResourceType hides a matching path only for resources whose
	// Type equals Rule.ResourceType.
	ScopeResourceType
)

// Rule hides any AttributeDiff whose Path ends with PathSuffix (matched
// via planmodel.AttributePath.HasSuffix), optionally narrowed to one
// resource type.
//
// Matching by suffix rather than full-path equality is deliberate: the
// motivating case is the same attribute (e.g.
// metadata.labels["app.kubernetes.io/version"]) changing identically
// across many resources with different full addresses — a suffix match
// is what actually collapses that noise. See docs/DATA_MODEL.md.
type Rule struct {
	PathSuffix   planmodel.AttributePath
	Scope        Scope
	ResourceType string
}

// Matches reports whether r hides the given diff on a resource of type
// resourceType.
func (r Rule) Matches(resourceType string, d planmodel.AttributeDiff) bool {
	if r.Scope == ScopeResourceType && resourceType != r.ResourceType {
		return false
	}
	return d.Path.HasSuffix(r.PathSuffix)
}

// Set holds the active filter rules for the current session.
type Set struct {
	rules []Rule
}

// Add appends r to the set.
func (s *Set) Add(r Rule) {
	s.rules = append(s.rules, r)
}

// Remove deletes the rule at index i. It panics if i is out of range,
// matching slice semantics — callers (the filters panel) always index
// against a slice obtained from Rules.
func (s *Set) Remove(i int) {
	s.rules = append(s.rules[:i], s.rules[i+1:]...)
}

// Clear removes every active rule.
func (s *Set) Clear() {
	s.rules = nil
}

// Rules returns the active rules, in the order they were added. The
// returned slice is a copy; mutating it does not affect s.
func (s *Set) Rules() []Rule {
	out := make([]Rule, len(s.rules))
	copy(out, s.rules)
	return out
}

// Hides reports whether any active rule hides d on a resource of type
// resourceType. A nil *Set hides nothing.
func (s *Set) Hides(resourceType string, d planmodel.AttributeDiff) bool {
	if s == nil {
		return false
	}
	for _, r := range s.rules {
		if r.Matches(resourceType, d) {
			return true
		}
	}
	return false
}
