package main

import (
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

	cmd.AddCommand(newPlanCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newVersionCmd())

	return cmd
}

// Execute runs the root command and returns the process exit code.
func Execute() int {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "tfp:", err)
		return exitCodeForError(err)
	}
	return 0
}
