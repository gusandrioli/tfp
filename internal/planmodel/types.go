// Package planmodel turns a parsed terraform plan (via
// github.com/hashicorp/terraform-json) into tfp's own domain model: a
// module tree of resources, each carrying only the attribute-level diffs
// that actually changed.
package planmodel

import (
	"strconv"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

// ChangeKind is tfp's normalized view of tfjson's Actions slice.
type ChangeKind int

const (
	// ChangeUnknown is the zero value: an action set tfp doesn't
	// recognize. It must never be treated as "no change".
	ChangeUnknown ChangeKind = iota
	// ChangeNoOp is a resource with no changes.
	ChangeNoOp
	// ChangeCreate is a new resource.
	ChangeCreate
	// ChangeUpdate is an in-place update.
	ChangeUpdate
	// ChangeDelete is a resource being destroyed.
	ChangeDelete
	// ChangeReplace is create+delete or delete+create.
	ChangeReplace
	// ChangeForget is removal from state without destroying.
	ChangeForget
	// ChangeRead is a data source read.
	ChangeRead
)

// String renders the kind the way terraform's own plan output labels it.
func (k ChangeKind) String() string {
	switch k {
	case ChangeNoOp:
		return "no-op"
	case ChangeCreate:
		return "create"
	case ChangeUpdate:
		return "update"
	case ChangeDelete:
		return "delete"
	case ChangeReplace:
		return "replace"
	case ChangeForget:
		return "forget"
	case ChangeRead:
		return "read"
	default:
		return "unknown"
	}
}

// Symbol renders the single-character marker terraform plan text uses.
func (k ChangeKind) Symbol() string {
	switch k {
	case ChangeCreate:
		return "+"
	case ChangeUpdate:
		return "~"
	case ChangeDelete:
		return "-"
	case ChangeReplace:
		return "-/+"
	case ChangeForget:
		return "."
	case ChangeRead:
		return "<="
	default:
		return "?"
	}
}

// Module is one node in the module tree. The root module has Path == nil.
type Module struct {
	Path      []string // e.g. ["eks_sp_foundation", "elastic_agent"]
	Children  []*Module
	Resources []*Resource
}

// Name returns the module's own name segment, or "" for the root module.
func (m *Module) Name() string {
	if len(m.Path) == 0 {
		return ""
	}
	return m.Path[len(m.Path)-1]
}

// Address renders the module's dotted terraform address, e.g.
// "module.eks_sp_foundation.module.elastic_agent". Empty for the root.
func (m *Module) Address() string {
	if len(m.Path) == 0 {
		return ""
	}
	var b strings.Builder
	for i, seg := range m.Path {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString("module.")
		b.WriteString(seg)
	}
	return b.String()
}

// Resource is one ResourceChange, tfp-shaped: only the leaves that
// actually changed are materialized into Diffs.
type Resource struct {
	Address   string // full resource address, as in tfjson
	Type      string
	Name      string
	Kind      ChangeKind
	Diffs     []AttributeDiff
	Sensitive bool // true if any changed leaf is marked sensitive

	// change is retained (not copied — just the same pointer already
	// held by the parsed tfjson.Plan) so FullAttributes can walk the
	// full before/after bodies on demand, without every Resource paying
	// for that walk up front when most are never expanded.
	change *tfjson.Change
}

// FullAttributes returns every leaf attribute of the resource, changed
// or not — the "expand" view's data source. Unlike Diffs, unchanged
// leaves are included too (with Changed = false), so a resource can be
// reviewed in full context rather than just its diff. Computed lazily;
// nil if the resource carries no change data (never happens via Parse,
// but keeps the zero value of Resource safe to call this on).
func (r *Resource) FullAttributes() []AttributeDiff {
	if r.change == nil {
		return nil
	}
	attrs := fullAttributeValues(r.change.Before, r.change.After, r.change.AfterUnknown, r.change.BeforeSensitive, r.change.AfterSensitive)
	markForcesReplacement(attrs, r.change.ReplacePaths)
	return attrs
}

// PathSegment is one step into an attribute tree: either a map/object key
// or a block attribute name (Key set), or a list/set index (Index set).
type PathSegment struct {
	Key   string
	Index *int
}

// AttributePath is a structured path into a resource's attribute tree,
// e.g. metadata.labels["app.kubernetes.io/version"] or ingress.rule[0].host
type AttributePath []PathSegment

// String renders the path the way a terraform user would recognize it:
// dotted block/attribute names (metadata.labels), bracket-quoted map keys
// that aren't plain identifiers (labels["app.kubernetes.io/version"]),
// and bracketed list indices (rule[0]).
func (p AttributePath) String() string {
	var b strings.Builder
	for i, seg := range p {
		switch {
		case seg.Index != nil:
			b.WriteByte('[')
			b.WriteString(strconv.Itoa(*seg.Index))
			b.WriteByte(']')
		case isIdent(seg.Key):
			if i > 0 {
				b.WriteByte('.')
			}
			b.WriteString(seg.Key)
		default:
			b.WriteByte('[')
			b.WriteString(strconv.Quote(seg.Key))
			b.WriteByte(']')
		}
	}
	return b.String()
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}

// HasSuffix reports whether p ends with the given suffix path, matching
// segment-by-segment (map keys must match by Key, indices by Index).
func (p AttributePath) HasSuffix(suffix AttributePath) bool {
	if len(suffix) > len(p) {
		return false
	}
	offset := len(p) - len(suffix)
	for i, seg := range suffix {
		if !segmentsEqual(seg, p[offset+i]) {
			return false
		}
	}
	return true
}

// HasPrefix reports whether p starts with the given prefix path, matching
// segment-by-segment. Used to test whether a diff falls under one of a
// resource's ReplacePaths — those name the attribute that forced
// replacement, which may be an ancestor of (or equal to) any given diff's
// own path (e.g. a diff at triggers["team"] falls under a replace path
// of just ["triggers"]).
func (p AttributePath) HasPrefix(prefix AttributePath) bool {
	if len(prefix) > len(p) {
		return false
	}
	for i, seg := range prefix {
		if !segmentsEqual(seg, p[i]) {
			return false
		}
	}
	return true
}

func segmentsEqual(a, b PathSegment) bool {
	if (a.Index == nil) != (b.Index == nil) {
		return false
	}
	if a.Index != nil {
		return *a.Index == *b.Index
	}
	return a.Key == b.Key
}

// AttributeDiff is one changed leaf value within a resource.
type AttributeDiff struct {
	Path              AttributePath
	Before            any
	After             any
	Unknown           bool // value not known until apply (tfjson AfterUnknown)
	Sensitive         bool // value redacted in display
	ForcesReplacement bool // this attribute is why the resource must be replaced
	// Changed is true for every entry in Resource.Diffs (they exist
	// because they changed). It's false for the unchanged leaves that
	// only appear in Resource.FullAttributes' "expand" view.
	Changed bool
}
