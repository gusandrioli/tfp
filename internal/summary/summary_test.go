package summary_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gusandrioli/tfp/internal/planmodel"
	"github.com/gusandrioli/tfp/internal/summary"
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

func TestBuild_Create(t *testing.T) {
	root := loadFixture(t, "create.json")
	r := summary.Build(root)

	want := summary.Counts{Create: 3}
	if r.Total != want {
		t.Fatalf("Total = %+v, want %+v", r.Total, want)
	}
	// Module rollups are cumulative (own resources + all descendants),
	// so the root's rollup equals the grand total.
	if got := r.ByModule[""]; got != want {
		t.Fatalf("ByModule[root] = %+v, want %+v", got, want)
	}
	if got := r.ByModule["module.child"]; got != (summary.Counts{Create: 1}) {
		t.Fatalf("ByModule[module.child] = %+v, want {Create: 1}", got)
	}

	entry, ok := r.ByResource["random_pet.this"]
	if !ok {
		t.Fatal("expected an entry for random_pet.this")
	}
	if entry.Kind != planmodel.ChangeCreate {
		t.Fatalf("random_pet.this kind = %v, want ChangeCreate", entry.Kind)
	}
	if entry.DiffCount == 0 {
		t.Fatal("expected random_pet.this to carry at least one diff")
	}
}

func TestBuild_Replace(t *testing.T) {
	root := loadFixture(t, "replace.json")
	r := summary.Build(root)

	// random_pet.this / random_password.this are no-op in this fixture
	// and must not appear anywhere in the report.
	if _, ok := r.ByResource["random_pet.this"]; ok {
		t.Fatal("no-op resource should not appear in ByResource")
	}

	want := summary.Counts{Replace: 1}
	if r.Total != want {
		t.Fatalf("Total = %+v, want %+v", r.Total, want)
	}
	if got := r.ByModule["module.child"]; got != want {
		t.Fatalf("ByModule[module.child] = %+v, want %+v", got, want)
	}
	// The root module has no changes of its own in this fixture, but
	// its rollup still includes module.child's replace.
	if got := r.ByModule[""]; got != want {
		t.Fatalf("ByModule[root] = %+v, want %+v", got, want)
	}
}

func TestBuild_Delete(t *testing.T) {
	root := loadFixture(t, "delete.json")
	r := summary.Build(root)

	want := summary.Counts{Delete: 3}
	if r.Total != want {
		t.Fatalf("Total = %+v, want %+v", r.Total, want)
	}
	if got := r.Total.Total(); got != 3 {
		t.Fatalf("Total.Total() = %d, want 3", got)
	}
}

func TestBuild_Update(t *testing.T) {
	root := loadFixture(t, "update.json")
	r := summary.Build(root)

	want := summary.Counts{Update: 1}
	if r.Total != want {
		t.Fatalf("Total = %+v, want %+v", r.Total, want)
	}

	entry := r.ByResource["module.eks_sp_foundation.module.elastic_agent.kubernetes_role_binding.elastic_agent"]
	if entry.DiffCount != 1 {
		t.Fatalf("DiffCount = %d, want 1", entry.DiffCount)
	}
}
