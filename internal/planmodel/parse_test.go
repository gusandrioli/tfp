package planmodel

import (
	"os"
	"path/filepath"
	"testing"
)

func loadFixture(t *testing.T, name string) *Module {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name)) //nolint:gosec // fixed test fixture directory, name is a compile-time constant at every call site
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	mod, err := Parse(data)
	if err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return mod
}

// findResource searches the tree (depth-first) for a resource by address.
func findResource(mod *Module, address string) *Resource {
	for _, r := range mod.Resources {
		if r.Address == address {
			return r
		}
	}
	for _, child := range mod.Children {
		if r := findResource(child, address); r != nil {
			return r
		}
	}
	return nil
}

func findDiff(r *Resource, path string) *AttributeDiff {
	for i := range r.Diffs {
		if r.Diffs[i].Path.String() == path {
			return &r.Diffs[i]
		}
	}
	return nil
}

func TestParse_Create(t *testing.T) {
	root := loadFixture(t, "create.json")

	if got := len(root.Resources); got != 2 {
		t.Fatalf("root resources = %d, want 2", got)
	}
	if got := len(root.Children); got != 1 {
		t.Fatalf("root children = %d, want 1", got)
	}
	child := root.Children[0]
	if got, want := child.Name(), "child"; got != want {
		t.Fatalf("child module name = %q, want %q", got, want)
	}
	if got, want := child.Address(), "module.child"; got != want {
		t.Fatalf("child module address = %q, want %q", got, want)
	}

	pet := findResource(root, "random_pet.this")
	if pet == nil {
		t.Fatal("random_pet.this not found")
	}
	if pet.Kind != ChangeCreate {
		t.Fatalf("random_pet.this kind = %v, want ChangeCreate", pet.Kind)
	}
	// keepers/prefix are null before and after: pruned.
	if d := findDiff(pet, "keepers"); d != nil {
		t.Fatalf("expected keepers to be pruned (nil == nil), got %+v", d)
	}
	// id is computed: absent from `after`, present only as an
	// after_unknown marker — must still surface as a diff.
	id := findDiff(pet, "id")
	if id == nil {
		t.Fatal("expected a diff for computed attribute \"id\"")
	}
	if !id.Unknown {
		t.Fatal("id diff should be marked Unknown")
	}
	if id.Sensitive {
		t.Fatal("id diff should not be marked Sensitive")
	}

	pw := findResource(root, "random_password.this")
	if pw == nil {
		t.Fatal("random_password.this not found")
	}
	if !pw.Sensitive {
		t.Fatal("random_password.this should be marked Sensitive")
	}
	result := findDiff(pw, "result")
	if result == nil {
		t.Fatal("expected a diff for \"result\"")
	}
	if !result.Unknown || !result.Sensitive {
		t.Fatalf("result diff = %+v, want Unknown=true Sensitive=true", *result)
	}

	labeled := findResource(root, "module.child.null_resource.labeled")
	if labeled == nil {
		t.Fatal("module.child.null_resource.labeled not found")
	}
	if got := findDiff(labeled, `triggers["app.kubernetes.io/version"]`); got == nil {
		t.Fatal(`expected a diff at triggers["app.kubernetes.io/version"]`)
	}
}

func TestParse_Update_PrunesUnchangedAndDropsNoOp(t *testing.T) {
	root := loadFixture(t, "update.json")

	// The no-op sibling resource must never reach the tree.
	if r := findResource(root, "module.eks_sp_foundation.module.elastic_agent.kubernetes_role_binding.other"); r != nil {
		t.Fatal("no-op resource should have been dropped from the tree")
	}

	r := findResource(root, "module.eks_sp_foundation.module.elastic_agent.kubernetes_role_binding.elastic_agent")
	if r == nil {
		t.Fatal("kubernetes_role_binding.elastic_agent not found")
	}
	if r.Kind != ChangeUpdate {
		t.Fatalf("kind = %v, want ChangeUpdate", r.Kind)
	}

	changed := findDiff(r, `metadata.labels["app.kubernetes.io/version"]`)
	if changed == nil {
		t.Fatal(`expected a diff at metadata.labels["app.kubernetes.io/version"]`)
	}
	if changed.Before != "9.2.2" || changed.After != "9.2.4" {
		t.Fatalf("version diff = %+v, want 9.2.2 -> 9.2.4", *changed)
	}

	// Unchanged sibling label and the untouched role_ref block must be
	// pruned entirely, not materialized as no-op diffs.
	for _, path := range []string{
		`metadata.labels["app.kubernetes.io/name"]`,
		`metadata.labels["app.kubernetes.io/managed-by"]`,
		"metadata.name",
		"metadata.namespace",
		"role_ref.kind",
		"role_ref.name",
		"id",
	} {
		if d := findDiff(r, path); d != nil {
			t.Fatalf("expected %s to be pruned (unchanged), got %+v", path, *d)
		}
	}

	if got, want := len(r.Diffs), 1; got != want {
		t.Fatalf("len(Diffs) = %d, want %d: %+v", got, want, r.Diffs)
	}
}

func TestParse_Replace_ModuleAddressAndMapChurn(t *testing.T) {
	root := loadFixture(t, "replace.json")

	// random_pet.this and random_password.this are no-op in this
	// fixture and must not appear.
	if r := findResource(root, "random_pet.this"); r != nil {
		t.Fatal("no-op random_pet.this should have been dropped")
	}

	r := findResource(root, "module.child.null_resource.labeled")
	if r == nil {
		t.Fatal("module.child.null_resource.labeled not found")
	}
	if r.Kind != ChangeReplace {
		t.Fatalf("kind = %v, want ChangeReplace", r.Kind)
	}

	changed := findDiff(r, `triggers["app.kubernetes.io/version"]`)
	if changed == nil || changed.Before != "9.2.2" || changed.After != "9.2.4" {
		t.Fatalf("version diff = %+v, want 9.2.2 -> 9.2.4", changed)
	}
	// "region" is a plain identifier, so it renders dotted rather than
	// bracket-quoted (bracket-quoting is reserved for keys that aren't
	// valid identifiers, like the label key above).
	if d := findDiff(r, `triggers.region`); d == nil || d.Before != nil || d.After != "us-east-1" {
		t.Fatalf(`triggers.region diff = %+v, want nil -> "us-east-1"`, d)
	}
	if d := findDiff(r, `triggers.team`); d == nil || d.Before != "platform" || d.After != nil {
		t.Fatalf(`triggers.team diff = %+v, want "platform" -> nil`, d)
	}

	// terraform reported replace_paths: [["triggers"]] for this resource
	// (triggers is a ForceNew attribute on null_resource) — every diff
	// under that path should be flagged as the reason for replacement,
	// so the UI can show it.
	for _, path := range []string{`triggers["app.kubernetes.io/version"]`, "triggers.region", "triggers.team"} {
		d := findDiff(r, path)
		if d == nil {
			t.Fatalf("expected a diff at %s", path)
		}
		if !d.ForcesReplacement {
			t.Errorf("%s: ForcesReplacement = false, want true", path)
		}
	}
	// "id" changing is a side effect of replacement, not itself the
	// cause — it must not be flagged.
	if d := findDiff(r, "id"); d == nil || d.ForcesReplacement {
		t.Errorf(`"id" diff = %+v, want ForcesReplacement = false`, d)
	}
	// The unrelated "app.kubernetes.io/name" trigger key is unchanged
	// and must be pruned.
	if d := findDiff(r, `triggers["app.kubernetes.io/name"]`); d != nil {
		t.Fatalf("expected triggers[\"app.kubernetes.io/name\"] to be pruned, got %+v", *d)
	}
}

func TestParse_Delete(t *testing.T) {
	root := loadFixture(t, "delete.json")

	for _, addr := range []string{"random_pet.this", "random_password.this", "module.child.null_resource.labeled"} {
		r := findResource(root, addr)
		if r == nil {
			t.Fatalf("%s not found", addr)
		}
		if r.Kind != ChangeDelete {
			t.Fatalf("%s kind = %v, want ChangeDelete", addr, r.Kind)
		}
		if len(r.Diffs) == 0 {
			t.Fatalf("%s expected non-empty Diffs (full before body removed)", addr)
		}
	}
}
