package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/gusandrioli/tfp/internal/planmodel"
	"github.com/gusandrioli/tfp/internal/render"
	"github.com/gusandrioli/tfp/internal/summary"
	"github.com/gusandrioli/tfp/internal/tfrunner"
	"github.com/gusandrioli/tfp/internal/ui"
)

// changesPresentSignal is returned (and specifically never printed as an
// error — see Execute in exitcode.go) when a non-interactive render
// (--output text|json, or --output auto against a non-terminal)
// completed successfully and the plan has changes. It's how `tfp show
// plan.json --output text` lets a script branch on exit code 3 without
// parsing output. See docs/CLI.md.
type changesPresentSignal struct{}

func (changesPresentSignal) Error() string { return "" }

// loadPlanJSON reads path (or stdin, for "-") and returns a `terraform
// show -json` payload: as-is if it's already JSON, or converted via a
// `terraform show -json` subprocess if it looks like a binary plan file.
func loadPlanJSON(ctx context.Context, path string) ([]byte, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path) //nolint:gosec // path is a user-supplied CLI argument, same trust level as any file the user names on their own command line
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	if json.Valid(bytes.TrimSpace(data)) {
		return data, nil
	}
	return tfrunner.Show(ctx, tfrunner.Options{}, path)
}

// renderPlan parses data and shows it: the interactive TUI when
// format is "auto" and stdout is a terminal, otherwise a static
// text/JSON render to cmd.OutOrStdout().
func renderPlan(cmd *cobra.Command, data []byte, format string) error {
	root, err := planmodel.Parse(data)
	if err != nil {
		return err
	}
	rep := summary.Build(root)

	if format == "auto" {
		if term.IsTerminal(int(os.Stdout.Fd())) { //nolint:gosec // standard idiom for x/term; a file descriptor never approaches the int/uintptr overflow range
			return ui.Run(root, rep)
		}
		format = "text"
	}

	switch format {
	case "text":
		err = render.Text(cmd.OutOrStdout(), root, rep, nil)
	case "json":
		err = render.JSON(cmd.OutOrStdout(), root, rep, nil)
	default:
		return newUsageError(fmt.Errorf("invalid --output %q (want auto, text, or json)", format))
	}
	if err != nil {
		return err
	}

	if rep.Total.Total() > 0 {
		return changesPresentSignal{}
	}
	return nil
}
