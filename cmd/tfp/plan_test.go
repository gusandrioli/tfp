package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// withFakeTerraform prepends internal/tfrunner/testdata (containing the
// fake-terraform shim used by tfrunner's own tests) to PATH, so
// tfrunner's default Binary ("terraform", resolved via PATH) resolves
// to it instead of a real terraform install.
func withFakeTerraform(t *testing.T) {
	t.Helper()
	shimDir, err := filepath.Abs(filepath.Join("..", "..", "internal", "tfrunner", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	linkDir := t.TempDir()
	if err := os.Symlink(filepath.Join(shimDir, "fake-terraform"), filepath.Join(linkDir, "terraform")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", linkDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestRunPlan_ChangesPresent(t *testing.T) {
	withFakeTerraform(t)
	t.Setenv("FAKE_TF_PLAN_EXIT", "2")
	t.Setenv("FAKE_TF_SHOW_JSON", mustAbsFixture(t, "create.json"))

	cmd := newPlanCmd()
	cmd.SetContext(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	f := &planFlags{output: "text", timeout: 10 * time.Second}

	err := runPlan(cmd, nil, f)

	var changes changesPresentSignal
	if !errors.As(err, &changes) {
		t.Fatalf("expected a changesPresentSignal, got %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("Plan:")) {
		t.Errorf("expected rendered plan output, got %q", out.String())
	}
}

func TestRunPlan_NoChanges(t *testing.T) {
	withFakeTerraform(t)
	t.Setenv("FAKE_TF_PLAN_EXIT", "0")

	cmd := newPlanCmd()
	cmd.SetContext(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	f := &planFlags{output: "text", timeout: 10 * time.Second}

	if err := runPlan(cmd, nil, f); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("No changes")) {
		t.Errorf("expected the no-changes message, got %q", out.String())
	}
}

func TestRunPlan_TerraformError(t *testing.T) {
	withFakeTerraform(t)
	t.Setenv("FAKE_TF_PLAN_EXIT", "1")

	cmd := newPlanCmd()
	cmd.SetContext(context.Background())
	cmd.SetErr(&bytes.Buffer{})
	f := &planFlags{output: "text", timeout: 10 * time.Second}

	err := runPlan(cmd, nil, f)
	if err == nil {
		t.Fatal("expected an error when terraform plan fails")
	}
	if got := exitCodeForError(err); got != 1 {
		t.Errorf("exitCodeForError = %d, want 1", got)
	}
}

func mustAbsFixture(t *testing.T, name string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "internal", "planmodel", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
