package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gusandrioli/tfp/internal/planmodel"
	"github.com/gusandrioli/tfp/internal/summary"
	"github.com/gusandrioli/tfp/internal/ui"
)

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

func TestFlatten_AllExpanded(t *testing.T) {
	root := loadFixture(t, "create.json")
	rep := summary.Build(root)

	rows := ui.Flatten(root, map[string]bool{}, rep)

	// 2 root resources + 1 module header + 1 resource under it.
	if got, want := len(rows), 4; got != want {
		t.Fatalf("len(rows) = %d, want %d: %+v", got, want, rows)
	}
	if rows[0].Kind != ui.RowResource || rows[0].Depth != 0 {
		t.Errorf("rows[0] = %+v, want a depth-0 resource row", rows[0])
	}
	if rows[2].Kind != ui.RowModule || rows[2].Module.Address() != "module.child" {
		t.Fatalf("rows[2] = %+v, want the module.child header", rows[2])
	}
	if rows[2].Counts.Create != 1 {
		t.Errorf("rows[2].Counts = %+v, want Create: 1", rows[2].Counts)
	}
	if rows[3].Kind != ui.RowResource || rows[3].Depth != 1 {
		t.Errorf("rows[3] = %+v, want a depth-1 resource row under module.child", rows[3])
	}
}

func TestFlatten_Collapsed(t *testing.T) {
	root := loadFixture(t, "create.json")
	rep := summary.Build(root)

	rows := ui.Flatten(root, map[string]bool{"module.child": true}, rep)

	// The collapsed module's resource must not appear, but the header
	// (with its rollup badge) still must.
	if got, want := len(rows), 3; got != want {
		t.Fatalf("len(rows) = %d, want %d: %+v", got, want, rows)
	}
	last := rows[len(rows)-1]
	if last.Kind != ui.RowModule || last.Counts.Create != 1 {
		t.Fatalf("last row = %+v, want the collapsed module.child header still carrying its rollup", last)
	}
}

func TestFlatten_NestedModuleDepth(t *testing.T) {
	// module.eks_sp_foundation.module.elastic_agent is a two-level nest.
	root := loadFixture(t, "update.json")
	rep := summary.Build(root)

	rows := ui.Flatten(root, map[string]bool{}, rep)

	var depths []int
	for _, r := range rows {
		depths = append(depths, r.Depth)
	}
	// module.eks_sp_foundation (0) -> module.elastic_agent (1) -> resource (2)
	want := []int{0, 1, 2}
	if len(depths) != len(want) {
		t.Fatalf("depths = %v, want %v", depths, want)
	}
	for i := range want {
		if depths[i] != want[i] {
			t.Errorf("depths[%d] = %d, want %d (full: %v)", i, depths[i], want[i], depths)
		}
	}
}

func TestDefaultCollapsed(t *testing.T) {
	root := loadFixture(t, "update.json")
	collapsed := ui.DefaultCollapsed(root)

	if collapsed["module.eks_sp_foundation"] {
		t.Error("top-level module should be open by default")
	}
	if !collapsed["module.eks_sp_foundation.module.elastic_agent"] {
		t.Error("nested module should be collapsed by default")
	}
}
