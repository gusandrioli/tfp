package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/gusandrioli/tfp/internal/tfrunner"
)

type planFlags struct {
	output  string
	chdir   string
	timeout time.Duration
}

func bindPlanFlags(cmd *cobra.Command) *planFlags {
	f := &planFlags{output: "auto", timeout: 30 * time.Minute}
	cmd.Flags().StringVar(&f.output, "output", f.output, "auto, text, or json")
	cmd.Flags().StringVarP(&f.chdir, "chdir", "C", "", "directory to run terraform in")
	cmd.Flags().DurationVar(&f.timeout, "timeout", f.timeout, "max time to wait for the terraform subprocess")
	return f
}

func newPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan [-- terraform-flags...]",
		Short: "Run terraform plan and review the result",
	}
	f := bindPlanFlags(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runPlan(cmd, args, f)
	}
	return cmd
}

func runPlan(cmd *cobra.Command, extraArgs []string, f *planFlags) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), f.timeout)
	defer cancel()

	result, err := tfrunner.Plan(ctx, tfrunner.Options{
		Dir:       f.chdir,
		ExtraArgs: extraArgs,
		Stderr:    cmd.ErrOrStderr(),
	})
	if err != nil {
		return err
	}
	if !result.ChangesPresent {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No changes. Infrastructure matches the configuration.")
		return err
	}
	return renderPlan(cmd, result.JSON, f.output)
}
