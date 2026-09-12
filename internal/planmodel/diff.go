package planmodel

import (
	"reflect"
	"sort"
)

// leafFunc is called once for every leaf walkTree visits: either a
// genuine scalar/nil leaf, or an entire subtree collapsed because it's
// marked unknown (in which case unknown is true and before/after cover
// the whole subtree, not a single scalar).
type leafFunc func(path AttributePath, before, after any, unknown, sensitive bool)

// diffValues walks before/after (both decoded from JSON, so composed of
// map[string]interface{}, []interface{}, and scalars/nil) in lockstep,
// alongside terraform's parallel afterUnknown/beforeSensitive/afterSensitive
// marker trees, and returns one AttributeDiff per leaf that actually
// changed. Equal subtrees are pruned entirely rather than materialized.
func diffValues(before, after, afterUnknown, beforeSensitive, afterSensitive any) []AttributeDiff {
	var out []AttributeDiff
	walkTree(before, after, afterUnknown, beforeSensitive, afterSensitive, nil, func(path AttributePath, b, a any, unknown, sensitive bool) {
		if !unknown && reflect.DeepEqual(b, a) {
			return // equal and not a forced-unknown leaf: prune
		}
		out = append(out, AttributeDiff{Path: path, Before: b, After: a, Unknown: unknown, Sensitive: sensitive, Changed: true})
	})
	return out
}

// fullAttributeValues is diffValues' counterpart for the "expand" view:
// it walks the same tree but keeps every leaf, changed or not, so the
// full resource can be shown for context — not just what's different.
func fullAttributeValues(before, after, afterUnknown, beforeSensitive, afterSensitive any) []AttributeDiff {
	var out []AttributeDiff
	walkTree(before, after, afterUnknown, beforeSensitive, afterSensitive, nil, func(path AttributePath, b, a any, unknown, sensitive bool) {
		changed := unknown || !reflect.DeepEqual(b, a)
		out = append(out, AttributeDiff{Path: path, Before: b, After: a, Unknown: unknown, Sensitive: sensitive, Changed: changed})
	})
	return out
}

// walkTree walks before/after in lockstep, alongside terraform's
// parallel afterUnknown/beforeSensitive/afterSensitive marker trees, and
// calls handle once per leaf (see leafFunc). It has no opinion on
// whether unchanged leaves matter — diffValues and fullAttributeValues
// each decide that for themselves in their handle callback.
func walkTree(before, after, afterUnknown, beforeSensitive, afterSensitive any, path AttributePath, handle leafFunc) {
	sensitive := boolMarker(beforeSensitive) || boolMarker(afterSensitive)

	if boolMarker(afterUnknown) {
		// The whole subtree at this path is computed at apply time.
		// Don't recurse into it — we have nothing truthful to compare
		// per-leaf, so it renders as one "(known after apply)" leaf.
		handle(clonePath(path), before, after, true, sensitive)
		return
	}

	beforeMap, beforeIsMap := before.(map[string]any)
	afterMap, afterIsMap := after.(map[string]any)
	auMap, auIsMap := afterUnknown.(map[string]any)
	bsMap, bsIsMap := beforeSensitive.(map[string]any)
	asMap, asIsMap := afterSensitive.(map[string]any)
	if beforeIsMap || afterIsMap || auIsMap || bsIsMap || asIsMap {
		// A computed/unknown leaf is often absent from `after` entirely
		// (its value isn't known yet) and only exists as a key in the
		// afterUnknown marker tree — so the key set must include the
		// marker trees too, not just before/after themselves.
		for _, key := range unionMapKeys(beforeMap, afterMap, auMap, bsMap, asMap) {
			walkTree(
				beforeMap[key], afterMap[key],
				subMapMarker(afterUnknown, key), subMapMarker(beforeSensitive, key), subMapMarker(afterSensitive, key),
				append(path, PathSegment{Key: key}), handle,
			)
		}
		return
	}

	beforeList, beforeIsList := before.([]any)
	afterList, afterIsList := after.([]any)
	auList, auIsList := afterUnknown.([]any)
	bsList, bsIsList := beforeSensitive.([]any)
	asList, asIsList := afterSensitive.([]any)
	if beforeIsList || afterIsList || auIsList || bsIsList || asIsList {
		n := max(len(beforeList), len(afterList), len(auList), len(bsList), len(asList))
		for i := range n {
			var b, a any
			if i < len(beforeList) {
				b = beforeList[i]
			}
			if i < len(afterList) {
				a = afterList[i]
			}
			idx := i
			walkTree(
				b, a,
				subListMarker(afterUnknown, i), subListMarker(beforeSensitive, i), subListMarker(afterSensitive, i),
				append(path, PathSegment{Index: &idx}), handle,
			)
		}
		return
	}

	// Scalar (or nil) leaf.
	handle(clonePath(path), before, after, false, sensitive)
}

func clonePath(path AttributePath) AttributePath {
	out := make(AttributePath, len(path))
	copy(out, path)
	return out
}

func unionMapKeys(maps ...map[string]any) []string {
	seen := make(map[string]struct{})
	var keys []string
	for _, m := range maps {
		for k := range m {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

// boolMarker reports whether a terraform marker-tree node (an
// afterUnknown/beforeSensitive/afterSensitive value) marks its whole
// subtree, i.e. the node itself is `true`.
func boolMarker(marker any) bool {
	b, ok := marker.(bool)
	return ok && b
}

// subMapMarker descends a marker tree by object key. A boolean marker
// propagates down as-is: `true` at a parent means every child is marked.
func subMapMarker(marker any, key string) any {
	switch m := marker.(type) {
	case bool:
		return m
	case map[string]any:
		return m[key]
	default:
		return nil
	}
}

// subListMarker descends a marker tree by list index, with the same
// boolean-propagation rule as subMapMarker.
func subListMarker(marker any, idx int) any {
	switch m := marker.(type) {
	case bool:
		return m
	case []any:
		if idx < len(m) {
			return m[idx]
		}
		return nil
	default:
		return nil
	}
}
