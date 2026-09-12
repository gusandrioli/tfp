package planmodel

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

// Parse decodes a `terraform show -json` payload and builds the module
// tree tfp's UI/renderer walk. No-op resources and data-source reads are
// dropped — they're not "changes" in the sense PLAN.md cares about.
func Parse(data []byte) (*Module, error) {
	var plan tfjson.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("planmodel: decode plan json: %w", err)
	}
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("planmodel: invalid plan json: %w", err)
	}
	return buildTree(plan.ResourceChanges), nil
}

func buildTree(changes []*tfjson.ResourceChange) *Module {
	root := &Module{}
	index := map[string]*Module{"": root}

	for _, rc := range changes {
		if rc.Change == nil {
			continue
		}
		kind := kindFromActions(rc.Change.Actions)
		if kind == ChangeNoOp || kind == ChangeRead {
			continue
		}

		res := &Resource{
			Address: rc.Address,
			Type:    rc.Type,
			Name:    rc.Name,
			Kind:    kind,
			Diffs:   diffValues(rc.Change.Before, rc.Change.After, rc.Change.AfterUnknown, rc.Change.BeforeSensitive, rc.Change.AfterSensitive),
			change:  rc.Change,
		}
		markForcesReplacement(res.Diffs, rc.Change.ReplacePaths)
		res.Sensitive = anySensitive(res.Diffs)

		mod := ensureModule(index, root, modulePathSegments(rc.ModuleAddress))
		mod.Resources = append(mod.Resources, res)
	}
	return root
}

func kindFromActions(a tfjson.Actions) ChangeKind {
	switch {
	case a.NoOp():
		return ChangeNoOp
	case a.Create():
		return ChangeCreate
	case a.Update():
		return ChangeUpdate
	case a.Delete():
		return ChangeDelete
	case a.Replace():
		return ChangeReplace
	case a.Forget():
		return ChangeForget
	case a.Read():
		return ChangeRead
	default:
		return ChangeUnknown
	}
}

// markForcesReplacement flags diffs that fall under one of a replace
// resource's ReplacePaths — the attribute(s) terraform says are why the
// resource must be recreated rather than updated in place. raw is
// tfjson's Change.ReplacePaths: a slice of paths, each itself a slice of
// string (map/object key) or float64 (list/set index) segments.
func markForcesReplacement(diffs []AttributeDiff, raw []any) {
	if len(raw) == 0 {
		return
	}
	paths := convertReplacePaths(raw)
	for i := range diffs {
		if slices.ContainsFunc(paths, diffs[i].Path.HasPrefix) {
			diffs[i].ForcesReplacement = true
		}
	}
}

func convertReplacePaths(raw []any) []AttributePath {
	paths := make([]AttributePath, 0, len(raw))
	for _, item := range raw {
		segs, ok := item.([]any)
		if !ok {
			continue
		}
		path := make(AttributePath, 0, len(segs))
		for _, seg := range segs {
			switch v := seg.(type) {
			case string:
				path = append(path, PathSegment{Key: v})
			case float64:
				idx := int(v)
				path = append(path, PathSegment{Index: &idx})
			}
		}
		paths = append(paths, path)
	}
	return paths
}

func anySensitive(diffs []AttributeDiff) bool {
	for _, d := range diffs {
		if d.Sensitive {
			return true
		}
	}
	return false
}

// modulePathSegments turns a dotted terraform module address, e.g.
// "module.eks_sp_foundation.module.elastic_agent", into path segments
// ["eks_sp_foundation", "elastic_agent"]. Empty for the root module.
func modulePathSegments(moduleAddress string) []string {
	if moduleAddress == "" {
		return nil
	}
	trimmed := strings.TrimPrefix(moduleAddress, "module.")
	return strings.Split(trimmed, ".module.")
}

// ensureModule walks/creates Module nodes along path, memoized in index
// by dotted-path key so siblings sharing a parent reuse the same node.
func ensureModule(index map[string]*Module, root *Module, path []string) *Module {
	parent := root
	key := ""
	for i, seg := range path {
		if i == 0 {
			key = seg
		} else {
			key = key + "." + seg
		}
		if m, ok := index[key]; ok {
			parent = m
			continue
		}
		modPath := make([]string, i+1)
		copy(modPath, path[:i+1])
		m := &Module{Path: modPath}
		index[key] = m
		parent.Children = append(parent.Children, m)
		parent = m
	}
	return parent
}
