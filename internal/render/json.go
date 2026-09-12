package render

import (
	"encoding/json"
	"io"

	"github.com/gusandrioli/tfp/internal/filter"
	"github.com/gusandrioli/tfp/internal/planmodel"
	"github.com/gusandrioli/tfp/internal/summary"
)

// jsonDiff mirrors planmodel.AttributeDiff with a rendered Path, so it
// serializes as a readable string rather than the internal segment slice.
type jsonDiff struct {
	Path              string `json:"path"`
	Before            any    `json:"before,omitempty"`
	After             any    `json:"after,omitempty"`
	Unknown           bool   `json:"unknown,omitempty"`
	Sensitive         bool   `json:"sensitive,omitempty"`
	ForcesReplacement bool   `json:"forces_replacement,omitempty"`
}

type jsonResource struct {
	Address       string     `json:"address"`
	Type          string     `json:"type"`
	Name          string     `json:"name"`
	Kind          string     `json:"kind"`
	Diffs         []jsonDiff `json:"diffs,omitempty"`
	DiffsFiltered int        `json:"diffs_filtered,omitempty"`
}

type jsonModule struct {
	Address   string         `json:"address"`
	Resources []jsonResource `json:"resources,omitempty"`
	Children  []jsonModule   `json:"children,omitempty"`
}

type jsonReport struct {
	Total    summary.Counts            `json:"total"`
	ByModule map[string]summary.Counts `json:"by_module"`
	Modules  jsonModule                `json:"modules"`
}

// JSON writes a machine-readable rendering of root and rep to w. filters
// may be nil for an unfiltered view; as with Text, filtering only hides
// diffs from the output, never rep's counts.
func JSON(w io.Writer, root *planmodel.Module, rep summary.Report, filters *filter.Set) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(jsonReport{
		Total:    rep.Total,
		ByModule: rep.ByModule,
		Modules:  toJSONModule(root, filters),
	})
}

func toJSONModule(m *planmodel.Module, filters *filter.Set) jsonModule {
	out := jsonModule{Address: m.Address()}
	for _, r := range m.Resources {
		out.Resources = append(out.Resources, toJSONResource(r, filters))
	}
	for _, c := range m.Children {
		out.Children = append(out.Children, toJSONModule(c, filters))
	}
	return out
}

func toJSONResource(r *planmodel.Resource, filters *filter.Set) jsonResource {
	out := jsonResource{
		Address: r.Address,
		Type:    r.Type,
		Name:    r.Name,
		Kind:    r.Kind.String(),
	}
	for _, d := range r.Diffs {
		if filters.Hides(r.Type, d) {
			out.DiffsFiltered++
			continue
		}
		out.Diffs = append(out.Diffs, jsonDiff{
			Path:              d.Path.String(),
			Before:            d.Before,
			After:             d.After,
			Unknown:           d.Unknown,
			Sensitive:         d.Sensitive,
			ForcesReplacement: d.ForcesReplacement,
		})
	}
	return out
}
