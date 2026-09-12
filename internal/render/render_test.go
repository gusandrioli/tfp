package render_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gusandrioli/tfp/internal/filter"
	"github.com/gusandrioli/tfp/internal/planmodel"
	"github.com/gusandrioli/tfp/internal/render"
	"github.com/gusandrioli/tfp/internal/summary"
)

var update = flag.Bool("update", false, "update golden files")

func loadFixture(t *testing.T, name string) *planmodel.Module {
	t.Helper()
	path := filepath.Join("..", "planmodel", "testdata", name)
	data, err := os.ReadFile(path) //nolint:gosec // fixed test fixture directory
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	mod, err := planmodel.Parse(data)
	if err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return mod
}

func checkGolden(t *testing.T, goldenName string, got []byte) {
	t.Helper()
	goldenPath := filepath.Join("testdata", goldenName)
	if *update {
		if err := os.WriteFile(goldenPath, got, 0o600); err != nil {
			t.Fatalf("write golden %s: %v", goldenName, err)
		}
		return
	}
	want, err := os.ReadFile(goldenPath) //nolint:gosec // fixed test fixture directory
	if err != nil {
		t.Fatalf("read golden %s: %v (run `go test ./internal/render/... -update` to create it)", goldenName, err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("output does not match %s\n--- want ---\n%s\n--- got ---\n%s", goldenName, want, got)
	}
}

func TestText(t *testing.T) {
	for _, fixture := range []string{"create", "update", "replace", "delete"} {
		t.Run(fixture, func(t *testing.T) {
			root := loadFixture(t, fixture+".json")
			rep := summary.Build(root)

			var buf bytes.Buffer
			if err := render.Text(&buf, root, rep, nil); err != nil {
				t.Fatalf("Text: %v", err)
			}
			checkGolden(t, fixture+".text.golden", buf.Bytes())
		})
	}
}

func TestJSON(t *testing.T) {
	for _, fixture := range []string{"create", "update", "replace", "delete"} {
		t.Run(fixture, func(t *testing.T) {
			root := loadFixture(t, fixture+".json")
			rep := summary.Build(root)

			var buf bytes.Buffer
			if err := render.JSON(&buf, root, rep, nil); err != nil {
				t.Fatalf("JSON: %v", err)
			}
			checkGolden(t, fixture+".json.golden", buf.Bytes())
		})
	}
}

func TestText_Filtering(t *testing.T) {
	root := loadFixture(t, "update.json")
	rep := summary.Build(root)

	var s filter.Set
	s.Add(filter.Rule{
		Scope:      filter.ScopeGlobal,
		PathSuffix: planmodel.AttributePath{{Key: "labels"}, {Key: "app.kubernetes.io/version"}},
	})

	var buf bytes.Buffer
	if err := render.Text(&buf, root, rep, &s); err != nil {
		t.Fatalf("Text: %v", err)
	}
	out := buf.String()

	if strings.Contains(out, `app.kubernetes.io/version`) {
		t.Errorf("expected the filtered attribute to be hidden from output, got:\n%s", out)
	}
	if !strings.Contains(out, "1 change(s) filtered") {
		t.Errorf("expected a filtered-count marker in the resource block, got:\n%s", out)
	}
	// Filtering must never change the summary line: it's built from the
	// unfiltered tree and must stay truthful about the real plan.
	if !strings.Contains(out, "Plan: 0 to add, 1 to change, 0 to destroy, 0 to replace, 0 to forget.") {
		t.Errorf("expected the summary line to be unaffected by filtering, got:\n%s", out)
	}
}

func TestJSON_Filtering(t *testing.T) {
	root := loadFixture(t, "update.json")
	rep := summary.Build(root)

	var s filter.Set
	s.Add(filter.Rule{
		Scope:      filter.ScopeGlobal,
		PathSuffix: planmodel.AttributePath{{Key: "labels"}, {Key: "app.kubernetes.io/version"}},
	})

	var buf bytes.Buffer
	if err := render.JSON(&buf, root, rep, &s); err != nil {
		t.Fatalf("JSON: %v", err)
	}

	var decoded struct {
		Total   summary.Counts `json:"total"`
		Modules struct {
			Children []struct {
				Children []struct {
					Resources []struct {
						Diffs         []any `json:"diffs"`
						DiffsFiltered int   `json:"diffs_filtered"`
					} `json:"resources"`
				} `json:"children"`
			} `json:"children"`
		} `json:"modules"`
	}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if decoded.Total != rep.Total {
		t.Errorf("Total = %+v, want %+v (must be unaffected by filtering)", decoded.Total, rep.Total)
	}

	res := decoded.Modules.Children[0].Children[0].Resources[0]
	if len(res.Diffs) != 0 {
		t.Errorf("expected no visible diffs, got %+v", res.Diffs)
	}
	if res.DiffsFiltered != 1 {
		t.Errorf("DiffsFiltered = %d, want 1", res.DiffsFiltered)
	}
}
