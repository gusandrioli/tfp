package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "internal", "planmodel", "testdata", name)
	data, err := os.ReadFile(path) //nolint:gosec // fixed test fixture path
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestRenderPlan_TextWithChanges(t *testing.T) {
	cmd := newShowCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)

	err := renderPlan(cmd, fixture(t, "update.json"), "text")

	var changes changesPresentSignal
	if !errors.As(err, &changes) {
		t.Fatalf("expected a changesPresentSignal, got %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("will be updated in-place")) {
		t.Errorf("expected text output, got %q", out.String())
	}
}

func TestRenderPlan_JSONWithChanges(t *testing.T) {
	cmd := newShowCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)

	err := renderPlan(cmd, fixture(t, "create.json"), "json")

	var changes changesPresentSignal
	if !errors.As(err, &changes) {
		t.Fatalf("expected a changesPresentSignal, got %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"total"`)) {
		t.Errorf("expected JSON output, got %q", out.String())
	}
}

func TestRenderPlan_NoChanges(t *testing.T) {
	noChanges := []byte(`{"format_version": "1.2", "resource_changes": [
		{"address": "random_pet.this", "type": "random_pet", "name": "this",
		 "change": {"actions": ["no-op"], "before": {}, "after": {}}}
	]}`)

	cmd := newShowCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)

	err := renderPlan(cmd, noChanges, "text")
	if err != nil {
		t.Fatalf("expected no error for a no-op-only plan, got %v", err)
	}
}

func TestRenderPlan_InvalidOutputFlag(t *testing.T) {
	cmd := newShowCmd()
	err := renderPlan(cmd, fixture(t, "create.json"), "bogus")

	if err == nil {
		t.Fatal("expected an error for an invalid --output value")
	}
	if got := exitCodeForError(err); got != 2 {
		t.Errorf("exitCodeForError = %d, want 2 (usage error)", got)
	}
}

func TestLoadPlanJSON_Stdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	want := fixture(t, "create.json")
	go func() {
		_, _ = w.Write(want)
		_ = w.Close()
	}()

	got, err := loadPlanJSON(context.Background(), "-")
	if err != nil {
		t.Fatalf("loadPlanJSON: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Error("loadPlanJSON(\"-\") did not return stdin's contents unchanged")
	}
}

func TestLoadPlanJSON_FileAlreadyJSON(t *testing.T) {
	path := filepath.Join("..", "..", "internal", "planmodel", "testdata", "create.json")
	got, err := loadPlanJSON(context.Background(), path)
	if err != nil {
		t.Fatalf("loadPlanJSON: %v", err)
	}
	want := fixture(t, "create.json")
	if !bytes.Equal(got, want) {
		t.Error("loadPlanJSON on an already-JSON file should return it unchanged, without shelling out")
	}
}
