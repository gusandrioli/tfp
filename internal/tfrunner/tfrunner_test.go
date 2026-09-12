package tfrunner_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gusandrioli/tfp/internal/tfrunner"
)

func fakeOptions(t *testing.T, stderr *bytes.Buffer) tfrunner.Options {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	opts := tfrunner.Options{Binary: filepath.Join(wd, "testdata", "fake-terraform")}
	// A nil *bytes.Buffer assigned into the io.Writer field would produce
	// a non-nil interface wrapping a nil pointer, which Options.stderr()'s
	// nil check can't see through — leave the field at its true zero
	// value instead so it falls back to io.Discard.
	if stderr != nil {
		opts.Stderr = stderr
	}
	return opts
}

func TestPlan_ChangesPresent(t *testing.T) {
	fixture := filepath.Join("..", "planmodel", "testdata", "create.json")
	want, err := os.ReadFile(fixture) //nolint:gosec // fixed test fixture path
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("FAKE_TF_PLAN_EXIT", "2")
	t.Setenv("FAKE_TF_SHOW_JSON", mustAbs(t, fixture))

	var stderr bytes.Buffer
	result, err := tfrunner.Plan(context.Background(), fakeOptions(t, &stderr))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !result.ChangesPresent {
		t.Error("expected ChangesPresent = true")
	}
	if !bytes.Equal(result.JSON, want) {
		t.Errorf("JSON = %q, want %q", result.JSON, want)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("fake plan stderr line")) {
		t.Errorf("expected terraform's stderr to be streamed through, got %q", stderr.String())
	}
	if bytes.Contains(stderr.Bytes(), []byte("fake plan progress")) {
		t.Error("terraform plan's stdout (the noisy human-readable text) must never be streamed")
	}
}

func TestPlan_NoChanges(t *testing.T) {
	t.Setenv("FAKE_TF_PLAN_EXIT", "0")
	// Deliberately no FAKE_TF_SHOW_JSON: if Plan tried to call `show`
	// when there are no changes, the fake would fail loudly.

	result, err := tfrunner.Plan(context.Background(), fakeOptions(t, nil))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if result.ChangesPresent {
		t.Error("expected ChangesPresent = false")
	}
	if result.JSON != nil {
		t.Errorf("expected nil JSON when there are no changes, got %q", result.JSON)
	}
}

func TestPlan_Error(t *testing.T) {
	t.Setenv("FAKE_TF_PLAN_EXIT", "1")

	_, err := tfrunner.Plan(context.Background(), fakeOptions(t, nil))
	if err == nil {
		t.Fatal("expected an error for a non-0/2 exit code")
	}
}

func TestShow_Error(t *testing.T) {
	t.Setenv("FAKE_TF_SHOW_FAIL", "1")

	_, err := tfrunner.Show(context.Background(), fakeOptions(t, nil), "irrelevant.tfplan")
	if err == nil {
		t.Fatal("expected an error when terraform show fails")
	}
}

func TestPlan_BinaryNotFound(t *testing.T) {
	opts := tfrunner.Options{Binary: filepath.Join(t.TempDir(), "does-not-exist")}
	_, err := tfrunner.Plan(context.Background(), opts)
	if err == nil {
		t.Fatal("expected an error when the terraform binary doesn't exist")
	}
	var exitErr interface{ ExitCode() int }
	if errors.As(err, &exitErr) {
		t.Fatal("a missing binary should not be reported as a process exit code")
	}
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
