package render_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

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
			if err := render.Text(&buf, root, rep); err != nil {
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
			if err := render.JSON(&buf, root, rep); err != nil {
				t.Fatalf("JSON: %v", err)
			}
			checkGolden(t, fixture+".json.golden", buf.Bytes())
		})
	}
}
