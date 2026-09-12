// Package tfrunner is the only part of tfp that shells out to terraform.
// It runs `terraform plan`/`terraform show -json` as subprocesses and
// hands back raw JSON bytes — it has no notion of tfp's domain model,
// that's internal/planmodel's job once the bytes come back.
package tfrunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Options configures how terraform is invoked. The zero value runs
// "terraform" against the current directory with output discarded.
type Options struct {
	// Dir is the working directory terraform runs in. Empty means the
	// current directory (matching exec.Cmd's own default).
	Dir string
	// Binary is the terraform executable to run. Empty means
	// "terraform", resolved via PATH. Overridable so tests can point it
	// at a fake shim instead of a real terraform install.
	Binary string
	// ExtraArgs is passed through verbatim to `terraform plan`, after
	// tfp's own flags (-out, -detailed-exitcode, -input=false).
	ExtraArgs []string
	// Stderr receives terraform's own live stderr (progress/errors).
	// Nil discards it. terraform plan's stdout (the noisy human-readable
	// plan text tfp exists to replace) is always captured, never shown.
	Stderr io.Writer
}

func (o Options) binary() string {
	if o.Binary != "" {
		return o.Binary
	}
	return "terraform"
}

func (o Options) stderr() io.Writer {
	if o.Stderr != nil {
		return o.Stderr
	}
	return io.Discard
}

// Result is the outcome of a Plan run.
type Result struct {
	// JSON is the `terraform show -json` payload. Nil when ChangesPresent
	// is false — there's nothing to show, so Plan skips that subprocess.
	JSON []byte
	// ChangesPresent reports terraform's own -detailed-exitcode verdict.
	ChangesPresent bool
}

// Plan runs `terraform plan -out=<tmp> -detailed-exitcode` in opts.Dir,
// then (only if the plan has changes) `terraform show -json <tmp>`,
// cleaning up the temporary plan file on every return path.
func Plan(ctx context.Context, opts Options) (Result, error) {
	tmp, err := os.CreateTemp("", "tfp-*.tfplan")
	if err != nil {
		return Result{}, fmt.Errorf("tfrunner: create temp plan file: %w", err)
	}
	tmpPath := tmp.Name()
	closeErr := tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()
	if closeErr != nil {
		return Result{}, fmt.Errorf("tfrunner: close temp plan file: %w", closeErr)
	}

	args := append([]string{"plan", "-out=" + tmpPath, "-detailed-exitcode", "-input=false"}, opts.ExtraArgs...)
	changesPresent, err := runPlan(ctx, opts, args)
	if err != nil {
		return Result{}, err
	}
	if !changesPresent {
		return Result{ChangesPresent: false}, nil
	}

	data, err := Show(ctx, opts, tmpPath)
	if err != nil {
		return Result{}, err
	}
	return Result{JSON: data, ChangesPresent: true}, nil
}

// Show runs `terraform show -json <path>` in opts.Dir against an
// existing (binary) plan file and returns its stdout.
func Show(ctx context.Context, opts Options, planPath string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, opts.binary(), "show", "-json", planPath) //nolint:gosec // opts.binary() and planPath are operator-controlled (CLI flags / our own temp file), same trust level as running terraform by hand
	cmd.Dir = opts.Dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = opts.stderr()
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("tfrunner: terraform show -json: %w", err)
	}
	return stdout.Bytes(), nil
}

// runPlan runs `terraform plan` with -detailed-exitcode and interprets
// the result: exit 0 means no changes, exit 2 means changes are present,
// anything else (including a non-ExitError failure, e.g. the binary
// isn't found) is a real error.
func runPlan(ctx context.Context, opts Options, args []string) (changesPresent bool, err error) {
	cmd := exec.CommandContext(ctx, opts.binary(), args...) //nolint:gosec // opts.binary()/args are operator-controlled (CLI flags), same trust level as running terraform by hand
	cmd.Dir = opts.Dir
	// terraform plan's human-readable progress goes to stdout — that's
	// exactly the noisy output tfp exists to replace, so it's discarded
	// rather than shown live. Only stderr (real errors/warnings) streams.
	cmd.Stdout = io.Discard
	cmd.Stderr = opts.stderr()

	runErr := cmd.Run()
	if runErr == nil {
		return false, nil
	}

	if exitErr, ok := errors.AsType[*exec.ExitError](runErr); ok && exitErr.ExitCode() == 2 {
		return true, nil
	}
	return false, fmt.Errorf("tfrunner: terraform plan: %w", runErr)
}
