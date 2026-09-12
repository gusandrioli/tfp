package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "tfp",
		Short:         "tfp makes terraform plan output easy to read and review",
		Long:          "tfp turns `terraform plan` output into a navigable, filterable view: grouped by module, summarized by change count, with noisy repeated changes filterable out.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	planCmd := newPlanCmd()
	cmd.AddCommand(planCmd)
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newVersionCmd())

	// `tfp` with no subcommand behaves exactly like `tfp plan`: same
	// flags (shared backing variables via AddFlagSet, not copies) and
	// the same RunE.
	cmd.RunE = planCmd.RunE
	cmd.Flags().AddFlagSet(planCmd.Flags())

	return cmd
}

// Execute runs the root command and returns the process exit code.
func Execute() int {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		if _, ok := errors.AsType[changesPresentSignal](err); ok {
			return 3
		}
		fmt.Fprintln(os.Stderr, "tfp:", err)
		return exitCodeForError(err)
	}
	return 0
}
